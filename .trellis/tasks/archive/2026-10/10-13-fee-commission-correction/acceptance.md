# 费用与提成联动更正验收

## 业务结果

- 受控申请覆盖漏录追加、金额/结算对象/方向更正、应收优惠、成本调减及整笔作废。现有审核人审批，展示原/新费用、理由、利润和提成影响；审批与真实资金执行分开。
- 原费用及账单、核销、票据和已执行提成保留。替换采用完整冲回和替代；未建账原费/冲回不进入账务待办，已建账部分通过调整账单清偿。
- 补录 1,000 实为 100：原系统 -100 尚未执行则取消后重建 -10；已执行则保留并增加 +90 补付。人工调整独立保留。
- 用户指定算例：应收 1,000、成本 400、实收 500、10% 利润提成，优惠至应收 800 后目标为 25，原 30 产生 -5 调整。退款及反转同步按实际净回款重算。
- 已有待审补录/更正阻止结案；已结案可继续受控更正，不反结案。结案不要求回款完成；费用完整性继续依靠人工判断。

## 验收证据

实现细节与定向测试见 [核心实现](research/core-implementation.md)、[账务实现](research/finance-implementation.md)、[提成实现](research/commission-implementation.md)。新规范见 `.trellis/spec/server/backend/order-fee-correction.md`。

| 验收项 | 对应证据 |
| --- | --- |
| AC1 受控更正及权限/版本 | EconomicOperations、RevenueAndParty、PermissionsVersionsAndConcurrency 实际 PostgreSQL；前端申请、审核与失败刷新测试 |
| AC2 各财务阶段及历史保护 | FinanceCorrection 四组实际 PostgreSQL，包含合并账、已收/未收、连续更正、原票整红后重开有效行 |
| AC3 补录及提成净额纠错 | CommissionPartialRevenue、PaidNetRebuild、FirstGeneration；确认/已付及已执行冲减、人工调整并存 |
| AC4 幂等、竞争与整笔回滚 | 同订单双并发提交、审批后注入重算错误、审批/付款数据库锁阻塞、退款及来源反转联动 |
| AC5 生命周期及各端展示 | PendingAndBusinessClosure；联合第一阶段自动锁单/结案；前端 25 元预览、冲回用途、开票有效金额、提成补付/扣回 |

## 独立审查修复

- 极小本位币金额多来源分配采用统一最大余数算法，避免逐行舍入导致最后一项负尾差，保持合计与容量守恒。
- 预览刷新失败时禁用批准，避免页面继续提交缓存金额；后端仍以指纹拒绝过期提交。
- 核销/对冲锁定关联提成的订单传递闭包，并在获得订单锁后重新验证集合；等待期间集合变化则返回冲突，禁止漏锁其他订单。
- Ent 生成约束测试同步新增审核动作与系统调整来源枚举，不关闭原检查。

## 最终门禁

2026-10-13 最终结果：

- `pnpm run check:fast` **PASS**，总耗时 77.6 秒：前端权限/常量生成一致性、架构边界、重复函数检查、lint/tsc 与 Vitest；后端分层、Proto lint、全包 Go 测试、vet 与漏洞扫描均通过。
- 前端 **227 个测试文件、1,215 个用例通过**；仓库原有 `sea-template.test.tsx` 中 1 个套件/12 个用例跳过。本任务没有新增跳过或关闭检查。
- 独立临时 PostgreSQL 最终组合 `go -C server test -p 32 ./internal/data -run 'Test(FeeCorrection|FinanceCorrection|FinanceEconomicOrderSet)' -count=1 -v`：**15 个顶层测试及 15 个子场景全部通过，零 SKIP，79.948 秒**。覆盖业务更正、账务、票据、提成、退款/反转、权限/幂等/回滚、实际付款锁等待和跨父单订单集合变化。
- 联合第一阶段 `TestAutoOrderLock_(MergedBillSettlement|PerBillCurrencySettlement|SettlementSources|EligibilityGuards)` 与 `TestOrderClosurePendingSupplement`：**7 个顶层测试及 21 个子场景全部通过，零 SKIP，21.171 秒**。
- [独立审查](research/check-review.md) 的三项问题已修，无待修确定缺陷；前端预览刷新错误保护另有定向 4/4 用例通过。`git diff --check` 通过。

普通 Go 全包测试在没有可选数据库环境时会跳过集成测试；上述独立 PostgreSQL 组合为实际运行的验收证据，两者不混淆。

## 环境与交付边界

所有真实数据库验证使用本任务独立临时数据库/角色及隔离 Schema，顺序执行正式 SQL 迁移，无自动建表，无开发数据库重置。Proto、Ent、Wire、HTTP 权限、OpenAPI 与前端客户端均经生成器更新。本次为本地代码交付，不涉及生产部署或现网数据操作；未执行浏览器端到端测试。
