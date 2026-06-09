# GateFrame

**English** · **[中文](README.zh-CN.md)**

**Enterprise Private Automation Platform · V0.1 Foundation**

> A privately deployable, multi-tenant operations foundation — secure, auditable, extensible, with architecture reserved for V0.2 AI capabilities.  
> **Automation First. AI Second.**

GateFrame is **delivered end-to-end by a solo full-stack developer**: architecture, Gateway, microservices, admin UI, CI, and local dev environment. It targets regulated industries (government, manufacturing, finance) where **data must not leave the network**, and serves as a **private-deployment template** for SaaS vendors.

---

## Why GateFrame

| Dimension | Capability |
|-----------|------------|
| **Data sovereignty** | Private deployment by default; no external SaaS dependency; PostgreSQL + MinIO + Redis; fully offline-capable |
| **Multi-tenancy** | Tenant isolation from JWT through Gateway, services, and repositories; tenant disable + refresh invalidation |
| **Zero-trust edge** | Rust Gateway for auth, RBAC, audit, rate limiting; strips client-forged internal headers |
| **Auditability** | Mutations recorded in audit-service; RBAC changes, user creation, etc. are traceable |
| **Permission-driven UI** | Nav, buttons, and routes gated by `resource.action` codes — not menu hiding alone |
| **Ops-friendly** | V0.1 search uses PostgreSQL FTS — no OpenSearch cluster; optional Prometheus |
| **Engineering rigor** | GitHub Actions CI + full smoke-test suite; one-command backend via ddev |
| **i18n** | Admin UI in `en` / `zh-CN`; docs and code comments in English |

---

## Architecture

```
                    ┌─────────────────────────────────────┐
  Browser / Admin   │  web/  React + TanStack Query       │
  (host dev)        │  Gateway only — no direct service   │
                    └─────────────────┬───────────────────┘
                                      │ HTTPS / JWT
                    ┌─────────────────▼───────────────────┐
                    │  gateway/  Rust · Axum               │
                    │  Auth · RBAC · Audit · Rate limit   │
                    │  Tenant check · OIDC · Routing      │
                    └─────────────────┬───────────────────┘
          ┌───────────────────────────┼───────────────────────────┐
          ▼                           ▼                           ▼
   user-service              workflow-service              file-service
   audit-service              search-service                (+ MinIO)
   notification-service       …                             …
          │                           │                           │
          └───────────────────────────┴───────────────────────────┘
                                      ▼
                            PostgreSQL 16 · Redis 7
```

**V0.1 rule:** Client → Gateway → Service → DB. No frontend-to-service calls; no business logic in the Gateway. Each microservice owns its schema; cross-service access uses Gateway routing and an internal token only.

---

## V0.1 Modules

| Module | API | Permissions |
|--------|-----|-------------|
| Users & auth | Login / refresh / permissions / OIDC (configurable) | JWT + active-tenant check |
| Tenants | CRUD, enable/disable | `tenant.read` / `tenant.manage` |
| RBAC | Roles, permissions, role-permission bindings | `rbac.read` / `rbac.manage` |
| Workflows | List / create / update / **delete** | `workflow.read` / `workflow.manage` |
| Search | PostgreSQL full-text search, index / **delete** docs | `search.read` / `search.manage` |
| Audit | Operation log query | `audit.read` |
| Notifications | In-app CRUD, mark read | `notification.read` / `notification.manage` |
| Files | Upload / download / delete; type & 50MB limits | `file.read` / `file.manage` |
| Dashboard | Aggregated stats | `user.read` |

Admin pages: `Dashboard` · `Users` · `Roles` · `Audit` · `Workflows` · `Search` · `Notifications` · `Files` · `Tenants`

---

## Security & Design Highlights

- **Gateway Pure** — Auth, authorization, audit, routing, tenant only; all business logic in Go services
- **Service Pure** — Each service owns its schema; cross-service access via Gateway + internal token only
- **Defense-in-depth RBAC** — Gateway enforces JWT claims; user-service re-checks sensitive operations
- **Login rate limiting** — Redis sliding window against brute force (429 covered in smoke tests)
- **OIDC** — Authorization-code flow at Gateway; integrates with enterprise IdP (Keycloak, Azure AD, etc.)
- **Production config guard** — Weak `JWT_SECRET` / `INTERNAL_TOKEN` rejected when `GATEFRAME_ENV=production`
- **Observability** — Gateway `/metrics` (Prometheus format); optional Prometheus scrape

---

## Tech Stack

| Layer | Tech | Notes |
|-------|------|-------|
| API Gateway | Rust, Axum | High-performance security boundary |
| Microservices | Go 1.23, Gin | Domain-split; vendor for offline builds |
| Admin UI | React, TypeScript, Vite, TanStack Query | Shadcn-style components |
| Database | PostgreSQL 16 | Structured data + FTS |
| Cache | Redis 7 | Sessions, rate limiting |
| Object storage | MinIO | Backend for file-service |
| Local dev | ddev + Docker | Containerized backend; Vite on host |
| CI | GitHub Actions | Gateway test → Go test → Web build |

---

## Quick Start

**Prerequisites:** Docker Desktop · [ddev](https://ddev.com/) · Node 20+

```bash
# 1. Start backend (PostgreSQL, Redis, Gateway, all services)
ddev start

# 2. End-to-end smoke (RBAC, CRUD, files, tenant disable, rate limit, …)
./scripts/smoke-test.sh

# 3. Admin UI (on host)
cd web && npm install && npm run dev
# Open http://127.0.0.1:5173
# Default login admin / Admin@123456
```

| Endpoint | URL |
|----------|-----|
| Gateway | `http://127.0.0.1:3002` |
| Frontend dev | `http://127.0.0.1:5173` |
| Prometheus (optional) | ddev compose overlay → `:9090` |

Proxy when network-restricted: `127.0.0.1:10808` (see `.ddev/config.yaml`).

## Quality Assurance

```bash
make test                 # Gateway + all Go services + web build
./scripts/smoke-test.sh   # requires ddev running
```

- **CI:** `.github/workflows/ci.yaml` — tests and build on push/PR  
- **Smoke:** login, RBAC 403, CRUD, large/invalid files, audit, disabled-tenant refresh 401, etc.  
- **Unit tests:** Gateway RBAC/proxy/JWT; Go service layer; file validation  

---

## Repository Layout

```
gateway/           # Rust API gateway
services/          # user · audit · workflow · search · notification · file
web/               # React admin
deploy/            # MinIO / Prometheus assets
.ddev/             # Local Docker Compose stack (local dev)
scripts/           # smoke-test.sh — integration verification
```

Internal service ports (via Gateway): user `8082` · audit `8084` · workflow `8085` · search `8086` · notification `8087` · file `8088`

---

## Roadmap

**V0.1 (current)** — Enterprise automation foundation: users, tenants, RBAC, workflows, search, audit, notifications, files.

**V0.2 (planned)** — Same security and audit boundaries: OpenSearch, event bus, agents / RAG / local models, etc.  
Principles: **data stays in the enterprise · AI output under RBAC · high-risk actions require approval**.

---

## About the Author (Mail:liluo293254@gmail.com)

This project is **production-grade engineering by a solo developer**, covering:

- Domain modeling and multi-tenant RBAC design  
- Rust gateway and Go microservices  
- Permission-driven admin UI and i18n  
- ddev workflow, Docker image optimization, CI, and automated smoke tests  

Suitable as a **private-deployment starter**, **portfolio piece**, or **forkable template** for extension.

---

## License

No open-source license specified yet. For commercial licensing or custom development, please open an Issue.
