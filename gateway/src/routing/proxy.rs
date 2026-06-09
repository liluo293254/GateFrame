use axum::{
    body::Body,
    extract::State,
    http::{HeaderMap, HeaderName, HeaderValue, Request},
    response::Response,
};
use bytes::Bytes;
use http_body_util::BodyExt;
use reqwest::Client;

use crate::error::AppError;
use crate::state::AppState;
use crate::tenant::TenantContext;

const HOP_BY_HOP: &[&str] = &[
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailers",
    "transfer-encoding",
    "upgrade",
    "host",
];

const STRIP_REQUEST: &[&str] = &[
    "authorization",
    "x-internal-token",
    "x-user-id",
    "x-tenant-id",
    "x-permissions",
    "x-request-id",
];

pub async fn forward_to_user_service(
    State(state): State<AppState>,
    req: Request<Body>,
) -> Result<Response, AppError> {
    let ctx = req.extensions().get::<TenantContext>().cloned();
    forward(&state, &state.config.user_service_url, req, ctx.as_ref()).await
}

/// Public proxy (login) — strips client-forged internal headers.
pub async fn forward_public_to_user_service(
    State(state): State<AppState>,
    req: Request<Body>,
) -> Result<Response, AppError> {
    forward(&state, &state.config.user_service_url, req, None).await
}

pub async fn forward_to_audit_service(
    State(state): State<AppState>,
    req: Request<Body>,
) -> Result<Response, AppError> {
    let ctx = req.extensions().get::<TenantContext>().cloned();
    forward(
        &state,
        &state.config.audit_service_url,
        req,
        ctx.as_ref(),
    )
    .await
}

pub async fn forward_to_workflow_service(
    State(state): State<AppState>,
    req: Request<Body>,
) -> Result<Response, AppError> {
    let ctx = req.extensions().get::<TenantContext>().cloned();
    forward(
        &state,
        &state.config.workflow_service_url,
        req,
        ctx.as_ref(),
    )
    .await
}

pub async fn forward_to_search_service(
    State(state): State<AppState>,
    req: Request<Body>,
) -> Result<Response, AppError> {
    let ctx = req.extensions().get::<TenantContext>().cloned();
    forward(
        &state,
        &state.config.search_service_url,
        req,
        ctx.as_ref(),
    )
    .await
}

pub async fn forward_to_notification_service(
    State(state): State<AppState>,
    req: Request<Body>,
) -> Result<Response, AppError> {
    let ctx = req.extensions().get::<TenantContext>().cloned();
    forward(
        &state,
        &state.config.notification_service_url,
        req,
        ctx.as_ref(),
    )
    .await
}

pub async fn forward_to_file_service(
    State(state): State<AppState>,
    req: Request<Body>,
) -> Result<Response, AppError> {
    let ctx = req.extensions().get::<TenantContext>().cloned();
    forward(
        &state,
        &state.config.file_service_url,
        req,
        ctx.as_ref(),
    )
    .await
}

async fn forward(
    state: &AppState,
    upstream_base: &str,
    req: Request<Body>,
    ctx: Option<&TenantContext>,
) -> Result<Response, AppError> {
    let (parts, body) = req.into_parts();
    let path_and_query = parts
        .uri
        .path_and_query()
        .map(|pq| pq.as_str())
        .unwrap_or("/");
    let target = format!(
        "{}{}",
        upstream_base.trim_end_matches('/'),
        path_and_query
    );

    let body_bytes = body
        .collect()
        .await
        .map_err(|_| AppError::Internal)?
        .to_bytes();

    let outbound_headers =
        build_outbound_headers(&parts.headers, &state.config.internal_token, ctx);

    let client = Client::new();
    let mut builder = client.request(parts.method, &target).headers(to_reqwest_headers(
        &outbound_headers,
    ));

    if !body_bytes.is_empty() {
        builder = builder.body(body_bytes.to_vec());
    }

    let upstream = builder
        .send()
        .await
        .map_err(|e| AppError::Upstream(e.to_string()))?;

    let status = upstream.status();
    let mut resp = Response::builder().status(status);
    for (k, v) in upstream.headers().iter() {
        if is_hop_by_hop(k.as_str()) {
            continue;
        }
        if let (Ok(name), Ok(val)) = (
            HeaderName::from_bytes(k.as_ref()),
            HeaderValue::from_bytes(v.as_bytes()),
        ) {
            resp = resp.header(name, val);
        }
    }

    let bytes = upstream
        .bytes()
        .await
        .map_err(|e| AppError::Upstream(e.to_string()))?;

    resp.body(Body::from(Bytes::from(bytes.to_vec())))
        .map_err(|_| AppError::Internal)
}

fn build_outbound_headers(
    inbound: &HeaderMap,
    internal_token: &str,
    ctx: Option<&TenantContext>,
) -> HeaderMap {
    let mut out = HeaderMap::new();
    for (k, v) in inbound.iter() {
        let key = k.as_str().to_ascii_lowercase();
        if STRIP_REQUEST.contains(&key.as_str()) || is_hop_by_hop(&key) {
            continue;
        }
        out.insert(k.clone(), v.clone());
    }
    out.insert(
        HeaderName::from_static("x-internal-token"),
        HeaderValue::from_str(internal_token).unwrap_or(HeaderValue::from_static("")),
    );
    if let Some(ctx) = ctx {
        out.insert(
            HeaderName::from_static("x-user-id"),
            HeaderValue::from_str(&ctx.user_id).unwrap_or(HeaderValue::from_static("")),
        );
        out.insert(
            HeaderName::from_static("x-tenant-id"),
            HeaderValue::from_str(&ctx.tenant_id).unwrap_or(HeaderValue::from_static("")),
        );
        if !ctx.permissions.is_empty() {
            let perms = ctx.permissions.join(",");
            out.insert(
                HeaderName::from_static("x-permissions"),
                HeaderValue::from_str(&perms).unwrap_or(HeaderValue::from_static("")),
            );
        }
    }
    out
}

fn to_reqwest_headers(map: &HeaderMap) -> reqwest::header::HeaderMap {
    let mut out = reqwest::header::HeaderMap::new();
    for (k, v) in map.iter() {
        if let (Ok(name), Ok(val)) = (
            reqwest::header::HeaderName::from_bytes(k.as_ref()),
            reqwest::header::HeaderValue::from_bytes(v.as_bytes()),
        ) {
            out.insert(name, val);
        }
    }
    out
}

fn is_hop_by_hop(name: &str) -> bool {
    HOP_BY_HOP.contains(&name.to_ascii_lowercase().as_str())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strips_client_internal_headers() {
        let mut inbound = HeaderMap::new();
        inbound.insert("x-internal-token", "forged".parse().unwrap());
        inbound.insert("x-user-id", "evil".parse().unwrap());
        inbound.insert("content-type", "application/json".parse().unwrap());
        let out = build_outbound_headers(&inbound, "real-token", None);
        assert_eq!(
            out.get("x-internal-token").unwrap().to_str().unwrap(),
            "real-token"
        );
        assert!(out.get("x-user-id").is_none());
        assert!(out.get("content-type").is_some());
    }

    #[test]
    fn injects_identity_when_authenticated() {
        let ctx = TenantContext {
            tenant_id: "tenant-1".into(),
            user_id: "user-1".into(),
            permissions: vec!["workflow.read".into(), "file.manage".into()],
        };
        let out = build_outbound_headers(&HeaderMap::new(), "tok", Some(&ctx));
        assert_eq!(out.get("x-tenant-id").unwrap().to_str().unwrap(), "tenant-1");
        assert_eq!(out.get("x-user-id").unwrap().to_str().unwrap(), "user-1");
        assert_eq!(
            out.get("x-permissions").unwrap().to_str().unwrap(),
            "workflow.read,file.manage"
        );
    }

    #[test]
    fn strips_client_permissions_header() {
        let mut inbound = HeaderMap::new();
        inbound.insert("x-permissions", "forged.perm".parse().unwrap());
        let ctx = TenantContext {
            tenant_id: "t".into(),
            user_id: "u".into(),
            permissions: vec!["audit.read".into()],
        };
        let out = build_outbound_headers(&inbound, "tok", Some(&ctx));
        assert_eq!(out.get("x-permissions").unwrap().to_str().unwrap(), "audit.read");
    }
}
