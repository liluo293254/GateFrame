use std::net::SocketAddr;

use axum::{
    body::Body,
    extract::{ConnectInfo, State},
    http::Request,
    middleware::Next,
    response::Response,
};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use tracing::warn;

use crate::error::AppError;
use crate::state::AppState;

pub struct LoginRateLimiter {
    redis: ConnectionManager,
    max: u32,
    window_secs: u64,
}

impl LoginRateLimiter {
    pub async fn connect(redis_url: &str, max: u32, window_secs: u64) -> Result<Self, redis::RedisError> {
        let client = redis::Client::open(redis_url)?;
        let redis = ConnectionManager::new(client).await?;
        Ok(Self {
            redis,
            max,
            window_secs,
        })
    }

    async fn current_count(&self, ip: &str) -> Result<u32, AppError> {
        let key = format!("rl:login:{}", ip);
        let mut conn = self.redis.clone();
        let count: Option<u32> = conn.get(&key).await.map_err(|_| AppError::Internal)?;
        Ok(count.unwrap_or(0))
    }

    /// Returns 429 when this IP already exceeded failed login attempts in the window.
    pub async fn ensure_not_blocked(&self, ip: &str, metrics: &crate::metrics::Metrics) -> Result<(), AppError> {
        let count = self.current_count(ip).await?;
        if count >= self.max {
            metrics.inc_login_blocked();
            warn!(
                target: "rate_limit",
                event = "login_blocked",
                client_ip = %ip,
                failure_count = count,
                limit_max = self.max,
                window_secs = self.window_secs,
                path = "/api/auth/login",
                "login rate limit exceeded"
            );
            return Err(AppError::TooManyRequests);
        }
        Ok(())
    }

    /// Counts a failed login (401/403) toward the per-IP limit.
    pub async fn record_failure(&self, ip: &str, metrics: &crate::metrics::Metrics) -> Result<(), AppError> {
        let key = format!("rl:login:{}", ip);
        let mut conn = self.redis.clone();
        let count: u32 = conn.incr(&key, 1u32).await.map_err(|_| AppError::Internal)?;
        if count == 1 {
            let _: () = conn
                .expire(&key, self.window_secs as i64)
                .await
                .map_err(|_| AppError::Internal)?;
        }
        metrics.inc_login_failure();
        warn!(
            target: "rate_limit",
            event = "login_failure_recorded",
            client_ip = %ip,
            failure_count = count,
            limit_max = self.max,
            window_secs = self.window_secs,
            path = "/api/auth/login",
            "failed login counted toward rate limit"
        );
        Ok(())
    }
}

pub fn client_ip<B>(req: &Request<B>) -> String {
    if let Some(forwarded) = req.headers().get("x-forwarded-for") {
        if let Ok(s) = forwarded.to_str() {
            if let Some(first) = s.split(',').next() {
                return first.trim().to_string();
            }
        }
    }
    req.extensions()
        .get::<ConnectInfo<SocketAddr>>()
        .map(|ci| ci.0.ip().to_string())
        .unwrap_or_else(|| "unknown".into())
}

pub async fn login_rate_limit_middleware(
    State(state): State<AppState>,
    req: Request<Body>,
    next: Next,
) -> Result<Response, AppError> {
    let ip = client_ip(&req);
    if let Some(limiter) = &state.login_limiter {
        limiter.ensure_not_blocked(&ip, &state.metrics).await?;
    }
    let response = next.run(req).await;
    if let Some(limiter) = &state.login_limiter {
        let status = response.status().as_u16();
        if status == 401 || status == 403 {
            limiter.record_failure(&ip, &state.metrics).await?;
        }
    }
    Ok(response)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn client_ip_from_forwarded_for() {
        let mut req = Request::builder()
            .header("x-forwarded-for", "203.0.113.1, 10.0.0.1")
            .body(())
            .unwrap();
        assert_eq!(client_ip(&req), "203.0.113.1");
        let _ = &mut req;
    }
}
