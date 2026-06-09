mod audit;
mod auth;
mod config;
mod error;
mod limiter;
mod metrics;
mod permission;
mod routing;
mod state;
mod tenant;

use std::net::SocketAddr;

use axum::{
    middleware,
    response::IntoResponse,
    routing::{get, post},
    Router,
};
use state::AppState;
use tower_http::{
    cors::{Any, CorsLayer},
    trace::TraceLayer,
};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

async fn health() -> &'static str {
    "ok"
}

async fn prometheus_metrics(
    axum::extract::State(state): axum::extract::State<AppState>,
) -> Result<impl IntoResponse, error::AppError> {
    let body = state
        .metrics
        .render()
        .map_err(|_| error::AppError::Internal)?;
    Ok((
        [(axum::http::header::CONTENT_TYPE, "text/plain; version=0.0.4")],
        body,
    ))
}

pub fn build_router(state: AppState) -> Router {
    let cors = if state.config.cors_origins.is_empty() {
        CorsLayer::new()
            .allow_origin(Any)
            .allow_methods(Any)
            .allow_headers(Any)
    } else {
        CorsLayer::new()
            .allow_origin(
                state
                    .config
                    .cors_origins
                    .iter()
                    .filter_map(|o| o.parse().ok())
                    .collect::<Vec<_>>(),
            )
            .allow_methods(Any)
            .allow_headers(Any)
    };

    let public = Router::new()
        .route("/health", get(health))
        .route("/metrics", get(prometheus_metrics))
        .route(
            "/api/auth/login",
            post(routing::forward_public_to_user_service)
                .route_layer(middleware::from_fn_with_state(
                    state.clone(),
                    audit::login_audit_middleware,
                ))
                .route_layer(middleware::from_fn_with_state(
                    state.clone(),
                    limiter::login_rate_limit_middleware,
                )),
        )
        .route(
            "/api/auth/oidc/config",
            get(auth::oidc::oidc_config),
        )
        .route(
            "/api/auth/oidc/login",
            get(auth::oidc::oidc_login),
        )
        .route(
            "/api/auth/oidc/callback",
            get(auth::oidc::oidc_callback),
        );

    Router::new()
        .merge(public)
        .merge(routing::protected_routes(&state))
        .layer(cors)
        .layer(TraceLayer::new_for_http())
        .with_state(state)
}

#[tokio::main]
async fn main() {
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(
            |_| "gateframe_gateway=info,tower_http=info,audit=info".into(),
        ))
        .with(tracing_subscriber::fmt::layer())
        .init();

    let config = config::Config::from_env().expect("invalid gateway config");
    let addr: SocketAddr = config
        .listen_addr
        .parse()
        .expect("invalid GATEWAY_LISTEN");
    let state = AppState::new(config)
        .await
        .expect("failed to initialize gateway state");
    let app = build_router(state);

    tracing::info!("gateway listening on {}", addr);
    let listener = tokio::net::TcpListener::bind(addr)
        .await
        .expect("bind failed");
    axum::serve(
        listener,
        app.into_make_service_with_connect_info::<SocketAddr>(),
    )
    .await
    .expect("server failed");
}

#[cfg(test)]
mod integration_tests {
    use super::*;
    use axum::body::Body;
    use axum::http::{Request, StatusCode};
    use tower::ServiceExt;
    use wiremock::matchers::{method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    fn test_config(user_url: &str) -> config::Config {
        config::Config {
            listen_addr: "127.0.0.1:0".into(),
            jwt_secret: "test-secret".into(),
            jwt_expiry_secs: 3600,
            internal_token: "internal".into(),
            user_service_url: user_url.into(),
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
            login_rate_limit_max: 3,
            login_rate_limit_window_secs: 60,
            cors_origins: vec!["http://localhost:5173".into()],
            file_max_request_body_bytes: 70 * 1024 * 1024,
        }
    }

    #[tokio::test]
    async fn health_is_public() {
        let app = build_router(AppState::new(test_config("http://127.0.0.1:9")).await.unwrap());
        let resp = app
            .oneshot(Request::builder().uri("/health").body(Body::empty()).unwrap())
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn protected_route_requires_jwt() {
        let app = build_router(AppState::new(test_config("http://127.0.0.1:9")).await.unwrap());
        let resp = app
            .oneshot(
                Request::builder()
                    .uri("/api/users")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn metrics_endpoint_is_public() {
        let app = build_router(AppState::new(test_config("http://127.0.0.1:9")).await.unwrap());
        let resp = app
            .oneshot(Request::builder().uri("/metrics").body(Body::empty()).unwrap())
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn login_proxies_to_upstream() {
        let mock = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/auth/login"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
                "code": "ok",
                "data": { "token": "upstream-token" }
            })))
            .mount(&mock)
            .await;

        let app = build_router(
            AppState::new(test_config(&mock.uri()))
                .await
                .unwrap(),
        );
        let resp = app
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/api/auth/login")
                    .header("content-type", "application/json")
                    .body(Body::from(r#"{"username":"admin","password":"admin"}"#))
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(resp.status(), StatusCode::OK);
    }
}
