# 实施验收记录

日期：2026-11-18（北京时间）。本任务未修改海管家数据；真实写入仅用于本地专用集成库中本轮创建的隔离Schema。未清理开发库或已有业务数据。

## 已完成的产品变化

- FinanceBillBusinessStatus与businessStatuses查询数组、businessStatus读取投影已由Proto生成Go/OpenAPI/TS客户端及枚举。
- 五态由biz解析、data筛选同源，页面与CSV仅映射投影；多选OR／其他条件AND，统一total和完整匹配汇总。
- 无票账单核销候选排除及现有账单锁内最终拒绝；部分票／票齐／免票可办理。建账附带核销先登记发票，删除登记行保留核销草稿但禁止无票提交。
- 最终补查将账单详情的“正常”改为同源五态，取消终态保留；新增详情6例通过，tsc与Biome通过。
- 受影响正向核销测试与验收流程补齐发票／免票前置，新增无票组合失败不留资金／占用检查。

## 定向与跨层验证

- 前端初轮8文件28例通过；增加隐藏恢复后账单查询文件5例通过；原建账工作台6例通过。tsc、手写变更文件Biome及git diff --check通过。独立check代理另跑6文件24例及tsc/Biome，结果见review.md。
- 后端biz/service及真实HTTP多选绑定定向通过；绑定覆盖重复枚举、蛇形、枚举名及Axios的businessStatuses[]形式。真实PG五态／OR／AND／分页完整汇总矩阵与核销开票前置、混合账单、差额资金回滚、同键回放和反核销专项通过。最终ISSUED票据夹具版10项PG专项通过（31.014秒），命令与矩阵见[后端记录](backend-validation.md)。整包真实PG扫描在15分钟超时，且出现航空迁移清理、单位初始化、人工审核锁旧夹具等范围外错误，未算通过。
- 独立本地运行环境（专用集成库内本轮新建Schema，正式SQL迁移、独立8010/8011端口）中，实际运行acceptance-finance-bill-batch --apply与acceptance-finance-payable --apply均PASS。覆盖应收应付、发票、资金、核销、提成、反向余额等现行API流程；应付票齐后全额核销正确为COMPLETED。
- 在上述环境实际运行acceptance-finance-foreign-currency PASS，覆盖USD本币／EUR业务币，并调用verifyCompositeFinanceWorkflow。新增“无票建账＋差额核销”拒绝、资金计数不变与费用未占用检查通过；同次两票登记／两笔差额核销、整组回滚、幂等及利润／收付／账龄链路通过。
- 真实Playwright页面观察五态多选“未开票＋部分开票”：浏览器请求businessStatuses[]=1&businessStatuses[]=2，返回记录businessStatus均属选择集合，total=matchedSummary.count；重置后请求去除该参数；无pageerror。截图与结果在仓库外/tmp/roncin-bill-status-qa/。

## 全量门禁的现存失败

已执行pnpm run check:fast，耗时287.9秒：服务端层级、测试、vet、Proto与漏洞扫描通过；前端类型／架构／生成物／lint阶段通过，Vitest结果为4文件失败、587通过、1跳过，20例失败、3050通过、11跳过。未将全量门禁宣称通过。

失败文件均为本任务未改的：

- src/pages/finance/commissions/index.test.tsx：5例，既有权限mock导致看不到导出／维护入口。
- src/pages/orders/new-create-idempotency.test.tsx：3例，useBlocker缺data router测试上下文。
- src/pages/orders/new.test.tsx：10例，同类data router上下文。
- src/pages/orders/new-source.test.tsx：2例，现行来源／路由夹具不匹配。

使用git archive HEAD从产品修改前基线导出到/tmp/roncin-bill-status-baseline/，复用同一依赖，分别原样运行上述4文件，复现同样20失败：前两文件8失败／5通过，后两文件12失败／8通过。由此确认不是本次账单改动引入，未扩展任务修复其他业务模块，也未关闭检查或降低规则。

原始log在/tmp/roncin-bill-status-qa/check-fast.log、baseline-failures.log、baseline-other-failures.log，不进入Git。

## 完整页面E2E

最终已实际通过 `pnpm --dir web exec playwright test tests/e2e/finance-bill-batch.e2e.ts`，耗时1.4分钟。当前链路为真实页面录费、订单详情审核锁、建账、收款、登记已有发票（含手填税额7.89）、核销、生成提成12.50、刷新重读、报表桌面／窄屏及同域资源验证。发票登记提前不反改费用税基，提成保持原来源净额口径，最终值一致。

执行中曾暴露旧E2E入口漂移：锁定入口已从费用页移到详情页、刷新快照改名刷新账单、工作台已由Drawer改Modal，以及异步选择费用科目尚未落入行数据时就填金额。仅校准测试的实际导航、按钮定位和等待业务字段落定的断言，未为通过E2E修改无关产品行为，没有sleep／自动重试。失败原始记录保留于e2e*目录，最终通过证据为 `/tmp/roncin-bill-status-qa/e2e-business.log` 与同名目录，不将早期失败隐藏或当通过。

## 提交与清理

功能主提交为1c054567。随后补齐详情五态及E2E现行关闭入口；最终补修提交随任务收尾记录。临时8010/9010/8011服务已停止，本轮新建隔离运行Schema已删除，临时进程环境中的凭据缓存已移除；原开发服务8000/8001继续正常运行。测试原始证据保留在仓库外。
