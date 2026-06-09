use axum::{
    extract::{Query, State},
    response::{IntoResponse, Redirect, Response},
};
use openidconnect::core::{
    CoreClient, CoreProviderMetadata, CoreResponseType,
};
use openidconnect::{
    AuthenticationFlow, AuthorizationCode, ClientId, ClientSecret, CsrfToken, EndpointMaybeSet,
    EndpointNotSet, EndpointSet, IssuerUrl, Nonce, RedirectUrl, Scope,
};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use serde::Deserialize;
use serde_json::json;
use tracing::warn;

use crate::error::AppError;
use crate::state::AppState;

type GateOidcClient = CoreClient<
    EndpointSet,
    EndpointNotSet,
    EndpointNotSet,
    EndpointNotSet,
    EndpointMaybeSet,
    EndpointMaybeSet,
>;

#[derive(Clone)]
struct OidcStateStore {
    redis: ConnectionManager,
    ttl_secs: i64,
}

impl OidcStateStore {
    async fn connect(redis_url: &str) -> Result<Self, redis::RedisError> {
        let client = redis::Client::open(redis_url)?;
        let redis = ConnectionManager::new(client).await?;
        Ok(Self {
            redis,
            ttl_secs: 600,
        })
    }

    async fn save(&self, state: &str, nonce: &str) -> Result<(), AppError> {
        let key = format!("oidc:state:{state}");
        let mut conn = self.redis.clone();
        conn.set_ex::<_, _, ()>(key, nonce, self.ttl_secs as u64)
            .await
            .map_err(|_| AppError::Internal)?;
        Ok(())
    }

    async fn take_nonce(&self, state: &str) -> Result<String, AppError> {
        let key = format!("oidc:state:{state}");
        let mut conn = self.redis.clone();
        let nonce: Option<String> = conn.get(&key).await.map_err(|_| AppError::Internal)?;
        let _: () = conn.del(&key).await.map_err(|_| AppError::Internal)?;
        nonce.ok_or(AppError::Unauthorized)
    }
}

struct OidcRuntime {
    client: GateOidcClient,
    store: OidcStateStore,
    http_client: reqwest::Client,
}

async fn runtime(state: &AppState) -> Result<OidcRuntime, AppError> {
    let cfg = &state.config;
    let issuer = cfg
        .oidc_issuer_url
        .as_ref()
        .ok_or(AppError::Internal)?;
    let client_id = cfg.oidc_client_id.as_ref().ok_or(AppError::Internal)?;
    let client_secret = cfg
        .oidc_client_secret
        .as_ref()
        .ok_or(AppError::Internal)?;
    let redirect = cfg.oidc_redirect_url.as_ref().ok_or(AppError::Internal)?;
    let redis_url = cfg.redis_url.as_ref().ok_or(AppError::Internal)?;

    let http_client = reqwest::Client::builder()
        .redirect(reqwest::redirect::Policy::none())
        .build()
        .map_err(|_| AppError::Internal)?;

    let issuer_url = IssuerUrl::new(issuer.clone()).map_err(|_| AppError::Internal)?;
    let metadata = CoreProviderMetadata::discover_async(issuer_url, &http_client)
        .await
        .map_err(|_| AppError::Internal)?;

    let client = CoreClient::from_provider_metadata(
        metadata,
        ClientId::new(client_id.clone()),
        Some(ClientSecret::new(client_secret.clone())),
    )
    .set_redirect_uri(
        RedirectUrl::new(redirect.clone()).map_err(|_| AppError::Internal)?,
    );

    let store = OidcStateStore::connect(redis_url)
        .await
        .map_err(|_| AppError::Internal)?;

    Ok(OidcRuntime {
        client,
        store,
        http_client,
    })
}

pub async fn oidc_config(State(state): State<AppState>) -> impl IntoResponse {
    axum::Json(json!({
        "enabled": state.config.oidc_enabled(),
    }))
}

pub async fn oidc_login(State(state): State<AppState>) -> Result<Response, AppError> {
    if !state.config.oidc_enabled() {
        return Err(AppError::BadRequest("oidc not configured".into()));
    }
    let rt = runtime(&state).await?;
    let (auth_url, csrf, nonce) = rt
        .client
        .authorize_url(
            AuthenticationFlow::<CoreResponseType>::AuthorizationCode,
            CsrfToken::new_random,
            Nonce::new_random,
        )
        .add_scope(Scope::new("openid".into()))
        .add_scope(Scope::new("profile".into()))
        .add_scope(Scope::new("email".into()))
        .url();

    rt.store
        .save(csrf.secret(), nonce.secret())
        .await?;

    Ok(Redirect::temporary(auth_url.as_str()).into_response())
}

#[derive(Deserialize)]
pub struct OidcCallbackQuery {
    code: String,
    state: String,
}

#[derive(Deserialize)]
struct ApiEnvelope<T> {
    data: T,
}

#[derive(Deserialize)]
struct OidcLoginData {
    token: String,
}

pub async fn oidc_callback(
    State(state): State<AppState>,
    Query(query): Query<OidcCallbackQuery>,
) -> Result<Response, AppError> {
    if !state.config.oidc_enabled() {
        return Err(AppError::BadRequest("oidc not configured".into()));
    }
    let rt = runtime(&state).await?;
    let nonce_val = rt.store.take_nonce(&query.state).await?;

    let token_response = rt
        .client
        .exchange_code(AuthorizationCode::new(query.code))
        .map_err(|_| AppError::Unauthorized)?
        .request_async(&rt.http_client)
        .await
        .map_err(|_| AppError::Unauthorized)?;

    let id_token = token_response
        .extra_fields()
        .id_token()
        .ok_or(AppError::Unauthorized)?;
    let nonce = Nonce::new(nonce_val);
    let claims = id_token
        .claims(&rt.client.id_token_verifier(), &nonce)
        .map_err(|_| AppError::Unauthorized)?;

    let subject = claims.subject().to_string();
    let email = claims.email().map(|e| e.to_string()).unwrap_or_default();
    let username = claims
        .preferred_username()
        .map(|u| u.as_str().to_string())
        .unwrap_or_default();

    let resolve_url = format!(
        "{}/internal/v1/auth/oidc/resolve",
        state.config.user_service_url.trim_end_matches('/')
    );
    let resolve_body = json!({
        "subject": subject,
        "email": email,
        "username": username,
        "tenant_slug": state.config.oidc_tenant_slug,
    });

    let resolve_resp = state
        .http_client
        .post(resolve_url)
        .header("X-Internal-Token", &state.config.internal_token)
        .json(&resolve_body)
        .send()
        .await
        .map_err(|_| AppError::Internal)?;

    if !resolve_resp.status().is_success() {
        warn!(target: "oidc", status = %resolve_resp.status(), "oidc user resolve failed");
        return Err(AppError::Unauthorized);
    }

    let body: ApiEnvelope<OidcLoginData> =
        resolve_resp.json().await.map_err(|_| AppError::Internal)?;
    let web = state.config.web_app_url.trim_end_matches('/');
    let redirect = format!(
        "{web}/auth/oidc/callback?token={}",
        urlencoding::encode(&body.data.token)
    );
    Ok(Redirect::temporary(&redirect).into_response())
}
