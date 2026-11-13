# 提成不含税口径与筛选完善：实施计划

前置：[design.md](./design.md) 与 [prd.md](./prd.md) 经用户审阅；`task.py start` 后才能实施。依赖顺序：费用台账任务先验证统一筛选设置与汇率快照口径，本任务在其后。

## 实施清单（按序，每组可验证后提交）

1. **biz 纯函数**：新增 `CommissionNetBase`，补表格测试（含税 106/未税 100、零总额、尾差、混合税率）。
2. **取值点替换**：`fee_correction_commission_projection.go` 的 `bases`；`finance_commission_calculation.go` 的 `billLineBaseByFee` 与费用回落；`biz.ProjectCommissionEconomics` 的回落分支。已建账取账单行 net/total，未建账取费用 net/total。
3. **指纹**：费用与账单行指纹纳入 `net_amount`。
4. **版本**：`CommissionCalculationVersion` 改为 `CUSTOMER_REALIZED_PROFIT_V4`；不新增 V3 双公式或为旧开发草稿补版本冻结；现有守卫的行为如实记录。
5. **前端**：`web/src/pages/finance/commissions` 的相关标签标为「不含税」，预览、列表、导出口径一致；接入统一筛选设置（费用台账任务验证后复用）。
6. **spec 更新**：`finance-commission-lock.md` 的数值例补不含税口径与版本说明；`.trellis/spec` 对应能力导航同步。
7. **开发数据**：旧计算版本的开发数据不作为兼容目标；不自动清库或重建。若另需数据操作，遵循明确授权要求。

## 验证

定向后端（`go -C server test -p 32 ./internal/biz/... ./internal/data/...` 中提成相关用例，必须使用正式迁移的真实 PostgreSQL，SKIP 不算验收）：

- 样例：应收含税 10,600/未税 10,000，应付含税 6,360/未税 6,000，核销占 40%，比例 10%：已实现利润 1,600.00，提成 160.00。
- 部分回款：不把未实现订单利润全部纳入；重复请求、重复来源不重复计提。
- 负利润：基数取 0，提成为 0。
- 混合税率：多费用不同税率，按费用逐行折算后汇总。
- 已建账与未建账费用并存；账单行与费用行税率不同时取账单行。
- 经济重算：新增/修改/删除费用、取消账单、来源反转后，调整差额按不含税目标计算；已付事实、手工奖罚不被吞。
- 不将旧开发版本草稿在重提时按当前公式刷新列为缺陷，不增加旧版本金额保留验收；仍验证正常事务、版本并发校验及已付事实保护。
- 预览、申请、导出同一数据口径一致。

前端：`pnpm --dir web exec vitest run <提成相关测试文件>`；覆盖真实 wire 形态（省略默认值、显式零、非零、必填缺失），不只用字段齐全的 mock。

提交前：`git diff --check`；类型影响时 `pnpm --dir web tsc`；任务整体完成后 `pnpm run check:fast`。

## 评审门与回滚点

- 评审门：design.md 与 implement.md 审阅通过后启动；用户已明确上线前不要求旧计算版本兼容；数据清理不属于本次授权。
- 回滚：步骤 1-3 为可独立回退的纯计算与取值改动；步骤 4 起结果已改变，回滚需同时还原常量与测试期望，不做运行期回退分支。
- 若发现 `net_amount` 与 `base_currency_amount` 同比例折算引入与费用台账不一致的汇率差，停止实施，回到设计阶段，不用隐藏分支掩盖。
