use axum::{
    body::Body,
    extract::State,
    http::{header, Request},
    middleware::Next,
    response::Response,
};

use crate::auth::validate_token;
use crate::error::AppError;
use crate::state::AppState;
use crate::tenant::TenantContext;

pub async fn require_auth(
    State(state): State<AppState>,
    mut req: Request<Body>,
    next: Next,
) -> Result<Response, AppError> {
    let token = extract_bearer(req.headers()).ok_or(AppError::Unauthorized)?;
    let claims = validate_token(&state.config.jwt_secret, token)?;
    crate::auth::tenant::ensure_tenant_active(&state, &claims.tenant_id).await?;
    req.extensions_mut().insert(claims.clone());
    req.extensions_mut().insert(TenantContext {
        tenant_id: claims.tenant_id.clone(),
        user_id: claims.sub.clone(),
        permissions: claims.permissions.clone(),
    });
    Ok(next.run(req).await)
}

/// Rejects JWTs whose embedded perm_ver is behind Redis (RBAC changed after login).
/// Skip on `/api/auth/permissions` and `/api/auth/refresh` so clients can recover.
pub async fn require_perm_version(
    State(state): State<AppState>,
    req: Request<Body>,
    next: Next,
) -> Result<Response, AppError> {
    if let Some(checker) = &state.perm_version {
        let claims = req
            .extensions()
            .get::<crate::auth::jwt::Claims>()
            .ok_or(AppError::Unauthorized)?;
        checker
            .ensure_current(&claims.sub, claims.perm_ver)
            .await?;
    }
    Ok(next.run(req).await)
}

fn extract_bearer(headers: &http::HeaderMap) -> Option<&str> {
    headers
        .get(header::AUTHORIZATION)?
        .to_str()
        .ok()?
        .strip_prefix("Bearer ")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extract_bearer_parses_header() {
        let mut headers = http::HeaderMap::new();
        headers.insert(
            header::AUTHORIZATION,
            "Bearer abc.def.ghi".parse().unwrap(),
        );
        assert_eq!(extract_bearer(&headers), Some("abc.def.ghi"));
    }

    #[test]
    fn extract_bearer_rejects_missing() {
        assert!(extract_bearer(&http::HeaderMap::new()).is_none());
    }
}
