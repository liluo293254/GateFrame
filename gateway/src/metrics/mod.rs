use prometheus::{
    Encoder, IntCounter, Opts, Registry, TextEncoder,
};

#[derive(Clone)]
pub struct Metrics {
    registry: Registry,
    login_blocked_total: IntCounter,
    login_failures_total: IntCounter,
    audit_persist_failures_total: IntCounter,
}

impl Metrics {
    pub fn new() -> Self {
        let registry = Registry::new();
        let login_blocked_total = IntCounter::with_opts(Opts::new(
            "login_blocked_total",
            "Login attempts blocked by per-IP rate limit",
        ))
        .expect("login_blocked_total metric");
        let login_failures_total = IntCounter::with_opts(Opts::new(
            "login_failures_total",
            "Failed login attempts counted toward per-IP rate limit",
        ))
        .expect("login_failures_total metric");
        let audit_persist_failures_total = IntCounter::with_opts(Opts::new(
            "audit_persist_failures_total",
            "Audit events that failed to persist after retries",
        ))
        .expect("audit_persist_failures_total metric");
        registry
            .register(Box::new(login_blocked_total.clone()))
            .expect("register login_blocked_total");
        registry
            .register(Box::new(login_failures_total.clone()))
            .expect("register login_failures_total");
        registry
            .register(Box::new(audit_persist_failures_total.clone()))
            .expect("register audit_persist_failures_total");
        Self {
            registry,
            login_blocked_total,
            login_failures_total,
            audit_persist_failures_total,
        }
    }

    pub fn inc_login_blocked(&self) {
        self.login_blocked_total.inc();
    }

    pub fn inc_login_failure(&self) {
        self.login_failures_total.inc();
    }

    pub fn inc_audit_persist_failure(&self) {
        self.audit_persist_failures_total.inc();
    }

    pub fn render(&self) -> Result<String, prometheus::Error> {
        let encoder = TextEncoder::new();
        let families = self.registry.gather();
        let mut buffer = Vec::new();
        encoder.encode(&families, &mut buffer)?;
        Ok(String::from_utf8(buffer).unwrap_or_default())
    }
}

impl Default for Metrics {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn render_includes_rate_limit_counters() {
        let m = Metrics::new();
        m.inc_login_blocked();
        m.inc_login_failure();
        let body = m.render().unwrap();
        assert!(body.contains("login_blocked_total"));
        assert!(body.contains("login_failures_total"));
        assert!(body.contains("audit_persist_failures_total"));
    }
}
