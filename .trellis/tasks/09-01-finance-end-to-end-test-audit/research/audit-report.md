# 财务全链路测试覆盖审计报告

审计日期：2026-09-01

审计范围：汇率配置与解析 → 订单录入与费用 → 账单与开票 → 资金流水与核销 →
提成预览、生成、调整、确认与支付。

## 一句话结论

**当前代码已经具备较强的分层测试和一条设计为通过真实 HTTP 执行的“本位币应收订单
→ 已确认提成”长链路脚本，但还不能据此声明财务全链路完全没有问题，也不建议在补齐
下列 P1 证据前把“多币种汇率 → 订单 → 账单 → 核销 → 提成支付”作为已经自动化验收的
上线承诺。**

置信边界如下：

- 可以对本次实际运行的单模块金额规则、八位精度、主要状态机有较高信心；若干共享事务
  回滚已有近期隔离库记录，本位币应收/应付 API 闭环则是断言较完整的脚本设计证据，
  但当前版本没有本次运行结果；
- 对同一业务 ID 贯穿的外币汇率链路、提成支付、非管理员权限和跨组织隔离，目前只能说
  “实现与部分单层测试存在”，不能说“完整链路已证明”；
- 本次未发现可由现有测试直接证实的生产逻辑缺陷。下面的 P1/P2 均是测试证据缺口或
  上线风险，不应误报为生产缺陷。

## 证据等级与本次执行口径

| 等级 | 证据 | 本报告采用的边界 |
|---|---|---|
| E1 | 领域/纯函数/组件单元测试 | 证明输入输出规则，不证明真实数据库、HTTP 或页面链路 |
| E2 | 仓储 SQL/mock、路由注册测试 | 证明谓词、排序、锁 SQL 或路由分发，不证明真实 PostgreSQL 事务 |
| E3 | 真实 PostgreSQL 集成测试 | 证明进入夹具的真实约束、锁和事务；直接写入上游表不算上游链路 |
| E4 | 真实 HTTP API 验收 | 证明管理员会话下路由、DTO、用例、仓储和数据库组合行为 |
| E5 | Playwright 浏览器验收 | 证明被实际操作的页面交互；读取预置数据不算浏览器创建上游数据 |

2026-09-01 本次只读审计实际执行：

```text
go -C server test ./internal/biz ./internal/data ./internal/service ./internal/server -count=1
结果：四个包通过。

pnpm --dir web test
结果：54 个测试文件、178 项测试全部通过。

env -u RONCIN_INTEGRATION_DATABASE_SOURCE go -C server test -v ./internal/data \
  -run '^(TestOrderCreateTransactionPostgres|TestFinanceBillCreateSharedTransactionPostgres|TestVerificationCreateSharedTransactionPostgres|TestCommissionCreateSharedTransactionPostgres|TestCommissionBillLockOrderConcurrentPostgres)$' \
  -count=1
结果：五个顶层测试均因未配置临时 PostgreSQL 数据库而 SKIP，命令最终 PASS。
```

本任务没有运行 `pnpm run acceptance:finance`，因为该入口依次执行两个带 `--apply` 的
脚本，会向所连接环境写入真实订单和财务数据；也没有运行 Playwright，因为它依赖上述
脚本预先建立的业务数据和正在运行的完整服务。`docs/plans/费用明细批量转账单.md:131`
保留了 2026-08-26 的“真实 API E2E 全部通过”历史验收声明，但没有原始命令输出、运行
环境或与当前脚本版本绑定的报告，且此后脚本继续加入核销和提成功能。因此本报告将其
作为历史人工验收记录，不把当前 E4/E5 记为“本次已通过”，也不据此推断当前完整脚本
仍然通过。

## 业务阶段 × 证据等级矩阵

符号：`强` 表示该层有直接行为断言；`部分` 表示只覆盖若干规则或依赖预置数据；`无`
表示没有找到该层的对应测试证据。

| 业务阶段 | E1 单元/领域 | E2 SQL/路由 | E3 PostgreSQL | E4 HTTP | E5 浏览器 | 审计结论 |
|---|---|---|---|---|---|---|
| 汇率配置与解析 | 强 | 部分 | 部分 | 无 | 无 | 单元规则充分，下游 E3 使用真实汇率表；没有从汇率 API/UI 建立配置再贯穿全链路 |
| 订单录入 | 强 | 部分 | 强 | 强（脚本断言；本次未运行） | 部分 | 订单创建 API 进入长链路；浏览器只查找预置订单 |
| 订单应收/应付费用 | 强 | 部分 | 部分 | 强（脚本断言；本次未运行） | 部分 | E4 覆盖应收/应付，但两条脚本都使用本位币；账单 E3 直接插入费用 |
| 账单与开票 | 强 | 部分 | 强（建账单） | 强（脚本断言；本次未运行） | 部分 | 外币账单快照有 E3；开票无独立 PostgreSQL 事务用例，E4 只覆盖应收开具 |
| 资金流水与核销 | 强 | 部分 | 强（核销） | 强（脚本断言；本次未运行） | 部分 | 外币核销和汇兑损益有 E1/E3，但 E3 直接插入账单、流水；E4 使用汇率 1 |
| 提成 | 强 | 强 | 强（创建/锁） | 强到确认/调整/取消（本次未运行） | 部分 | 有真实创建事务与 CNY 快照；HTTP 长链路未进入 PAID，浏览器未提交创建 |
| 完整外币链路 | 部分规则 | 无 | 无 | 无 | 无 | 不存在同一业务 ID 从外币汇率配置一直走到提成的测试 |
| 权限与组织隔离 | 部分 | 部分 | 部分组织谓词 | 弱 | 弱 | 管理员成功路径不能证明拒绝路径和跨组织不可见 |

## 分阶段证据

### 1. 汇率配置与解析

当前能证明：

- `server/internal/biz/exchange_rate_test.go:71` 起覆盖八位精度、支持的汇率类型、各业务
  节点独立业务日期、本位币固定为 1、专用汇率优先、继承开关、区间和时区规范；代表
  用例包括 `TestResolveBaseCurrencyUsesExactOne`、
  `TestResolveUsesDedicatedRateBeforeInheritance`、
  `TestResolveInheritsBaseCurrencyRateWhenEnabledAndDedicatedMissing` 和
  `TestNormalizeExchangeRateSettingConvertsToBusinessTimezone`。
- `server/internal/biz/exchange_rate_import_test.go:31` 起覆盖导入秒级时间、内部重叠、相邻
  区间、错误本位币和数据库冲突预检；`server/internal/service/exchange_rate_excel_test.go:12`
  起覆盖模板解析、公式拒绝和 Excel 日期时间格式化。
- `server/internal/biz/order_fee_test.go:119`、`:135`、`:158` 分别验证无权手工覆盖被拒、
  手工汇率快照和系统汇率快照；`:184` 验证超过八位的小数被拒。
- `server/internal/data/finance_bill_transaction_integration_test.go:163` 在真实 PostgreSQL
  中直接建立 USD/CNY 汇率设置和 USD 费用，验证并发更新汇率不会改变账单事务内
  `7.20000000` 快照。
- `server/internal/data/finance_verification_transaction_integration_test.go:81` 在真实
  PostgreSQL 中验证核销事务读取 `7.30000000` 后，外部并发更新需等待，已创建核销
  保留原快照。
- `server/internal/biz/finance_commission_test.go:335` 起验证 CNY 恒为 1、非 CNY 本位币
  按倒数派生、`1/7 → 0.14285714` 八位舍入、汇率日期非法/不一致拒绝；`:448` 起验证
  创建事务内重新解析、预览后变化和同核销重生成采用新快照。

当前不能证明：

- 没有 E4/E5 从“创建或导入一条 USD/CNY 汇率配置”开始，再通过订单费用、账单、资金
  流水、核销和提成验证同一汇率来源 ID、日期和金额的连续测试。
- 没有独立真实 PostgreSQL 汇率仓储测试覆盖重叠区间约束、组织继承和导入落库全过程；
  当前真实库证据来自账单/核销测试为下游直接插入的汇率设置。

### 2. 订单录入与订单费用

当前能证明：

- `server/internal/biz/order_fee_test.go:42` 起覆盖精确总额、八位乘积、超精度拒绝、应收/
  应付、税内价/税外价、本位币金额快照和费用创建幂等意图。
- `server/internal/data/order_transaction_integration_test.go:44` 使用真实 `OrderUsecase` 和
  PostgreSQL，验证四个并发创建订单发号不重不丢、无效客户回滚订单/编号/审计、相同
  版本并发修改只有一个成功。该夹具只直接建立组织、客户、用户和编号规则，订单本身
  经真实用例创建，因此是订单模块有效 E3。
- `scripts/acceptance-finance-bill-batch.mjs:128` 和
  `scripts/acceptance-finance-payable.mjs:146` 都通过 `/api/v1/orders` 创建订单；之后经
  `/orders/{id}/fees` 创建并确认费用，再用同一费用 ID 进入账单，因此是有效的订单 →
  费用 → 账单 E4 传递。
- 应收脚本在 `scripts/acceptance-finance-bill-batch.mjs:167` 创建两笔费用，验证费用台账
  和委托/结算单位；应付脚本在 `scripts/acceptance-finance-payable.mjs:180` 创建一笔
  应付费用并验证供应商方向。

当前不能证明：

- 两条 E4 脚本的费用币种均取 `options.baseCurrency`（应收脚本 `:177`，应付脚本
  `:188`），所以没有覆盖外币订单费用、系统汇率解析、手工覆盖和后续折本币传递。
- 账单 E3 的订单和费用由 `data.db.Order.Create`、`data.db.OrderFee.Create` 直接写入
  （`server/internal/data/finance_bill_transaction_integration_test.go:259`、`:299`），没有复用订单或费用
  Usecase；它不能证明订单创建到建账单的数据库级连续链路。
- 浏览器测试在 `web/tests/e2e/finance-bill-batch.e2e.ts:34` 直接查询并寻找
  `ACC-FIN-` 预置订单，随后只查看费用页（`:48`）；没有从订单录入表单创建订单或费用。

### 3. 账单与开票

当前能证明：

- `server/internal/biz/finance_bill_test.go:11` 起覆盖账单精确聚合、混合结算范围拒绝、重复
  费用、到期日、稳定分组和快照变化；
  `server/internal/biz/finance_bill_exchange_rate_test.go:11` 验证账单日汇率快照。
- `server/internal/biz/finance_invoice_test.go:10` 起覆盖已确认账单的发票聚合和混合结算
  单位拒绝，为开票提供 E1 规则证据，但不证明真实数据库事务或 HTTP 状态流转。
- `server/internal/biz/finance_bill_transaction_test.go:103` 验证账单创建只建立一次共享事务，
  响应在提交后读取。
- `server/internal/data/finance_bill_transaction_integration_test.go:79` 是真实 PostgreSQL
  E3，覆盖同幂等键并发重放、不同键争用同一费用只有一个成功、审计失败回滚账单/费用
  状态/编号，以及外币账单事务内汇率快照。
- 应收 E4 在 `scripts/acceptance-finance-bill-batch.mjs:238` 预览批量账单，`:266` 用同一
  幂等键并发创建，`:295` 验证不同键争用费用冲突，`:304` 确认账单；`:442` 创建发票，
  验证抬头、税号和税务明细，`:476` 登记开具。
- 应付 E4 在 `scripts/acceptance-finance-payable.mjs:236` 至 `:286` 使用同一应付费用 ID
  预览、创建并确认应付账单。
- 浏览器 E5 能打开账单生成向导（`web/tests/e2e/finance-bill-batch.e2e.ts:20`），但随即
  取消；它能查看预置发票详情（`:97`），不是浏览器创建账单或发票。

当前不能证明：

- 发票没有独立 E3 事务/并发/回滚测试；E4 仅覆盖应收开票的创建和 issue，不覆盖应付
  进项票、作废、红冲及其与核销/提成的组合。
- E4 账单金额都是本位币；不能证明外币费用经账单日汇率重新快照后继续进入 HTTP 核销。

### 4. 资金流水与核销

当前能证明：

- `server/internal/biz/finance_cashflow_test.go:39` 起覆盖流水幂等意图、USD 结算日系统汇率
  快照、无权手工覆盖拒绝和在汇率变化前重放旧流水。
- `server/internal/biz/finance_verification_test.go:103` 验证核销分摊顺序不影响幂等；`:132`
  对同一 USD 金额验证账单本币 `288`、资金本币 `290`、核销本币 `292`、应收汇兑收益
  `2` 以及应付汇兑损失 `-2`；`:157` 验证一个共享事务和提交后重读。
- `server/internal/data/finance_verification_transaction_integration_test.go:47` 是真实
  PostgreSQL E3，覆盖相同幂等键并发重放、审计失败回滚核销/分摊/编号，以及并发汇率
  更新时的快照锁定。
- 应收 E4 在 `scripts/acceptance-finance-bill-batch.mjs:499` 起并发登记两笔收款，`:579`
  起并发核销；验证候选按方向/结算单位/币种隔离、超额分摊拒绝、余额归零、有效核销时
  禁止取消、反核销后余额恢复，并把第一笔核销 ID 传给提成。
- 应付 E4 在 `scripts/acceptance-finance-payable.mjs:288` 起验证应收流水不能核销应付账单，
  `:346` 起创建/确认付款，`:384` 起完成应付核销并验证费用、账单和流水余额。
- 浏览器 E5 在 `web/tests/e2e/finance-bill-batch.e2e.ts:116` 实际打开核销工作台，选择预置
  资金和账单、自动分配并提交核销。这是实际页面写操作，但资金和账单不是该浏览器测试
  自己创建。

当前不能证明：

- 两条 E4 脚本都把 `exchangeRate` 明确设为 `1`（应收 `:504`，应付 `:295`、`:353`），
  且流水币种沿用本位币账单；因此真实 HTTP 长链路没有验证不同账单/结算/核销汇率造成
  的汇兑损益。
- 核销 E3 的 USD 账单和资金流水由 Ent 直接插入
  （`server/internal/data/finance_verification_transaction_integration_test.go:169`、`:174`），没有调用账单和
  资金流水 Usecase，故不能累计为外币订单 → 账单 → 核销 E3。
- 没有发现资金流水创建的独立 PostgreSQL 事务集成测试；真实库核销测试从已确认流水
  开始。

### 5. 提成

当前能证明：

- `server/internal/biz/finance_commission_test.go:125` 起覆盖金额计算、逐订单成本分摊、亏损
  订单按零、规则角色、调整、反核销计划、CNY 快照、日期筛选和导出审计。
- `server/internal/data/finance_commission_test.go:206` 验证事务内首次读取核销单即
  `FOR UPDATE`；`:401` 验证账单按主键升序加锁，覆盖锁 SQL 形态。
- `server/internal/data/finance_commission_transaction_integration_test.go:107` 使用真实
  `CommissionUsecase`、真实仓储和 PostgreSQL，验证成功创建、完整主单/行/CNY/审计
  快照、提交后普通上下文重读、汇率失败、持久化失败和审计最晚失败的整体回滚。
- 同文件 `:712` 的真实 PostgreSQL 并发测试验证相反输入顺序锁定账单不会死锁。仓库
  日志 `.trellis/workspace/mam15mon/journal-1.md:136` 记录该测试于 2026-08-31 实跑通过；
  `:176` 记录提成创建四个 PostgreSQL 子用例于 2026-09-01 在隔离临时库实跑通过。
- 应收 E4 把 `verificationA.id` 传给预览和创建
  （`scripts/acceptance-finance-bill-batch.mjs:651`、`:671`），并断言明细 `orderId` 等于
  最初创建的订单（`:663`）；随后覆盖并发幂等、来源规则变化冲突、草稿费用关账拦截、
  确认提成、订单财务锁、调整单幂等、有效金额、超额冲减拒绝和存在提成时反核销拒绝。
  因此这是一条真实的本位币订单 → 提成确认 E4 长链路。
- 提成前端 Vitest 覆盖 CNY 预览依据、创建后刷新、详情双口径、月份筛选、CSV 安全转义
  和导出权限按钮；锚点包括
  `web/src/pages/finance/commissions/components/CommissionCreateModal.test.tsx:57`、
  `web/src/pages/finance/commissions/components/CommissionDetailDrawer.test.tsx:8`、
  `web/src/pages/finance/commissions/commissionExport.test.ts:10`、
  `web/src/pages/finance/commissions/index.test.tsx:73`。

当前不能证明：

- E4 脚本把提成确认到 `CONFIRMED` 后做调整和取消，没有调用
  `/api/v1/finance/commissions/{id}/paid`；现有后端测试也没有直接覆盖 `MarkPaid` 状态流转。
  `PAID` 只出现在反核销计划、筛选/导出和 DTO 映射测试中，不能证明支付动作本身。
- 提成 E3 的订单、两笔费用、账单、流水和核销都由 Ent 夹具直接写入
  （`server/internal/data/finance_commission_transaction_integration_test.go:276` 至 `:455`），只证明“给定合法
  上游快照时提成创建的真实事务”，不证明上游真实用例连续生成这些数据。
- 提成 E3 成功场景的组织本位币是 CNY（`:228`），CNY 汇率恒为 1；非 CNY 倒数换算只有
  E1/mock 用例，没有同一条真实 PostgreSQL 或 HTTP 上游链路证据。
- Playwright 在 `web/tests/e2e/finance-bill-batch.e2e.ts:166` 直接查询任意已有、带调整的
  `ORDER_LINE_V1` 提成并查看详情；`:208` 只打开生成弹窗并断言预览按钮禁用，没有选择
  核销、计算预览、提交创建、确认或支付。

## 现有跨阶段链路的真实起止点

### A. 应收真实 HTTP 长链路（现有最强证据）

```text
POST 订单
  → POST/confirm 两笔本位币应收费用
  → 费用台账
  → 账单预览/并发创建/确认
  → 发票创建/issue
  → 两笔本位币收款创建/确认
  → 部分核销（同一核销 ID）
  → 订单人员 + 提成规则
  → 提成预览/并发创建/确认
  → 调整确认/取消、提成取消
  → 第二笔核销、反核销和余额恢复
```

证据：`scripts/acceptance-finance-bill-batch.mjs:128-1142`。

它确实跨越订单、费用、账单、开票、收款、核销和提成，并用 `order.id`、`feeIds`、
`bill.id`、`cashflow.id`、`verification.id` 连续传递。因此不能说仓库“完全没有端到端
测试”。但它有三个明确边界：费用始终等于本位币、流水汇率固定为 1、提成只到确认而
非支付。

### B. 应付真实 HTTP 长链路

```text
POST 供应商
  → POST 订单
  → POST/confirm 一笔本位币应付费用
  → 应付账单预览/创建/确认
  → 应收/应付方向错配拒绝
  → 本位币付款创建/确认
  → 应付核销
  → 费用和两端余额归零
```

证据：`scripts/acceptance-finance-payable.mjs:113-433`。终点是核销，不包含发票和提成；
同样没有外币换算。

### C. Playwright 页面路径

`web/tests/e2e/finance-bill-batch.e2e.ts:9-217` 使用管理员登录，依赖 A 脚本已经创建的
`ACC-FIN-` 数据：查看账单向导、订单费用、费用台账、客户开票资料、发票和流水页面，
实际通过页面新建一次核销；随后查看任意既有可解释提成并只打开生成弹窗。它不是从页面
订单录入到提成的浏览器端到端测试，也不保证查看的提成属于它刚提交的核销。

### D. 真实 PostgreSQL 分阶段路径

| 测试 | 真正进入真实用例的起点 | 上游夹具方式 | 可证明的终点 |
|---|---|---|---|
| `TestOrderCreateTransactionPostgres` | 订单创建 | 组织/客户直接建夹具 | 订单、编号、审计和版本并发 |
| `TestFinanceBillCreateSharedTransactionPostgres` | 账单创建 | 订单/费用直接写表 | 账单、费用占用、汇率快照、审计和回滚 |
| `TestVerificationCreateSharedTransactionPostgres` | 核销创建 | 账单/流水直接写表 | 核销、分摊、汇率快照、编号、审计和回滚 |
| `TestCommissionCreateSharedTransactionPostgres` | 提成创建 | 订单到核销全部直接写表 | 提成主单/行/CNY快照/审计/回滚/提交后重读 |
| `TestCommissionBillLockOrderConcurrentPostgres` | 提成计算所需账单锁 | 账单直接写表 | 相反输入顺序不会因锁序死锁 |

这些测试组合起来是重要的模块 E3 防线，但由于没有复用上游 Usecase 和同一个跨阶段业务
标识，不能拼接成一条完整 E3。

## 横向风险维度

| 维度 | 已覆盖证据 | 主要缺口 |
|---|---|---|
| 金额与精度 | E1 覆盖八位金额、税内/税外、账单聚合、核销汇兑损益、提成分摊；E4 验证 125、40/85、4、1.25 等金额 | 没有外币同 ID 全链路和舍入累计/多行尾差的 E4/E5 |
| 币种与汇率方向 | E1 覆盖应收/应付方向、系统/手工、继承、CNY 倒数；E3 覆盖 USD 账单/核销快照 | E4/E5 都没有外币汇率配置到提成；应付外币长链路缺失 |
| 日期 | E1 覆盖业务节点日期、时区、区间、提成归属；E3 固定日期快照 | 没有跨月、时区午夜、汇率生效边界的 HTTP 连续链路 |
| 事务与回滚 | 订单、账单、核销、提成均有 E3；提成最晚审计失败证据最完整 | 资金流水和发票缺专用 E3；四段 E3 没有统一跨阶段事务场景 |
| 并发、幂等和锁 | 订单发号/更新、账单费用争用、核销幂等、提成幂等/账单锁有测试；应收 E4 也发并发请求 | E3 默认 CI 跳过；没有高并发长链路和故障恢复/重试压力证据 |
| 权限 | Proto 访问规则与生成表一致由 `server/internal/server/access_rules_test.go:18` 验证；部分 E1 验证汇率覆盖权限，前端测试验证提成导出按钮 | E4/E5 全用 bootstrap 管理员；没有低权限用户 403、仅查看/管理/导出分离的真实 HTTP 验收 |
| 组织隔离 | 仓储查询普遍携带组织谓词；E3 按测试组织断言，提成行和主单包含组织过滤 | 没有两个组织的数据交叉读取/写入拒绝 E3/E4；管理员当前组织成功路径不能证明租户隔离 |
| 审计 | 订单/账单/核销/提成创建 E3 验证审计数量与回滚；提成创建核对详情 JSON，导出 E1/E2 验证审计内容 | E4 不查询审计；汇率、费用、发票、流水、确认/取消/支付的全链路审计完整性未验收 |
| API 契约 | 应收/应付 E4 脚本覆盖大量真实路由和错误 reason；`server/internal/server/commission_route_test.go:52` 以真实 Kratos Router 验证提成静态路由 | 没有非管理员鉴权和外币链路；API 脚本不在 CI |
| UI | 178 项 Vitest 当前通过；Playwright 能提交核销并查看财务页面 | E5 依赖预置数据，未创建订单/费用/账单/发票/流水/提成，也未操作提成支付 |
| 日常质量门 | GitHub CI 执行前端 Vitest、后端 `go test ./...` 和 vet | CI 未提供 PostgreSQL 变量，E3 会 Skip；未执行 `acceptance:finance` 和 Playwright |

## 风险与缺口分级

### 已证实生产缺陷

本次没有发现。本次执行的安全只读检查均通过，审计中也没有观察到测试断言与实现直接矛盾。
“没有证实缺陷”不等于“已证明没有缺陷”。

### P0

无。当前没有证据表明已覆盖场景存在会阻断所有财务操作、造成确定性账务破坏或越权
泄露的生产缺陷；下列项目均按“缺少自动化证据”分级。

### P1：上线承诺前必须补齐的证据

1. **缺少外币完整链路。** 当前最长 E4 全部使用本位币且流水汇率为 1；E3 的 USD 场景
   被拆在账单和核销模块，并由直接写表夹具衔接。若对外承诺多币种正确性，应先补一条
   USD（建议再加一条应付）从汇率设置到提成 CNY 快照的同 ID 长链路。
2. **真实 PostgreSQL 核心测试未进入默认 CI。** `.github/workflows/ci.yml:57` 运行普通
   `go test ./...`，但未配置 `RONCIN_INTEGRATION_DATABASE_SOURCE`；本次已复现五个核心
   E3 全部 Skip 而命令 PASS。代码回归可以在 CI 绿灯下绕过事务、锁和真实约束测试。
3. **权限与组织隔离缺少真实负向链路。** 财务数据和提成属于高敏感信息，当前 E4/E5
   只用 bootstrap 管理员。至少需要验证只读、管理、导出权限分离，以及组织 A 无法读取
   或修改组织 B 的订单、账单、流水、核销和提成。
4. **提成支付没有行为级测试。** 当前 HTTP 长链路只到 CONFIRMED，随后取消；没有 E1
   用例直接调用支付状态流转，没有 E3/E4/E5 调用 `/paid` 并验证 `paid_at`、版本、审计和
   非法重复支付。若“直到提成”包含实际支付，这是明确的验收断点。
5. **当前版本的 E4/E5 没有可复核的运行结果。** 2026-08-26 的历史计划只声明当时真实
   API E2E 通过，之后脚本新增了核销和提成行为；本次因数据写入边界没有执行
   `acceptance:finance`。所以 E4/E5 的“强”表示源码中有直接断言，不代表当前提交已经
   实跑通过。上线前至少要在隔离环境固化一次当前提交的完整输出和业务 ID。

### P2：建议在上线前或首个发布迭代补齐

1. 将 Playwright 改为消费 API 脚本输出的精确业务 ID，或自行从页面创建数据；禁止从
   全表找“任意 ACC-FIN 订单”和“任意带调整提成”，避免串用旧数据产生假阳性。
2. 为资金流水和发票增加真实 PostgreSQL 成功/最晚失败回滚测试，并检查审计；当前两者
   主要依赖 E1 和 E4。
3. 在 E4 增加关键审计查询：汇率变更、费用确认、账单确认、发票 issue、流水确认、
   核销/反核销、提成确认/支付/导出，核对组织、操作人、资源 ID 和关键详情。
4. 增加跨月、汇率生效区间边界、Asia/Shanghai 午夜、1/7 舍入、多费用多账单分摊的
   HTTP 组合测试，避免单元规则在 DTO/持久化边界丢失。
5. 应付链路补进项发票和外币汇兑损益；若业务明确不对付款生成提成，应在验收里显式
   断言该业务边界，而不是由脚本止步核销来隐含表达。

### 功能测试无法单独覆盖的生产风险

- 生产数据库迁移、历史数据质量、真实组织汇率配置完整性；
- 性能容量、锁等待阈值、连接池耗尽和长事务；
- 安全渗透、凭据泄漏、导出数据外泄和审计防篡改；
- 备份恢复、灾备切换、网络中断后的人工对账与失败恢复；
- 生产监控、告警、账务差异报表和人工复核流程。

这些不是本次发现的缺陷，但即使补齐全部功能测试，也不能据此宣称生产“完全没有问题”。

## 上线前最低补测路线图

### 优先级 0：固化当前验收基线

完成定义：在一次性 PostgreSQL 数据库和完整服务上运行当前提交的
`pnpm run acceptance:finance`，保存提交哈希、服务配置摘要、两个 API 脚本和 Playwright
的逐项结果；失败即形成独立缺陷，成功后销毁临时数据库并确认无角色/数据库残留。该步骤
只回答“现有本位币脚本当前能否通过”，不能补足下列外币、权限或精确 ID 缺口。

### 优先级 1：外币应收全链路 E4

完成定义：

1. 通过真实 API 创建/启用 USD→CNY 的订单、账单、结算、核销汇率，使用明确不同的
   汇率（例如 7.20、7.25、7.30）；若当前 API 不能创建测试组织或切换组织本位币，应
   在隔离库中预建组织，不能借用开发库既有配置；
2. 同一个订单 ID 创建 USD 应收费用，确认其费用汇率来源、日期、设置 ID 和本币金额；
3. 同一费用 ID 创建/确认账单，断言账单日快照；创建 USD 收款，断言结算日快照；
4. 同一账单/流水完成部分核销，断言账面本币、资金本币、核销本币和汇兑损益；
5. 同一核销 ID 生成并确认提成，断言逐订单来源、归属日期、CNY 汇率依据和 CNY 金额；
   对 CNY 本位币组织应明确断言 `BASE_CURRENCY / 1`，它只能证明前序 USD 财务链路，
   不能代替非 CNY 本位币的提成倒数派生测试；
6. 另用非 CNY 本位币组织贯穿至少一条真实核销到提成路径，断言 CNY→本位币来源配置
   ID、日期、倒数八位舍入、`DERIVED` 来源和 CNY 提成金额；
7. 所有业务阶段用返回 ID 串联；若仅为建立隔离组织而使用数据库夹具，报告必须把该
   环境前置与业务链路分开标注，不得把直接写业务表称为 E4；测试后清理全部夹具。

### 优先级 2：提成支付 E3 + E4

完成定义：真实 PostgreSQL 和 HTTP 各覆盖一次 CONFIRMED→PAID，精确断言版本、
`paid_at`、有效金额和审计；同时覆盖草稿支付拒绝、重复旧版本冲突、已支付后的取消/反核销
计划。若支付实际需要外部付款单，应按真实生产入口联动，而不是直接改状态。

### 优先级 3：PostgreSQL 质量门

完成定义：CI 为订单、账单、核销、提成和锁顺序测试启动一次性 PostgreSQL 服务，显式
注入 `RONCIN_INTEGRATION_DATABASE_SOURCE`，日志必须显示这些测试为 PASS 而不是 SKIP；
测试结束验证夹具无残留。普通单元测试与集成测试可分 Job，但集成 Job 必须是合并门禁。

### 优先级 4：权限与双组织隔离 E4

完成定义：创建管理员、只读财务、提成管理员、仅导出或无导出权限等最小角色；用组织 A
和 B 的真实记录验证每个敏感路由的 200/403/404 边界，前端按钮与后端拒绝同时成立；
尤其验证提成列表、详情、导出不会泄露另一组织员工收入。

### 优先级 5：稳定的浏览器主链路

完成定义：Playwright 使用本次运行创建的精确订单和提成 ID，从 UI 至少完成外币费用
查看、账单快照查看、核销提交、提成预览/生成/确认/支付；不得以查找任意历史数据替代。

## 面向上线决策的最终回答

- 如果上线范围只是“开发环境内由管理员操作、以本位币为主的应收/应付、提成先确认不
  支付”，现有 E1、E4 脚本加分阶段 E3 已提供较好的开发阶段回归基础，但仍应先固化
  当前 E4/E5 运行结果，并把 PostgreSQL 测试纳入门禁。
- 如果上线承诺包括“多币种从汇率配置贯穿订单、账单、核销、汇兑损益和提成支付”，
  **当前证据不足，不建议宣称该链路已经完全验收。** 最少完成优先级 0 至 4 后，才能给出
  有条件的高置信上线判断。
- 自动化测试无论多完整，都只能降低已建模场景的风险，不能证明系统“完全没有问题”。
