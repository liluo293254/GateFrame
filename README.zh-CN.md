# GateFrame

**[English](README.md)** · **中文**

**企业私有自动化平台 · V0.1 基础设施**

> 可私有化部署的多租户运营底座 —— 安全、可审计、可扩展，为 V0.2 AI 能力预留架构空间。  
> **Automation First. AI Second.**

GateFrame 由**独立全栈开发者**从架构设计到 Gateway、微服务、管理端、CI 与本地开发环境完整交付，面向政企、制造、金融等对**数据不出域**有硬性要求的场景，也可作为 SaaS 厂商的私有化交付模板。

---

## 为什么选择 GateFrame

| 维度 | 能力 |
|------|------|
| **数据主权** | 默认私有化部署，不依赖外部 SaaS；PostgreSQL + MinIO + Redis，可完全离线运行 |
| **多租户** | 租户隔离贯穿 JWT、Gateway、Service、Repository；支持租户禁用与 Refresh 失效闭环 |
| **零信任边界** | Rust Gateway 统一鉴权、RBAC、审计、限流；剥离客户端伪造的内部 Header |
| **可审计** | 变更操作写入 audit-service；RBAC 变更、用户创建等可回溯 |
| **权限驱动 UI** | 菜单、按钮、路由均按 `resource.action` 权限码控制（非仅隐藏菜单） |
| **运维友好** | V0.1 搜索用 PostgreSQL FTS，无需 OpenSearch 集群；Prometheus 可选接入 |
| **工程化** | GitHub Actions CI + 全链路 smoke-test；ddev 一键拉起后端栈 |
| **国际化** | 管理端 i18n（`en` / `zh-CN`），文档与代码注释英文规范 |

---

## 架构一览

```
                    ┌─────────────────────────────────────┐
  Browser / Admin   │  web/  React + TanStack Query       │
  (host dev)        │  仅连接 Gateway，不直连微服务          │
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

**硬性约束（V0.1）：** Client → Gateway → Service → DB。禁止前端直连 Service，禁止 Gateway 写业务逻辑。各微服务独立 schema，跨服务访问仅经 Gateway 路由与 Internal Token。

---

## V0.1 已交付模块

| 模块 | API 能力 | 权限模式 |
|------|----------|----------|
| 用户与认证 | 登录 / Refresh / 权限列表 / OIDC（可配置） | JWT + 租户状态校验 |
| 租户 | CRUD、启用/禁用 | `tenant.read` / `tenant.manage` |
| RBAC | 角色、权限、角色-权限绑定 | `rbac.read` / `rbac.manage` |
| 工作流 | 列表 / 创建 / 更新 / **删除** | `workflow.read` / `workflow.manage` |
| 搜索 | PostgreSQL 全文检索、文档索引 / **删除** | `search.read` / `search.manage` |
| 审计 | 操作日志查询 | `audit.read` |
| 通知 | 站内通知 CRUD、已读 | `notification.read` / `notification.manage` |
| 文件 | 上传 / 下载 / 删除；类型与 50MB 限制 | `file.read` / `file.manage` |
| 仪表盘 | 聚合统计 | `user.read` |

管理端页面：`Dashboard` · `Users` · `Roles` · `Audit` · `Workflows` · `Search` · `Notifications` · `Files` · `Tenants`

---

## 安全与设计亮点

- **Gateway Pure**：只做认证、授权、审计、路由、租户 —— 业务逻辑全部在 Go 微服务
- **Service Pure**：每个服务独立 schema，跨域能力仅通过 Gateway + Internal Token
- **双层 RBAC**：Gateway 按 JWT claims 拦截；user-service 对敏感操作二次校验
- **登录限流**：Redis 滑动窗口，防暴力破解（smoke 覆盖 429）
- **OIDC**：Gateway 发起授权码流程，对接企业 IdP（Keycloak / Azure AD 等）
- **生产配置校验**：`GATEFRAME_ENV=production` 时拒绝弱 `JWT_SECRET` / `INTERNAL_TOKEN`
- **可观测**：Gateway `/metrics`（Prometheus 格式）；可选 Prometheus 抓取

---

## 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| API 网关 | Rust, Axum | 高性能安全边界 |
| 微服务 | Go 1.23, Gin | 按领域拆分，vendor 离线构建 |
| 管理端 | React, TypeScript, Vite, TanStack Query | Shadcn 风格组件 |
| 数据 | PostgreSQL 16 | 结构化数据 + FTS |
| 缓存 | Redis 7 | Session、限流 |
| 对象存储 | MinIO | file-service 后端 |
| 本地开发 | ddev + Docker | 后端容器化；前端宿主机 Vite |
| CI | GitHub Actions | Gateway test → Go test → Web build |

---

## 快速开始

**前置：** Docker Desktop · [ddev](https://ddev.com/) · Node 20+

```bash
# 1. 启动后端（PostgreSQL、Redis、Gateway、全部微服务）
ddev start

# 2. 端到端冒烟（RBAC、CRUD、文件、租户禁用、限流等）
./scripts/smoke-test.sh

# 3. 管理端（宿主机）
cd web && npm install && npm run dev
# 打开 http://127.0.0.1:5173
# 默认账号 admin / Admin@123456
```

| 端点 | 地址 |
|------|------|
| Gateway | `http://127.0.0.1:3002` |
| 前端 dev | `http://127.0.0.1:5173` |
| Prometheus（可选） | ddev compose overlay → `:9090` |

网络受限时可配置代理：`127.0.0.1:10808`（见 `.ddev/config.yaml`）。

## 质量保障

```bash
make test                 # Gateway + 全部 Go 服务 + Web 构建
./scripts/smoke-test.sh   # 需 ddev 运行中
```

- **CI**：`.github/workflows/ci.yaml` — push/PR 自动跑测试与构建  
- **Smoke**：覆盖登录、RBAC 403、CRUD、大文件/非法类型、审计、租户禁用 Refresh 401 等  
- **单测**：Gateway RBAC/代理/JWT；Go service 层测试；file 校验测试  

---

## 仓库结构

```
gateway/           # Rust API 网关
services/          # user · audit · workflow · search · notification · file
web/               # React 管理端
deploy/            # MinIO / Prometheus 等部署资产
.ddev/             # 本地 Docker Compose 栈（本地开发）
scripts/           # smoke-test.sh — 集成验证
```

微服务端口（经 Gateway 对外，内部）：user `8082` · audit `8084` · workflow `8085` · search `8086` · notification `8087` · file `8088`

---

## 路线图

**V0.1（当前）** — 企业自动化底座：用户、租户、RBAC、工作流、搜索、审计、通知、文件。

**V0.2（规划）** — 在相同安全与审计边界内扩展：OpenSearch、事件总线、Agent / RAG / 本地模型等。  
原则：**企业数据不出域 · AI 输出受 RBAC 约束 · 高风险操作需审批**。

---

## 关于作者 (Mail:liluo293254@gmail.com)

本项目为**个人开发者**独立完成的产品级工程实践，涵盖：

- 领域建模与多租户 RBAC 设计  
- Rust 网关与 Go 微服务实现  
- 权限驱动管理端与 i18n  
- ddev 开发体验、Docker 镜像优化、CI 与自动化冒烟  

适合作为**私有化交付起点**、**技术面试作品集**或**二次开发模板** Fork 扩展。

---

## License

暂未指定开源协议。如需商业授权或定制开发，请通过 Issue 联系。
