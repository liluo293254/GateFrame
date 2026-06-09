use serde::Deserialize;

use crate::error::AppError;
use crate::state::AppState;

#[derive(Deserialize)]
struct StatusEnvelope {
    data: StatusData,
}

#[derive(Deserialize)]
struct StatusData {
    status: String,
}

pub async fn ensure_tenant_active(state: &AppState, tenant_id: &str) -> Result<(), AppError> {
    let base = state.config.user_service_url.trim_end_matches('/');
    let url = format!("{base}/internal/v1/tenants/{tenant_id}/status");
    let resp = state
        .http_client
        .get(&url)
        .header("X-Internal-Token", &state.config.internal_token)
        .send()
        .await
        .map_err(|_| AppError::Internal)?;

    if resp.status() == reqwest::StatusCode::NOT_FOUND {
        return Err(AppError::Unauthorized);
    }
    if !resp.status().is_success() {
        return Err(AppError::Internal);
    }

    let body: StatusEnvelope = resp.json().await.map_err(|_| AppError::Internal)?;
    if body.data.status != "active" {
        return Err(AppError::Unauthorized);
    }
    Ok(())
}
