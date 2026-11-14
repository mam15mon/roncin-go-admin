# 费用录入与费用台账：实施计划

前置：[design.md](./design.md)、[prd.md](./prd.md) 经用户审阅；`task.py start` 后才能实施。
契约变更顺序：先改 `.proto` → `make -C server api` → `pnpm run generate:web-client`；权限变更后 `pnpm run generate:permission-keys`。生成物与源文件放在同一提交。
每组改动后提交，提交信息用 Conventional Commits，中文说明。

## 阶段 A：R1 汇率日期与联动（后端）

1. **核对**：LAND／RAIL／内贸品类的 `trade_direction` 是否有值；发票、核销、对冲提交路径是否先锁 Order（决定 1.3 的并发校验点）。结论写入 task 说明。
2. `OrderRateDay` 纯函数 + 表格测试（出口 ETD、进口 ETA、缺日期报错、业务时区跨日）；`resolveExchangeRate` 与 `OrderETDDate` 改造；专用缺汇率错误与 ETD/ETA 缺失文案。
3. `feeHasProtectedFinanceFact` 判定（发票 DRAFT/ISSUED、核销 ACTIVE、对冲 CONFIRMED，账单级）。
4. `refreshFeeRatesForOrders`：升序锁、先全量解析后写入、缺汇率整体失败、版本递增、`reconcileOrdinaryFeeEconomics`、审计。
5. 接入入口：`UpdateDraft`、单票 SE 内联改期、共享海运 TE 更新（同步成员订单 `etd/eta`）、空运主单更新（同步成员订单列）；其后单独一步接入改配、拆票、恢复。
6. 同步 `exchange-rate-single-rate.md` 与相关 spec。

## 阶段 B：R3 权限与 R4 双口径（后端契约）

7. `manifest.go` 新增税率权限；`generate:permission-keys`；服务层传 bool、用例校验、`ErrOrderFeeTaxRateForbidden`；`resolveCatalog` 不覆盖显式税率；审计税率变化。
8. `OrderFee` 增 `base_currency_net_amount` 计算字段（proto、`make api`、生成物）。

## 阶段 C：R2 整表保存

9. `SaveOrderFees` proto 与用例：单事务、行级结构化错误、请求级派生幂等键、回放判定、手工汇率覆盖按提交后集合。
10. 前端：两表多行常开编辑、各自保存、只刷新本侧、保留另一侧草稿；取消「编辑」按钮；新建费用页 15 秒外部变动检测。

## 阶段 D：R4 / R5 / R6 前端

11. 页头三卡双口径、`Decimal` 汇总、缺失显示 `-`、未折算计数。
12. 列宽按设计第 5 节调整。
13. 台账导出加主单号；补主单号验收用例。

## 阶段 E：R7 统一筛选设置

14. `components/ui/filter-settings` 公共能力 + `SearchFilterTemplate` 扩展 + 页签恢复剔除隐藏字段。
15. 接入费用台账、账单、提成三页；更新 `.trellis/spec/web/frontend/capability-navigation.md`。

## 验证

**后端（真实 PostgreSQL，SKIP 不算验收；运行时只传 `RONCIN_INTEGRATION_DATABASE_SOURCE`，不要把整个 `.env.local` 导入环境，里面的 OTEL 变量会令无关测试失败）**

- 日期：出口取 ETD、进口取 ETA，跨周汇率不同；改非基准日期不触发。
- 联动：未建账系统汇率更新；已建账无关联更新；有发票／核销／对冲保留；手工汇率保留；缺目标周汇率整体失败、日期与费用都不变；无可更新费用时不要求汇率。
- 共享多票：逐票判定；成员订单列被同步；任一成员缺汇率则整次失败。
- 并发：联动与开票／核销／对冲并发，锁内最新事实为准。
- 提成：已确认／已付提成经 `reconcileOrdinaryFeeEconomics` 重算，已付事实与手工调整保留。
- 整表：单侧提交不影响另一侧；任一行失败整体不落库；响应丢失重试不重复新增；越权税率 403；已建账带税率被拒；税率显式保存后不被目录覆盖。
- 双口径：含税 106／未税 100 与 53／50 的样例；本币未税与 `CommissionNetBase` 一致。
- 全包：`go -C server test -p 32 ./internal/data` 全部通过（约 15 分钟，放后台跑，超时设 40 分钟）。

**前端**

- `pnpm --dir web exec vitest run <相关测试>`；覆盖真实 wire 形态（省略默认值、显式零、非零、必填缺失），不只用字段齐全的 mock。
- 页头缺失本币显示 `-` 与未折算计数；两侧草稿隔离；筛选移除清值并刷新、折叠保留、存储失败提示、不同用户／公司／页面隔离、页签恢复剔除隐藏字段。
- 一次编写多个 `*.test.tsx` 时按文件拆分，避免单文件超过 20 秒。

**提交前**：`git diff --check`、`pnpm --dir web tsc`；任务整体完成后 `pnpm run check:fast`（约 15 分钟，放后台）。

## 评审门与回滚点

- 评审门：设计与本计划审阅通过后启动；阶段 A 步骤 1 的核对结论若推翻设计 1.1 或 1.3，回到设计阶段修订再继续。
- 每个阶段结束合并前做独立 review（`/code-review`），确认问题后再合并，核对结论不直接采信。
- 回滚：阶段 A、B 为后端内部与契约增量，可逐提交回退；阶段 C 的接口与前端要同组回退；联动不做运行期回退分支。
- 若发现联动在并发下无法保证与发票／核销／对冲串行，停止实施回到设计，不用隐藏分支掩盖。
