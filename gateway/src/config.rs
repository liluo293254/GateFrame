use std::env;

#[derive(Clone, Debug)]
pub struct Config {
    pub listen_addr: String,
    pub jwt_secret: String,
    pub jwt_expiry_secs: u64,
    pub internal_token: String,
    pub user_service_url: String,
    pub audit_service_url: String,
    pub workflow_service_url: String,
    pub search_service_url: String,
    pub notification_service_url: String,
    pub file_service_url: String,
    pub redis_url: Option<String>,
    pub web_app_url: String,
    pub oidc_issuer_url: Option<String>,
    pub oidc_client_id: Option<String>,
    pub oidc_client_secret: Option<String>,
    pub oidc_redirect_url: Option<String>,
    pub oidc_tenant_slug: String,
    pub login_rate_limit_max: u32,
    pub login_rate_limit_window_secs: u64,
    pub cors_origins: Vec<String>,
    /// Max JSON request body for file uploads (base64 expands ~4/3 over raw file size).
    pub file_max_request_body_bytes: usize,
}

const DEV_JWT_SECRET: &str = "dev-only-change-me-in-production";
const DEV_INTERNAL_TOKEN: &str = "dev-internal-token-change-me";
const MIN_SECRET_LEN: usize = 32;

fn is_production_env() -> bool {
    match std::env::var("GATEFRAME_ENV") {
        Ok(v) => {
            let v = v.to_ascii_lowercase();
            v == "production" || v == "prod"
        }
        Err(_) => false,
    }
}

impl Config {
    pub fn from_env() -> Result<Self, String> {
        let cfg = Self {
            listen_addr: env::var("GATEWAY_LISTEN").unwrap_or_else(|_| "0.0.0.0:3000".into()),
            jwt_secret: env::var("JWT_SECRET").unwrap_or_else(|_| DEV_JWT_SECRET.into()),
            jwt_expiry_secs: env::var("JWT_EXPIRY_SECS")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(3600),
            internal_token: env::var("INTERNAL_TOKEN").unwrap_or_else(|_| DEV_INTERNAL_TOKEN.into()),
            user_service_url: env::var("USER_SERVICE_URL")
                .unwrap_or_else(|_| "http://127.0.0.1:8082".into()),
            audit_service_url: env::var("AUDIT_SERVICE_URL")
                .unwrap_or_else(|_| "http://127.0.0.1:8084".into()),
            workflow_service_url: env::var("WORKFLOW_SERVICE_URL")
                .unwrap_or_else(|_| "http://127.0.0.1:8085".into()),
            search_service_url: env::var("SEARCH_SERVICE_URL")
                .unwrap_or_else(|_| "http://127.0.0.1:8086".into()),
            notification_service_url: env::var("NOTIFICATION_SERVICE_URL")
                .unwrap_or_else(|_| "http://127.0.0.1:8087".into()),
            file_service_url: env::var("FILE_SERVICE_URL")
                .unwrap_or_else(|_| "http://127.0.0.1:8088".into()),
            redis_url: env::var("REDIS_URL").ok().filter(|s| !s.is_empty()),
            web_app_url: env::var("WEB_APP_URL")
                .unwrap_or_else(|_| "http://127.0.0.1:5173".into()),
            oidc_issuer_url: env::var("OIDC_ISSUER_URL").ok().filter(|s| !s.is_empty()),
            oidc_client_id: env::var("OIDC_CLIENT_ID").ok().filter(|s| !s.is_empty()),
            oidc_client_secret: env::var("OIDC_CLIENT_SECRET").ok().filter(|s| !s.is_empty()),
            oidc_redirect_url: env::var("OIDC_REDIRECT_URI").ok().filter(|s| !s.is_empty()),
            oidc_tenant_slug: env::var("OIDC_TENANT_SLUG").unwrap_or_else(|_| "default".into()),
            login_rate_limit_max: env::var("LOGIN_RATE_LIMIT_MAX")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(10),
            login_rate_limit_window_secs: env::var("LOGIN_RATE_LIMIT_WINDOW_SECS")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(60),
            cors_origins: env::var("CORS_ORIGINS")
                .unwrap_or_else(|_| "http://127.0.0.1:5173,http://localhost:5173".into())
                .split(',')
                .map(|s| s.trim().to_string())
                .filter(|s| !s.is_empty())
                .collect(),
            file_max_request_body_bytes: env::var("FILE_MAX_REQUEST_BODY_BYTES")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(70 * 1024 * 1024),
        };
        cfg.validate_production()?;
        Ok(cfg)
    }

    fn validate_production(&self) -> Result<(), String> {
        if !is_production_env() {
            return Ok(());
        }
        if self.jwt_secret.trim().is_empty() {
            return Err("JWT_SECRET must be set when GATEFRAME_ENV=production".into());
        }
        if self.internal_token.trim().is_empty() {
            return Err("INTERNAL_TOKEN must be set when GATEFRAME_ENV=production".into());
        }
        if self.jwt_secret == DEV_JWT_SECRET {
            return Err("JWT_SECRET must not use the development default in production".into());
        }
        if self.internal_token == DEV_INTERNAL_TOKEN {
            return Err(
                "INTERNAL_TOKEN must not use the development default in production".into(),
            );
        }
        if self.jwt_secret.len() < MIN_SECRET_LEN {
            return Err(format!(
                "JWT_SECRET must be at least {MIN_SECRET_LEN} characters in production"
            ));
        }
        if self.internal_token.len() < MIN_SECRET_LEN {
            return Err(format!(
                "INTERNAL_TOKEN must be at least {MIN_SECRET_LEN} characters in production"
            ));
        }
        Ok(())
    }

    pub fn oidc_enabled(&self) -> bool {
        self.oidc_issuer_url.is_some()
            && self.oidc_client_id.is_some()
            && self.oidc_client_secret.is_some()
            && self.oidc_redirect_url.is_some()
            && self.redis_url.is_some()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn production_rejects_dev_jwt_secret() {
        let cfg = Config {
            listen_addr: "0.0.0.0:3000".into(),
            jwt_secret: DEV_JWT_SECRET.into(),
            jwt_expiry_secs: 3600,
            internal_token: "a".repeat(MIN_SECRET_LEN),
            user_service_url: "http://127.0.0.1:8082".into(),
            audit_service_url: "http://127.0.0.1:8084".into(),
            workflow_service_url: "http://127.0.0.1:8085".into(),
            search_service_url: "http://127.0.0.1:8086".into(),
            notification_service_url: "http://127.0.0.1:8087".into(),
            file_service_url: "http://127.0.0.1:8088".into(),
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
            file_max_request_body_bytes: 1024,
        };
        std::env::set_var("GATEFRAME_ENV", "production");
        assert!(cfg.validate_production().is_err());
        std::env::remove_var("GATEFRAME_ENV");
    }
}
