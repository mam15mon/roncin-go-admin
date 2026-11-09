# 研究： 全站业务标签契约、权限与运行期退役

- 研究问题： 清点业务标签 RPC/DTO/领域/仓储/Provider/Wire/权限依赖，给出不兼容历史的最薄完整退役方案。
- 范围： internal（仓库源与现行规范；未访问数据库）
- 日期： 2026-11-09

## 研究结果

### 1. 唯一删除对象与必须保留边界

这里的“标签”是企业资源 TAG 资产及其与订单、订单费用、财务账单的人工多对多关联，不是 AntD Tag 组件、运输方式 SEA 等字典属性、客商人员候选标识、汇差提示或 TagsView 页签。

- `server/internal/biz/business_tag.go:24` 的 BusinessTagRepo 明确三类业务目标；全部能力由该仓储与 BusinessTagUsecase 持有，不参与金额计算、财务状态机或订单流程。
- `server/internal/data/business_tag.go:14` 实现候选、读关联、批量写关联和使用计数，可在正式退役后整个删除。
- `server/internal/biz/enterprise_resource.go:31` 的 EnterpriseResourceTagType 是通用企业资源类型之一，删除该分支，不删除资源备忘本身。保留地址、备注、图片、发货人/收货人/通知人和英文货名/HS/唛头能力。

### 2. 完整 RPC 删除清单（18 个）

| Proto / 现行位置 | 待删除 RPC | HTTP 前缀 |
| --- | --- | --- |
| `server/api/order/v1/order_tag.proto:13` | ListOrderTagOptions、BatchAssignOrderTags、BatchRemoveOrderTags（整个独立 service/source 文件） | `/api/v1/order-tag-options`、`/api/v1/order-tags/batch-assign`、`batch-remove` |
| `server/api/order/v1/order_fee.proto:152` | ListOrderFeeTagOptions、BatchAssignOrderFeeTags、BatchRemoveOrderFeeTags | `/api/v1/orders/{order_id}/fee-tag-options`、`fee-tags/batch-assign`、`batch-remove` |
| `server/api/finance/v1/settlement.proto:70` | ListFinanceBillTagOptions、ListFinanceBillTagAssignmentOptions、BatchAssignFinanceBillTags、BatchRemoveFinanceBillTags | `finance/bill-tag-options`、`bill-tag-assignment-options`、`bill-tags/batch-*` |
| 同文件 `:88` | ListFinanceFeeTagOptions、ListFinanceFeeTagAssignmentOptions、BatchAssignFinanceFeeTags、BatchRemoveFinanceFeeTags | `finance/fee-tag-options`、`fee-tag-assignment-options`、`fee-tags/batch-*` |
| `server/api/enterprise_resource/v1/enterprise_resource.proto:80` | ListEnterpriseTagGroups、CreateEnterpriseTagGroup、UpdateEnterpriseTagGroup、DeleteEnterpriseTagGroup | `/api/v1/enterprise-tag-groups` 及 `/{id}` |

相应独占 request/response messages 一并删除，不留 Unimplemented/空返回兼容端点。订单标签独立 Proto 删除后，Buf 不保证删除旧生成文件；精确移除对应 `order_tag.pb.go`、`order_tag_grpc.pb.go`、`order_tag_http.pb.go`，正常生成其他聚合文件，禁止手工修补生成内容。前端 `orderTagService.ts` 通过客户端生成器清理。

### 3. 删除字段、消息与编号保留

下表均应在**仍存在的所属消息**同时 `reserved <编号>; reserved "<原字段名>";`，避免把历史标识重用于其他用途；不是建立旧接口兼容。

| 源文件/所属类型 | 移除字段 | 当前锚点 |
| --- | --- | --- |
| `order.proto` / Order | `tags = 79` | :407 |
| `order.proto` / ListOrdersRequest | `tag_ids = 35` | :572 |
| `order_fee.proto` / OrderFee | `tags = 36` | :208 |
| `settlement.proto` / ListFeeLedgerRequest | `tag_ids = 15` | :294 |
| `settlement.proto` / FeeLedgerItem | `tags = 34` | :335 |
| `settlement.proto` / ListBillsRequest | `tag_ids = 10` | :486 |
| `settlement.proto` / FinanceBill | `tags = 36` | :592 |
| `enterprise_resource.proto` / EnterpriseResource | oneof detail 内 `tag = 13` | :216 |
| 同文件 / EnterpriseResourceInput | oneof detail 内 `tag = 12` | :237 |

- `enterprise_resource.proto:111` 移除 enum `ENTERPRISE_RESOURCE_TYPE_TAG = 4`，在枚举内保留编号 4 和名称；SHIPPER=5、CONSIGNEE=6、NOTIFY_PARTY=7、后续8/9/10不重排。
- `order.proto:278` 的 BusinessTagSummary 与 `settlement.proto:1105` 的同名概要均只为标签功能服务；三类字段与全部候选 RPC 删除后一起删除。
- `enterprise_resource.proto:200` EnterpriseResourceTag、`:277-286` EnterpriseTagGroup/Input 和组 CRUD 消息均删除。
- `order_fee.proto:113` 的费用删除注释提到标签级联，退役后清理这句即可；历史账单行来源引用置空与金额/科目快照语义必须保留。

### 4. 领域、传输、仓储读取链清理

#### 独占文件

- 删除 `internal/biz/business_tag.go` 及其专属测试。
- 删除 `internal/data/business_tag.go`，清理纯标签专属测试；不要删除承载其他行为的测试文件。
- 删除 `internal/service/order_tag.go`、`settlement_fee_tag.go`、`settlement_bill_tag.go` 及纯标签候选服务测试 `settlement_tag_test.go`。

#### 共享文件中的最小修改

| 代码入口 | 必要变更 |
| --- | --- |
| `biz/order.go:115-123` | 移除 tagRepo 字段和 NewOrderUsecase 的 BusinessTagRepo 参数；同步所有正式调用及测试（当前检索约33个构造调用） |
| `biz/order_usecase.go:26,38,91` | 删除 attachOrderTags 和详情/列表附加标签读取，不再因标签查询失败使订单详情/列表失败 |
| `biz/order_types.go:315,398` | 删除 Order.Tags、OrderListOptions.TagIDs |
| `biz/settlement.go:46`、`biz/finance_bill_types.go:133` | 删除筛选 TagIDs；保持已批准 matchedSummary/分页/其他筛选 |
| `service/order_query.go:209-213` | 删除 tag IDs 解析，保留其他边界校验 |
| `service/order_convert.go:37,229` | 删除 Tags 投影和 businessTagSummariesToAPI |
| `service/order_fee.go:22-29,86-93,426-494` | 移除 tagUsecase 注入、列表附加查询和全部标签 RPC/helper；保留本体费用 CRUD/补审/批量校验 |
| `service/settlement.go:32,124-125` | 移除 tagUsecase 注入；NewOrderFeeService/NewSettlementService测试和构造调用一起同步（当前约7处） |
| `service/settlement_bill.go:44-48,61-68,211-216` | 删除 tag筛选解析、列表/详情附加读取和投影；保留账单明细、权限、汇总 |
| `service/settlement_fee_ledger.go:71,104` | 删除标签关联读取与DTO转换参数，保留feeLedger本体/summary |
| `data/order_query.go:178-179` | 删除标签筛选谓词与ent标签导入 |
| `data/settlement.go:459-460` | 删除费用台账标签筛选谓词；matched与顶部summary继续从剩余完整谓词派生 |
| `data/finance_bill.go:87-88` | 删除账单标签筛选谓词，其他筛选/业务summary不动 |

- `biz/enterprise_resource.go:18,20` 删除 ErrEnterpriseTagGroupNotEmpty、ErrEnterpriseTagInUse；另删标签资源类型、Tag详情结构、EnterpriseTagGroup、repo组接口、TagGroup用例/normalize与TAG专用audit分支。保留资源名/图片/地址等通用校验。
- `service/enterprise_resource.go:327-368,450-455,486-487,525-532` 删除组CRUD、Tag oneof解析/投影和group转换。资源类型 switch 要明确拒绝未声明类型，而不是把旧4映射成新类型。
- `data/enterprise_resource.go:43` 的通用查询移除 WithTag；`:309-339` 删除只用于标签的使用计数限制；`:369` 的 TAG detail清理、`:531-602` 组CRUD、`:794-799` 组存在校验、`:860/:883` tag detail写入、`:1005-1011` tag/group转换一起清理。其他类型的使用限制、partner链接/assignees/address_types保留。

### 5. 注册、构造与生成顺序

- `service/service.go:6` 删除 NewOrderTagService；`biz/biz.go:6` 删除 NewBusinessTagUsecase；`data/data.go:35` 删除 NewBusinessTagRepo。
- `server/http.go:26,60`、`grpc.go:25,57` 删除 OrderTagService 参数及注册。其他 Order/OrderFee/Settlement/EnterpriseResource service仍保留，只删除已不存在RPC。
- `server/auth.go:333-338` 的订单businessType提取移除3类 OrderTag 请求 switch；`:265` 移除独立 FinanceFeeTag 权限分支。其余订单权限/fee.read/update及bill.read/update不删。
- `cmd/server/wire_gen.go:83,90,160,168,182,186-196` 当前多处直接构造业务标签，必须由 Wire 正式生成更新；`cmd/server/wire.go:23` 依赖ProviderSet，本体不加手工补丁。
- `server/internal/server/access_rules.go:8` 是访问规则生成入口，`make -C server api` 已顺带 `go generate ./internal/server`，更新 `access_rules_gen.go` 不手改。
- Ent Schema source删标签专属实体/edges后，通过 `go generate ./internal/data/ent` 更新实体集合；删除源文件对应的残留generated文件仅按生成器不清理行为精确处理。
- 契约与实现源同步后标准生成：`make -C server api` → Ent生成（对应Schema变更）→ Wire生成（`go -C server generate ./cmd/server`）→ `pnpm run generate:web-client` → `pnpm run generate:permission-keys`。也可按编译依赖分阶段执行，最终重跑应幂等；不要用手工生成物补编译错误。

### 6. 权限退役与历史审计

- **唯一独立删除权限**：`access/manifest.go:48,222` 的 `FinanceFeeTag = "system.finance.fee.tag"`。
- 企业资源资源备忘CRUD权限（manifest:209-212）仍控制地址等真实功能，保留Key，只把说明中的“标签/标签组”移除。订单标签复用订单read/update，订单费用复用fee.read/update，账单复用bill.read/update，不能为去标签把这些业务权限一起删。
- `web/src/access.ts:39,239-240` 删除对应permission键和canManageFinanceFeeTags，随后生成permissions.generated.ts；删除前端实际引用与角色测试中该专属键。
- `internal/data/permission_manifest_sync.go:82-85` 已按manifest删除清单外Permission；`cmd/migrate/main.go:47` 迁移后现成同步入口。复用当前事务与role_permissions外键清理，不额外写重复权限删除平台或回退旧码。
- `biz/business_tag.go:67-139,198-203` 曾产出 order.tag / order.fee.tag / finance.fee.tag / finance.bill.tag 的 batch_assign/remove 审计；enterprise_resource还有enterprise_tag/group动作。**删除新的事件生产代码不等于删除历史审计记录**。
- `features/audit/audit-presentation.ts:651-685` 仍含enterprise_tag/group显示描述。最薄退役可删当前功能注册映射并沿用既有未知动作显示；也可仅保留明确为历史展示的无业务依赖文本。无论如何不清理AuditLog记录，不为此建立新兼容分支；在最终范围明确展示处理。

### 7. Ent/迁移与种子清单

持久化退役细节由数据库研究单独覆盖，契约研究确认源依赖：
- 专属Schema：`schema/enterprise_tag.go`（EnterpriseTagGroup、EnterpriseTag）与 `schema/business_tag_relation.go`（OrderEnterpriseTag、OrderFeeEnterpriseTag、FinanceBillEnterpriseTag）。
- 通用实体edge清理：`schema/enterprise_resource.go:46,50-52`、`schema/organization.go:74-77`、`schema/order.go:141`、`schema/order_fee.go:83`、`schema/finance_bill.go:76`。
- `schema/enterprise_resource.go:12` resource_type枚举去TAG，但不能删除其他资源类别。
- `cmd/sync-dev`、`internal/data/seed*`、当前`server/seeds`脚本的定向检索未找到业务标签初始化（检索结果里的运输方式“标签”、航司名称含Tag不属于业务标签）。应再以编译/全量扫描确认，不为关键字清零修改行业种子。
- 历史正式迁移保留，不回写过去迁移。删除表/数据属于正式迁移方案且需主会话核对用户数据处置授权，研究阶段不运行数据库。

### 8. 风险匹配验证

1. 编译级：删除constructor参数后，所有`NewOrderUsecase`/`NewOrderFeeService`/`NewSettlementService`生产和测试调用对齐；provider与HTTP/gRPC真实注册一致。
2. 契约级：正式Proto反射/HTTP注册测试断言18个RPC与HTTP端点不再存在，Order/OrderFee/FeeLedger/FinanceBill响应无tags字段；被删字段/枚举编号reserved；保留核心RPC继续可用。
3. 权限：manifest不含fee.tag，迁移/权限同步后旧权限及角色关联消失，其他financeFeeRead、billUpdate、orderUpdate等授权不变。
4. 领域相邻：订单/费用/账单详情与列表不再附加标签查询；原分页、组织隔离、筛选、金额汇总、版本、费用CRUD与账单状态机仍正确。
5. 企业资源相邻：地址/备注/图片/单证主体CRUD和候选保留；TAG=4无新合法映射；当前resource-save/resource-types回归须调整已删除标签类型用例但仍保护其他类型切换与表单校验。
6. 测试不是全删：`order_fee_bulk_integration_test.go:155-191`、`order_fee_delete_integration_test.go:216-247` 中仅删除标签夹具/断言，保留删除费用后账单来源置空、审计/原子性等相邻断言；其他构造测试同步签名。
7. 正式迁移隔离库：前态含标签及各类关联，升级后标签存储清理且订单/费用/账单/非TAG资源/历史审计不变；空库全迁移末态与Ent一致；主会话明确真实开发数据处置后再决定是否执行。
8. 生成物重跑幂等；最终 `pnpm run check:fast`。单组本地最小检查：`buf lint server`、受影响Go包测试/vet、前端定向/tsc/Biome、`git diff --check`；按风险补真实注册和迁移集成。不把缺专用库而SKIP算PASS。

## 相关规范

- `.trellis/spec/server/backend/index.md`、`directory-structure.md`：纯领域/DTO/PO边界。
- `.trellis/spec/server/backend/order-milestone-retirement.md`：现有纵向能力彻底退役先例；精确移除生成残留、标准生成、保留审计、不保留兼容端点。
- `.trellis/spec/server/backend/quality-guidelines.md`：真实HTTP注册与正式迁移校验。
- `.trellis/spec/web/frontend/type-safety.md`：生成字段源、枚举reserved/非业务视觉Tag区分。
- `.trellis/spec/server/backend/capability-navigation.md:37`：当前业务标签入口，实施后需更新，不能继续指向已删biz文件。

## 外部依据与版本

无外部网页。本次只依据当前仓库Proto、Go/Ent/Wire生成入口和现有迁移规范，不引入依赖版本升级。

## 限制与未确认项

- 本轮纯静态研究，未修改产品、生成物、测试、规范或Git状态；仅写本研究文件，未运行任何数据库查询/迁移。
- 未确认当前数据库是否有标签及关联数据，不把“目前没有使用场景”理解成“数据为空”；数据删除授权由主会话向用户明确。
- 行号以本次研究工作树fd650008为准，实施时随删除变动。
- 全仓字符串tag并不等于业务标签；历史迁移、reserved、审计原始记录、普通状态Tag和行业种子名称不能机械删。

### 生成目录清理复核（2026-11-09）

- `web/scripts/generate-services.mjs:5-17` 当前只导入 `readdir/readFile/writeFile` 并直接调用 `generateService`，**没有** `rmSync` 或其他显式删除 `src/services/roncin` 输出目录的步骤。上文“前端 orderTagService.ts 通过客户端生成器清理”应理解为最终验收要求，不能仅凭此脚本断定旧 service 文件一定自动消失；生成后需检查该残留，必要时精确移除已退役服务对应文件。
- `server/buf.gen.yaml:1-19` 当前没有 `clean` 配置；Go 插件输出目录 `api` 与 Proto 源目录重叠，不能为清理旧绑定而直接清空整个输出目录。继续采用上文精确移除 `order_tag.pb.go`、`order_tag_grpc.pb.go`、`order_tag_http.pb.go` 的处理，其他绑定与 OpenAPI 走正式生成。
- 本复核只查看上述两个配置/脚本，未执行生成、未断言 `@umijs/openapi` 内部清理行为，未写入产品文件。
