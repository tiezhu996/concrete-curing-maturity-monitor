# 混凝土养护成熟度监测

```bash
docker compose up -d
```

打开 `http://localhost:18535`。首次构建会启动 PostgreSQL、Go API 与 Nginx 前端，三个服务通过健康检查后才依次就绪。

本项目是面向工程试验室、施工技术团队与独立审核人的离线决策支持工具。它维护浇筑区段、版本化配合比、真实温度观测序列和不可覆写的强度预测，并保存请求级审计证据。

> 预测结果不能替代实体试块强度报告、现场核验或持证工程师签认。系统不连接、控制加热、喷淋、模板或任何现场设备。

## 主要功能

- 浇筑区段：建档、关联已发布配合比、乐观锁编辑及受控养护状态迁移。
- 配合比版本：维护基准温度和单调成熟度—强度标定点，按草稿、校验、发布、废止流转。
- 温度序列：导入真实 JSON 观测点，校验时间顺序、温度范围、缺失率并生成 SHA-256 校验和。
- 成熟度预测：使用 Nurse-Saul 公式和分段线性插值，冻结输入、公式版本、计算证据与阈值 ETA。
- 评审与审计：预测发起人不能确认自己的结果；写操作保存操作者、Request ID、前后快照和计算摘要。

## 角色账号

| 账号 | 密码 | 角色 | 主要权限 |
| --- | --- | --- | --- |
| `admin` | `admin123` | admin | 全部管理与审核权限 |
| `lab` | `lab123` | lab_engineer | 配合比、温度导入、运行预测 |
| `site` | `site123` | site_engineer | 区段维护与状态推进、运行预测 |
| `reviewer` | `reviewer123` | reviewer | 发布、评审、确认、审计读取 |
| `auditor` | `auditor123` | auditor | 只读数据与审计记录 |

演示账号只适用于本地环境。生产部署必须更换密码与 `JWT_SECRET`。

## 页面与工作流

| 页面 | 实体消费 | 关键操作 |
| --- | --- | --- |
| `/sections` | PourSection + MixDesign | 建档、搜索、状态迁移、查看最新预测 |
| `/mix-designs` | MixDesign + PourSection 引用 | 新建版本、标定证据、校验、发布、废止 |
| `/temperatures` | TemperatureSeries + PourSection | 导入、质量检查、确认、作废、成熟度曲线 |
| `/forecasts` | StrengthForecast + TemperatureSeries + MixDesign | 运行、证据、重放、评审、确认、作废 |
| `/audit` | 四实体审计投影 | 按实体和 Request ID 检索前后快照 |

## 架构与目录

```text
Browser -> Nginx :80 -> React SPA
                    -> /api/v1 -> Gin :8080 -> GORM -> PostgreSQL 16

backend/   cmd + config/constants/model/dto/repository/service/handler/router
frontend/  api/stores/types/hooks/components/pages/router/utils
database/  PostgreSQL 初始化脚本
output/    本地执行报告与 Browser 截图
```

后端使用构造器注入；handler 不直接访问数据库。状态变化使用带当前状态与版本条件的更新，预测幂等键同时绑定发起用户。正式服务使用 PostgreSQL，runtime smoke 使用内存 SQLite 并走同一迁移、种子与 HTTP 服务入口。

## API 清单

所有业务接口使用 `/api/v1`，响应包含统一 `code`、`message`、`data` 和 `request_id`。

| 方法与路径 | 说明 |
| --- | --- |
| `GET /healthz` | 数据库连通性健康检查 |
| `POST /api/v1/auth/login` | 登录并获取 JWT |
| `GET/POST /api/v1/pour-sections` | 区段列表与建档 |
| `GET/PUT /api/v1/pour-sections/:id` | 区段详情与乐观锁编辑 |
| `POST /api/v1/pour-sections/:id/transition` | 受控养护状态迁移 |
| `GET/POST /api/v1/mix-designs` | 配合比列表与版本草稿 |
| `POST /api/v1/mix-designs/:id/{validate,publish,retire}` | 配合比状态流 |
| `GET/POST /api/v1/temperature-series` | 温度列表与导入 |
| `POST /api/v1/temperature-series/:id/{confirm,invalidate}` | 序列确认或作废 |
| `GET/POST /api/v1/strength-forecasts` | 历史预测与幂等运行 |
| `POST /api/v1/strength-forecasts/:id/{review,confirm,void,replay}` | 预测审核与确定性重放 |
| `GET /api/v1/strength-forecasts/:id/compare/:other_id` | 同区段历史结果对比 |
| `GET /api/v1/audit-logs` | 审计记录筛选 |

错误使用明确 HTTP 状态：认证失败 `401`、权限不足 `403`、非法迁移或乐观锁冲突 `409`、无效标定或温度质量 `422`、限流 `429`。

## 公式与证据边界

逐区间使用梯形平均温度计算 Nurse-Saul 成熟度：

```text
M = Σ max(Tavg - T0, 0) × Δt
```

- 温度单位为摄氏度，时间单位为小时，成熟度单位为 `°C·h`。
- 当区间平均温度低于基准温度 `T0` 时，该段贡献截断为零，不累积负成熟度。
- 温度点严格按时间递增；推断缺失率超过 `MAX_MISSING_RATIO` 时拒绝导入并返回 `422`。
- 强度仅在已发布标定点上做单调分段线性插值。超出范围时夹取边界值并明确标记，禁止无依据外推。
- 目标强度超出标定范围时不提供 ETA。每次预测冻结区段、序列校验和、配合比版本和公式版本。

## 共享枚举位置

`CuringState = prepared | poured | curing | suspended | threshold_reached | closed`

- 后端：`constants/curing_state.go`，并由 model/dto、service 状态机、handler/router、算法上下文和测试消费。
- 前端：`types/enums/curing-state.ts`，并由 store、`CuringStateBadge`、区段/温度/预测页面和测试消费。

`ConfidenceLevel = low | medium | high`

- 后端：`constants/confidence_level.go`，并由预测算法、model/dto、service 和测试消费。
- 前端：`types/enums/confidence-level.ts`，并由预测 type/store、标签组件、页面和测试消费。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | React 18、TypeScript、Vite、Ant Design、Zustand、ECharts |
| 后端 | Go 1.22、Gin、GORM、JWT、bcrypt |
| 数据库 | PostgreSQL 16；SQLite runtime smoke |
| 部署 | Docker Compose、Nginx Alpine |

## 环境变量与端口

复制 `.env.example` 为 `.env` 并修改敏感值。仓库默认开发端口固定如下：

| 服务 | 宿主机 | 容器内 |
| --- | ---: | ---: |
| 前端 | `18535` | `80` |
| 后端 | `19535` | `8080` |
| PostgreSQL | `57535` | `5432` |
| runtime smoke | `20535` | 本机进程 |

关键变量包括 `DB_DSN`、`JWT_SECRET`、`JWT_EXPIRY`、三类接口限流值和 `MAX_MISSING_RATIO`。JWT 密钥至少 16 个字符。

## 本地开发

先启动 PostgreSQL，或使用 SQLite 开发配置，再分别运行：

```bash
cd backend
PORT=19535 DB_DRIVER=sqlite DB_DSN='file:local.db' JWT_SECRET='local-development-secret' go run ./cmd/server

cd frontend
npm ci
npm run dev
```

Vite 将 `/api/v1` 代理到 `http://127.0.0.1:19535`。前端代码始终使用相对 `/api` 路径，不保存模拟业务数据。

## 构建与测试

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...

cd backend
go build ./...
go vet ./...
go test ./...
go test -race ./...

cd ../frontend
npm ci
npm run test
npm run typecheck
npm run build
npm run audit:prod
```

运行标准 runtime smoke：

```bash
python3 /Users/gaobo/.codex/skills/go-annotation-pipeline/scripts/runtime_smoke.py .
```

脚本读取根目录 `runtime_smoke.json`，在 `20535` 启动真实 Gin 服务、访问 `/healthz` 后停止进程。

## Docker 部署与停止

```bash
docker compose config --quiet
docker compose up -d --build
docker compose ps
```

后端也可从宿主机访问 `http://localhost:19535/healthz`；前端代理健康端点为 `http://localhost:18535/api/healthz`。

停止并清理本项目容器、网络和数据库卷：

```bash
docker compose down -v --remove-orphans
```

## 常见问题

- `401`：JWT 缺失、过期或账号已停用，重新登录。
- `403`：当前角色没有对应权限；预测发起人也不能确认自己的结果。
- `409`：客户端版本或实体状态已变化，刷新列表后按最新状态重试。
- `422`：检查标定点单调性、温度时间顺序、缺失率或发布状态。
- ETA 为空：目标强度超出已发布标定范围，或近期成熟度速率不足，系统不会猜测时间。

## License

MIT License。工程使用方仍须自行遵守当地试验、质量和施工签认规范。
