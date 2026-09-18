# 实施计划：锁单后费用补录与提成冲减调整

## 0. 开发前复核

- 完整读取 prd.md、design.md、本文件及 implement.jsonl 引用规范。
- 检查 git status，保留所有与本任务无关的用户改动。
- 核验普通费用门禁、直接解锁资格谓词、提成调整状态机和汇率解析入口，禁止复制第二套规则。
- 本功能跨越契约、Schema、权限、事务和前端，必须作为一个原子闭环交付，不得先上线可绕过锁但没有冲减建议的半成品。
- 自动锁定同属本任务，只由有效应收结清及符合条件的费用草稿终态事件触发；不得对所有余额为零的新建订单直接加锁，也不得把自动锁单归责给触发业务人员。

## 1. Schema、领域对象与迁移

- 新增 OrderFeeSupplementRequest Ent Schema、边、PENDING/APPROVED/REJECTED/WITHDRAWN 状态枚举、BUSINESS/FINANCIAL/BOTH 锁依据、可空业务锁代次、可空财务锁证据版本/哈希/净额快照、版本化 request_fingerprint、乐观锁版本、CHECK 和唯一索引；同一组织幂等键只接受同一指纹的语义重放。
- 给 OrderFee 增加补录申请来源关联。
- 给 FinanceCommissionAdjustment 增加 LOCKED_FEE_SUPPLEMENT 来源及补录申请关联；数据库 CHECK 强制“该来源当且仅当 supplement_request_id 非空”，外键使用 RESTRICT，其他来源不得携带该关联。
- 给 FinanceCommissionLine 增加历史总应收/总应付分母、READY/UNAVAILABLE 状态、NATIVE/MIGRATED 来源、回填算法版本、证据哈希和不可用原因码；新提成原生固化，存量行只从原提成计算快照形成时点的不可变事实确定性回填，禁止读取迁移时点当前汇总或当前规则配置。
- 迁移对可回填行复算并核对已存提成结果后标记 READY；无法还原或复算不一致的行标记 UNAVAILABLE 并输出迁移报告，不填猜测值、不改原提成金额、不自动清库。
- 在 internal/biz 增加补录申请领域对象、命令、错误和仓储接口；扩展调整来源类型。
- 扩展现有 NotificationDelivery 模板枚举，增加待审批与员工知情模板；通知继续使用 BackgroundTask + NotificationDelivery，不新增抽象 outbox 表。
- 生成 Ent 代码与正式迁移，保证数据库 CHECK、外键删除策略和唯一约束与 Schema 同源。
- 先完成实现，再补 Schema 元数据及真实 PostgreSQL 迁移测试；禁止 TDD。

## 2. 服务端补录申请与审批事务

- 在订单费用 Proto 源文件增加创建、列表/详情、通过、驳回、发起人撤回及【作废补录费用】接口和 DTO；在提成契约增加本人专属冲减来源详情接口；创建与审批双重拒绝 RECEIVABLE。
- Service 只做 UUID、版本、分页和 DTO 转换；创建保留 fee.create，审批/驳回按目标订单实时 lock grant 校验，专用申请读取在领域层执行“fee.read 或本人发起或实时 lock grant”逐行授权，本人冲减来源详情执行“employee_id = 当前用户 + 组织成员”授权。专用读取不得被组织级 fee.read/commission.read 注解提前挡住，也不得扩张到通用费用或提成详情。
- Biz 用例负责状态机、参数规则、审计语义和共享事务编排。
- Data 层实现申请持久化、直接解锁资格复用、专用费用创建和固定锁序；财务锁证据计算复用现有净额口径，对参与净额的提成行和调整行按“组件类型 + 主键 + 方向 + 符号化 8 位金额”固定排序并按版本规范编码，CONFIRMED/PAID 统一编码为 ACTIVE，不另写一套锁定公式。
- 创建申请时由服务端对订单、不可变费用快照和补录原因生成版本化 request_fingerprint；同键同指纹返回原申请，同键不同指纹返回 FEE_SUPPLEMENT_IDEMPOTENCY_CONFLICT。
- 创建事务解析当前有效直接解锁人员；一个也没有时返回 FEE_SUPPLEMENT_APPROVER_UNAVAILABLE，申请、审计和通知零写入；存在审批人时，在同一事务写入申请、审计及逐收件人的待审批 BackgroundTask + NotificationDelivery。提交后全部资格失效时申请保持 PENDING 并展示暂无审批人，发起人仍可撤回；以后新获得实时资格的人员可以处理。
- 审批通过在同一共享事务中完成费用、冲减草稿、申请终态、审计，以及 BackgroundTask + NotificationDelivery 通知任务与明细入队。
- 实现两类钉钉模板渲染与消费：FEE_SUPPLEMENT_APPROVAL_PENDING 只链接目标申请审批最小详情，COMMISSION_DECREASE_SUGGESTED 只链接员工本人来源详情；每名收件人一任务一明细，任务 ID/idempotency_key 分别按“申请 + 审批人”和“调整 + 员工”确定性生成。任务/明细入库失败回滚业务事务，事务提交后的发送失败沿用现有重试与 DEAD_LETTER。
- 发起时在 Order 行锁内判定并固化 BUSINESS/FINANCIAL/BOTH；财务锁依据固化证据版本、当时净额及参与事实的规范化证据哈希。审批时复核提交时的原始依据，BUSINESS 要求同一业务锁代次，FINANCIAL 要求当前净额仍大于零且同版本证据哈希一致，BOTH 只需二者至少一项匹配。CONFIRMED 与 PAID 之间转换不改变证据；原依据全部失效时返回 LOCK_BASIS_CHANGED，并根据当前是否仍有新锁分别提示重新申请补录或改走普通新增；不得自动嫁接提交后新出现的锁。
- 所有会改变财务锁净额或证据集合的提成/调整状态迁移统一先按 UUID 排序锁定受影响 Order，再锁提成父单和调整，避免补录审批复核证据后被并发改写；补齐反向并发测试和既有财务锁投影一致性测试。
- 审批创建费用前按 calculation_version 路由成本敏感性：REALIZED_PROFIT 等成本敏感版本要求 snapshot_status = READY 且原提成行 base_currency 与新费用一致；REALIZED_REVENUE 等明确不受应付成本影响的版本不因历史成本分母缺失而阻断；未知版本失败关闭。快照或币种不满足时返回稳定冲突并整体回滚，不允许只创建费用或跳过某张应处理提成。
- 审批生成的 CONFIRMED 费用不增加建账特例，验证现有建账候选、单张/批量建账及可取消账单恢复 CONFIRMED 的链路能够识别 supplement_request_id 来源。
- 实现专用作废命令：Order → 申请/费用 → 提成父单 → 调整固定锁序；只允许最新有效、CONFIRMED、无活动账单行，且关联调整 confirmed_at/paid_at 从未写入、当前全为 DRAFT/CANCELLED 的补录，原子取消 DRAFT 调整和费用。APPROVED 申请保持不变；普通 RemoveFee 门禁不放宽。
- 普通费用入口保持原门禁，禁止增加公开的 skipLock、force 或布尔绕过参数。

针对性验证：

- 业务锁和财务锁均不存在时拒绝补录并提示普通新增；仅业务锁、仅财务锁和双锁订单均允许提交；
- BUSINESS 申请只在同一业务锁代次仍有效时可审批；FINANCIAL 申请只在当前净额大于零且财务证据哈希一致时可审批；BOTH 申请在同一业务锁代次或财务证据任一匹配时可审批；原依据全部失效但出现新锁时拒绝并提示重新申请，当前已无锁时拒绝并提示普通新增；
- CONFIRMED 与 PAID 互转不改变财务锁证据；参与集合、方向、金额或主键变化时 FINANCIAL 申请因证据变更拒绝；财务锁释放后由新提成重新形成时不得承接旧申请；证据复核与并发提成/调整状态迁移只能一方先提交且结果可串行解释；
- 无 fee.create 不可提交，无实时 lock grant 不可审批；
- 提交时无任何合格审批人返回 FEE_SUPPLEMENT_APPROVER_UNAVAILABLE；提交后审批人全部失效时仍可读取、展示无审批人并由发起人撤回，新获得资格的人可接手；
- 合格发起人可以自行审批；
- 同一幂等键同一 request_fingerprint 返回原申请，不同指纹稳定冲突；版本冲突、旧锁代次和并发审批均稳定失败或返回原结果；
- 无 fee.read 的发起人只能读取本人申请，无 fee.read 的实时审批人只能读取目标申请最小详情；其他申请及通用费用不可见；
- 驳回不创建费用；
- 只有发起人可按 expectedVersion 撤回 PENDING 申请；撤回与审批并发只有一个成功，撤回成功不创建费用或调整；
- 通过创建且只创建一条 CONFIRMED 费用，订单仍保持锁定；
- CONFIRMED 补录可正常单张/批量建账并转为 BILLED；符合现有取消条件的账单取消后恢复 CONFIRMED；
- 无账单且建议从未 CONFIRMED/PAID、当前为 DRAFT/CANCELLED 的最新有效补录可专用作废，费用与 DRAFT 建议原子转为 CANCELLED；存在更晚有效补录、活动账单行、曾确认/扣回建议或版本竞争时零写入；
- 任一步错误时费用、调整、申请、审计、BackgroundTask 和 NotificationDelivery 全部回滚。
- 重试创建/审批不会为同一收件人重复插入通知任务；提交时审批人快照逐人通知，后来资格变化不补发旧申请；只有实际生成冲减建议的员工收到知情通知，模板链接不能越权读取其他申请或员工数据。

## 2.1 自动业务锁定（独立可验收工作流）

- 扩展 Order / OrderLockRecord 的锁定来源与触发审计字段，生成迁移并把已有锁定事实标记为 MANUAL；更新锁状态 DTO，使自动锁定显示为【系统自动锁定】而不是触发人。
- 抽取现有锁单事务的内部核心；人工入口保留 lock grant 校验，自动入口只接受内部可信触发并在 Order 行锁内重验有效结清事实、订单级未结应收、费用草稿、生命周期、版本和 SE 快照条件。禁止复用触发用户权限或伪装 bootstrap admin。
- 在应收核销创建生效、应收对冲确认生效、费用草稿确认和费用草稿作废的成功提交后触发独立自动锁定事务；跨订单分摊收集全部受影响订单，去重排序后逐单检查。
- 非共享订单逐单检查；共用 MBL 的 SE 订单提升为活动成员组级检查，按 UUID 锁定全部 Order，再锁 MBL、全部活动 Link、共享运输执行和成员 HBL，重验成员集合并以全有或全无方式锁定尚未锁成员。复用同一 MBL/运输执行版本，每个订单保留独立锁定记录与 HBL 快照。
- 重构反核销和反对冲：锁来源单前只读定位受影响 Order，按 UUID 统一取得 Order 行锁，再锁来源单并重验版本、状态和有效分摊；该 Order 锁只做线性化，不得用业务锁阻止财务反转。
- 自动事务失败不改变原财务动作；写入带稳定原因码的审计。资格暂不满足或执行失败只由后续上述事件重试，不增加全量定时扫描或后台补偿调度。
- 固定 Order 优先锁序并用锁定状态幂等收敛自动与人工并发；最多形成一个锁代次和一份 SE 快照。

针对性验证：

- 纯成本、从未产生应收、仅有作废应收和部分结清订单不自动锁；
- 未建账已确认应收、费用草稿均阻止自动锁；费用草稿确认/作废后重新检查；
- 核销加对冲恰好结清、跨订单账单分摊逐单判断；已反转核销/对冲不计入结清；
- 触发人无 lock grant 仍可由系统自动锁定，响应显示系统来源且审计保留触发人；
- 自动锁失败不回滚核销、对冲或费用流转，后续事件可重试；
- 自动与手动锁定并发只成功一次，SE 快照、锁定记录和锁代次均不重复。
- 自动锁定后的反核销/反对冲不自动解锁、不改原锁定事实；显式解锁流程保持可用。
- 反转先于自动锁定提交时，自动锁定锁内重验后放弃；自动锁定先提交时，反转仍成功但订单保持锁定；
- 共用 MBL 任一未锁成员未结清、存在草稿或从未有有效结清事实时整组不自动锁；全部成员合格时整组原子锁定、共享版本只生成/复用一次；锁前后成员集合变化、两成员并发触发和手工/自动竞争均无部分锁定或死锁；

## 3. 提成影响与现有调整复用

- 新增按 calculation_version 路由的历史快照纯计算函数，以原提成行冻结的已实现范围、历史总应收/总应付、比例和版本计算本次补录的边际差额；禁止复用会查询当前 FinanceCommissionRule 的创建/预览入口。
- 为 calculation_version 建立显式元数据：当前 REALIZED_PROFIT 为成本敏感，REALIZED_REVENUE 为非成本敏感；未知版本返回稳定错误，不猜测。只为成本敏感口径的负向差额创建 DECREASE + DRAFT + LOCKED_FEE_SUPPLEMENT 调整。
- 按提成父单 UUID 固定顺序加锁；来源唯一键保证审批重试不重复创建。
- 同一订单多张原提成分别基于自身订单行快照计算，不做 FIFO/LIFO 或跨父单分配；草稿创建同时以 commission_id + order_id 的订单行余额和整张父单余额封顶，并预留其他 DRAFT DECREASE，不允许父单其他订单行补贴目标行。理论超出额同时进入订单行/父单审计而非员工负债。
- 继续使用现有 ConfirmCommissionAdjustment 和 MarkCommissionAdjustmentPaid，不新增确认状态机。
- 重构现有 TransitionAdjustment：所有余额相关迁移先锁 Order，再按 UUID 锁提成父单与调整；确认 LOCKED_FEE_SUPPLEMENT 时重算订单行与父单两层有效余额，任一层不足均返回 COMMISSION_ADJUSTMENT_EXCEEDS，不静默缩小金额。
- 对 LOCKED_FEE_SUPPLEMENT 的 DRAFT 调整开放【忽略建议】，复用 CancelCommissionAdjustment，必填原因后转为 CANCELLED；增加来源专属状态门禁，CONFIRMED/PAID 不得经通用取消接口转为 CANCELLED。
- 增加可分页的调整列表查询，显式支持系统来源、状态、员工、组织和关键字过滤。
- 新增本人专属最小来源详情查询，后端同时限定 adjustment ID、employee_id = 当前用户和组织成员关系，只返回订单号、原提成号、补录费用摘要、建议金额与状态。

针对性验证：

- 已确认与已发放提成都能生成建议；
- DRAFT 和 CANCELLED 父提成不参与；
- 收入口径补录应付不生成建议，即使其 snapshot_status = UNAVAILABLE 也不阻断；成本敏感行 UNAVAILABLE、未知计算版本分别稳定阻断；
- 毛利口径按部分实现范围计算，不把未实现收入对应成本提前全部冲减；
- 多员工、多父提成分别生成，金额与来源可追溯；
- 两张原提成单的影响由各自历史已实现收入和分母决定；一张余额不足不顺延到另一张；同一父单 A/B 两条订单行中 A 额度不足时不得占用 B 的余额；
- 修改或停用当前规则后仍使用原提成快照得出同一结果，且复算路径不查询规则表；未来即使规则允许删除也不依赖动态规则内容；
- 存量 READY + MIGRATED 行的回填只取原提成计算快照形成时点的事实，原提成后新增普通费用和当前规则变化不影响结果；无法确定唯一截止时点或历史事实已被覆盖的成本敏感行标记 UNAVAILABLE，命中补录审批时费用、建议、申请终态和通知任务零写入；
- 新费用与任一受影响成本敏感行 base_currency 不一致时返回 COMMISSION_BASE_CURRENCY_MISMATCH，不使用当前汇率或费用发生日汇率换算历史影响；
- 锁后补录应收在 API 与领域边界均被拒绝，补录应付不改变历史总应收分母；
- 连续两笔补录应付按各原提成行逐次舍入后的差额计提；前一次建议被忽略也不重复计算前一次成本；
- 零差额不生成；
- 两笔补录并发审批不会重复或覆盖调整；
- 财务确认后变 CONFIRMED，标记已扣回后变 PAID；建议创建后若其他调整抢占订单行或父单额度，确认在锁内稳定冲突；
- 财务忽略 DRAFT 建议后变 CANCELLED，原因、操作人与时间可追溯且不再占据待处理列表；
- LOCKED_FEE_SUPPLEMENT 已 CONFIRMED/PAID 时调用通用取消接口稳定拒绝；其关联费用也不能借由状态洗白作废；
- 被通知员工无需组织级 commission.read 即可查看本人的最小来源详情，查询其他员工调整返回无权限或不存在且不泄露记录；
- 人工调整导致订单行或父单额度不足时均拒绝确认，不产生负数有效金额。

## 4. 前端订单费用体验

- 使用生成客户端接入补录申请接口。
- 在订单锁定或财务锁定时保留普通写入禁用，同时按 fee.create 能力显示【补录费用】。
- 复用费用表单字段，增加必填补录原因和“不会修改原费用”的说明。
- 补录表单固定为应付方向，不展示应收选项；服务端仍校验方向。
- 展示申请历史、状态、发起人、审批人和生成费用；申请区不因缺少通用 fee.read 对合法发起人/审批人整体隐藏，界面只消费后端逐条返回的 canApprove/canWithdraw 等能力。提交后暂无合格审批人时展示明确提示，发起人可撤回自己的 PENDING 申请。
- APPROVED 申请展示生成费用的 CONFIRMED/BILLED/CANCELLED 状态；仅按后端能力显示【作废补录费用】，提交必填原因。BILLED、存在更晚补录或冲减已确认时展示服务端稳定阻断原因，不引导用户使用普通删除。
- 订单身份变化时清理申请、弹窗和异步状态，遵循页面复用隔离规范。
- 增加发起、自审、驳回、发起人撤回、撤回与审批竞争、锁状态和迟到响应的定向测试。

## 5. 前端提成体验

- 在现有提成页增加【待处理冲减】视图，使用服务端分页接口，不在前端循环翻页。
- 默认只展示 DECREASE + DRAFT + LOCKED_FEE_SUPPLEMENT。
- 展示员工、订单、原提成、补录费用、建议金额、原因和时间；支持下钻来源。
- 【确认冲减】调用现有确认接口；【忽略建议】调用现有取消接口并要求原因；CONFIRMED 后沿用现有【标记已扣回】。
- 明确状态文案：待处理不等于已确认，已确认不等于已扣回。
- 员工通知跳转本人专属来源详情，不跳转组织级提成工作台；页面只展示订单、原提成、补录费用摘要、建议金额和状态。
- 增加列表筛选、权限、确认/忽略冲突、取消原因和刷新行为测试。

## 5.1 前端锁定来源展示

- 锁状态与锁定记录使用生成客户端返回的 lock_source；MANUAL 展示实际锁定人，AUTO_SETTLEMENT 固定展示【系统自动锁定】并可查看触发类型、触发单据和触发操作人。
- 不根据 locked_by 为空在前端自行猜测来源；锁定来源与触发信息由服务端统一投影。
- 增加人工/自动来源文案、自动锁定无 locked_by、触发详情以及自动锁后显式解锁入口保持可用的定向测试。

## 6. 生成与验证

契约和 Schema 修改后按顺序运行：

    make -C server api
    go -C server generate ./...
    pnpm run generate:web-client

只有新增或修改 Manifest 权限码时运行：

    pnpm run generate:permission-keys

开发期先运行受影响的定向测试：

    go -C server test ./internal/biz ./internal/data ./internal/service -run 'Test.*(FeeSupplement|CommissionAdjustment|CommissionSource|Notification|AutoOrderLock|SharedMBL)' -count=1
    pnpm --dir web exec vitest run src/pages/orders/fees.test.tsx src/pages/finance/commissions/index.test.tsx src/pages/orders/use-order-lock-state.test.ts src/pages/orders/components/detail/OrderLockControl.test.tsx
    git diff --check

Biome 定向检查不预写尚未创建的新文件名；实现时先用 `git diff --name-only` 确认本组实际修改的前端文件，再把这些文件逐项传给 `pnpm --dir web exec biome check`，不得用全量检查代替开发期定向检查。

事务、锁、唯一约束和迁移必须注入专用测试库执行真实 PostgreSQL 用例，并确认是 PASS 而不是 SKIP：

    RONCIN_INTEGRATION_DATABASE_SOURCE="<专用测试库连接串>" go -C server test -v ./internal/data -run 'Test.*(FeeSupplement|CommissionAdjustment|Notification|AutoOrderLock|SharedMBL).*Postgres$' -count=1

任务最终验收执行一次：

    go -C server test ./...
    go -C server vet ./...
    pnpm --dir web tsc
    pnpm run check:web
    pnpm run check
    git diff --check

不把 pnpm run build 作为普通固定门禁；只有生产入口、依赖或构建配置变化时才运行。

## 7. 提交与独立检查

每完成一组可独立验证的修改，由主会话复核差异后提交：

1. feat: 增加应收结清自动锁定
2. feat: 增加锁单后费用补录与冲减建议后端
3. feat: 接入费用补录审批与待处理冲减页面

实现后独立检查：

- 普通费用门禁没有被放宽；
- 专用补录入口不能被当作通用绕过；
- 同一事务、固定锁序、乐观锁和唯一索引真实生效；
- 建议确实复用现有调整状态机，DRAFT、CONFIRMED、PAID 文案不混淆；
- LOCKED_FEE_SUPPLEMENT 的来源关联 CHECK、来源专属取消门禁、订单行/父单双层额度和本人专属读取均有真实数据库或权限测试；
- 生成物来自生成器，迁移与 Ent Schema 同源；
- 所有验收条件均有代码和测试证据。

## 8. 停止条件

以下情况必须停止并回到规划，不自行扩大范围：

- 必须支持既有费用减额、负数费用、红字或已核销冲销；
- 客户要求系统自动控制工资、打款或跨月员工余额；
- 现有提成快照不足以复算锁后成本影响，且需要改变已批准计算口径；
- 产品要求作废已进入 CONFIRMED/PAID 冲减的补录费用并自动正向恢复提成，或绕过现有账单/发票/核销/对冲撤销链直接红冲；
- 产品范围改为首期同时支持补录应收，或要求跨期重分摊历史成本；
- 自动锁定若被要求覆盖从未发生有效应收结清事件的零应收订单，或要求增加全量定时扫描/后台补偿调度；
- 需要清空、重建或自动修复数据库。
