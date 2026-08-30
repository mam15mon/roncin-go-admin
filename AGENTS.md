# Roncin Go Admin 开发约束

本文件是 `roncin-go-admin` 仓库的协作规范，适用于根目录及其子目录。目标是
支持 Roncin 货代后台的持续重构：保持边界清晰、变更可验证、生成物可追溯。

如果某个子目录存在更近的 `AGENTS.md`，先遵循更近文件中的目录专属约束；
它不能违反本文件的安全、依赖管理和提交要求。用户明确指令优先于本文件。

## 基本原则

- 所有文档、设计说明、提交说明和面向开发者的注释使用中文；代码标识符遵循
  现有语言惯例。
- 先确认目标、影响范围和验证方式，再修改代码。只实现当前需求，不做猜测性
  重构、无关格式化或一次性抽象。
- 禁止 TDD；功能实现后按风险补充针对性验证。修复缺陷时，至少验证触发场景
  和受影响的相邻路径。
- 未经明确要求，不添加自动回退、旧数据兼容、静默纠错或“顺便支持”的逻辑。
  发现兼容性风险时记录在变更说明中，不能用隐藏分支掩盖。
- 保留用户已有的未提交改动；修改前检查 `git status`，不要重置、覆盖或删除
  不属于本次任务的文件。
- 每完成一组可验证的修改就提交 Git。提交信息准确描述变更内容，使用
  Conventional Commits 前缀（如 `feat:`、`fix:`、`refactor:`、`docs:`、
  `test:`、`chore:`）。

## 仓库结构与职责

```text
server/                  Go 服务端（Kratos、Ent、PostgreSQL）
  api/                    Protobuf 契约与生成代码
  cmd/                    server、bootstrap-admin 等入口
  configs/                非敏感运行配置
  internal/access/        权限 Manifest 与权限码
  internal/service/       传输层 DTO 转换、校验、用例调用
  internal/biz/           领域对象、用例、仓储接口、业务规则
  internal/data/          Ent 仓储实现与持久化转换
  internal/platform/      日志、请求元数据等平台能力
  internal/server/        HTTP/gRPC 注册、中间件、静态资源服务
web/                      React + Ant Design Pro 管理后台
  config/                  Umi 配置、路由、代理、OpenAPI 输入
  src/pages/               按业务领域组织的页面
  src/services/roncin/    OpenAPI 生成的请求客户端
  src/components/         可复用界面组件
  tests/                   前端测试基础设施
scripts/                  根目录开发与构建辅助脚本
```

### 后端分层边界

- `server/api/` 是 HTTP/gRPC 契约的唯一真相源。业务接口先修改 `.proto`，再
  生成绑定代码；不得手改 `*.pb.go`、`*_grpc.pb.go`、`*_http.pb.go` 或
  `server/openapi.yaml` 等生成物。
- `internal/service` 只负责请求 DTO 与领域对象转换、边界参数校验和用例调用，
  不读取 Ent、不写业务规则、不直接访问数据库。
- `internal/biz` 持有纯 Go 领域对象、用例、仓储接口、领域错误和业务规则；不
  依赖 `service` 或 `data`，不得把 Protobuf/Ent 类型泄漏到领域对象。
- `internal/data` 实现 `biz` 仓储接口，负责 Ent 查询、持久化对象转换和驱动
  错误映射；Ent 类型只能停留在该层。
- `internal/server` 只负责传输层注册、中间件、健康检查和静态资源服务，不承
  担 DTO 转换或业务决策。
- `internal/access/manifest.go` 统一定义后端权限码。前端只消费
  `/api/v1/auth/me` 返回的权限集，不复制另一套权限真相。
- 运行日志使用结构化 JSON；业务审计日志独立存储。禁止记录密码、Cookie、
  令牌或完整敏感请求/响应报文。

### 后端列表分页约定

- 所有常规列表接口的 `pageSize` 最大值统一为 `200`，默认值仍按具体接口的交互
  需要设置；禁止在领域层或传输层另写 `100`、`500` 等不一致的上限。
- 分页上限统一复用 `internal/biz` 提供的公共常量与校验函数。新增列表接口时，
  必须验证 `page >= 1` 且 `1 <= pageSize <= 200`，并补充边界测试。
- 面向下拉框、联想输入等选择器的查询应支持服务端关键字过滤；前端无需为了
  获取候选项而自行循环翻页或一次性加载全量数据。
- 所有动态候选项的 `keyword` 查询必须覆盖业务代码、中文名称、英文名称、别名
  （字段存在时）、中文无声调全拼及拼音首字母；新增可搜索的中文候选实体时，
  应同步维护后端拼音检索键、索引和历史数据回填逻辑。

### 数据库事务与并发控制规范

- 事务一律通过 `internal/data` 提供的统一封装调用（`WithTx(ctx, fn)` 风格，
  封装内部负责 Begin、出错自动 Rollback、panic 安全回滚、成功 Commit）；
  禁止在仓储中手写 `db.Tx(ctx)` 加逐点 `tx.Rollback()` 再 `tx.Commit()`
  的分散模板，禁止混用 `sqlDB.BeginTx` 原生事务。存量手写事务在触碰
  相关文件时迁移到封装，迁移进度见根目录 `TODO.md`。
- 并发修改防护统一采用「悲观锁 + 乐观锁」双层模式，范本是
  `internal/data/order_write.go` 的 `UpdateDraft`：
  1. 事务内 `ForUpdate()` 锁定目标行；
  2. 比对 `existing.Version != expectedVersion`，不匹配立即返回
     `errors.Conflict` 业务错误（HTTP 409，中文提示「已被更新，请刷新后重试」）；
  3. 通过状态机检查后才允许变更；
  4. 更新时 `SetVersion(existing.Version + 1)`。
- 并发防护按实体分层取舍：多人协作编辑的业务实体（订单、账单、结算等）
  必须具备 `version` 字段与 `expectedVersion` 校验；状态流转、号码分配等
  原子操作使用 `ForUpdate` 加状态检查；低频单人维护的主数据可不加版本
  检测。新增实体时按此判定，判定不清时先向用户确认。
- 多行加锁必须先按主键排序再 `ForUpdate()`，固定加锁顺序防止死锁。
- 唯一性由数据库唯一索引兜底，驱动错误用 `ent.IsConstraintError` 统一
  判断并映射为业务错误。
- 设计事务、锁或版本字段时，遇到范围、命名、分层等不确定的情况，必须
  先向用户确认，禁止默认直接生成配置或代码。

### 前端边界

- 路由级权限使用 Ant Design Pro 的 `access`；按钮级权限复用同一权限判断结
  果，不在页面中硬编码第二套规则。
- 所有后端请求经过统一请求配置或 OpenAPI 生成客户端，禁止页面自行拼接后端
  主机地址。接口数据优先由 React Query 等服务端状态工具管理，不复制到全局
  可变状态。
- 页面按业务领域放在 `web/src/pages/`，页面专属请求、类型和样式就近存放；
  避免引入无关的大型聚合组件。
- `web/src/services/roncin/`、`web/types/` 中由 OpenAPI/Umi 生成的文件不得
  手工修改。应修改服务端契约后运行生成命令，并把源文件与生成物放在同一组
  变更中审阅。
- 前端依赖、脚本和校验统一使用 `pnpm`。不得新增 `npm`、`npx`、Yarn 入口；
  历史文档中的其他命令以本文件和 `package.json` 为准。

### 前端 UI 与页面模板规范

全站页面统一遵循纯白高密度企业级视觉规范，公共模板组件统一由 `@/components/ui` 导出：

- **整页分节表单与详情页 (Form / Detail Pages)**：
  - 页面顶部统一使用 `PageHeaderShell` 呈现吸顶导航（包含统一「返回列表」操作、层级路径、主副标题、数据编码 Tag 与主要操作按钮组）。
  - 各业务数据区块统一使用 `SectionCard` 包装，保持标题左侧 `3px × 15px` 品牌蓝竖标（`#1677ff`）、纯白卡片背景与细边框（`#f0f0f0`），支持 `collapsible` 折叠与右上角操作插槽。
  - 大表单/详情页底部统一使用 `StickyFooterBar` 提供吸底操作栏（左侧展示当前档案摘要，右侧放置取消与主提交按钮），避免用户必须滚动到最底部才能保存。
  - 订单多品类表单统一复用 `OrderFormTemplate` 及其配置式分节结构。
- **基础主数据与通用 CRUD 列表页 (Master Data / CRUD Pages)**：
  - 标准化配置类数据统一复用 `MasterDataTemplate`，内置顶部指标统计卡、关键字与下拉筛选栏、标准分页表格与快捷模态表单。
- **表格列表页 (Table List Pages)**：
  - 统一采用 Ant Design ProTable 高密度样式，搜索卡片与表格卡片外层统一细边框与微圆角，操作列靠右对齐。
- **侧边栏与菜单交互规范**：
  - 侧边栏折叠收起宽度基准为 48px，菜单项固定 36px 居中圆角卡片，折叠时彻底隐藏文本与展开箭头，保证图标正中居中。
  - 含有子级的菜单项（如「订单管理」）在折叠态下鼠标 Hover 必须弹出纯白圆角子菜单浮层（`.ant-menu-submenu-popup`）供点击直达。

## 生成代码与契约变更

契约变更遵循以下顺序：

1. 修改 `server/api/**/*.proto` 或配置源文件。
2. 在 `server/` 执行对应生成目标，检查生成差异。
3. 执行 `pnpm run generate:web-client` 更新 OpenAPI 输入和前端客户端。
4. 完成服务端、前端实现及验证后，在同一提交中包含必要生成物。

生成文件只通过生成器更新，不手工修补。若生成结果异常，先修正源文件或生成
配置，再重新生成。

修改 `server/internal/access/manifest.go`（新增、改名或删除权限码，或调整
`Requires` 依赖）后，执行 `pnpm run generate:permission-keys` 重新生成
`web/src/permissions.generated.ts`，使 `web/src/access.ts` 的权限键名在
编译期与后端清单对齐；该生成物同样与源文件放在同一提交中。

## 日常命令

在仓库根目录执行：

```bash
pnpm install
pnpm dev
pnpm run dev:web
pnpm run dev:server
pnpm run dev:permit
pnpm run generate:web-client
pnpm run generate:permission-keys
pnpm run check:web
pnpm run check:server
pnpm run check
pnpm run build
```

服务端生成与局部校验：

```bash
go -C server test ./...
go -C server vet ./...
go -C server generate
make -C server api
make -C server config
make -C server all
```

前端局部命令：

```bash
pnpm --dir web lint
pnpm --dir web test
pnpm --dir web tsc
pnpm --dir web biome:lint
```

优先运行与改动直接相关的最小检查，再根据风险运行 `pnpm run check` 或
`pnpm run build`。不要为了通过检查而关闭规则、跳过类型错误或提交临时产物。

## 配置、数据与部署

- 当前本地开发环境在 WSL 中直接运行 PostgreSQL（监听 `127.0.0.1:5432`）。
  `pnpm dev` 仅检查数据库是否就绪并执行迁移，不会启动或管理数据库服务；运行前
  确保 PostgreSQL 已启动，且具备 `DATABASE_SOURCE` 对应的数据库和用户凭据。
- 私密配置只通过环境变量注入；仓库只提交 `.env.example` 和不含凭据的示例
  配置。不要把真实密码、令牌、Cookie、连接串或生产数据写入 Git。
- PostgreSQL Schema 以 Ent Schema 为真相源。生产数据库变更必须生成、审阅
  并随代码提交迁移；不得绕过 Ent 在业务代码中散落手写 SQL（确有必要时说明
  原因并集中封装在 `internal/data`）。
- 权限目录随 `cmd/migrate` 自动同步：`pnpm run migrate:server`（生产发版的
  迁移步骤同样执行该命令）在迁移完成后按 `internal/access` 的 Manifest 幂等
  同步 `permissions` 表，并为 `administrator` 角色补挂缺失权限。新增权限码
  不需要单独跑 `pnpm run dev:permit`，该脚本仅保留作开发期手工兜底。
- 生产采用同域部署：Go 服务提供 `/api/*`、`/health/*` 以及 React 静态资源。
  修改静态资源打包或路由时，同时验证开发期代理和生产同域路径。

## 变更检查清单

提交前确认：

- 改动只涉及需求范围，且没有覆盖其他未提交工作。
- 契约、权限、日志和配置变更符合对应边界。
- 生成代码已按源文件重新生成，没有手工编辑生成物。
- 已运行与风险匹配的前端/后端检查，并记录无法运行的检查及原因。
- 没有提交秘密、调试输出、构建缓存或本地环境文件。
- 已用准确的 Conventional Commit 信息提交本组变化。
