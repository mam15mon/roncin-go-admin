# Research: 对冲提成候选、计算与草稿确认

- Query: 已确认对冲单如何形成计提候选、按 10% 已实现毛利计提并确认草稿；定位可复用验收入口，不预设引擎缺陷。
- Scope: internal
- Date: 2026-11-08
- 研究边界：只读仓库；未访问业务数据库、未运行会写库的集成测试、未修改业务代码或测试。现场数据由主会话独占只读核验。

## Findings

### 1. 两层候选与来源状态

| 阶段 | 入口与证据 | 契约 |
| --- | --- | --- |
| 对冲单候选 | `server/internal/data/finance_commission_selectors.go:154` | 同公司、`status=CONFIRMED`、`correction_request_id IS NULL`、存在 `active=true AND direction=RECEIVABLE` 分摊，按创建时间倒序分页；这里只保证来源可选，尚不表示任一员工可计提 |
| 员工/身份/方案候选 | 同文件 `:56`、`:68`、`:120` | 来源有正已实现收入的订单上，按 `OrderCommissionAttribution` 的员工×身份去重；来源日期唯一命中启用方案及该员工有效分配，再调用同一计算函数 |
| 来源读取 | `server/internal/data/finance_commission_calculation.go:249`、`:260`、`:270` | 再次要求对冲 `CONFIRMED`、`confirmed_at != nil`、非更正内部对冲，只消费活动应收分摊；没有合法分摊或正收入返回 `FINANCE_COMMISSION_SOURCE` |

候选查询对 `ErrCommissionRuleNotResolved`、`ErrCommissionSource`、`ErrCommissionEmployeeRole` 的组合跳过，其他解析/计算错误外传（`finance_commission_selectors.go:96`）。因此无方案、日期不命中、无固化归属都可合法返回空候选。

主会话现场核验说明：`NT2026110800001` 当前已为 `REVERSED`、分摊 `active=false`，北京公司方案及该来源提成均为零。该现状不能继续用作“已确认来源有方案却无候选”的复现，也不能据此认定引擎有缺陷；应保留状态门禁，不自动恢复旧对冲。

### 2. 来源日期与有效方案

- 对冲归属日明确取 `confirmed_at.UTC().Format("2006-01-02")`，不是对冲创建日、订单日或浏览器日期：`finance_commission_calculation.go:126`、`:157`、`:264`。预览、创建、确认共用该口径。
- `resolveCommissionRuleForDate`（同文件 `:82`）：同公司、身份、`enabled=true`；方案区间和该员工 `cancelled_at IS NULL` 分配区间都必须包含归属日（闭区间，空终点不限制）。零命中或多命中均 `FINANCE_COMMISSION_RULE_NOT_RESOLVED`，不随便取一条。
- 历史离职员工是否可计提由来源日期上的资格判断，不额外要求当前员工启用（同文件 `:208`）。但新创建启用方案的分配员工仍须满足方案写入侧当前成员有效约束。
- 新建启用方案必须有合法起始日，起始日不得早于 `FinanceBusinessDate(time.Now())`；终点可空，比例 `0 < rate <= 100`：`server/internal/biz/finance_commission_rules.go:220`、`:232`。
- 系统财务业务日使用 Asia/Shanghai：`server/internal/biz/finance_commission.go:321`。这与现行对冲来源日 UTC 口径存在跨本地午夜差异；本次现场确认时刻本地 22:27，对应 UTC 仍为同一天，不能据此判成当前故障，也不要在本任务中静默更改日期契约。

### 3. 订单人员归属

- 创建订单事务内的 `snapshotOrderCommissionAttributions` 只从同公司、同订单的 `OrderPersonnel` 三岗（SALES/OPERATOR/CUSTOMER_SERVICE）生成快照，保留 `source_assignment_id`、姓名和归属时间：`server/internal/data/order_write.go:256`、`:261`、`:282`。
- 候选与实际计算均查询此固化快照，不按当前客户档案责任人或登录人的角色猜员工。
- 计算时用 `organization_id + source.orderIDs + employee_id + personnel_role` 限定归属，未命中任何来源订单返回 `ErrCommissionEmployeeRole`：`finance_commission_calculation.go:214`、`:467`。
- 主会话核验的订单 `SE2026110600002` 三岗均为张冠楠，账号和北京公司成员有效；只要新合法来源及方案/分配命中，可预期至少 SALES 组合有资格，但仍须实际候选接口验证。

### 4. 收入、分母与 10% 金额

完整链路：`loadCommissionCalculationSource` → `loadCommissionEconomicProjection` → `ProjectFinanceBillLineSettlements` → `ProjectCommissionEconomics` → `SourceRevenue("NETTING", sourceID)` → `CalculateCommissionLine`。

关键代码：

- `server/internal/data/fee_correction_commission_projection.go:18`：全量加载该订单费用、活动账单行，以及完整核销/对冲清偿事件，构成经济投影。
- 当前分母费用复用 `biz.IsCurrentFee`：`economic_effect != DECREASE AND superseded_by_request_id IS NULL`（`server/internal/biz/order_fee_correction.go:37`），不因 `accounting_required=false` 或未建账而从经济分母排除。
- 普通费用优先采用同订单活动账单行、父账单 `DRAFT/CONFIRMED`、`purpose=NORMAL` 的本位币金额之和；费用带 `correction_request_id` 或 `supplement_request_id` 时用费用自身本位币快照。没有账单快照的费用也用自身本位币金额（投影仓储 `:43-62`、`server/internal/biz/finance_commission_economics.go:83-108`）。
- 按方向分别汇总当前总应收 R 和总应付 P。收入来自统一行清偿重放，普通对冲按当时剩余行余额比例分摊，八位最大余数处理尾差；反转撤销原贡献，不把它再分给后来来源（`server/internal/biz/finance_settlement_projection.go:49`、`:120-157`、`:168`）。
- 更正内部清偿不形成收入；收入沿费用经济根保留与冲回，并限制在当前有效经济金额内。最终按每票 `SourceRevenue("NETTING", id)` 累计本次来源的 BaseQuota（`finance_commission_economics.go:135`、`:232`、`:301`）。所以不能直接用整张账单已核销金额，或将对冲的应付分摊直接当该订单成本。
- `server/internal/biz/finance_commission_lifecycle.go:12`、`:27`：每票计算 `cost=Round8(realized × P / R)`；`profit=Round8(realized-cost)`；毛利口径 `base=max(profit,0)`；`commission=Round8(base × 10 / 100)`。先逐票 Round(8)，再汇总各行，前端展示两位。
- 主会话进一步只读核验原订单全量当前费用：应收 300，应付另有 600+200=800；三条均 INCREASE/current/accounting_required=true/非更正补录，均已有 NORMAL/CONFIRMED 的活动账单行（另600应付在 BI2026110600002）。原来源应收分摊200时，若来源仍有效，本应分摊成本 533.33333333、已实现毛利 -333.33333333、10%提成0。当前原 NT 已反转，实际来源收入为0，不能为这条失效来源生成。
- 负毛利只把计提基数归零，**不会因金额为0移除员工候选，也不阻止预览、生成零金额草稿或确认**：计算最终只要求 Lines 非空且 RealizedRevenue>0（calculation.go:576）；Create 不要求 CommissionAmount>0，正式 Schema `commission_amount_nonnegative` 允许0（`server/migrations/20260826200000_finance_commissions.sql:1`）；确认无正金额门禁。前端生成按钮只要求有效预览，提交检查预览签名和候选，未加正金额条件（`web/src/pages/finance/commissions/components/CommissionCreateModal.tsx:314`、`:332-354`）。仍须满足普通来源/资格/确认门禁。
- 主会话已收敛为另建最小独立正毛利订单（R300/P200）、已确认对冲贡献200、销售10%，保留旧费用及原已反转NT；预期分摊成本133.33333333、毛利66.66666667、提成6.66666667（界面6.67）。这是计划验收的独立数据预期，尚未执行。

### 5. 预览、生成、幂等与占用

HTTP 契约集中于 `server/api/finance/v1/settlement.proto:156-190`：

| 动作 | 路径 | 必要字段 |
| --- | --- | --- |
| 计提员工候选 | `GET /api/v1/finance/commissions/candidates` | `organizationId`、`nettingId`，核销来源必须缺省，分页合法 |
| 预览 | `POST /api/v1/finance/commissions/preview` | `organizationId`、`nettingId`、`employeeId`、`personnelRole="SALES"` |
| 生成草稿 | `POST /api/v1/finance/commissions` | 预览同组来源/员工/身份，另带 `idempotencyKey`，不指定 ruleId |
| 确认草稿 | `POST /api/v1/finance/commissions/{id}/confirm` | `id`、当前 `expectedVersion` |

- 预览、生成、候选都要求 `commission.manage` 对目标公司可写，方案维护另需 `commission.configure`；不能用可配置权限推断经营办理权限。service 对来源同公司检查见 `server/internal/service/settlement_commission.go:462`、`:495`。
- 预览调用统一计算，再读取归属日/本币上下文生成 CNY 快照（`server/internal/biz/finance_commission.go:301`）。创建不信任预览，事务中重新计算、固化来源指纹与逐行快照（`finance_commission_lifecycle.go:47`、`server/internal/data/finance_commission_write.go:25`）。
- 同公司幂等键若来源/员工/身份/备注意图相同，重放同一记录；不同意图返回重复错误（`finance_commission_lifecycle.go:40`、`:56`、`:86`）。
- 即使换幂等键，同公司、同来源、同员工、同身份已有非 CANCELLED 提成（含 DRAFT/CONFIRMED/PAID）仍拒绝；锁内查询及部分唯一索引兜底（`finance_commission_write.go:33`；`server/migrations/20260912000000_commission_netting_source.sql:18`）。取消后可重新生成。
- **现行 ListCandidates 没有按已存在提成过滤**；候选可仍出现，但 Create 负责最终拒绝重复。因此“没有候选”不能由已有提成占用直接解释。本次现场本就没有来源提成。

### 6. 草稿确认前置与错误

`server/internal/data/finance_commission_write.go:99`：先员工、完整来源订单集合锁，再锁来源和账单，统一重算；确认是单独业务动作，不是预览/生成的自动后续。

| 条件 | 结果 |
| --- | --- |
| 来源反转/方案或归属已失效/经济事实指纹变化 | `FINANCE_COMMISSION_SOURCE_CHANGED`，保留旧草稿并要求取消重建（`:136-146`） |
| 关联订单有待审补录 | `COMMISSION_PENDING_FEE_SUPPLEMENT`（`finance_commission_payment.go:94`） |
| 有待审费用更正 | `FEE_CORRECTION_PENDING`（同文件 `:102`） |
| 当前需建账费用无有效账单行 | `FINANCE_COMMISSION_UNCONFIRMED_FEES`（`finance_commission_write.go:154`） |
| 草稿状态或 expectedVersion 不符 | `FINANCE_COMMISSION_TRANSITION`（`:163-175`） |
| 正常草稿、来源指纹稳定、无待审与未建账阻断 | 原子改为 CONFIRMED、version+1，并记确认人/时间（`:177`） |

确认的未建账门禁与经济分母不同：`accountingOrderFeePredicate = accounting_required=true AND superseded_by_request_id IS NULL`（`server/internal/data/order_fee_correction_predicate.go:14`）；有效账单行为 `active=true AND parent.status != CANCELLED`（`fee_bill_association.go:13`）。不要求额外先做旧式费用审核；业务状态锁与发放不在本次扩展范围。

### 7. 可复用测试与最薄验收

- `server/internal/data/finance_commission_netting_integration_test.go:380` 的 `TestCommissionNettingSourcePostgres/纯对冲结清走候选预览创建并断言金额`：真实创建并确认对冲，预览 R=800、成本=300、毛利=500、10%提成=50，两票分别20/30，再生成 DRAFT 与确认 CONFIRMED。`:456` 覆盖同源重复、幂等重放、取消后重建。
- `server/internal/data/finance_commission_cross_order_netting_integration_test.go:55`：跨票对冲只用应收订单集合，预览/生成金额及指纹一致，确认通过；用于防止应付账单属于其他票时错误扩大来源订单集合。
- `server/internal/data/finance_commission_resolution_integration_test.go:403` 的 `TestCommissionResolutionPostgres`：多员工/身份、加入退出日期、离职历史资格、复制衔接与稳定指纹；当前夹具是核销来源，可复用资格断言，不能称已覆盖本次 UI→对冲员工候选全链。
- 现有对冲夹具直接 Ent 创建规则和分配（netting 测试 `:290-311`），主正例只调用“对冲单候选”，随后直接传已知员工预览，**未覆盖从实际 CreateRule 到 ListCandidates(nettingId) 的员工候选链路**。应新增或扩展一条针对性用例，使用合法业务日期通过 CreateRule 创建启用方案及员工分配，再检查员工候选、逐票金额、草稿与确认。
- `scripts/acceptance-finance-bill-batch.mjs:805` 有实际规则创建请求样例，以及核销来源预览/生成/确认；脚本包含完整账务变更与发放，不能为了本次截至确认的验收整段执行。可抽取其合法请求形状，按本期边界运行。
- 规划阶段不要改业务数据。批准后，若在开发环境验收，先确认当前来源已反转，不自动恢复旧 NT；另建受控验收对冲来源，或用专用测试库隔离 Schema 复现。方案起止日显式覆盖新来源确认日；名称/日期/人员需按实际合法输入记录，不伪造历史日期、归属或恢复活动分摊。
- 建议定向验证命令：`go -C server test ./internal/data -run '^TestCommissionNettingSourcePostgres$/^纯对冲结清走候选预览创建并断言金额$' -count=1 -v`；另按变更风险补 `TestCommissionCrossOrderNettingPostgres` 及规则创建/日期边界测试。必须安全注入专用 `RONCIN_INTEGRATION_DATABASE_SOURCE`，受影响真实库用例不能把 SKIP 当通过。

## Files Found

- `server/api/finance/v1/settlement.proto`：权限注解、来源 XOR 请求、计提与确认契约。
- `server/internal/service/settlement_commission.go`：权限/公司/来源检查与 DTO 转换。
- `server/internal/biz/finance_commission.go`：用例、错误、预览及系统财务日期。
- `server/internal/biz/finance_commission_rules.go`：有效方案日期/比例/初始员工分配约束。
- `server/internal/biz/finance_commission_lifecycle.go`：金额纯函数、生成事务、幂等意图和确认入口。
- `server/internal/data/finance_commission_selectors.go`：对冲单与员工候选两层查询。
- `server/internal/data/finance_commission_calculation.go`：归属日、唯一方案/分配解析、按来源计提及指纹。
- `server/internal/data/fee_correction_commission_projection.go`：完整订单费用与统一清偿投影接入。
- `server/internal/biz/finance_settlement_projection.go`：按发生时间重放与八位尾差分摊。
- `server/internal/biz/finance_commission_economics.go`：经济根、来源收入额度与反转/冲回。
- `server/internal/data/finance_commission_write.go`：锁内生成、去重与确认重算门禁。

## Related Specs

- `.trellis/spec/server/backend/commission-scheme-assignment.md`
- `.trellis/spec/server/backend/operating-company-commission-attribution.md`
- `.trellis/spec/server/backend/finance-commission-lock.md`
- `.trellis/spec/server/backend/finance-bill-currency.md`
- `.trellis/spec/server/backend/database-guidelines.md`

## External References

无外部依赖选型；本研究以当前仓库的 proto、实现与现有测试为依据，未引用外部资料。

## Caveats / Not Found

- 本研究未运行测试，因此上述测试金额是源码中的验收期望，不是本次新执行结果。
- 未自行读取或更改实际 NT/订单/员工/方案。现场事实均来自主会话只读结果，状态可能继续变化，实施前须重查。
- 尚无证据认定提成引擎在有效方案条件下失效。表单提交/创建成功反馈由另一研究代理负责。
- 当前原 NT 已 REVERSED；用户已选择另建开发环境验收单，主会话计划独立正毛利最小数据链路，不扩展到发放，也不自动恢复已反转单据。
