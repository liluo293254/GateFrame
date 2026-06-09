use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::AppError;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Claims {
    pub sub: String,
    pub tenant_id: String,
    pub permissions: Vec<String>,
    #[serde(default)]
    pub perm_ver: u64,
    pub exp: usize,
    pub iat: usize,
}

pub fn issue_token(
    secret: &str,
    user_id: Uuid,
    tenant_id: Uuid,
    permissions: Vec<String>,
    expiry_secs: u64,
) -> Result<String, AppError> {
    let now = jsonwebtoken::get_current_timestamp();
    let claims = Claims {
        sub: user_id.to_string(),
        tenant_id: tenant_id.to_string(),
        permissions,
        perm_ver: 1,
        iat: now as usize,
        exp: (now + expiry_secs) as usize,
    };
    encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(secret.as_bytes()),
    )
    .map_err(|_| AppError::Internal)
}

pub fn validate_token(secret: &str, token: &str) -> Result<Claims, AppError> {
    decode::<Claims>(
        token,
        &DecodingKey::from_secret(secret.as_bytes()),
        &Validation::default(),
    )
    .map(|data| data.claims)
    .map_err(|_| AppError::Unauthorized)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn issue_and_validate_roundtrip() {
        let uid = Uuid::new_v4();
        let tid = Uuid::new_v4();
        let perms = vec!["user.read".into()];
        let token = issue_token("test-secret", uid, tid, perms.clone(), 3600).unwrap();
        let claims = validate_token("test-secret", &token).unwrap();
        assert_eq!(claims.sub, uid.to_string());
        assert_eq!(claims.tenant_id, tid.to_string());
        assert_eq!(claims.permissions, perms);
    }

    #[test]
    fn reject_invalid_secret() {
        let uid = Uuid::new_v4();
        let tid = Uuid::new_v4();
        let token = issue_token("secret-a", uid, tid, vec![], 3600).unwrap();
        assert!(validate_token("secret-b", &token).is_err());
    }
}
