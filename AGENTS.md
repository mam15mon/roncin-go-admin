# Roncin Go Admin 开发约束

本文件是 `roncin-go-admin` 仓库的协作规范，适用于根目录及其子目录。目标是
支持 Roncin 货代后台的持续重构：保持边界清晰、变更可验证、生成物可追溯。

如果某个子目录存在更近的 `AGENTS.md`，先遵循更近文件中的目录专属约束；
它不能违反本文件的安全、依赖管理和提交要求。用户明确指令优先于本文件。

## 当前交付阶段

- 项目当前处于正式上线前的快速交付阶段。默认优先完成可上线的最小业务闭环，
  在满足当前需求、核心数据正确性和必要验证的前提下，选择实现与维护成本最低的
  方案；没有已批准的现实用例时，不预建通用平台、复杂安全框架或面向远期扩展的
  重型抽象。可延期能力记录到任务说明或后续任务，不阻塞当前上线。
- 项目尚无需要维持的生产历史契约。除非用户或任务明确要求，默认不兼容历史数据、
  旧接口、旧行为或旧配置，不添加双读、双写、自动回退、兼容分支或跨版本适配层；
  契约变化时一次性同步仓库内服务端、前端、生成物、测试和开发数据。
- “不兼容历史”不等于可以跳过正式 Schema 迁移、生成流程或验证，也不授权自动删除、
  重置开发数据。需要清空或重建数据库时必须先取得用户明确授权。
- 快速上线不降低权限判定、组织范围、写入目标、事务原子性、并发一致性、秘密保护等
  当前业务正确性底线；但只实现已进入本期范围的最薄闭环，不为尚不存在的生产规模、
  多租户形态或合规要求提前建设完整治理体系。

## 基本原则

- 所有文档、设计说明、提交说明和面向开发者的注释使用中文；代码标识符遵循
  现有语言惯例。
- 先确认目标、影响范围和验证方式，再修改代码。只实现当前需求，不做猜测性
  重构、无关格式化或一次性抽象。
- 禁止 TDD；功能实现后按风险补充针对性验证。修复缺陷时，至少验证触发场景
  和受影响的相邻路径。
- 未经明确要求，不添加自动回退、历史兼容、静默纠错或“顺便支持”的逻辑。
  发现契约变更影响时一次性更新仓库内调用方，并在变更说明中记录；不能用隐藏
  分支掩盖。
- 保留用户已有的未提交改动；修改前检查 `git status`，不要重置、覆盖或删除
  不属于本次任务的文件。
- 每完成一组可验证的修改就提交 Git。提交信息准确描述变更内容，使用
  Conventional Commits 前缀（如 `feat:`、`fix:`、`refactor:`、`docs:`、
  `test:`、`chore:`）。
## 仓库结构与职责

> **AI 会话领域上手**：业务术语、模块地图、核心数据流与实体关系见
> `.trellis/spec/domain/`（glossary.md + architecture-map.md）；先读它再定位代码。


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
- 主数据维护页（非选择器列表页）按数据量选择分页模式，以服务器总资源（交互
  成本）最小化为目标而非单次请求成本：小数据集（约 5000 行以内，如货币、中国
  行政区划）采用「全量接口 + 前端分页」组合，后端提供无 `page/pageSize` 的
  全量列表接口（范本 `ListCurrencies`），前端 `MasterDataTemplate` 客户端模式
  本地完成搜索、筛选、翻页与统计，交互零服务器请求；大数据集（全球港口、
  全球机场等）一律服务端分页 + 服务端 `keyword` 过滤（范本 `PortsPanel`），
  全量加载反而放大传输、序列化与前端内存成本。
- 禁止前端为绕过 200 条上限循环翻页拼接全量；数据量跨过上述阈值时应调整接口
  契约，而不是在调用方堆补丁。
- 所有动态候选项的 `keyword` 查询必须覆盖业务代码、中文名称、英文名称、别名
  （字段存在时）、中文无声调全拼及拼音首字母；新增可搜索的中文候选实体时，
  应同步维护后端拼音检索键、索引和历史数据回填逻辑。

### 数据库事务与并发控制规范

- 事务一律通过 `internal/data` 提供的统一封装调用（`WithTx(ctx, fn)` 风格，
  封装内部负责 Begin、出错自动 Rollback、panic 安全回滚、成功 Commit）；
  禁止在仓储中手写 `db.Tx(ctx)` 加逐点 `tx.Rollback()` 再 `tx.Commit()`
  的分散模板，禁止混用 `sqlDB.BeginTx` 原生事务（存量手写事务已全部
  迁移到封装，触碰旧写法时直接改写为封装调用）。
- 跨仓储共享事务统一由 `biz.Transactor` 的 `WithinTransaction` 建立
  （实现见 `internal/data/transaction.go`）：用例在回调内取得 `txCtx`，
  多个仓储在同一事务中读写；嵌套 `WithinTransaction` 与 `WithTx` 自动
  合并到外层事务，事务结束后再使用 `txCtx` 会返回业务错误，不会静默
  开新事务。
- 仓储方法取得 Ent 客户端必须经由 `Data.client(ctx)`，禁止直连
  `d.db`/`r.data.db`：直连查询会静默落到共享事务之外，破坏原子性且没有
  编译期防护。事务内参与的读取按并发要求加 `ForShare`。
- `withSQLTx` 原生 SQL 事务不加入共享事务，禁止在 `txCtx` 回调中调用走
  原生事务的仓储方法；共享事务回调只做校验与写入，完整业务响应在事务
  提交后用普通上下文重读（范本：`internal/biz/finance_bill.go` 的
  `Create`），回调错误必须原样外传，不得吞掉后继续提交。
- 并发修改防护统一采用「悲观锁 + 乐观锁」双层模式，范本是
  `internal/data/order_write_draft.go` 的 `UpdateDraft`：
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

- **页面容器与宽度规范 (Container & Width Standards)**：
  - 全站页面统一以 `<PageContainer>` 作为顶级骨架，全局布局遵循 `web/config/defaultSettings.ts` 中的 `contentWidth: 'Fluid'` 流式全屏自适应，背景统一使用 `#f5f7fa`。
  - **严禁在任何页面、工作台或模板中私自硬编码 `maxWidth: 1440` 或自定义水平居中外层容器**；吸顶页头（`PageHeaderShell`）、业务分节卡片（`SectionCard`）、数据表格与吸底操作栏（`StickyFooterBar`）必须在任何分辨率下保持 100% 满屏平铺与贴边对齐，仅保留全局统一的 12px 内容区内边距。
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
pnpm run migrate:dev
pnpm run sync:all
pnpm run generate:web-client
pnpm run generate:permission-keys
pnpm run check:fast
pnpm run check:web
pnpm run check:server
pnpm run check
pnpm run build
```

服务端生成与局部校验：

```bash
go -C server test -p 32 ./...
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
pnpm --dir web test:changed
pnpm --dir web tsc
pnpm --dir web biome:lint
```

前端验证与全栈门禁加速规范：

- 开发过程中只运行受影响的定向测试和修改文件的 Biome 检查。
  - 定向单文件/少量文件测试：`pnpm --dir web exec vitest run <test-file> [more-test-files...]`（路径必须相对 `web/` 目录，如 `src/pages/...`；禁止使用 `pnpm --dir web test -- <test-file>` 冒充定向测试，它会触发全量执行）。
  - Git 改动文件增量测试：`pnpm --dir web test:changed`（利用 Vitest 仅测试受当前改动影响的用例）。
- 全量门禁极速验证：全栈验收时推荐优先运行 `pnpm run check:fast`。该脚本由 `scripts/check-parallel.mjs` 调度：
  - 前端门禁（`check:web`）与后端门禁（`check:server`）**全并发并行执行**，告别串行等待；
  - 自动挂载系统 `/dev/shm` 内存虚拟盘（tmpfs），将 `GOCACHE`、`GOTMPDIR` 与 Vitest 转换缓存全部内存化，消除磁盘写放大与 IOPS 瓶颈；
  - 后端单元测试统一指定 `-p 32` 充分发挥多核性能。
- 测试长尾效应防护（Longest Processing Time 约束）：
  - Vitest 按测试文件（Suite）粒度分发给 Worker 进程并发。**严禁在单个测试文件中堆砌超过 10 个重型业务组件/深层级树渲染用例**（如批量建账、复杂单据大表单）。
  - 当单测试文件执行耗时超过 20 秒时，必须按功能维度拆分文件（例如常规流程放 `*.test.tsx`，超大批量/规模压测放 `*.scale.test.tsx`），以便被调度器分配到不同的 CPU 核心并发执行，杜绝「1 个核心跑长尾、其他几十核围观」的 CPU 饥饿。
  - 前端单元测试已全局注入无动画规则（`setupTests.ts` 中置空过渡与关键帧时间），禁止在用例中写无意义的 `setTimeout` 帧等待。
- 一组代码准备普通本地提交时，运行相关测试、`git diff --check`，并根据类型影响
  决定是否运行 `pnpm --dir web tsc`。提交动作本身不要求运行全量测试、
  `check:web` 或构建。
- 整个任务或批准阶段完成、准备最终验收/归档/推送时，执行 `pnpm run check:fast`
  或风险匹配的前端门禁。`pnpm run build` 仅用于构建配置、依赖、生产入口等相关变更，或发布
  前验收及用户明确要求；不作为普通 UI 修改的固定步骤。
- 契约、权限、生成物、跨领域公共组件或大范围状态管理等高风险变更，可以提前或
  增加完整检查，但必须说明风险依据。

始终优先运行与改动直接相关的最小检查，再根据上述阶段与风险选择
`pnpm run check:fast`、`pnpm run check:web`、`pnpm run check` 或 `pnpm run build`。不要为了通过检查而
关闭规则、跳过类型错误或提交临时产物。

## 配置、数据与部署

- 当前开发环境在 Ubuntu / Linux 中直接运行 PostgreSQL（监听 `127.0.0.1:5432`）。
  `pnpm dev` 仅检查数据库是否就绪并执行迁移，不会启动或管理数据库服务；运行前
  确保 PostgreSQL 已启动，且具备 `DATABASE_SOURCE` 对应的数据库和用户凭据。
- 私密配置只通过环境变量注入；仓库只提交 `.env.example` 和不含凭据的示例
  配置。不要把真实密码、令牌、Cookie、连接串或生产数据写入 Git。
- PostgreSQL Schema 以 Ent Schema 为真相源。生产数据库变更必须生成、审阅
  并随代码提交迁移；不得绕过 Ent 在业务代码中散落手写 SQL（确有必要时说明
  原因并集中封装在 `internal/data`）。
- Ent 迁移全局状态与并发禁忌：Ent 包级表对象（`migrate.Tables`）在运行期
  执行 `client.Schema.Create` 或 Diff 时会被框架就地回写，存在隐式顺序依赖与
  全局指针污染；禁止在集成测试中并发执行 `Schema.Create`（涉及 Schema 初始化的
  测试禁止 `t.Parallel()`）；生产与集成验证严格以有序的正式 SQL 迁移文件
  （`server/migrations/*.sql`）为唯一真相源，不得依赖 Ent 运行期自动建表。
- 权限目录随 `cmd/migrate` 自动同步：`pnpm run migrate:server`（生产发版的
  迁移步骤同样执行该命令）在迁移完成后按 `internal/access` 的 Manifest 幂等
  同步 `permissions` 表，并为 `administrator` 角色补挂缺失权限。新增权限码
  不需要单独跑 `pnpm run dev:permit`，该脚本仅保留作开发期手工兜底。
- 行业参考大数据（全球船公司、联合国海港、全球航司、全球机场、中国省市区行政区划）
  源文件归档于 `server/seeds/`，新环境初始化或日常更新执行 `pnpm run sync:all`
  一键幂等落库，不重复新增。
- 迁移校验和按环境区分：`pnpm dev` 与 `pnpm run migrate:dev` 允许把已应用迁移
  的校验和重录为当前文件（开发期迁移文件在应用后继续修改属正常迭代）；
  `pnpm run migrate:server` 保持严格校验，禁止对生产库做任何自动修复。
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
