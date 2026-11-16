# 研究：账号展示偏好的跨浏览器持久化

- 研究问题：全站筛选和表头的所有展示设置如何按账号、当前工作区、稳定区域标识保存，并涵盖专有列设置、列宽、密度与高级配色。
- 研究范围：仓库内部；服务端契约、分层、Ent Schema、迁移与前端工作区／公共 UI 接入。
- 研究日期：2026-11-16（Asia/Shanghai）。

## 已检查的范围

读取了 `.trellis/workflow.md`、后端索引和数据库规范、前端状态管理／目录结构／列设置规范；检索 `server/api`、`server/internal/biz`、`server/internal/data/ent/schema` 的 preference、用户设置及 JSON 字段；精读以下入口。未运行迁移、生成器、产品测试或修改产品代码。

| 文件／锚点 | 职责与发现 |
| --- | --- |
| `server/api/auth/v1/auth.proto:67` | `/auth/me` 只返回当前主体、组织与权限，当前没有展示偏好。 |
| `server/api/auth/v1/passkey.proto:34` | 本人管理接口范式：AUTHENTICATED，无需新增业务权限码；个人展示偏好可采用同一访问模式。 |
| `server/internal/service/auth.go:181` | `RequirePrincipal(ctx)` 取得本人主体；展示偏好所有者必须由此得出，不能接受任意 userId。 |
| `server/internal/server/auth.go:42` | 未声明访问规则的 RPC 会拒绝；新增 proto 以后必须生成 access_rules。 |
| `server/internal/data/ent/schema/user.go:32` | 用户字段含登录身份、通讯录字段，没有通用用户偏好字段。 |
| `server/internal/data/ent/schema/finance_custom_setting.go:11` | 公司财务业务策略，有业务权限、审计和版本，不能挪用为个人 UI 存储。 |
| `server/internal/data/ent/schema/mixin.go:19`、`:39` | 新独立表可复用 UUIDv7 ID 和创建／更新时间。 |
| `server/internal/data/transaction.go:49`、`:100` | 多步写入走 WithTx；仓储取客户端走 Data.client，禁止自行 Begin／Rollback 模板。 |
| `server/internal/data/ent/generate.go:8` | 已启用 Ent `sql/lock,sql/upsert,sql/execquery`；可使用生成的 upsert 完成单区域原子整包写。 |
| `server/internal/data/migration_manifest_test.go:21` | 正式迁移链必须覆盖每个 Ent 表，创建顺序及外键亦被检查。 |
| `web/src/components/layout/OrganizationWorkspace.tsx:28`、`:55` | 每个 userId + currentOrganization.id 拥有独立 QueryClient；切换时同步 clear，适合作为偏好 Provider 生命周期边界。 |
| `web/src/utils/queryClient.ts:13` | 共用 Query／Mutation 错误策略，不自动重试、聚焦或网络恢复重查；新能力不能假设已有自动同步。 |
| `web/src/utils/requestClient.ts:38`、`:100`、`:121` | workspaceRequestScope 只约束旧响应展示，不终止网络，不向后端提供身份票据。 |
| `web/src/requestErrorConfig.ts:234` | 现有请求拦截器仅加 X-Request-ID 和写请求 in-flight 防重，不携带可校验的工作区 ID。 |
| `web/src/components/ui/filter-settings/useFilterSettings.ts:34` | 当前按本人 + 组织 + pageKey 读写 localStorage，只存显隐。 |
| `web/src/components/ui/column-settings/useColumnSettings.tsx:263`、`:326` | 通用表头偏好读写入口；当前只有同步本地存储，设置即时生效。 |
| `web/src/components/ui/column-settings/types.ts:16` | 当前存 order、hidden、freeze.left/right；不含列宽或密度。 |
| `web/src/components/ui/responsive-table/useResizableColumns.tsx:27`、`:31`、`:56` | 拖宽／双击展宽是独立实例状态，刷新会丢；必须接入账号保存，不能只改 useColumnSettings。 |
| `web/src/pages/orders/components/fees/feeColumnPreference.ts:274`、`:297` | 费用专有列偏好，按用户+组织，所有订单和应收／应付共用；字段权限与必显规则不能丢。 |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx:372`、`:400` | 专有偏好消费与保存位置，应与公共表头同时迁移。 |
| `web/src/pages/finance/fees/feeLedgerViewPreference.ts:30`、`:45` | 台账高级行配色独立存在 localStorage，必须纳入“所有设置”。 |
| `web/src/pages/finance/fees/index.tsx:200` | 7 类财务进度配色的即时保存和重置入口。 |
| `web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx:130`、`:246` | 行密度为实例 useState，onSizeChange 更新；不能漏掉已有密度选择。 |
| `web/src/components/ui/master-data-template/MasterDataTemplate.tsx:739` | 已开启 Pro density；与子实体抽屉等实例需一并盘点和接入，不能只改财务模板。 |
| `web/src/components/ui/column-settings/preference.ts:33` | 当前通用存储及合并逻辑；合并逻辑可保留，localStorage 读写应被服务端来源替代。 |

## 结论

在已检查的源目录中未发现现成的“个人展示偏好”后端。最薄方案是新增本人偏好服务及一张独立表，保存明确类型的筛选／表格展示配置，不复用公司财务业务策略，也不建立可写任意业务 JSON 的设置平台。

本次不能只替换通用列设置的 localStorage；必须覆盖四条已有路径：通用表头、费用专有列、响应式表格列宽、费用台账高级配色，以及各表格实际开放的密度选择。刷新、全屏是临时动作，不持久化；登录记住账号、通行密钥提示、表单草稿及未知提交意图不属于本任务。

## 推荐契约（规划候选，主会话收敛后实施）

### 批量读取，区域独立写入

建议独立 `DisplayPreferenceService`，放在 `server/api/auth/v1/display_preference.proto`；全部 RPC 使用 AUTHENTICATED：

| RPC／HTTP | 语义 |
| --- | --- |
| `GetMyDisplayPreferences`，GET `/api/v1/auth/me/display-preferences` | 返回本人当前工作区的完整展示配置聚合，按稳定区域标识索引；不是业务列表分页，也不循环请求分页。只返回已配置区域，缺失区域表示当前产品默认值。 |
| `PutMyDisplayPreference`，PUT `/api/v1/auth/me/display-preferences/{kind}/{region_key}` | 原子替换一个区域的一类完整配置，返回服务端成功确认的记录；不能替换整工作区而误覆盖其他区域。 |
| `DeleteMyDisplayPreference`，DELETE 同一路径 | 仅清除对应区域／类型，恢复当前产品默认；不存在时仍成功。 |

`kind` 最少 FILTER / TABLE 两类。FILTER 为显隐、完整顺序、字段跨度、折叠展示数；TABLE 为完整顺序、隐藏列、左右冻结、字段宽度、密度及可选高级行配色。使用 typed protobuf 消息，不提供任意 `settings_json` 写入入口；Ent 层可用自有领域结构的 JSON 字段保存，biz 不导入 proto／Ent。

每次 GET／PUT／DELETE 建议携带 `expected_scope`，其中 `user_id` 与 `organization_id` 都是纯前置条件，必须分别等于 principal.UserID 与 principal.Organization.ID；真正写入的所有者和工作区仍全部来自 principal。它不允许前端任选用户或公司目标，也不接受通过管理员权限修改他人偏好。

保留这一前置条件的依据：现有 request scope 仅丢弃旧响应，没有服务端身份票据。旧工作区排队回调若在 Cookie 已切换后误发，隐式 principal 方案可能把旧布局写入新公司；单独 expectedOrganizationId 又不能区分同公司切换账号。显式 expectedUserId + expectedOrganizationId 能在服务端拒绝这两类错位，仅作两次等值校验，不增加查询或授权模型。Provider 队列仍必须在销毁时作废所有未发送项，不能只依靠这些字段掩盖生命周期问题。批量读取响应也返回工作区标识，前端仅采纳匹配当前身份的结果。

### 最小存储

独立 Ent `DisplayPreference`：IDMixin、TimeMixin、不可变 user_id、不可变 organization_id、不可变 kind、不可变 region_key、typed 配置 JSON。数据库唯一索引 `(user_id, organization_id, kind, region_key)`。一条配置只属于一个稳定业务视图，不使用订单／账单 ID，不为每张单据单独生成配置。

同区域整包写可经 Data.client(ctx) 取得 Ent client 后，以唯一索引对应的原子 upsert 完成，不需“先查存在再决定插入”。如果最终实现出现多步写入需求，必须使用统一 WithTx，不自行 Begin。当前是本人低风险 UI 配置，不是多人业务单据，建议不增加 expectedVersion、ForUpdate 或业务冲突流程；同区域以最后成功提交的配置为准。此并发取舍应在最终规划摘要中由用户确认，不能把它当成当前仓库已有规则。

`region_key` 由代码稳定声明，限制长度与允许格式，不以中文标题、数组位置或实体 ID 猜测；当前 MasterDataTemplate／SubEntityDrawer 的默认 key 含 title／entityName，需要在覆盖清单中明确稳定 ID，不能简单实施 ASCII 限制后让这些既有入口保存失败。

数据边界校验应限制配置大小、字段数量／标识长度、重复字段、宽度范围、span、密度枚举与颜色格式；数值范围由主方案确定，不在研究中擅自改变产品行为。服务端无需复制每个页面完整字段元数据，客户端仍按真实声明执行必显、权限裁剪、结构列、分组和冻结规则；偏好永远不能授权业务数据。

## 前端最薄集成

### 中立公共能力，不反向依赖 features

推荐 `web/src/components/ui/display-preferences/`：api、typed mapper、DisplayPreferencesProvider、useDisplayPreference 与公开 index。它只依赖生成的 services/roncin、React Query、appInitialStateContext 及中立 utils；不能放在 features 后被 components/ui 导入。

依据 `.trellis/spec/web/frontend/directory-structure.md:43`，components/ui、hooks、utils、constants 均禁止导入 pages/features。列设置、筛选模板、响应式表格是中立 UI；领域特有的费用字段定义、行颜色默认值继续由所属页面传入，不提升到中立 Provider。

在 OrganizationWorkspace 的 keyed QueryClient 内、Antd App 内挂载 Provider；每个用户／组织生命周期只批量读取一次偏好聚合，各区域从同一份工作区 query cache 取得配置。该聚合是服务端状态，不把副本存到全局可变对象。

读取未完成／失败须与真正不存在偏好区分。失败显示可重试，不读取旧 localStorage，也不把默认值标作已同步；读取完成前禁用设置写入，避免默认布局覆盖远端配置。首屏加载、页签恢复与初次业务查询需要使用同一就绪状态，避免先按默认字段请求、随后恢复设置又错误清除查询。具体 UI 阻塞粒度由主设计选择，复用现有加载表现，避免额外启动闪屏。

### 同区域顺序与错误处理

- 用户每次操作即时应用本次布局；保存由 Provider 按 user+org+kind+regionKey 独立串行。队列属于工作区，不属于打开的弹窗，因此关闭设置面板或普通换页不应丢失已接受的保存动作。
- 同一区域快速连续操作始终在最新本地配置上组合，再入队完整快照；一个在途写结束后发送最新待提交快照，不能让较早响应覆盖用户更晚操作。不同区域可并发。
- 成功响应只更新对应区域的确认状态，不能以其旧 payload 覆盖更新的展示意图；读聚合迟到也不能覆盖已接受本地操作。
- 写失败保留当前可见设置并明确提示“尚未保存到账户”；提供当前区域重试。不要宣称换浏览器已保存，也不默默退回本地保存模式。
- 恢复默认走同一队列，不能与在途旧 PUT 乱序，不能在失败时把错误当成默认已经保存。
- 账号／组织变化当次丢弃旧工作区显示状态、旧错误与旧成功回调；销毁时取消读取并作废未发送写项。已发送写入只能作用于原身份／工作区，不回写新 QueryClient。
- 同账号两个浏览器同时改同区域：最后成功提交整包生效，不做实时推送或多人合并。新登录／刷新读取最终值；是否需要前台已打开旧页面即时感知不属于用户当前明确要求，应在最终范围说明“不做实时推送”，而不能暗示已实现实时同步。

### 表格专有接入风险

- `useColumnSettings`、费用专有配置和 `useResizableColumns` 必须共用稳定区域标识与账号 TABLE 配置。不能从行数据、列中文标题或当前列下标反推保存身份。
- 建议显式把区域 key 或受控宽度／更新入口传入三类 ResponsiveTable，而不是在列数组上附带隐式全局元数据。共享模板只声明一次常量，再传给列设置和响应式组件。
- 列宽按字段稳定身份保存；分组头／选择／展开等辅助列仍按原交互规则处理。窄屏临时解冻、浏览器实际多分配的空白宽度不能写回用户偏好。
- 拖宽松手和双击展宽／还原都是用户设置动作，需保存；测量过程与拖动参考线不保存。保留编辑输入、分页、选择、官方请求及实例，不通过重挂表格恢复列宽。
- 财务 advanced 行颜色是七个稳定状态槽，按领域适配并验证合法颜色；普通公共 Provider 不依赖 FinanceLedgerTemplate 的领域类型。
- 密度应把现有 ProTable `size`／`onSizeChange` 合成受控持久化，但保留调用方已有回调、默认密度和 disabled 约束，不能新增全站第二套密度入口。
- 公共“恢复默认”应明确重置范围：表格整体恢复可重置该 TABLE 全部配置；高级“重置默认颜色”只更新颜色部分，不能顺便清空列宽、冻结或密度。操作最终写入仍是最新合并 TABLE 全快照。

## 生成、迁移与验证计划建议

### 生成链路

1. 先修改 `.proto` 和 Ent Schema；`make -C server api` 生成绑定、OpenAPI 及 access_rules。
2. `go -C server generate ./...` 更新 Ent、Wire 等生成物；检查新增服务 ProviderSet、HTTP／gRPC 注册与命令入口 Wire 结果（注册模式见 `server/internal/server/http.go:48`、`grpc.go:48`、`service/service.go:6`、`data/data.go:35`）。
3. 基于 Ent Schema 生成并审阅正式 SQL 迁移，提交源与生成物；当前未找到仓库已提交的通用 schema-diff 命令，实施时核对工具，不手改生成代码。不能为迁移生成或验证重置真实开发库。
4. `pnpm run generate:web-client` 同步前端输入、请求客户端和 proto 常量。若无新权限码，无需修改 manifest 或生成权限键；AUTHENTICATED 仍必须生成 RPC 访问规则。
5. 新增独立 PostgreSQL Schema，按正式 migration.Apply 路径验证空库与升级；禁止 Schema.Create／并行全局迁移对象污染。

### 有价值的针对性验证

- Biz／Service：未登录、缺组织、expectedUserId／expectedOrganizationId 不匹配、本人所有权、非法类型／字段／跨度／宽度、无配置与空数组／显式零值存在性语义。
- Data：真实 PostgreSQL upsert 唯一性，两次并发首次保存不重复，同区域最后提交有效，不同用户／组织／区域独立；DELETE 只清目标，正式迁移与 Ent 元数据一致。
- Provider：一次批量读取、缓存共享、失败可重试、连续 A→B→C 串行且旧响应不覆盖 C、失败→新操作→重试、DELETE 与 PUT 顺序、组件关闭／切页、账号／组织切换与迟到读写。
- 通用表头、费用专有列、列宽三类 ResponsiveTable、密度与高级配色逐项回归；重新进入页面的设置不得与分页／请求条件／编辑草稿混淆。
- 真实跨浏览器验收使用两个独立 BrowserContext／profile，禁复制 localStorage；浏览器 A 保存，B 同账号同公司读取，B 改后 A 刷新验证；另账号／另公司不得串用；断网保存明确未同步。
- 生成客户端测试使用真实 wire 形态：省略默认值、显式零值、非零值、optional 未设置、真正必填缺失；不只用字段齐全 mock。
- 分组提交前定向测试 + 修改文件 Biome + diff check；跨层公共状态与生成物任务完成后 `pnpm run check:fast`，必要真实 PostgreSQL 复验；并非构建配置变更，不固定增加生产 build。

## 相关规范

- `.trellis/spec/server/backend/database-guidelines.md`
- `.trellis/spec/server/backend/directory-structure.md`
- `.trellis/spec/server/backend/auth-session-org-switch.md`
- `.trellis/spec/server/backend/passkey-authentication.md`
- `.trellis/spec/web/frontend/column-settings.md`（本次批准会替代“列宽只存在当前实例”和“偏好仅本浏览器”的旧规则）
- `.trellis/spec/web/frontend/directory-structure.md`
- `.trellis/spec/web/frontend/state-management.md`
- `.trellis/spec/web/frontend/organization-switch.md`
- `.trellis/spec/web/frontend/type-safety.md`

## 外部引用与限制

未使用外部资料。React Query 版本由 `web/package.json:40` 和 pnpm-lock.yaml 确认是 5.103.1；未找到可直接访问的 query-core 源文件，不据此声称某个 mutation scope API 已验证可用。规划只要求区域顺序约束，实施时可据已安装官方 API 选择最薄实现。

本研究是方案依据，不是已经实施或通过验证的证明。全站筛选逐页覆盖清单由另一研究负责；本文件未声称完整逐页阅读全部表格。未发现通用偏好的结论限于上文明确检索目录。是否采用本人配置无业务版本锁、最后成功提交生效、首次读取阻塞粒度及恢复默认范围，需要主会话按最终规划摘要收敛。
