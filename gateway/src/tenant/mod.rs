#[derive(Debug, Clone)]
pub struct TenantContext {
    pub tenant_id: String,
    pub user_id: String,
    pub permissions: Vec<String>,
}

impl TenantContext {
    pub fn has_permission(&self, required: &str) -> bool {
        self.permissions.iter().any(|p| p == required || p == "*")
    }
}
