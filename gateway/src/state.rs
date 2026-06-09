use std::sync::Arc;

use reqwest::Client;

use crate::auth::perm_version::PermVersionChecker;
use crate::config::Config;
use crate::limiter::LoginRateLimiter;
use crate::metrics::Metrics;

#[derive(Clone)]
pub struct AppState {
    pub config: Config,
    pub http_client: Client,
    pub login_limiter: Option<Arc<LoginRateLimiter>>,
    pub perm_version: Option<Arc<PermVersionChecker>>,
    pub metrics: Arc<Metrics>,
}

impl AppState {
    pub async fn new(config: Config) -> Result<Self, redis::RedisError> {
        let mut login_limiter = None;
        let mut perm_version = None;
        if let Some(url) = &config.redis_url {
            login_limiter = Some(Arc::new(
                LoginRateLimiter::connect(
                    url,
                    config.login_rate_limit_max,
                    config.login_rate_limit_window_secs,
                )
                .await?,
            ));
            perm_version = Some(Arc::new(PermVersionChecker::connect(url).await?));
        }
        Ok(Self {
            config,
            http_client: Client::new(),
            login_limiter,
            perm_version,
            metrics: Arc::new(Metrics::new()),
        })
    }
}
