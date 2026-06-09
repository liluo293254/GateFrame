use redis::aio::ConnectionManager;
use redis::AsyncCommands;

use crate::error::AppError;

pub struct PermVersionChecker {
    redis: ConnectionManager,
}

impl PermVersionChecker {
    pub async fn connect(redis_url: &str) -> Result<Self, redis::RedisError> {
        let client = redis::Client::open(redis_url)?;
        let redis = ConnectionManager::new(client).await?;
        Ok(Self { redis })
    }

    /// Rejects tokens whose perm_ver is older than the current Redis version.
    pub async fn ensure_current(&self, user_id: &str, token_ver: u64) -> Result<(), AppError> {
        let key = format!("perm:ver:{user_id}");
        let mut conn = self.redis.clone();
        let current: Option<u64> = conn.get(&key).await.map_err(|_| AppError::Internal)?;
        if is_stale_perm_ver(token_ver, current.unwrap_or(1)) {
            return Err(AppError::Unauthorized);
        }
        Ok(())
    }
}

fn is_stale_perm_ver(token_ver: u64, current: u64) -> bool {
    let token_ver = if token_ver == 0 { 1 } else { token_ver };
    token_ver < current
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stale_perm_ver_detected() {
        assert!(is_stale_perm_ver(1, 2));
        assert!(!is_stale_perm_ver(2, 2));
        assert!(!is_stale_perm_ver(3, 2));
    }

    #[test]
    fn legacy_zero_perm_ver_treated_as_one() {
        assert!(!is_stale_perm_ver(0, 1));
        assert!(is_stale_perm_ver(0, 2));
    }
}
