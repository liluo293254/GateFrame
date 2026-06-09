use std::time::Duration;

use axum::{
    body::Body,
    extract::State,
    http::Request,
    middleware::Next,
    response::Response,
};
use http_body_util::BodyExt;
use serde::Serialize;
use tracing::{error, info};

use crate::state::AppState;
use crate::tenant::TenantContext;

#[derive(Serialize, Clone)]
pub struct AuditPayload {
    pub tenant_id: String,
    pub user_id: String,
    pub action: String,
    pub path: String,
    pub status_code: u16,
}

/// Logs mutating requests and persists to audit-service asynchronously.
pub async fn audit_middleware(
    State(state): State<AppState>,
    req: Request<Body>,
    next: Next,
) -> Response {
    let method = req.method().clone();
    let path = req.uri().path().to_string();
    let tenant_ctx = req.extensions().get::<TenantContext>().cloned();
    let response = next.run(req).await;
    if should_audit(&method, &path) {
        emit_audit(
            &state,
            AuditPayload {
                tenant_id: tenant_ctx
                    .as_ref()
                    .map(|c| c.tenant_id.clone())
                    .unwrap_or_default(),
                user_id: tenant_ctx
                    .as_ref()
                    .map(|c| c.user_id.clone())
                    .unwrap_or_default(),
                action: method.to_string(),
                path,
                status_code: response.status().as_u16(),
            },
        );
    }
    response
}

/// Audits POST /api/auth/login outcomes (success and failure).
pub async fn login_audit_middleware(
    State(state): State<AppState>,
    req: Request<Body>,
    next: Next,
) -> Response {
    let response = next.run(req).await;
    let status = response.status().as_u16();
    let (parts, body) = response.into_parts();
    let bytes = body
        .collect()
        .await
        .map(|collected| collected.to_bytes())
        .unwrap_or_default();

    let mut tenant_id = String::new();
    let mut user_id = String::new();
    if status == 200 {
        if let Ok(json) = serde_json::from_slice::<serde_json::Value>(&bytes) {
            if let Some(data) = json.get("data") {
                tenant_id = data
                    .get("tenant_id")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                user_id = data
                    .get("user_id")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
            }
        }
    }

    emit_audit(
        &state,
        AuditPayload {
            tenant_id,
            user_id,
            action: "POST".into(),
            path: "/api/auth/login".into(),
            status_code: status,
        },
    );

    Response::from_parts(parts, Body::from(bytes))
}

fn emit_audit(state: &AppState, payload: AuditPayload) {
    info!(
        target: "audit",
        action = %payload.action,
        path = %payload.path,
        tenant_id = %payload.tenant_id,
        user_id = %payload.user_id,
        status = payload.status_code,
        "audit_event"
    );
    persist_event(state, payload);
}

fn persist_event(state: &AppState, payload: AuditPayload) {
    let client = state.http_client.clone();
    let base = state.config.audit_service_url.clone();
    let token = state.config.internal_token.clone();
    let metrics = state.metrics.clone();
    tokio::spawn(async move {
        let url = format!("{}/internal/v1/events", base.trim_end_matches('/'));
        let mut last_err = None;
        for attempt in 0..3u32 {
            match client
                .post(&url)
                .header("X-Internal-Token", token.clone())
                .json(&payload)
                .send()
                .await
            {
                Ok(resp) if resp.status().is_success() => return,
                Ok(resp) => {
                    last_err = Some(format!("status {}", resp.status()));
                }
                Err(err) => {
                    last_err = Some(err.to_string());
                }
            }
            if attempt < 2 {
                tokio::time::sleep(Duration::from_millis(100 * (attempt as u64 + 1))).await;
            }
        }
        metrics.inc_audit_persist_failure();
        error!(
            target: "audit",
            path = %payload.path,
            error = last_err.unwrap_or_else(|| "unknown".into()),
            "audit persist failed after retries"
        );
    });
}

fn should_audit(method: &http::Method, path: &str) -> bool {
    use http::Method;
    if matches!(
        *method,
        Method::POST | Method::PUT | Method::PATCH | Method::DELETE
    ) {
        return true;
    }
    if *method == Method::GET && is_file_download(path) {
        return true;
    }
    false
}

fn is_file_download(path: &str) -> bool {
    path.starts_with("/api/files/") && path.ends_with("/content")
}

#[cfg(test)]
mod tests {
    use super::*;
    use http::Method;

    #[test]
    fn post_is_audited() {
        assert!(should_audit(&Method::POST, "/api/users"));
    }

    #[test]
    fn file_download_is_audited() {
        assert!(should_audit(
            &Method::GET,
            "/api/files/00000000-0000-0000-0000-000000000001/content"
        ));
    }

    #[test]
    fn ordinary_get_is_not_audited() {
        assert!(!should_audit(&Method::GET, "/api/users"));
    }
}
