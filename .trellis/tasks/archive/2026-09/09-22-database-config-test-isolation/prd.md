# 清理数据库配置与测试库回退

## 目标与用户价值

让仓库只暴露实际生效且边界清晰的数据库配置，避免开发者误以为服务使用多个业务库，
并确保真实 PostgreSQL 集成测试在缺少专用测试库配置时不会使用硬编码凭据或隐式连接
任何本地数据库。

## 背景与确认事实

- 服务端开发、验收和生产配置均只读取 `DATABASE_SOURCE`；仓库中没有运行时代码读取
  `.env.example` 的 `KRATOS_DATABASE_SOURCE`。
- `DATABASE_SOURCE` 指向日常开发/部署数据库；
  `RONCIN_INTEGRATION_DATABASE_SOURCE` 专供真实 PostgreSQL 集成测试使用。
- 公共集成测试助手会在测试库内创建随机隔离 Schema、执行正式迁移，并在测试结束后
  `DROP SCHEMA ... CASCADE`，因此测试库必须与日常开发库隔离。
- 大多数集成测试在未配置 `RONCIN_INTEGRATION_DATABASE_SOURCE` 时会明确跳过；但订单锁
  相关的三个入口会回退到包含本地用户名、密码和库名的硬编码连接：
  `server/internal/data/order_lock_integration_test.go`、
  `server/internal/data/order_auto_lock_integration_test.go`、
  `server/internal/data/order_lock_grant_integration_test.go`。
- 一次性财务验收脚本创建的 `roncin_acc_fin_*` 数据库具有独立的显式管理连接和生命周期
  清理机制，不属于本次问题。

## 需求

- 保留开发库和集成测试库的现有隔离设计，不修改数据库 Schema、迁移、数据或运行时连接
  逻辑。
- 从示例环境配置中移除未被运行时代码消费的 `KRATOS_DATABASE_SOURCE`，以
  `DATABASE_SOURCE` 作为服务端数据库连接的唯一示例配置。
- 删除上述三个订单锁测试入口对固定 `roncin_go_admin_integration` 地址及本地密码的
  回退；未显式提供 `RONCIN_INTEGRATION_DATABASE_SOURCE` 时，统一复用公共助手的明确
  `Skip` 行为。
- 更新与修改直接相关的注释或文档表述，保证文档仍准确说明开发库和测试库为何隔离、
  如何配置。
- 不引入兼容分支、默认测试凭据或自动探测本地数据库。

## 验收标准

- [x] 仓库有效配置中不再出现 `KRATOS_DATABASE_SOURCE`，服务配置仍通过
  `DATABASE_SOURCE` 正常表达唯一业务数据库连接。
- [x] 三个目标订单锁测试文件中不再包含硬编码 PostgreSQL 连接串或
  `roncin_local_dev` 密码。
- [x] 清除 `RONCIN_INTEGRATION_DATABASE_SOURCE` 后运行三个相关测试入口，测试均因缺少
  专用测试库配置而明确跳过，且不会尝试连接 `roncin_go_admin` 或
  `roncin_go_admin_integration`。
- [x] 显式配置专用测试库时，相关测试仍通过公共随机 Schema 机制运行；若本次环境未注入
  该变量，则以公共助手代码路径检查和未配置场景定向测试为验收依据，并记录未执行真实库
  场景的原因。
- [x] 修改通过 `gofmt`（如 Go 文件发生格式变化）、相关定向 Go 测试与
  `git diff --check`。

## 范围外

- 不合并开发库和集成测试库，也不创建、删除、重建或清空任何本地数据库。
- 不改变一次性验收数据库 `roncin_acc_fin_*` 的创建与清理流程。
- 不调整迁移、Ent Schema、业务数据模型、测试夹具内容或 PostgreSQL 系统库。
- 不顺带清理其他环境变量或重构全部集成测试基础设施。

## 关键决策与风险

- 此任务按轻量配置/测试安全修复处理，仅使用本 PRD，不新增设计文档和执行计划。
- 开发库与测试库隔离属于必要安全边界，本次只减少配置歧义，不减少隔离层级。
- 删除隐式回退后，未配置专用测试库的开发者会看到测试 `SKIP`；这是显式且安全的预期
  行为，真实 PostgreSQL 验证必须由调用方主动注入连接串。
- 工作区另有未提交的 Trellis 任务目录，属于其他并行工作，本次不得修改或提交；若实现
  期间用户的并行工作继续变化，也只提交本任务明确涉及的文件。

## 验收记录

- `env -u RONCIN_INTEGRATION_DATABASE_SOURCE go -C server test -v ./internal/data -run
  '^(TestOrderLock_PostgresFlows|TestOrderLockGrantScopes_PostgresFlows|TestAutoOrderLock_.*Postgres)$'
  -count=1`：包编译成功，六个真实数据库入口均在连接前明确 `SKIP`；此结果不记为真实
  PostgreSQL 测试通过。
- `go -C server vet ./internal/data`：通过。
- 三个 Go 文件的 `gofmt -d`：无输出。
- `git diff --check`：通过。
- 未执行显式连接专用 PostgreSQL 的路径，因为本任务禁止访问或写入数据库；已由公共助手
  路径审查确认显式配置后仍使用随机隔离 Schema。
- 规范回顾结论：`.trellis/spec/server/backend/quality-guidelines.md` 已明确记录“专用
  PostgreSQL 测试必须显式注入、禁止回退开发库、SKIP 不得算 PASS”，没有新增知识需要
  重复写入规范。
