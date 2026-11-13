# 提成不含税口径与筛选完善：技术设计

对应 [prd.md](./prd.md)。本文只覆盖设计，不含实施步骤（见 implement.md）。

## 1. 目标与边界

- 利润提成基数改为**不含税已实现利润**；公式与比例机制不变：提成 = 不含税已实现利润 × 固定比例。
- 保留：回款实现占比、成本分摊、员工方案与分配区间、月度申请、发放、经济重算与调整机制。
- 不做：利润金额分档、累计范围、含税/不含税开关、手工改写已计提金额。

## 2. 现状（已核对代码）

| 位置 | 事实 |
|---|---|
| `biz/finance_commission_types.go:28` | `CommissionCalculationVersion = "CUSTOMER_REALIZED_PROFIT_V3"` |
| `biz/finance_commission_lifecycle.go:13-38` | `CalculateCommissionAmount/Line`：基数 = 已实现利润（负数取 0）× 比例；成本 = 核销收入 × 应付 ÷ 应收 |
| `biz/finance_commission_economics.go:92-108` | 应收、应付累加 `base`（费用或账单行本位币**含税**额） |
| 同文件约 `:256-262` | 来源额度 `baseTotal = quota × base ÷ f.TotalAmount`，`quota` 为原币含税清偿额 |
| `data/finance_commission_calculation.go:425-431` | 建账费用的 `billLineBaseByFee` 累加账单行 `base_currency_amount`（含税） |
| `data/finance_commission_calculation.go:529-540` | 逐费用快照 `aggregateBase`，已建账取账单行本位币，否则取费用本位币 |
| `data/fee_correction_commission_projection.go:54-62` | 经济投影的 `bases` 同样累加账单行含税本位币 |
| `order_fee`、`finance_bill_line` | 有原币 `total_amount`、`net_amount`、`tax_amount`；本位币只有含税 `base_currency_amount` |

结论：含税基数全部来自「本位币含税额」这一个值，改口径集中在三处取值点，不需要改核销分摊、清偿投影的数据结构。

## 3. 核心设计：本位币不含税额

新增纯函数（放 `biz`，遵守分层，不依赖 Ent）：

```text
CommissionNetBase(baseCurrencyAmount, netAmount, totalAmount) =
  totalAmount == 0 ? 0 : (baseCurrencyAmount × netAmount ÷ totalAmount).RoundBank(8)
```

- 沿用该费用或账单行自己的汇率快照，不另取汇率，与费用台账含税/未税卡片同源。
- 逐费用、逐账单行折算，因此混合税率天然成立，部分回款沿用现有「核销额占应收」比例。
- 取值规则：已建账（非更正、非补录）取**账单行**自己的 `net_amount/total_amount` 折算后再累加；未建账取费用自身 `net_amount/total_amount`。
- 三处取值点统一替换：
  1. `fee_correction_commission_projection.go` 构造 `bases` 时；
  2. `finance_commission_calculation.go` 的 `billLineBaseByFee` 与费用回落值；
  3. `biz.ProjectCommissionEconomics` 中 `base := f.BaseCurrencyAmount` 的回落分支。
- `quota × base ÷ f.TotalAmount` 的结构不变：`quota/TotalAmount` 是已清偿占比，乘以不含税 `base` 即得不含税已实现收入。
- `REALIZED_REVENUE` 依据一并走不含税（应收、已实现收入均为不含税），保持两种依据口径一致；不为其保留含税路径。
- 指纹：费用与账单行指纹需纳入 `net_amount`，使税率或未税额变化触发来源变化检测与重算；现有指纹只含 `TotalAmount`、汇率与本位币额。

## 4. 计算标识与上线前取舍

- 当前实现使用 `CUSTOMER_REALIZED_PROFIT_V4`，作为计算标识；用户在 review 后明确，尚未上线，不要求兼容旧开发版本或保留旧开发草稿的公式结果。
- 不建设 V3/V4 并存的双公式，不为旧开发版本增加冻结保护。驳回月度申请重提时，既有 `refreshApplicationLines` / `refreshCommissionSnapshot` 可按当前公式刷新 DRAFT 提成及申请金额；该行为不再列为缺陷。
- 当前 `CommissionEconomicTarget`、费用更正及 `CommissionSupplementCostSensitivity` 的部分路径仍会拒绝非当前版本，这是现有实现行为；本次文档修订不等于已经统一这些路径，也不要求为旧开发数据补适配层。
- 保留正常并发控制、幂等及实际支付事实保护；本次取舍不授权自动清库、重建开发数据或改写真实付款事实。
- `Schema` 默认值 `CUSTOMER_REALIZED_PROFIT_V2` 不变，业务路径显式写入计算版本；本期未修改列定义。

## 5. 契约与前端

- 提成行快照里的 `TotalReceivableSnapshot`、`TotalPayableSnapshot`、`RealizedRevenue` 等语义改为不含税；字段名与 proto 不变，避免无必要的契约变更。
- 页面与导出把相关标签明确为「不含税」，并说明基数与比例；预览、申请、导出同源，复用同一计算结果。
- 逐费用展示税额（PRD 待确认第 3 项）会新增 `FeeDetail` 字段，涉及 proto、`make api`、`pnpm run generate:web-client`。**默认本期不做**，只展示不含税基数；需要时再单独提契约变更，由用户决定。
- 不改后端权限码；不改列表分页上限。

## 6. 统一筛选设置

- 复用费用台账任务已验证的公共能力，本任务在其之后接入。
- 偏好按「用户＋公司＋页面」存浏览器本地；公司工作区与权限范围不可移除；查询、汇总、导出同一筛选快照。
- 页面入口 `web/src/pages/finance/commissions`，遵循 AGENTS.md 的 ProTable `request` 与 `onLoad` 快照约定，不自建列表状态。

## 7. 风险与取舍

| 风险 | 处理 |
|---|---|
| 三处取值点漏改，预览与重算口径不一致 | 统一经 `CommissionNetBase`；验证预览金额与重算目标一致 |
| 未税额与本位币额同比例折算的精度 | `RoundBank(8)`，尾差沿用现有稳定分摊；用 12.00 差额类样例回归 |
| 已建账费用账单行与费用行税率不一致 | 已建账一律取账单行自身 net/total，不混用 |
| 旧开发版本不兼容 | 不建设双公式，不为草稿重提刷新增加额外版本阻断；不自动清库 |
| 与费用台账任务汇率快照口径分叉 | 共用同一汇率字段，不新增汇率来源 |

## 8. 不做的事

不为含税规则预留开关；不引入发票税额、账单税额作为提成税源；不按开票进度拆税；不照搬参考系统整票参考利润与可编辑最终金额。
