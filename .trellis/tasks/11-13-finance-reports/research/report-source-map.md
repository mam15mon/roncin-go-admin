# 四类报表的现有事实与复用地图（2026-11-14）

本文件仅为静态源码研究，不是产品实现或数据库验证。以下路径均相对仓库根目录。

## 1. 已批准的业务口径

- 利润按出口 ETD／进口 ETA 归期，包含未建账的当前有效费用，含税与不含税并列。
- 实际收付款按流水日期，剔除撤销流水。
- 账龄按账单到期日，未到期、1–30、31–60、61–90、90 天以上；首版只查当前余额，历史截止余额延期。
- 新权限 `system.finance.report.read`，限当前公司；设计已获审阅并完成首轮实现，验证证据见 [report-implementation.md](report-implementation.md)。

## 2. 经营费用与利润

| 现有真相 | 位置 | 报表消费方式 |
| --- | --- | --- |
| 当前费用叶子为 INCREASE 且 superseded_by_request_id IS NULL | `server/internal/biz/order_fee_correction.go` 的 `IsCurrentFee`；`server/internal/data/order_fee_correction_predicate.go` 的 `currentOrderFeePredicate` | 利润必须用这一集合，不能用台账所有可见行 |
| 反向 DECREASE 是清偿/更正过程事实，保留历史且可能仍待建账 | `EconomicAmount`、`SettlementDirection`、`RequiresAccounting`；`server/internal/data/settlement.go` | 台账有必要展示反向行，但经营利润不能再把原费用、反向行、替换叶子全部相加 |
| 含税本币金额为费用 `BaseCurrencyAmount` | `server/internal/biz/order_fee.go`；`server/internal/data/ent/schema/order_fee.go` | 汇总已保存费用快照，不按报表查询当天重取周汇率 |
| 本币不含税为已保存含税本币 × 原币净额 ÷ 原币总额，银行家舍入到 8 位；总额 0 时为 0 | `OrderFee.BaseCurrencyNetAmount()`，同 `CommissionNetBase`；`server/internal/biz/order_fee_base_net_test.go` | 复用这一领域公式与逐行精度，不重算税额、不按整票统一税率 |
| 当前经营汇总只汇总当前正常行，含税应收减应付 | `server/internal/data/settlement.go` 的 `listFeeLedger`、`getFeeLedgerOrderDetail` | 同方向、同本币聚合；报表增加未税列，不拿台账分页回传行自行拼全量 |
| 账单行及提成来源可能用账单日汇率、来源重放、已实现额度 | `server/internal/data/fee_correction_commission_projection.go`、`server/internal/biz/finance_commission_economics.go` | 不能把提成已实现利润当作本需求的经营费用利润；本报表不依赖是否开票/核销 |
| 多 SELECT 一致读取已有只读 REPEATABLE READ | `server/internal/data/transaction.go` 的 `withReadSnapshot`；`settlement.go` 的列表入口 | 同一次响应的行、完整合计、图表用同一快照 |

订单保留有效费用时，不因未审核、未建账、未开票、未收款、已结案自动排除。订单终止不能据状态悄悄删去仍存在的真实有效费用；需要过滤时由明确筛选处理。

## 3. 业务日期、人员与客户

| 字段/能力 | 位置与事实 | 设计约束 |
| --- | --- | --- |
| 出口 ETD／进口 ETA 自然日 | `biz.OrderRateDay`，使用 `ExchangeRateBusinessLocation`，要求 RFC3339，贸易方向来自 `trade_direction` | 使用同一业务时区归日，不能 UTC 字符截断或按创建日代替 |
| AE 权威 ETD/ETA 来自主单最新关联 | `server/internal/data/air_order_query.go` 的 `writeOrderTransportTimeExpression`；`order_query.go` 的 `applyOrderStringDateRange` | 报表日期查询复用权威表达式，不能只读可能陈旧的 `orders.etd/eta`；SE 继续订单同步投影 |
| 订单客户 | `ent/schema/order.go` 的 `customer_id`；订单和费用台账 `.WithCustomer()` | 利润按订单客户筛选；客户不等于某条费用的结算单位 |
| 订单协作人员 | `ent/schema/order_personnel.go`；`order_query.go` 的 `applyOrderPersonnelFilter` | 默认沿订单列表按 SALES/OPERATOR/CUSTOMER_SERVICE 当前岗位筛选，使用 EXISTS，避免一单多人 join 放大金额 |
| 固化提成归属 | `ent/schema/order_commission_attribution.go` | 属于提成来源，不默认替代本报表操作人员筛选；不凭客户档案当前责任人反推历史订单 |
| 人员候选 | `data/company_personnel.go`；`OrderRepo.ListPersonnelOptions`；`api/order/v1/order.proto` 的 `ListPersonnelOptions` | 底层候选可复用，但旧 RPC 要求订单权限，不能让“只有报表权限”用户被迫拥有订单维护权 |
| 客商候选 | `data/partner.go`/伙伴检索及别名拼音键 | 按报表权限包装薄只读候选入口，复用后端 keyword/拼音搜索，不前端全量加载 |

## 4. 实际收付款

`server/internal/data/ent/schema/finance_cashflow.go`：流水本身有 `transaction_date`、`amount`、`currency`、`base_amount`、`base_currency`、方向、结算单位、本方账户、支付方式与状态。实际发生只取 `CONFIRMED`；`DRAFT` 为未确认登记，`CANCELLED` 已取消。

`server/internal/data/finance_cashflow.go` 的现有列表 `Summary` 已只计 CONFIRMED，而 `MatchedSummary` 跟随任意列表状态。因此新报表固定确认态，不能直接照搬含草稿/取消行的列表汇总。

流水核销前也是真实收付款。核销/反核销只改变分配，不创造/撤销银行事实；不能把 verification.amount 或 netting.amount 再加到实收实付金额。流水与订单没有直接一对一关系，首版按流水一行、结算单位/币种/方向/支付方式/账户筛选，不伪造订单客户或人员维度。

汇兑损益是另一种事实：`biz/finance_verification.go` 有头与分摊 `ExchangeGainLoss`，`biz/finance_netting.go` 定义 `payable_base_amount - receivable_base_amount` 为对冲汇差。它们不改变原流水金额，也未获批准加入本报表的经营利润，不再二次折算或擅自拼成“实际利润”。

## 5. 当前账龄

- `server/internal/data/finance_bill_credit.go`：`writeBillSettledSum`、`billUnsettledPredicate`；`server/internal/data/finance_bill.go`：`enrichVerificationAmounts`。当前余额 = 账单总额 − 有效核销 − 有效对冲，带符号，不按发票进度扣减。
- 发票/免票改变开票和费用保护状态，不清偿债务；草稿对冲不计余额，撤销核销/对冲后余额恢复。
- `partnerCreditSummaries` 已以未结金额乘账单 `exchange_rate` 折本币。报表沿账单汇率，不能改用费用汇率或实时周汇率。
- 到期日是 `FinanceBill.DueDate` 可空。`CalculateOverdueDays` 对无到期日不计算逾期；不可回退账单日或发票日。五个指定分段之外单列“未设置到期日”异常余额/数量，保证总额守恒。
- 到期当天为未到期；业务日差 1/30/31/60/61/90/91 分别落到明确边界。只返回本次快照日 `as_of_date`，不接收历史截止日，不宣称历史重演。
- 账单可能跨多个订单、客户和岗位。首版一账单一行，按结算单位、方向、币种筛选，不 join 多订单后重复计整张余额。

## 6. 前端可复用入口及缺口

- 已有 PageContainer、SectionCard、FinanceLedgerTemplate、SearchFilterTemplate、useFilterSettings、useColumnSettings、React Query、ProTable request/onLoad、财务快照契约。
- `features/finance/ledger-summary` 可复用金额展示结构，但利润双口径与账龄分段需要本领域适配，不把字段硬套成费用 totalAmount。
- `utils/format.ts` 的 formatAmount/formatDate、`utils/csv.ts` 的 csvAmount/serializeCsv、提成同步限额导出路径可复用。
- `web/package.json` 未声明 charts/plots/echarts/recharts 等图表库，源码没有已有业务图表入口。首版局部 SVG 分组柱图（按月、含税/未税并列）加同源可读数表，避免引入新图表依赖或建设通用 BI 组件；不需要 AI 图片。
- `web/config/routes.ts` 没有四报表入口。新增 4 个路由，利润图和利润表消费同一领域请求/映射，其他页就近实现；路由标题只写在 routes.ts。

## 7. 暂不阻塞本期的延期项

历史截止余额、已实现/现金制利润、汇兑损益并入经营利润、任意维度透视、物化视图/定时快照、跨公司合并、报表订阅、异步导出平台。上述均不能借当前需求提前落地。

实施复核补充：`order_sync.go` 的客户引用锁内校验强制 Partner.OrganizationIDEQ(organizationID)，报表客户同公司关联与写契约一致。结算单位候选复用 `financePartyKeywordPredicate`，覆盖 profile 英文名称及别名/拼音。账龄沿全部 CONFIRMED 有效账单余额，包含已有更正清偿用途，不按 purpose 过滤掉真实债权债务。
