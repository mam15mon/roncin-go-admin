# 阶段 F 独立审查与修复（2026-11-14）

本次检查针对普通逐组/NETTING整批建账附带多票、多核销及差额的既定范围；不以实现代理先前通过的测试替代源码复核。未提交 Git，共享契约与生成物由父任务统一提交。

## 已修复

### 1. 核销金额在落库时静默舍入

`VerificationUsecase.create` 原先只校验金额为正，合法解析的九位小数可到达仓储并被 `StringFixed(8)` 改写；普通既有资金路径与差额路径的边界不一致。

- 普通与差额统一复用 `validVerificationAmount`：逐项与合计须为正、最多八位小数，且落在 `numeric(28,8)` 正值范围内。
- 指数须先有界，再做金额比较。F 专用传输解析亦先限制指数/范围，避免 `1e1000000000` 在请求摘要的 Decimal 转串阶段展开成巨型文本。
- 无效边界仍保留原始字符串的 JSON 指纹，只使本组失败并定位 `step/item_index`，不会提前拒绝同批合法组。
- 新增公开独立/组合核销入口的零、负、八/九位、范围、合计越界及巨大指数验证；真实PG确认九位/零核销整组回滚、合法八位原样保存。

涉及：`biz/finance_verification.go`、`finance_verification_shortfall.go`、`service/settlement_bill_operations.go`，新增金额测试与服务测试扩展。

### 2. 组合建账与独立差额核销的跨入口锁序环

源码原顺序为：F先持订单及既有资金锁，再申请流水编号；独立差额核销先创建流水并持编号锁，再锁订单/既有资金。同订单不同账单可形成 `订单→编号` / `编号→订单`；不同订单共享同一资金可形成 `资金→编号` / `编号→资金`。

经主会话确认采用现有仓储边界最薄修复：

- `VerificationRepo.LockBillOrders` 只允许共享事务，经 `Data.client(ctx)` 按当前公司确认账单集合，复用经济订单闭包展开、UUID排序锁定与锁后集合复核；不提前持账单头锁。
- `CreateWithCashflow` 及其组合变体先预锁完整订单闭包，再复用 `LockCashflows` 对全部非空既有资金按UUID预锁，之后才创建流水/分配编号。F已有订单、资金锁可重入，不扩大锁范围。
- 普通既有资金核销的 `LoadCashflowContextScoped` 为无锁SELECT；仓储实际顺序原已是订单→资金→账单，不存在额外的资金→订单反转。
- 发票登记先锁订单再编号，F预锁订单覆盖本组/整批；本次未新增反向的发票/编号获取路径，未引入通用锁框架。
- 新PG用例以真实持锁与渠道屏障固定两入口竞争位置，覆盖同订单不同账单、不同订单共享资金。两入口均完成且各核销50、两核销/三资金事实准确，避免靠定时等待碰运气。

涉及：`biz.VerificationRepo`、`finance_verification_shortfall.go`、`data/finance_verification.go`、新增 `finance_bill_composition_lock_order_integration_test.go`。全局顺序已写入 `finance-bill-currency.md`。

### 3. NETTING错误缺少可识别的业务组

服务端已返回 `group_key`，前端整体失败原只显示第几条发票/核销，多组时无法定位。

`requestMessage` 现消费本次冻结的提交快照，将组摘要映射为「结算单位 · 应收/应付 · 币种」，再显示步骤及序号；不显示内部摘要。补现有错误回显回归。

涉及：`billWorkbenchHelpers.ts`、`BillCreationWorkbench.tsx`、`billCompositionPayload.test.ts`。

## 独立复核结论

- NORMAL每组共享事务，失败组不会降为只建账；NETTING全部账单、草稿对冲与附带事实同事务。
- 发票/核销/差额资金创建、确认、显式汇率覆盖各自按权限及当前公司判定。
- 意图摘要覆盖全部附带动作；不同意图同键冲突，同意图并发回放，不在未知结果后换键。
- 子命令仅写/返回身份，完整进度提交后重读；重读失败按未知结果处理，同键回放也重读。
- 组内错误保留步骤与序号；共享错误常量不被原位修改。
- 前端候选绑定公司/方向/单位/币种，搜索保留已选快照；普通未完成/旧会话与未知提交保持已有输入保护和原载荷确认。
- 本次未发现仍待产品决策的F范围内问题。最终HTTP/UI、生产构建及全量门禁仍由父任务执行，不在本记录中冒称完成。

## 验证

- 专用PostgreSQL、正式迁移：4顶层/22子场景全部PASS，无SKIP（13.416秒，`/tmp/composition-check-postgres.log`）。包含原F10场景、原差额7场景、新锁序2场景、新精度3场景。
- Go biz/service相关定向通过（`/tmp/composition-check-unit.log`），新增金额边界及无效发票隔离测试另有精确命令日志 `composition-check-amount-boundaries.log`。
- 前端4文件11用例PASS（19.36秒，`/tmp/composition-check-web.log`），包含本次NETTING业务组定位及既有多行、部分成功、未知结果与会话隔离。
- `pnpm --dir web tsc` PASS；F七文件Biome PASS；`go vet ./internal/biz ./internal/data ./internal/service` PASS；四HTTP脚本 `node --check` 与 `git diff --check` PASS。

数据库与日志在仓库外，不含开发数据重置；未在记录中写入连接、凭据、Cookie或令牌。
