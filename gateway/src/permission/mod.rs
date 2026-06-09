//! Route-based RBAC permission checks (JWT claims).

use crate::tenant::TenantContext;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn wildcard_grants_all() {
        let ctx = TenantContext {
            tenant_id: "t".into(),
            user_id: "u".into(),
            permissions: vec!["*".into()],
        };
        assert!(ctx.has_permission("user.delete"));
    }

    #[test]
    fn exact_permission_required() {
        let ctx = TenantContext {
            tenant_id: "t".into(),
            user_id: "u".into(),
            permissions: vec!["user.read".into()],
        };
        assert!(ctx.has_permission("user.read"));
        assert!(!ctx.has_permission("user.create"));
    }
}
