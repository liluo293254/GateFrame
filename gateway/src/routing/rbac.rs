use axum::{
    body::Body,
    extract::Request,
    middleware::{self, Next},
    response::Response,
    routing::{get, patch, post, MethodRouter},
    Router,
};
use tower_http::limit::RequestBodyLimitLayer;

use crate::audit;
use crate::auth;
use crate::error::AppError;
use crate::routing::proxy;
use crate::state::AppState;
use crate::tenant::TenantContext;

async fn require_permission_code(
    code: &'static str,
    req: Request<Body>,
    next: Next,
) -> Result<Response, AppError> {
    let ctx = req
        .extensions()
        .get::<TenantContext>()
        .ok_or(AppError::Unauthorized)?;
    if !ctx.has_permission(code) {
        return Err(AppError::Forbidden);
    }
    Ok(next.run(req).await)
}

async fn require_any_permission_code(
    codes: &'static [&'static str],
    req: Request<Body>,
    next: Next,
) -> Result<Response, AppError> {
    let ctx = req
        .extensions()
        .get::<TenantContext>()
        .ok_or(AppError::Unauthorized)?;
    for code in codes {
        if ctx.has_permission(code) {
            return Ok(next.run(req).await);
        }
    }
    Err(AppError::Forbidden)
}

async fn require_user_read(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("user.read", req, next).await
}

async fn require_user_create(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("user.create", req, next).await
}

/// Permission for /api/users/{id} depends on HTTP method.
async fn require_user_id_permission(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    use axum::http::Method;
    let code = match *req.method() {
        Method::GET => "user.read",
        Method::PUT => "user.update",
        Method::DELETE => "user.delete",
        _ => return Err(AppError::BadRequest("method not allowed".into())),
    };
    require_permission_code(code, req, next).await
}

async fn require_rbac_read(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("rbac.read", req, next).await
}

async fn require_rbac_manage(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("rbac.manage", req, next).await
}

async fn require_audit_read(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("audit.read", req, next).await
}

async fn require_workflow_read(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("workflow.read", req, next).await
}

async fn require_workflow_manage(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("workflow.manage", req, next).await
}

async fn require_workflow_read_or_manage(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_any_permission_code(&["workflow.read", "workflow.manage"], req, next).await
}

async fn require_search_read(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("search.read", req, next).await
}

async fn require_notification_read(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("notification.read", req, next).await
}

async fn require_search_manage(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("search.manage", req, next).await
}

async fn require_notification_manage(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("notification.manage", req, next).await
}

async fn require_tenant_read(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("tenant.read", req, next).await
}

async fn require_tenant_manage(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("tenant.manage", req, next).await
}

async fn require_file_read(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("file.read", req, next).await
}

async fn require_file_manage(req: Request<Body>, next: Next) -> Result<Response, AppError> {
    require_permission_code("file.manage", req, next).await
}

fn with_auth_audit(state: &AppState, router: Router<AppState>) -> Router<AppState> {
    router
        .route_layer(middleware::from_fn_with_state(
            state.clone(),
            audit::audit_middleware,
        ))
        .route_layer(middleware::from_fn_with_state(
            state.clone(),
            auth::middleware::require_auth,
        ))
}

macro_rules! with_perm {
    ($state:expr, $perm:ident, $router:expr) => {
        $router
            .route_layer(middleware::from_fn($perm))
            .route_layer(middleware::from_fn_with_state(
                $state.clone(),
                auth::middleware::require_perm_version,
            ))
            .route_layer(middleware::from_fn_with_state(
                $state.clone(),
                audit::audit_middleware,
            ))
            .route_layer(middleware::from_fn_with_state(
                $state.clone(),
                auth::middleware::require_auth,
            ))
    };
}

fn proxy_get() -> MethodRouter<AppState> {
    get(proxy::forward_to_user_service)
}

fn proxy_post() -> MethodRouter<AppState> {
    post(proxy::forward_to_user_service)
}

fn proxy_user_by_id() -> MethodRouter<AppState> {
    get(proxy::forward_to_user_service)
        .put(proxy::forward_to_user_service)
        .delete(proxy::forward_to_user_service)
}

fn proxy_put() -> MethodRouter<AppState> {
    axum::routing::put(proxy::forward_to_user_service)
}

pub fn protected_routes(state: &AppState) -> Router<AppState> {
    let auth_only = with_auth_audit(
        state,
        Router::new()
            .route("/api/auth/permissions", proxy_get())
            .route("/api/auth/logout", proxy_post())
            .route("/api/auth/refresh", proxy_post()),
    );

    let user_read = with_perm!(
        state,
        require_user_read,
        Router::new().route("/api/users", proxy_get())
    );

    let user_create = with_perm!(
        state,
        require_user_create,
        Router::new().route("/api/users", proxy_post())
    );

    // Single route for all /api/users/{id} methods — merging separate routers
    // for the same path drops handlers and returns 404.
    let user_by_id = with_perm!(
        state,
        require_user_id_permission,
        Router::new().route("/api/users/:id", proxy_user_by_id())
    );

    let rbac_read = with_perm!(
        state,
        require_rbac_read,
        Router::new()
            .route("/api/rbac/roles", proxy_get())
            .route("/api/rbac/roles/:id/permissions", proxy_get())
            .route("/api/rbac/permissions", proxy_get())
            .route("/api/rbac/role-permissions", proxy_get())
    );

    let rbac_manage = with_perm!(
        state,
        require_rbac_manage,
        Router::new().route("/api/rbac/roles/:id/permissions", proxy_put())
    );

    let audit_read = with_perm!(
        state,
        require_audit_read,
        Router::new().route("/api/audit", get(proxy::forward_to_audit_service))
    );

    let workflow_read = with_perm!(
        state,
        require_workflow_read,
        Router::new()
            .route("/api/workflows", get(proxy::forward_to_workflow_service))
            .route(
                "/api/workflows/:id",
                get(proxy::forward_to_workflow_service),
            )
            .route(
                "/api/workflows/:id/events",
                get(proxy::forward_to_workflow_service),
            )
    );

    let workflow_request = with_perm!(
        state,
        require_workflow_read_or_manage,
        Router::new()
            .route("/api/workflows", post(proxy::forward_to_workflow_service))
            .route(
                "/api/workflows/:id",
                axum::routing::put(proxy::forward_to_workflow_service),
            )
            .route(
                "/api/workflows/:id/submit",
                post(proxy::forward_to_workflow_service),
            )
    );

    let workflow_review = with_perm!(
        state,
        require_workflow_manage,
        Router::new()
            .route(
                "/api/workflows/:id/approve",
                post(proxy::forward_to_workflow_service),
            )
            .route(
                "/api/workflows/:id/reject",
                post(proxy::forward_to_workflow_service),
            )
            .route(
                "/api/workflows/:id/request-changes",
                post(proxy::forward_to_workflow_service),
            )
            .route(
                "/api/workflows/:id",
                axum::routing::delete(proxy::forward_to_workflow_service),
            )
    );

    let search_read = with_perm!(
        state,
        require_search_read,
        Router::new().route("/api/search", get(proxy::forward_to_search_service))
    );

    let notification_read = with_perm!(
        state,
        require_notification_read,
        Router::new().route("/api/notifications", get(proxy::forward_to_notification_service))
    );

    let search_manage = with_perm!(
        state,
        require_search_manage,
        Router::new()
            .route(
                "/api/search/documents",
                post(proxy::forward_to_search_service),
            )
            .route(
                "/api/search/documents/:id",
                axum::routing::delete(proxy::forward_to_search_service),
            )
    );

    let notification_manage = with_perm!(
        state,
        require_notification_manage,
        Router::new()
            .route(
                "/api/notifications",
                post(proxy::forward_to_notification_service),
            )
            .route(
                "/api/notifications/:id/read",
                patch(proxy::forward_to_notification_service),
            )
    );

    let tenant_read = with_perm!(
        state,
        require_tenant_read,
        Router::new().route("/api/tenants", get(proxy::forward_to_user_service))
    );

    let tenant_manage = with_perm!(
        state,
        require_tenant_manage,
        Router::new()
            .route("/api/tenants", post(proxy::forward_to_user_service))
            .route(
                "/api/tenants/:id",
                axum::routing::put(proxy::forward_to_user_service),
            )
    );

    let dashboard_read = with_perm!(
        state,
        require_user_read,
        Router::new().route(
            "/api/dashboard/stats",
            get(proxy::forward_to_user_service),
        )
    );

    let file_read = with_perm!(
        state,
        require_file_read,
        Router::new()
            .route("/api/files", get(proxy::forward_to_file_service))
            .route(
                "/api/files/:id/content",
                get(proxy::forward_to_file_service),
            )
    );

    let file_manage = with_perm!(
        state,
        require_file_manage,
        Router::new()
            .route("/api/files", post(proxy::forward_to_file_service))
            .route(
                "/api/files/:id",
                axum::routing::delete(proxy::forward_to_file_service),
            )
            .layer(RequestBodyLimitLayer::new(
                state.config.file_max_request_body_bytes,
            ))
    );

    Router::new()
        .merge(auth_only)
        .merge(user_read)
        .merge(user_create)
        .merge(user_by_id)
        .merge(rbac_read)
        .merge(rbac_manage)
        .merge(audit_read)
        .merge(workflow_read)
        .merge(workflow_request)
        .merge(workflow_review)
        .merge(search_read)
        .merge(search_manage)
        .merge(notification_read)
        .merge(notification_manage)
        .merge(tenant_read)
        .merge(tenant_manage)
        .merge(dashboard_read)
        .merge(file_read)
        .merge(file_manage)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::auth::issue_token;
    use axum::body::Body;
    use axum::http::{Request, StatusCode};
    use tower::ServiceExt;
    use uuid::Uuid;
    use wiremock::matchers::{method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    fn test_state(upstream: &str) -> AppState {
        AppState {
            config: crate::config::Config {
                listen_addr: "127.0.0.1:0".into(),
                jwt_secret: "test-secret".into(),
                jwt_expiry_secs: 3600,
                internal_token: "internal".into(),
                user_service_url: upstream.into(),
                audit_service_url: "http://127.0.0.1:9".into(),
                workflow_service_url: "http://127.0.0.1:9".into(),
                search_service_url: "http://127.0.0.1:9".into(),
                notification_service_url: "http://127.0.0.1:9".into(),
                file_service_url: "http://127.0.0.1:9".into(),
                redis_url: None,
                web_app_url: "http://127.0.0.1:5173".into(),
                oidc_issuer_url: None,
                oidc_client_id: None,
                oidc_client_secret: None,
                oidc_redirect_url: None,
                oidc_tenant_slug: "default".into(),
                login_rate_limit_max: 10,
                login_rate_limit_window_secs: 60,
                cors_origins: vec![],
                file_max_request_body_bytes: 70 * 1024 * 1024,
            },
            http_client: reqwest::Client::new(),
            login_limiter: None,
            perm_version: None,
            metrics: std::sync::Arc::new(crate::metrics::Metrics::new()),
        }
    }

    async fn mount_active_tenant_status(mock: &MockServer, tenant_id: Uuid) {
        Mock::given(method("GET"))
            .and(path(format!("/internal/v1/tenants/{tenant_id}/status")))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
                "data": { "status": "active" }
            })))
            .mount(mock)
            .await;
    }

    #[tokio::test]
    async fn list_users_requires_user_read_permission() {
        let mock = MockServer::start().await;
        let tenant_id = Uuid::new_v4();
        mount_active_tenant_status(&mock, tenant_id).await;
        Mock::given(method("GET"))
            .and(path("/api/users"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({"code":"ok","data":[]})))
            .mount(&mock)
            .await;

        let state = test_state(&mock.uri());
        let app = protected_routes(&state).with_state(state);

        let token = issue_token(
            "test-secret",
            Uuid::new_v4(),
            tenant_id,
            vec!["user.create".into()],
            3600,
        )
        .unwrap();

        let resp = app
            .oneshot(
                Request::builder()
                    .uri("/api/users")
                    .header("Authorization", format!("Bearer {}", token))
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn list_users_allowed_with_user_read() {
        let mock = MockServer::start().await;
        let tenant_id = Uuid::new_v4();
        mount_active_tenant_status(&mock, tenant_id).await;
        Mock::given(method("GET"))
            .and(path("/api/users"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({"code":"ok","data":[]})))
            .mount(&mock)
            .await;

        let state = test_state(&mock.uri());
        let app = protected_routes(&state).with_state(state);

        let token = issue_token(
            "test-secret",
            Uuid::new_v4(),
            tenant_id,
            vec!["user.read".into()],
            3600,
        )
        .unwrap();

        let resp = app
            .oneshot(
                Request::builder()
                    .uri("/api/users")
                    .header("Authorization", format!("Bearer {}", token))
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn bare_user_by_id_route_without_middleware() {
        let user_id = Uuid::new_v4();
        let mock = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path(format!("/api/users/{}", user_id)))
            .respond_with(ResponseTemplate::new(200).set_body_string("ok"))
            .mount(&mock)
            .await;

        let state = test_state(&mock.uri());
        let app = Router::new()
            .route("/api/users/:id", get(proxy::forward_to_user_service))
            .with_state(state);

        let resp = app
            .oneshot(
                Request::builder()
                    .uri(format!("/api/users/{}", user_id))
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn get_user_by_id_proxies_with_user_read_permission() {
        let user_id = Uuid::new_v4();
        let mock = MockServer::start().await;
        let tenant_id = Uuid::new_v4();
        mount_active_tenant_status(&mock, tenant_id).await;
        Mock::given(method("GET"))
            .and(path(format!("/api/users/{}", user_id)))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"code":"ok","data":{"id": user_id}})),
            )
            .mount(&mock)
            .await;

        let state = test_state(&mock.uri());
        let app = protected_routes(&state).with_state(state);

        let token = issue_token(
            "test-secret",
            Uuid::new_v4(),
            tenant_id,
            vec!["user.read".into()],
            3600,
        )
        .unwrap();

        let resp = app
            .oneshot(
                Request::builder()
                    .uri(format!("/api/users/{}", user_id))
                    .header("Authorization", format!("Bearer {}", token))
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn delete_user_by_id_proxies_with_user_delete_permission() {
        let user_id = Uuid::new_v4();
        let mock = MockServer::start().await;
        let tenant_id = Uuid::new_v4();
        mount_active_tenant_status(&mock, tenant_id).await;
        Mock::given(method("DELETE"))
            .and(path(format!("/api/users/{}", user_id)))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"code":"ok","data":{"deleted":true}})),
            )
            .mount(&mock)
            .await;

        let state = test_state(&mock.uri());
        let app = protected_routes(&state).with_state(state);

        let token = issue_token(
            "test-secret",
            Uuid::new_v4(),
            tenant_id,
            vec!["user.delete".into()],
            3600,
        )
        .unwrap();

        let resp = app
            .oneshot(
                Request::builder()
                    .method("DELETE")
                    .uri(format!("/api/users/{}", user_id))
                    .header("Authorization", format!("Bearer {}", token))
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn list_workflows_requires_workflow_read_permission() {
        let user_mock = MockServer::start().await;
        let workflow_mock = MockServer::start().await;
        let tenant_id = Uuid::new_v4();
        mount_active_tenant_status(&user_mock, tenant_id).await;
        Mock::given(method("GET"))
            .and(path("/api/workflows"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({"code":"ok","data":[]})))
            .mount(&workflow_mock)
            .await;

        let mut state = test_state(&user_mock.uri());
        state.config.workflow_service_url = workflow_mock.uri();
        let app = protected_routes(&state).with_state(state);

        let token = issue_token(
            "test-secret",
            Uuid::new_v4(),
            tenant_id,
            vec!["user.read".into()],
            3600,
        )
        .unwrap();

        let resp = app
            .oneshot(
                Request::builder()
                    .uri("/api/workflows")
                    .header("Authorization", format!("Bearer {}", token))
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::FORBIDDEN);
    }

    #[tokio::test]
    async fn list_workflows_allowed_with_workflow_read() {
        let user_mock = MockServer::start().await;
        let workflow_mock = MockServer::start().await;
        let tenant_id = Uuid::new_v4();
        mount_active_tenant_status(&user_mock, tenant_id).await;
        Mock::given(method("GET"))
            .and(path("/api/workflows"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({"code":"ok","data":[]})))
            .mount(&workflow_mock)
            .await;

        let mut state = test_state(&user_mock.uri());
        state.config.workflow_service_url = workflow_mock.uri();
        let app = protected_routes(&state).with_state(state);

        let token = issue_token(
            "test-secret",
            Uuid::new_v4(),
            tenant_id,
            vec!["workflow.read".into()],
            3600,
        )
        .unwrap();

        let resp = app
            .oneshot(
                Request::builder()
                    .uri("/api/workflows")
                    .header("Authorization", format!("Bearer {}", token))
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }
}
