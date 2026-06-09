# GateFrame Kubernetes Network Policies

## Purpose

Enforce **Client → Gateway → Service** at the network layer. Microservice ports (`8082`–`8088`) accept traffic **only** from the Gateway pod; browsers and other namespaces cannot reach services directly.

## Prerequisites

- CNI that supports `NetworkPolicy` (Calico, Cilium, etc.)
- Deployments label pods with `app.kubernetes.io/name` matching the selectors in `network-policy.yaml`
- Gateway Deployment label: `app.kubernetes.io/name: gateframe-gateway`

## Apply

```bash
kubectl apply -f deploy/k8s/network-policy.yaml
```

Adjust `namespace: gateframe` and ingress namespace selectors (`ingress-nginx`) to match your cluster.

## Verify

From a debug pod **without** the gateway label, a direct call to `user-service:8082` should time out or be refused. Through the Gateway Ingress URL, `/api/users` should work when authenticated.

## Related env (Gateway OIDC / perm version)

| Variable | Purpose |
|----------|---------|
| `REDIS_URL` | Login rate limit, `perm_ver`, OIDC state |
| `OIDC_ISSUER_URL` | Authentik/Keycloak issuer |
| `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` | OAuth client |
| `OIDC_REDIRECT_URI` | Gateway callback, e.g. `https://api.example.com/api/auth/oidc/callback` |
| `WEB_APP_URL` | Frontend base for post-SSO redirect |
| `OIDC_TENANT_SLUG` | Tenant slug for OIDC user linking (default `default`) |
