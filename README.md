# SmartEstate 智慧社区物业管理系统

> 面向业主、物业人员和管理员的数字化社区服务平台：在线报修、模拟缴费、公告发布、个人房产资料和物业工作台一体化管理。

## Docker 一键启动（推荐）

```bash
docker compose up -d
```

启动完成后访问：

- 前端：<http://localhost:18412>
- 后端健康检查：<http://localhost:19412/healthz>
- 后端 API：<http://localhost:19412/api/v1>

演示帐号密码均为 `password123`：业主 `13800000001`、物业 `13800000002`、管理员 `13800000003`。

## 主要功能

- **物业工作台**：汇总待办报修、待审访客凭证、本月已收费用和近期公告。
- **访客通行**：业主登记访客（姓名、手机号、到访时段、楼栋、事由）生成仅本时段有效的凭证；同一访客同一楼栋重叠时段仅一张有效凭证；物业审核、门岗核对进出；楼栋在场容量实时增减，满员暂停审核；取消/审核/进出/逾期全程留痕。
- **报修管理**：业主创建水电/家具/公共设施等报修；物业筛选、分配和更新进度。
- **费用缴纳**：按业主展示账单，通过支付宝沙箱模拟完成支付和记录查询。
- **社区公告**：置顶、发布、详情查看与阅读计数。
- **个人中心**：更新昵称、头像 URL，并绑定楼栋、单元和房间。
- **安全与治理**：JWT 登录态、RBAC、操作日志、敏感接口内存限流、统一 JSON 响应。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | Vue 3 + TypeScript + Vite + Element Plus + ECharts |
| 后端 | **Go 1.22 + Gin + GORM** |
| 数据库 | MySQL 8.0（本地开发未设置 DSN 时回退 SQLite） |
| 认证 | JWT + RBAC |
| 部署 | Docker Compose + Nginx 反向代理 |

## 本地开发（备选）

前端：

```bash
cd frontend
npm install
npm run dev
```

后端：

```bash
cd backend && go mod tidy && go run ./cmd/server
```

构建后端：

```bash
cd backend && go build ./...
```

默认本地后端采用 SQLite 文件 `backend/smartestate.db`。若要连接 MySQL，请设置 `DB_DRIVER=mysql` 和 `DB_DSN`。

## 常用 API 清单

所有业务接口均以 `/api/v1` 开头，并使用 `{ "code": 0, "message": "ok", "data": ... }` 响应包裹。除登录与健康检查外均需 `Authorization: Bearer <token>`。

| 方法 | 接口 | 用途 / 权限 |
| --- | --- | --- |
| POST | `/auth/login` | 登录（限流） |
| GET/PUT | `/users/me` | 获取或更新个人资料 |
| GET | `/users/staff` | 获取处理人员，`repair:manage` |
| GET/POST | `/repairs` | 工单列表 / 创建工单 |
| PATCH | `/repairs/:id/assign` | 分配处理人，`repair:manage` |
| PATCH | `/repairs/:id/status` | 更新进度，`repair:manage` |
| GET/POST | `/payments` | 账单列表 / 生成账单 |
| POST | `/payments/:id/pay` | 模拟支付（限流） |
| GET/POST | `/announcements` | 公告列表 / 发布，发布需 `announcement:publish` |
| GET | `/announcements/:id` | 公告详情并记录阅读 |
| GET | `/dashboard/summary` | 工作台汇总（含待审访客凭证数） |
| GET | `/operation-logs` | 操作日志，`log:read` |

访客通行模块（业主端、门岗 `visitor:gate`、物业审核 `visitor:review`）：

| 方法 | 接口 | 用途 / 权限 |
| --- | --- | --- |
| GET/POST | `/visitor/passes` | 凭证列表（业主仅本人）/ 业主登记 |
| GET | `/visitor/passes/:id` | 凭证详情与全生命周期留痕 |
| POST | `/visitor/passes/:id/cancel` | 业主/物业取消（仅待审核/已通过可取消） |
| POST | `/visitor/passes/:id/approve` | 物业审核通过，容量满返回 409，`visitor:review` |
| POST | `/visitor/passes/:id/reject` | 物业驳回，`visitor:review` |
| GET/PUT | `/visitor/capacity` | 楼栋容量总览 / 调整上限，`visitor:review` |
| GET | `/gate/verify?pass_no=` | 门岗核对凭证，返回 `allow` 与原因，`visitor:gate` |
| POST | `/gate/passes/:id/checkin` | 门岗办理进入，`visitor:gate` |
| POST | `/gate/passes/:id/checkout` | 门岗办理离开，`visitor:gate` |
| GET | `/gate/records` | 门岗最近进出/审核留痕，`visitor:gate` |

OpenAPI 摘要位于 `backend/api/openapi.yaml`。

## 项目结构

```text
.
├── frontend/
│   ├── src/api/                # user、repair、payment、announcement、visitor 请求
│   ├── src/stores/             # authStore、userStore、repairStore、paymentStore、visitorStore
│   ├── src/types/              # 共享实体和 permission 类型
│   ├── src/components/common/  # StatCard、RepairStatusBadge、RepairCard、PassStatusBadge、PassCard 等
│   ├── src/hooks/              # useAuth、useRepairStats、usePermission
│   ├── src/pages/              # Dashboard、Repairs、Payments、Announcements、Profile、Visits、VisitorReview、Gate
│   ├── src/router/             # 路由及 guards
│   ├── src/utils/              # request、roleText、feeCalculator、visitTime
│   └── src/constants/          # repair、user、visitor、errorCodes
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/{config,model,repository,service,handler,router,middleware,dto,constants,util}
│   ├── migrations/
│   ├── api/openapi.yaml
│   └── Dockerfile
├── database/init.sql
├── docker-compose.yml
└── .env.example
```

## 贯穿全栈的实体与分层

`User` 依次存在于数据库/GORM 模型、`model/user.go`、`repository/user_repository.go`、`service/user_service.go`、`handler/user_handler.go`、`router/users.go`、前端 `api/user.ts`、`stores/userStore.ts` 与共享类型。`Repair`、`Payment`、`Announcement` 均按模型→仓储→服务→处理器→路由→前端 API→页面/组件分层，CRUD 没有合并在单一文件。

**严禁合并职责到单一文件。** 当前实现按 `handler → service → repository → model` 单向依赖拆分。为了兼容原始练习的“牵一发动全身”约束，日志模板、权限码、枚举、格式化与提示文案也分散在指定常量/工具/路由/前端文件中；这是题目规定的高耦合演示设计，不应作为新生产系统的推荐模式。

## 枚举出现位置清单

### RepairStatus

- 后端定义：`backend/internal/constants/repair.go`；数据库 `Repair.status`；模型 `backend/internal/model/repair.go`。
- 后端使用：`backend/internal/service/repair_service.go` 状态机、`backend/internal/handler/repair_handler.go` DTO 校验、`backend/internal/constants/log_templates.go`、`backend/internal/util/formatter.go`。
- 前端定义：`frontend/src/constants/repair.ts`、`frontend/src/types/index.ts`。
- 前端使用：`frontend/src/components/common/RepairStatusBadge.vue`、`RepairCard.vue`、`frontend/src/pages/Repairs.vue` 的筛选器、`frontend/src/api/repair.ts`、`frontend/src/hooks/useRepairStats.ts`。

### UserRole

- 后端定义：`backend/internal/constants/user.go`；数据库 `User.role`；模型 `backend/internal/model/user.go`。
- 后端使用：`backend/internal/service/permission_service.go`、`backend/internal/middleware/auth.go`、`middleware/rbac.go`、路由权限与 `backend/internal/util/formatter.go`。
- 前端定义：`frontend/src/constants/user.ts`、`frontend/src/types/index.ts`。
- 前端使用：`frontend/src/stores/authStore.ts`、`frontend/src/hooks/useAuth.ts`、`usePermission.ts`、`frontend/src/router/index.ts` 的 meta、`router/guards.ts`、`components/common/PermissionButton.ts`、`utils/roleText.ts` 与 `App.vue`。

### PassStatus（访客凭证状态）

- 值：PENDING = 'pending'（待审核）、APPROVED = 'approved'（已通过）、CHECKED_IN = 'checked_in'（在场）、COMPLETED = 'completed'（已完成/已离开）、CANCELLED = 'cancelled'（已取消）、REJECTED = 'rejected'（已驳回）、EXPIRED = 'expired'（已逾期）。
- 状态机：`pending → approved → checked_in → completed`；`pending/approved → cancelled`；`pending → rejected`；`approved/checked_in → expired`（定时任务兜底）。
- 后端定义：`backend/internal/constants/visitor.go`；模型 `backend/internal/model/visitor.go` 的 `VisitorPass.Status`。
- 后端使用：`service/visitor_service.go` 状态机与事务、`repository/visitor_repository.go`（重叠与在场统计）、`util/formatter.go`（PassStatusText/PassActionText）、`constants/log_templates.go`、`constants/messages.go`、`constants/permissions.go`（`visitor:review`/`visitor:gate`）、`handler/visitor_handler.go`、`handler/gate_handler.go`、`handler/building_capacity_handler.go`、`router/visitors.go`、`cmd/server/main.go`（装配、迁移、逾期扫描协程）。
- 前端定义：`frontend/src/constants/visitor.ts`、`frontend/src/types/index.ts`。
- 前端使用：`components/common/PassStatusBadge.vue`、`PassCard.vue`、`pages/Visits.vue`（业主端）、`pages/VisitorReview.vue`（物业审核/容量）、`pages/Gate.vue`（门岗核对/进出记录）、`api/visitor.ts`、`stores/visitorStore.ts`、`router/index.ts`、`App.vue` 导航。
- 留痕：每次取消、审核、进入、离开、逾期都在同一数据库事务内写 `visitor_events` 与 `operation_logs`（见 `migrations/002_visitor_passes.sql`）。

### 容量规则（BuildingCapacity）

- 在场人数不做累加计数，而由 `visitor_passes.status='checked_in'` 实时统计：进入即减剩余、离开或逾期标记后自动恢复，杜绝计数器与实际状态不一致。
- 审核通过与办理进入两个节点都在事务内复核名额，达到上限返回业务冲突码 `40901` 并暂停新凭证审核。
- **并发安全**：所有状态变更在数据库事务内执行。先对 `building_capacities` 楼栋行加写锁（MySQL `SELECT … FOR UPDATE`，SQLite 用 WAL + busy_timeout 排队）将同楼栋操作串行化；再以条件更新 `UPDATE … WHERE id=? AND status IN (?)`（CAS，按 RowsAffected 判定）迁移状态。因此两名物业并发审核同一凭证或同一楼栋一批凭证时只有一个成功、已承诺名额（已通过+在场）绝不越上限；同一凭证并发办理进入/离开只成功一次，终态不被改写，失败请求明确返回 409 且不产生任何留痕。SQLite 文件库默认开启 `_journal_mode=WAL&_busy_timeout=5000`。
- **授权**：登记仅业主可发起（路由 `RequireRole(resident)` + service 双重校验）；取消仅凭证登记业主本人，物业/门岗/管理员不能代他人创建或取消；审核需 `visitor:review`，门岗进出需 `visitor:gate`。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `smartestate` | Compose 项目及容器名前缀 |
| `DB_NAME` / `DB_USER` / `DB_PASSWORD` | `smartestate_db` / `smartestate_user` / `smartestate_pwd` | MySQL 应用数据库与账号 |
| `DB_ROOT_PASSWORD` | `smartestate_root` | MySQL root 密码 |
| `JWT_SECRET` | `change_me_to_a_long_random_string` | JWT 签名密钥，生产必须替换 |
| `FRONTEND_PORT` / `BACKEND_PORT` / `DB_PORT` | `18412` / `19412` / `3306` | 对外端口 |

## Docker 部署说明

Nginx 提供 SPA 静态资源并将 `/api/` 代理至 Docker 内部的 `backend:8080`；浏览器前端只请求同源 `/api`。MySQL 使用命名卷 `smartestate_mysql_data` 持久化。数据库健康后才启动后端，后端通过 `/healthz` 健康后才启动前端。

常见问题：

1. 端口被占用时，修改 `.env` 中的 `FRONTEND_PORT`、`BACKEND_PORT` 或 `DB_PORT` 后重新执行 `docker compose up -d`。
2. 需重置演示数据时运行 `docker compose down -v`，这会删除 MySQL 持久化数据。
3. 中文路径可正常使用：Compose 的 build context 采用相对路径，未将宿主绝对路径传入容器。

## License

MIT
