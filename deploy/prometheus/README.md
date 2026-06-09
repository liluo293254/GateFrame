# Prometheus for GateFrame

## ddev (local)

Prometheus is **optional** (avoids blocking `ddev start` when not enabled).

```bash
chmod +x scripts/enable-prometheus-ddev.sh
./scripts/enable-prometheus-ddev.sh
ddev restart
```

**No `docker pull` from Docker Hub** — the image is **built** from `deploy/prometheus/Dockerfile`, which downloads the official GitHub release during `docker build` using the same proxy pattern as Gateway:

- Build args: `HTTP_PROXY=http://host.docker.internal:10808`
- Ensure your proxy listens on **host** `127.0.0.1:10808`

If build still fails, check proxy is running and retry `ddev restart`.

- UI: http://127.0.0.1:9090
- Scrape config: `prometheus.ddev.yml` → `gateway:3000/metrics`
- Disable: `rm .ddev/docker-compose.prometheus.yaml && ddev restart`

After `ddev restart`, verify:

```bash
curl -s http://127.0.0.1:9090/-/healthy
curl -s 'http://127.0.0.1:9090/api/v1/query?query=login_failures_total'
curl -s http://127.0.0.1:3002/metrics | grep login_
```

## Production

1. Copy `prometheus.prod.example.yml` to your Prometheus config path.
2. Point `targets` at your Gateway instances (`host:3000` or service DNS).
3. Ensure `/metrics` is reachable from the Prometheus network (not public internet unless intended).

## Grafana (optional)

Import a dashboard with queries on `login_blocked_total` and `login_failures_total`.
