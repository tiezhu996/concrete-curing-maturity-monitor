# gb-535 执行与验证记录

## 基本信息

| 项目 | 实测值 |
| --- | --- |
| 项目编号 | `gb-535` |
| 项目名 | `concrete-curing-maturity-monitor` |
| 验证完成时间 | `2026-08-22T06:52:02+08:00` |
| 实现提交 | `637384e9668beacf21b7f9da234102e7b584c05b` |
| Git 本地身份 | `blueship581 <brysj.hhrhl.g@gmail.com>` |
| 前端端口 | `18535` |
| 后端端口 | `19535` |
| PostgreSQL 端口 | `57535` |
| SQLite runtime smoke 端口 | `20535` |
| Go 功能规模 | 3858 行 / 42 个 `.go` 文件 |

本记录只陈述本项目的真实执行结果。系统仅提供离线成熟度计算和工程决策支持，不连接或控制现场设备；预测不能替代实体试块试验、现场核验或持证工程师签认。

## 构建、静态检查与测试

下列命令均在本项目中真实执行并以退出码 0 完成；报告落盘前又对无需恢复 Compose 数据的命令进行了复跑。

| 范围 | 命令 | 结果 |
| --- | --- | --- |
| 根工作区 | `go work sync` | 通过 |
| 根工作区 | `go build ./backend/...` | 通过 |
| 根工作区 | `go vet ./backend/...` | 通过 |
| 根工作区 | `go test ./backend/...` | 通过 |
| 根工作区 | `go test -race ./backend/...` | 通过，无数据竞争报告 |
| 后端目录 | `go build ./...` | 通过 |
| 后端目录 | `go vet ./...` | 通过 |
| 后端目录 | `go test ./...` | 通过 |
| 后端目录 | `go test -race ./...` | 通过，无数据竞争报告 |
| 前端依赖 | `npm --prefix frontend ci` | 通过，安装 270 个包 |
| 前端测试 | `npm --prefix frontend test` | 2 个测试文件、4 个测试全部通过 |
| 前端类型 | `npm --prefix frontend run typecheck` | 通过 |
| 前端生产构建 | `npm --prefix frontend run build` | 通过，Vite 6.4.3，5185 个模块 |
| 前端完整依赖审计 | `npm --prefix frontend audit --audit-level=high --registry=https://registry.npmjs.org` | 通过，0 vulnerabilities |
| 前端生产依赖审计 | `npm --prefix frontend run audit:prod` | 通过，0 vulnerabilities |
| 官方规模脚本 | `project_scale.py .` | 通过，3858 行 / 42 个功能 Go 文件，带前端页面 |
| 官方运行脚本 | `runtime_smoke.py .` | 通过，真实启动 Gin + SQLite，`http://127.0.0.1:20535/healthz` 返回 HTTP 200 |
| Compose 配置 | `docker compose config --quiet` | 通过 |

Go 普通项目测试覆盖 Nurse-Saul 成熟度、分段线性插值、预测证据、养护/预测状态机和温度序列质量规则。前端测试覆盖共享枚举及其迁移表、图表数据转换。生产构建生成的主脚本为 2289.71 kB（gzip 743.94 kB）；Vite 给出大块性能提示，但不影响本次功能、类型和部署验收。

## Compose 与健康检查

真实执行结果：

```text
docker compose config --quiet       PASS
docker compose up -d --build        PASS
docker compose ps                   db/backend/frontend 均 healthy
GET :19535/healthz                  HTTP 200
GET :18535/api/healthz              HTTP 200（经 Nginx 代理）
```

Compose 使用 PostgreSQL 16，三个服务实际监听 `18535/19535/57535`。前端只使用相对 `/api` 路径，Nginx 将 `/api/v1` 原样代理到 `backend:8080`；数据库、后端和前端均配置 healthcheck，并按健康条件启动。

## API 冒烟

在 Compose 的 PostgreSQL 场景中完成 39 条自动化 HTTP 检查，39/39 全部通过。检查包含响应体业务断言，不只是状态码；实际覆盖 HTTP 200、201、401、403、409、422。

| 验证组 | 真实链路与关键断言 | 实测状态 |
| --- | --- | --- |
| 健康与认证 | 后端健康检查、角色登录和无令牌访问 | 200、401 |
| RBAC | 管理、试验室、现场、审核、审计角色的写入边界，以及预测发起人与确认人隔离 | 200、403 |
| 浇筑区段 | 列表、详情、建档、合法状态迁移与非法迁移；非法跳转不写库 | 200、201、409 |
| 配合比版本 | 列表、详情、创建、校验、发布及无效标定点拒绝；历史发布版本保持不可变 | 200、201、422 |
| 温度序列 | 列表、详情、导入、checksum、质量校验、确认及无效时序/缺失率拒绝 | 200、201、422 |
| 强度预测 | 运行、读取、幂等重放、输入哈希防重、评审、独立确认与结果对比 | 200、201、403 |
| 审计投影 | 按 request ID、实体、操作者筛选四实体写入事件，并核对前后快照 | 200 |

预测幂等验证中，相同 `Idempotency-Key` 以及不同 key 但完全相同的冻结输入和公式版本均返回 forecast ID `1`，证明数据库复合唯一约束与服务层防重同时生效。另一历史结果的对比接口返回成熟度差 `10`、强度差 `0.5 MPa`，未覆盖原预测记录。

## Codex 内置 Browser 验证

浏览器验收全程只使用 Codex 内置 Browser，未启动或使用外部 Chrome。

1. 登录后在 `/sections` 真实创建区段 `B2-W09`，刷新列表确认落库，并完成 `prepared -> poured` 状态变化。
2. 在 `/mix-designs` 检查版本化配合比、基准温度、标定点及区段引用信息，数据来自真实 `/api/v1`。
3. 在 `/temperatures` 为区段导入 `TC-B2-W09-BROWSER`，核对温度点、checksum 和质量摘要，并真实确认该序列可用。
4. 在 `/forecasts` 运行并打开 `FC-0003` 的计算证据，核对成熟度、插值、ETA、置信等级与冻结版本；随后以独立审核身份完成 `completed -> reviewed -> confirmed`。
5. 在 `/audit` 核对上述区段、温度序列和预测的写入事件、request ID 及 before/after 快照。
6. 新开干净标签页依次访问五个核心页面，路由和标题均正确；`tab.dev.logs()` 返回 `[]`，捕获到 9 个 `/api/v1` 请求且全部 HTTP 200。
7. 桌面视口 `1440 x 1000` 下文档宽度为 1440，无页面级横向溢出、遮挡或阻断交互。
8. 移动视口 `390 x 844` 下文档 `clientWidth = scrollWidth = 390`；审计证据抽屉 `clientWidth = scrollWidth = 360`，长 JSON 可换行且没有横向溢出。

五个核心页面均消费真实后端数据；创建、导入、状态变化、运行预测、审核、确认和审计查询均不是静态 mock。

## 截图与校验值

| 页面/证据 | 视口 | 文件 | SHA-256 |
| --- | --- | --- | --- |
| 审计中心 | 1440 x 1000 | [audit-desktop-1440x1000.png](audit-desktop-1440x1000.png) | `3637090d3281f7e05b43fa91c2622f51addc19bc28d8a1bb619e1b8333698d4d` |
| 审计证据抽屉 | 1440 x 1000 | [audit-evidence-desktop-1440x1000.png](audit-evidence-desktop-1440x1000.png) | `3e63554b15ab41d19660c49756fe116988c06996e0d56fbca5aaeb054b80113a` |
| 审计证据抽屉 | 390 x 844 | [audit-evidence-mobile-390x844.png](audit-evidence-mobile-390x844.png) | `1d6c876eadddf314e1956c1d845483805b3def15c2ee387dfc9fd47a79b2435e` |
| 强度预测 | 1440 x 1000 | [forecast-desktop-1440x1000.png](forecast-desktop-1440x1000.png) | `3c8eafe0c2aa4b4a8fc6c5310be14f0658b576aa53c63d1d20e12f99f46c370e` |
| 温度序列 | 1440 x 1000 | [temperature-desktop-1440x1000.png](temperature-desktop-1440x1000.png) | `ed0191f630d361552875ac879bd5832f0a3b97665d7ccc23d5ba25a91214b0e4` |

## 需求覆盖

| 要求 | 实现与验证 |
| --- | --- |
| 四核心实体全链路 | `PourSection`、`MixDesign`、`TemperatureSeries`、`StrengthForecast` 均有 PostgreSQL 表和独立 Go model/dto/repository/service/handler/router、React type/api/store/page；API 冒烟逐一穿透验证 |
| 五核心页面 | `/sections`、`/mix-designs`、`/temperatures`、`/forecasts`、`/audit` 均加载真实 `/api/v1`；Browser 完成跨实体、跨角色业务流 |
| 共享前端能力 | `CuringStateBadge`、`MaturityChart`、`CalculationDrawer` 以及 `useAuth`、`useForecastRun` 已被多个页面复用 |
| 成熟度与强度算法 | Nurse-Saul 按梯形平均温度逐段累计，低于基准温度截零；版本化标定点执行单调分段线性插值，并保存覆盖率、外插标志、ETA、置信度和逐步证据 |
| 状态与历史 | 养护和预测状态机执行受控迁移；非法跳转 409；预测冻结输入、公式和配合比版本，幂等重放及结果对比不覆盖历史 |
| JWT 与 RBAC | 五类角色、JWT claims、后端 auth/rbac、React guard 和按钮显隐联动；401、403 与预测发起人不得自确认均真实验证 |
| 横切能力 | request ID、recovery、auth、RBAC、audit、统一错误、限流、幂等、结构化日志和优雅停机均实现 |
| 共享枚举 | `CuringState` 与 `ConfidenceLevel` 在 Go 常量、模型/DTO/算法/状态机和 React 类型/store/组件/页面/测试中保持一致，README 列明位置 |
| 技术与规模 | Go 1.22、Gin、GORM、PostgreSQL 16、SQLite smoke、React 18、TypeScript、Ant Design、Zustand、ECharts；3858 行 / 42 个功能 Go 文件 |
| 部署 | `db/backend/frontend` 三服务 healthcheck、健康依赖、命名卷、固定端口和 Nginx `/api` 代理均真实运行 |
| 工程安全边界 | README 与页面均明确不连接或控制加热、喷淋、模板或现场设备，预测不替代试块报告和工程签认 |

## 修复摘要

- 将预测防重从单纯依赖客户端 key 加固为“冻结输入哈希 + 公式版本”的服务校验与数据库复合唯一索引；相同输入即使更换 key 也复用原预测。
- 更新并锁定前端工具链依赖，完整依赖与生产依赖审计最终均为 0 vulnerabilities；随后复跑测试、类型检查和生产构建。
- 修复移动端审计抽屉中长 JSON 的换行和溢出，加入 `pre-wrap`、`overflow-wrap: anywhere` 与 `word-break: break-all`，并在 390 x 844 视口重新验证。

## 停服与残留检查

验收后真实执行：

```text
docker compose down -v --remove-orphans     PASS
docker compose ps -a                        仅表头，无服务
docker ps -a（项目名过滤）                  空
docker volume ls（项目名过滤）              空
docker network ls（项目名过滤）             空
lsof :18535/:19535/:57535/:20535            均无监听进程
```

仅删除本项目容器、默认网络和 PostgreSQL 命名卷，未执行全局 prune，也未停止或删除其他项目资源。报告落盘前再次核对，四个端口仍无监听进程。
