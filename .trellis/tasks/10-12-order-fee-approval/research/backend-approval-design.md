# Research: 订单整票费用审批后端设计与并发边界

- Query: 最小 Schema、内容版本、不可变审批依据、费用/建账/拆票锁序、独立审核权限及生成验证入口。
- Scope: internal
- Date: 2026-10-12
- 阶段：仅规划研究，本文建议须合入 design.md 后统一审阅；未修改产品代码、未执行测试、生成或迁移。
- 业务依据：用户最新认可 `research/recommended-flow.md` 的整体方案；该文件仍含“待确认”旧措辞，不能据此否认最新认可，PRD 由主会话合并。

## Findings

### 1. 最小存储建议：订单当前投影 + 追加式审核事件

现有 `server/internal/data/ent/schema/order.go:63` 有业务状态、两把锁与通用 `version`，没有费用审核事实；`schema/order_fee.go:58` 的版本同时被内容修改与账单关联操作递增。建议：

| 对象 | 最小变化 | 意义 |
| --- | --- | --- |
| Order | `fee_review_status`：NOT_SUBMITTED/PENDING/APPROVED/REJECTED/REVOKED | 列表、台账引用同一整票事实，不在每笔费用复制审核状态 |
| Order | `fee_content_version`，默认 1 | 仅费用内容或审核范围变化递增；一次事务修改多笔只递增一次 |
| Order | `fee_review_version`，默认 1 | 提交、撤回、批准、退回、撤销、补录状态变化递增；请求携带 expected_review_version，避免批准旧提交 |
| Order | `current_fee_review_event_id`、`approved_fee_review_event_id` 可空 | 当前提交/状态与最近有效批准的定位；待审补录保留原 APPROVED 事实 |
| 新 OrderFeeReviewEvent | 事件 ID、订单/组织、审核代次、动作、提交事件 ID、操作者/姓名及授权来源快照、意见、时间、费用内容版本、指纹、整票 JSON 快照、来源补录申请 ID（可空） | SUBMIT/WITHDRAW/APPROVE/REJECT/REVOKE/SUPPLEMENT_APPROVE 等只追加，不回写历史 |

不必先建通用审批平台或会签表。上述一个业务历史表足够表达首次审核与补录后整票新批准。复杂度选择也可采用专用审核申请表 + 不可变事件，但当前单人任一有资格者批准无此必要。历史中的费用 ID/姓名/科目/结算单位采用快照，不能因费用硬删除丢失依据或给费用历史建立阻止删除的强依赖外键。

`OrderFee.version` 保留现有乐观锁用途。建账/取消账单仍可递增它，但不得递增 `Order.fee_content_version` 或改变批准。证据：`finance_bill_write.go:90`、`:270`，`finance_bill_batch.go:175` 都只有关联变化也 AddVersion(1)。审批指纹不得包含费用通用 version、updated_at、有效账单关联、账单版本或订单通用 version，否则分次建账、取消账单、一般订单编辑都会错误废弃批准。

审批 JSON 固化完整费用集合，按费用 ID 稳定排序；保留方向、科目、结算方 ID 与名称、数量、单价、含税口径、税率及金额、币种、汇率/来源/日期/设置 ID、本位币/折算金额、发生日期、备注、补录来源，以及收入/成本/利润/利润率。decimal 用规范字符串，金额聚合使用 decimal。审核显示与批准校验用同一纯 Go 计算函数；CNY 不是任意硬编码，须确认整票各行本位币一致，混合口径拒绝提交。

建议审核利润统一使用费用本身已保存的汇率快照，不混入建账日汇率。现有提成使用账单行本位币作为已建账费用的分母口径（`finance_commission_calculation.go:382`），因此“审批利润”和“提成计提利润”可能不同，页面须分别标注。建账日汇率不能反向修改费用审批快照，也不能让普通建账使批准失效。更新汇率主数据本身不重算历史费用快照；修改费用汇率才改变内容版本。

建账准入：Order=APPROVED + 批准事件的 content_version/指纹匹配当前完整费用集合 + 选中费用确实包含在该快照 + 无 PENDING 补录 + 无有效账单占用 + 原有组织与建账权限。预览与提交均消费同一资格函数；真正写入时必须锁内复核。撤销批准须确认整票没有任何有效账单关联（含 DRAFT），否则只取消某一张账单就撤销整票会使其它有效账单失去审批依据。

待审补录不把订单审核状态覆写成 PENDING：投影 `has_pending_fee_supplement` 来自补录申请 EXISTS，可作为独立筛选条件。现有 `order_fee_supplement_request.go:57` 已有 PENDING 状态，优先沿用，不在 Order 复制未经必要性证明的计数。若限制一票同一时刻只允许一笔待审补录，增加组织/订单的 PENDING 部分唯一索引，并与 Ent 注解同源。

### 2. 全部费用内容写路径

以下是对 data 非生成、非测试文件中 OrderFee.Create/Update/Delete 的搜索结果；实施时每条都必须参与内容版本和冻结校验。

| 路径 | 当前锁/行为 | 需要变化 |
| --- | --- | --- |
| `server/internal/data/order_fee.go:380` Add | `lockOrderForFeeMutation` 先 Order.ForUpdate（:280） | PENDING/APPROVED 拒绝普通新增，成功后内容版本 +1 |
| `order_fee.go:460` Update | Order → 已有关联 Bill（:481）→ BillLine（:485）→ Fee（:490） | 审核冻结优先阻断；既有已建账编辑配置不能放行 |
| `order_fee.go:681` Remove | Order → Fee，查有效账单占用，物理删除 | 冻结、版本、有效账单与补录专用删除限制同时保留 |
| `order_fee.go:802` BulkUpdate / 约 :875 后 BulkRemove | Order → 费用 ID 排序锁；单票整批事务 | 同一冻结门禁；成功批量操作只增一次整票内容版本 |
| `order_fee_supplement_approval.go:159` CreateApprovedFee | 补录 usecase 的共享事务内新增正式费用 | 唯一受控批准后新增入口；新成本、内容版本、新整票批准快照、CONFIRMED 冲减、通知和审计原子提交 |
| `order_fee_supplement_cancel.go:264` 物理删除补录费 | 专用撤销要求未建账、最新有效、无确认过的调整 | 删除也必须更新整票内容版本与审批状态；不能绕过下述已确认冲减禁撤销约束 |
| `sea_order_change_split_execute.go:1894` / `:1952` | 克隆费用到新票，再删除原费用 | 拆票改变审批范围，不能复制批准；PENDING/APPROVED 原票先撤回/撤销才能拆，结果票为 NOT_SUBMITTED |
| `finance_bill_write.go:90`、`:270`，`finance_bill_batch.go:175` | 只递增费用 version | 关联操作不改内容版本；但写入需统一订单锁与准入复核 |

费用标签关联属于旁路元数据，目前不写 OrderFee 主记录；若继续允许冻结后维护标签，必须明确标签不属于审核内容，不因此让财务事实解冻。种子代码也要同步新状态：新订单/费用不能伪造 APPROVED，演示建账数据须经明确服务端审核流程或受控种子构造，不做自动批准存量迁移。

拆票已有 Order 锁（`sea_order_change_split_execute.go:92`），随后共享单证锁、费用锁（:553），且已有账单占用门禁（:567）。在最早 Order 锁内新增审核冻结校验，避免锁到费用后才判定。改配若不改变费用审核范围/金额不必废弃批准；若改变费用归属需与拆票同处理。首次迁移统一 NOT_SUBMITTED，不伪造历史人工批准；原费用、账单和提成历史继续保存，原账单不能充当新增建账的审批授信。此项是单一耦合交付任务，实施步骤可分验证包但不能将审批门禁与补录/提成安全衔接拆开独立上线。

### 3. 锁顺序：必须在第一次事务读时调整

当前并没有一个全链路一致的锁序，不能以“已有事务”当成并发验收。

**确定的反序证据：**

1. 普通费用 Update 是 Order → Bill → BillLine → Fee；取消账单现在 Bill → BillLine → Fee（`finance_bill_write.go:212`、`:249`、`:261`）。若新增取消账单的 Order 锁却放在 Bill 之后，A 编辑持有 Order 等 Bill，B 取消持有 Bill 等 Order，将死锁。
2. 单/批建账 `LoadBillableFeesScoped` **在共享事务的首次读取就已经 Fee.ForUpdate**（`finance_bill.go:362`）。只在 Create 或 CreateBatch 末端补 Order 锁，会与 Order→Fee 的普通写入口反序。`biz/finance_bill.go:175`、`biz/finance_bill_batch.go:540` 是应调整的首次共享事务入口。
3. 提成确认/生成 `GetGenerationContext` 先锁来源（`finance_commission_calculation.go:130`），随后 calculateCommission 锁 Bill（:61）→ Order（:354）→ Fee（:370）→ BillLine（:389）。费用编辑是 Order→Bill，存在反序；现有 `finance_commission_write.go:100` 注释不能视为整体安全证明。
4. 反核销与反对冲已经 Order 优先再锁来源（`finance_verification.go:352`、`finance_netting.go:444`），与提成生成来源优先也有反序。这不是新功能独有，审批新增订单锁会增加触发机会，需要定向联动。
5. 取消账单 :260 在 Go 排序 feeIDs，但 :261 查询没有 Order(ByID)，**IN 列表排序不等于数据库实际锁序**；BillLine :249 同样没有主键排序。必须在锁查询本身显式排序。
6. `confirmCommissionsForApplicationApproval` 目前逐提成单调用 calculateCommission，再锁全部 Orders/父单（`finance_commission_write.go:201` 起）。多张提成跨相同订单可能按来源顺序交叉持锁，不能只给最后一步排序就宣称一致。

**建议收敛为当前受影响实体的固定相对顺序：**

`全部受影响 Order（主键升序） → 财务来源（如核销/对冲，主键升序） → Bill（主键升序） → BillLine（主键升序） → OrderFee（主键升序） → SupplementRequest/审核事件 → Commission 父单（主键升序） → Adjustment（主键升序）`

首次/补录审批不需要全部实体，可以跳过不用的类型；共享 MBL 类保留已有 Order→MBL→Link→运输执行的单证相对锁序，它们在费用锁之前。员工/授权、币种、账户、汇率等关联读取必须在已锁订单之后完成；若某行后续需更新，初读直接 ForUpdate，不能先 ForShare 升级。使用此顺序前应逐条核对未列出的邻接路径，不能宣称此处已证明整个财务系统无死锁。

可执行改法：

- 建账：在 `LoadBillableFeesScoped(txCtx, …)` 中只读定位费用→订单集合，锁所有 Order，再锁费用，重验集合/组织/审核批准与补录状态。Create 仓储独立调用也必须同样起始顺序；不能依赖“通常已有外层事务”省门禁。
- 取消账单：只读定位活动账单行的全部 Order，先锁完整集合，然后锁 Bill→BillLine→Fee，重验集合与账单版本，维持发票/核销/对冲禁取消规则。取消关联不撤销批准。
- 已建账费编辑：批准冻结后通常拒绝；撤销后的历史允许路径仍遵守 Order→Bill→BillLine→Fee。
- 提成生成/确认：在 GetGenerationContext 首次加来源锁之前预定位受影响 Orders，先锁 Orders；calculateCommission 改为锁后重验来源/账单行关联，BillLine→Fee 顺序一致。月度申请批准应预解析整批全部 Orders，再集中加锁，不能逐提成先取得下游锁后再扩张订单集合。
- PAID/调整迁移现有 Order→Commission→Adjustment 可保留相对顺序（`finance_commission_write.go:123`，`finance_commission_adjustment.go:206`），在 Order 锁后检查待审补录。
- 未加锁的预定位仅用于决定锁集合，不能作为批准依据；集合变化应返回版本冲突要求刷新。禁止锁了 O2 后临时追加比它小的 O1，禁止用自动重试掩盖锁序缺陷。
- 批量整票批准按推荐方案逐票独立事务，返回逐票成功/失败；多票建账则一个共享事务，先取得完整订单 ID 排序锁。
- 暂不把任意“账单取消/冻结”辅助函数放到提成父单锁之后调用，否则会逆转上述顺序。

用例通过 `biz.Transactor.WithinTransaction`（`data/transaction.go:61`）建共享事务，仓储读走 `Data.client(ctx)`（:76），局部写用 WithTx（:48）自动合并。注意纯 WithTx 回调的原 ctx 不携带事务标识，辅助函数应显式接 tx.Client，或用外层 WithinTransaction 的 txCtx。回调错误原样返回；通知只能入队，不能在事务中调用钉钉网络。完整响应提交后普通 ctx 重读，禁止共享事务里调用 withSQLTx（:92）。

### 4. 单一费用审核权限与实时授权

建议 Manifest 新增 `FinanceFeeReview = "system.finance.fee.review"`，显示名「费用审核」，`Requires` 为 `FinanceFeeRead` + 现有全部已支持订单业务类型的 `OrderRead`（由现有业务类型注册枚举生成依赖列表，避免硬编码另一份）。这符合现有统一财务台账读权限（`access/manifest.go:43`、`:218`）与单一跨订单品类审核权限要求，不扩展每品类的 fee.review 权限，不依赖 FinanceBillCreate/OrderLock/OrderUpdate。批准、退回、撤销和补录审批共用该权限；提交复用所属品类费用写权限，撤回复用提交人本人关系，仍需组织准入。

这是让“只勾费用审核”角色也能从现有订单列表看到审核入口的最薄策略：同一授权角色闭包增加各品类只读，不增加写/单证子资源权限，不写一套放宽订单列表的 OR 授权旁路，也不建第二个审核订单列表。依据 `access/manifest.go:408` 的订单子资源只读本来就依赖 OrderRead，:430 ResolveDependencies 有现成依赖闭包；`web/src/access.ts:387` 和 :400-404 的路由列表访问消费各品类 read。审核完整费用预览走 FinanceFeeRead 与审核权限对应组织交集，不要求另勾每品类 fee.read。所有新增依赖仍保留同一角色 DataScope 与 `workspacePermissionScope` 的本公司上限；不得拿独立 OrderRead 角色的广范围扩大 fee.review 的审批组织。权限 UI 应清楚展示“费用审核依赖订单查看和费用台账查看”。

已有能力：`biz/auth.go:258` ResolvePermissionOrganizationScope / :286 CanAccessOrganizationForPermission 将同一权限与其所属角色 scope 绑定；不能合并一个角色的审核权限与另一角色的 ALL scope。`data/auth.go:360` ResolvePrincipal 复核用户、启用 Membership、启用组织、启用角色，且 :418 按实际工作台收集角色；`biz/auth.go:792` 的 workspacePermissionScope 应保持系统管理与公司经营范围限制。中间件 ctx Principal 用于 UI/普通授权，批准事务里要重读合格授权，防止权限撤销与审批同时发生。

旧补录 `data/order_fee_supplement_lock.go:187` 给 bootstrap admin 显式放行，`:169` 调用 queryQualifiedBusinessLockCandidates；资格谓词位于 `order_lock_qualification.go:108`。这套资格**不能直接复用**：权限是 lock，且 ALL/祖先匹配不等于当前公司工作台授权，新要求是有效用户+成员+角色+费用审核权限。建议提取当前费用审核专用的查询，用同一查询规则服务候选与决策，并保留 grant/membership/role 的来源 ID；不新增 bootstrap 绕过分支。

实时批准：先锁目标 Orders；在同一客户端按稳定顺序读取/ForShare 锁 user、membership、role_assignment、role 与权限边，并检查组织启用、所在公司锚点、目标订单是否在该权限 writable 集合。权限边删除也必须与锁规则串行（ForShare 角色行不能自动阻止单独 DELETE role_permissions，需锁实际边或让角色授权变更取得同一角色写锁）。无法用非锁定 EXISTS 宣称“提交时实时有效”。审阅当时合格授权来源与 actor 姓名保存快照，未来权限收回不追溯删除既有批准，新的动作重新检查。

候选提醒：使用同样业务授权口径，按 UserID 去重；未绑钉钉仍具备资格并可在 Roncin 待审列表处理；不把候选快照当永久审批白名单，不通知到的人不因此失权。复用 `order_fee_supplement_notifications.go:22` 的 BackgroundTask + NotificationDelivery，同事务写确定性事件+人幂等键，后台重试；新增待审与结果模板需扩展 Ent Enum 和正式 SQL CHECK（不能只改 Go 枚举）。提醒最好只含订单号与“有待审费用”，完整金额/利润在登录授权后查看。

### 5. 补录必须同步修正的旧契约

- 当前补录只接受 BUSINESS/FINANCIAL/BOTH 锁依据（`schema/order_fee_supplement_request.go:32`、`:44`），`biz/order_fee_supplement_usecase.go:313` 拒绝双锁都不存在。现在仅费用 APPROVED 的订单也需能补录；应一次性改为费用批准事件/内容版本的证据，加保留存在的业务/财务锁证据，不能伪造 business_lock_generation。锁类型 CHECK、proto 注释/枚举、映射、测试、种子都需同改，不添加旧资格自动回退。
- 现有补录批准已通过共享事务锁 Order→Request（`biz/order_fee_supplement_usecase.go:453`），实时资格仍是 lock（:471）。替换审核资格后保留两把锁的修改限制及证据复核。
- `data/order_fee_supplement_approval.go:233` 创建 DRAFT 建议。改为 CONFIRMED 必须复用 `finance_commission_adjustment.go:196` 的订单行与父单两层可冲减余额、同事务锁、confirmed_at/by、版本及幂等，不能只替换状态字符串。冲减 SourceType 沿用 LOCKED_FEE_SUPPLEMENT 或统一新命名需全仓同改；财务不得 IGNORE/CANCEL 系统确认调整，允许登记追回。
- 补录预览/批准需指纹覆盖完整整票费用与提成影响集合，批准锁内重算，对前端看到的批准结果做版本比对；退回/撤回不产生费用或调整。
- PENDING 补录要同时拦单/批建账、提成单确认、月度申请整批批准、相关提成 PAID 登记；当前 PAID 分支 `finance_commission_write.go:155` 只要求 CONFIRMED，必须新增统一门禁。不要阻断该员工其它订单。

### 6. 契约、注册、生成、迁移、验证入口

| 范围 | 源文件/入口 |
| --- | --- |
| 整票审核 API | `server/api/order/v1/order_fee.proto`：复用现有 OrderFeeService，新增摘要/完整预览、提交/撤回、批准/退回/撤销、批量预览/批准；请求含 order ID、review version、内容版本/预览指纹、意见 |
| 订单状态与服务端筛选 | `server/api/order/v1/order.proto`、`biz/order*.go`、`data/order_query.go:60`；将过滤加到 Count 与分页共用的 query（:171） |
| 财务台账筛选/候选 | `server/api/finance/v1/settlement.proto`、`service/settlement_fee_ledger.go`、`data/settlement.go:412`（共用 baseQuery :478）与 `finance_bill.go:354` |
| 权限中间件 | proto 的 `(access.v1.rule)` 是源；`internal/server/access_rules.go:8` 生成，禁止手改 access_rules_gen.go；当前补录多处 AUTHENTICATED 后 biz 授权（order_fee.proto:96）需同步改 |
| 注册与注入 | 复用服务已在 `internal/server/http.go:72`、`grpc.go:69` 注册；新 repo/usecase 加 `data/data.go:34`、`biz/biz.go:6`，修改 service 构造器；运行 Wire 生成 `cmd/server/wire_gen.go` |
| Schema | `server/internal/data/ent/schema/` 源；CHECK/索引/外键策略与正式迁移同源；不可变历史不跟费用硬删除级联丢失 |
| 提醒 | `schema/notification_delivery.go` + 通知模板/跳转映射与正式迁移 CHECK；复用后台发送器 |

已核实命令（只记录，未执行）：

```bash
go -C server generate ./internal/data/ent
make -C server api
go -C server generate ./cmd/server
pnpm run generate:permission-keys
pnpm run generate:web-client
pnpm run migrate:dev
```

Ent 生成器源 `internal/data/ent/generate.go:6` 使用 sql/lock、sql/upsert；make api 同时执行 access rule 生成。正式迁移由 `server/cmd/migrate/main.go:17` 读取 server/migrations，生产入口 `pnpm run migrate:server` 严格校验且同步权限；禁止用 Schema.Create 代替正式迁移。仓库目前未找到独立受控的“生成迁移”脚本/Atlas 配置，实施前需补足基于 Ent Schema 差异的迁移生成方式，不能在计划中虚构现有命令。新枚举/部分索引/CHECK 须审查生成元数据与 SQL 等价。

后续验收入口建议（需实施授权后按风险执行；本研究没有执行）：

```bash
go -C server test -p 32 ./internal/biz ./internal/service ./internal/server -run 'FeeReview|OrderFee|FeeSupplement|FinanceBill|Principal|Access'
go -C server test -p 32 ./internal/data -run 'FeeReview|FeeSupplement|FinanceBill|CommissionBillLockOrderConcurrent|SeaOrderSplit|OrderFeeDelete|OrderFeeBulk'
pnpm run check:fast
```

真实库用例需 `RONCIN_INTEGRATION_DATABASE_SOURCE`（`data/integration_test_helper_test.go:30`），未配置时会 Skip，不能把普通 go test 通过当事务验证完成。helper :101 用正式迁移建临时 schema。现有关键用例：`TestFinanceBillCreateSharedTransactionPostgres`（finance_bill_transaction_integration_test.go:78）、`TestFeeSupplementDecisionPostgres`、`TestFeeSupplementFinancialEvidencePostgres`、`TestFeeSupplementEvidenceConcurrencyPostgres`（补录集成文件 :598/:729/:969）、`TestCommissionBillLockOrderConcurrentPostgres`（提成事务集成 :753）、海运拆票集成 :460。正式迁移初始化不并行跑 Schema.Create、不重置开发库。

必要新增验证场景：提交与编辑竞争；批准/退回/撤回争抢只一方成功；按版本批准旧预览失败；负利润空白意见拒绝；权限/成员/角色停用及权限与组织 scope 混搭拒绝；批准冻结覆盖普通/批量/拆票入口；分次建账不使批准失效；取消账单不使批准自动撤销；补录待审与建账/提成确认/发放竞争；补录批准任一写步骤失败全回滚；确认冲减两层封顶及实际登记；不倒序追加订单锁；未经授权不能读取 hover 摘要。

## Related specs

- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：业务实体与财务链路。
- `.trellis/spec/server/backend/database-guidelines.md`：共享事务、并发锁、正式 CHECK 同源、锁升级禁忌。
- `.trellis/spec/server/backend/finance-commission-lock.md`：提成净额财务锁与历史不可变。
- `.trellis/spec/server/backend/order-lock-and-document-version.md`：Order 优先共享单证锁、拆票范围。
- `.trellis/spec/server/backend/auth-session-org-switch.md`：真实公司工作台准入与授权重算。
- `.trellis/spec/server/backend/index.md`：权限与组织 scope 必须来自同一 grant，生成物边界。

## External references

本次只核实仓库实际源码、现有测试与规范，无外部文档依赖或版本推荐；不会用网上通用审批架构覆盖仓库契约。

## Caveats / Not Found

1. **批准补录即确认冲减，会压缩纠错空间。** 当前 `order_fee_supplement_cancel.go:138` 明确只要调整曾确认/扣回就不允许取消补录。新方案发生冲减即确认后，这类补录将不能直接撤销，即使尚未建账/发放；不能靠撤销整票批准绕过。用户已认可推荐中的“不绕过专用撤销约束”，但若希望补录录错也能完整纠错，需要另定“恢复提成调整”业务，不在本期自动加恢复/INCREASE。最终规划须明确告知这个限制。
2. **提成实际少发/扣回的 PAID 含义**：现有提成有名义金额、调整单 PAID 只登记追回；新方案“已确认未发放自动少发”必须在界面/契约明确名义金额、系统冲减、实际支付、执行日期，不能把父单 PAID 自动等同全额支付或暗改历史金额。代码只能说明现状，不能替用户决定实际支付业务口径。
3. **审批利润与提成利润汇率口径不同**：建议分别展示，不以审核表的毛利百分比直接算所有历史提成，复用既有边际计算、原规则与快照。
4. **原始订单没有应付/应收的特殊业务**：代码不能判断“录齐”的完整性；不新增未经确认的“必须两侧都有正数费用”规则，提交人的完整性确认及审批是本期判定依据。
5. **财务/系统 bootstrap 通用旁路**：旧补录有 bootstrap 放行，但新规则要求真实有效 role/membership，不能默认继承它。用户角色权限设计已明确，设计应选明确授权无旁路。
6. 费用标签是否可冻结后维护、拆票是否必须先撤销批准、补录多个并发待审的体验、有效审核被撤销后申请中的补录如何终结：属于边界产品规则，应在最终设计集中陈述，可按用户认可的整体方向收敛，不能用现有代码当产品答案。
8. 原币/本位币必须有有效三位币种，金额、汇率及折本币字段必须解析为 decimal；同币种应满足恒等汇率和金额，跨币种须有正数汇率及正确折算快照，不能拿缺失值当 0；不一致拒绝提交/批准/建账，禁止临时查询当前周汇率静默修正。R5 的“汇率变化”明确是源费用快照变化，周汇率主数据维护与创建账单使用其它日期汇率不应使原批准失效。
7. 检索只覆盖仓库当前 data 主记录写路径，未证明 DB 外部写入或所有未列出的财务路径遵循全局锁序。最终实现需按受影响用例验证，不扩大成通用锁管理平台。
