# 费用审核实施审查记录

日期：2026-10-12。角色：trellis-check。审查依据：check.jsonl、PRD、design、implement、领域地图及后端/前端质量、事务、权限、金额、表格规范。工作区仍由主线程与实现同伴共同维护；本报告记录本次审查和定向验证，完整收尾门禁与浏览器最终验收由主线程追加。

## 已核对的关键路径

- 审核事实在 Order 与 OrderFeeReviewEvent 上，每订单一个整票费用包；审批涵盖全部应收、应付和利润，未引入费用行审批事实。自审仍分提交与批准两个动作；亏损意见由服务端复核。
- 快照指纹排除账单关联、通用行版本、时间和结算单位展示名。普通费用五个写入口及拆票转移均受 PENDING/APPROVED 冻结限制；单张、批量建账再次复核批准、内容集合和待审补录。
- 费用/审核/建账参与同一 Order 行锁，授权读取与角色/成员授权写入在父行共享锁/写锁上互斥。权限和范围来自同一 grant，未复用职位或原锁单人；日后撤权不抹掉历史合法批准。
- 新补录依有效整票批准，批准原子刷新完整快照与 CONFIRMED 冲减；原账单和提成金额保持历史事实。父提成净发与调整执行原子关联；实际现金证据和来源反转共同复核，未知旧付款拒绝新现金调整。
- 正式迁移与 Ent CHECK 同源，迁移预检不自动处理旧待审申请/草稿冲减/非法付款组合。种子仅为本轮新建的演示订单记录明确批准，不将存量订单默认为已审。

## 已修复问题

1. `server/internal/data/order_fee_review.go`：新 authenticated 审核接口使用旧锁资格谓词时未排除停用组织，all 范围可借停用成员组织授权。候选和实际决策同步校验目标组织启用，grant 同时要求成员组织启用；规范同步至 order-fee-review.md。真实 PG 验证两类停用场景均拒绝，候选为空。
2. `server/internal/data/finance_bill_transaction_integration_test.go`、`finance_bill_batch_integration_test.go`：旧账单夹具无有效人工批准证据，先被新审核门禁挡住，未到达原并发/账户/回滚/预览断言。补真实操作用户、显式整票快照事件与清理；陈旧费用事实在重新建立当前批准后仍验证旧预览令牌拒绝。补结算账户按测试组织清理，避免错误结算单位子场景残留。
3. `server/internal/data/order_fee_bulk_integration_test.go`、`order_fee_delete_integration_test.go`：旧测试要求批准后可删同订单未建账行、取消账单自动可删，与整票冻结冲突。同步冻结优先错误；取消账单仍冻结、显式撤销夹具批准后删除并继续验证历史行快照/来源置空。删除各子场景独立订单，保留版本、审计回滚、标签级联及建账竞争断言。
4. 新增 `server/internal/data/order_fee_review_concurrency_integration_test.go`：真实 PG 验证提交与删除、批准与撤回只有一方成功，并校验完整费用包和批准指针终态；补停用目标和停用授权成员组织反例。
5. 四个 `web/src/pages/orders/detail-*.test.tsx`：费用抽屉新增 React Query 后，原订单动作/草稿/共享箱测试未提供其查询上下文。按测试职责隔离费用 panel facade，保留所有原业务断言，费用审核本身由独立测试及浏览器覆盖。
6. `web/src/pages/orders/components/PartnerQuickAddSelect.failure.test.tsx`：happy-dom 下只触发 animationEnd 不能保证弹窗退出。等待 leave-active 并在 act 中触发 transitionEnd，保留失败内容、重试载荷一致、关闭和重新打开清表断言。

以下由负责文件的实现同伴据审查意见修复并验证：

- supplement 同伴补独立调整执行的实际付款证据/现金余额门禁，与可执行投影和来源反转共用 commissionCashExposure；未执行增提不能扩张扣回现金额度。其定向真实 PG 验证通过。
- web 同伴为批量、明细、历史审核表格接入 useColumnSettings 与 ResponsiveTable；应收应付共享偏好，稳定表格键不含订单 ID。嵌套表头的全局 CSS 精确直属选择器修复由同伴维护，最终真实浏览器由主线程复核。

## 验证结果

- `git diff --check`：PASS。
- `pnpm --dir web lint`（Biome + TypeScript）：PASS；`/tmp/roncin-fee-review-reviewer-final-lint.log`。Biome 仅有 7 条已有其它模块 info，无 error/warning。
- `go -C server vet ./internal/biz ./internal/service ./internal/data ./cmd/sync-dev`：PASS；`/tmp/roncin-fee-review-reviewer-vet.log`。
- 注入隔离 PG，四组账单/费用相邻回归及两组新竞态：PASS，17.740 秒，`/tmp/roncin-fee-review-reviewer-pg.log`。各目标实际执行，无 SKIP。
- 注入隔离 PG，全部 OrderFeeReview 生命周期/授权/并发/通知/批量：9 组 PASS，28.704 秒，`/tmp/roncin-fee-review-reviewer-auth.log`；增强停用成员组织反例后单组复验 PASS，2.618 秒，`/tmp/roncin-fee-review-reviewer-disabled-org.log`。
- 前端上述 5 文件：33 测试 PASS，8.76 秒；`/tmp/roncin-fee-review-reviewer-web.log`，无 act/deprecated 噪音。
- 主线程提供的前次完整 `check:server` PASS 已核对；前次完整真实 PG data 失败定位的账单夹具已在上述定向修复。supplement 同伴最终广定向真实 PG（Commission/OrderCommission/FeeSupplement/Workbench）PASS，215.620 秒。

## 未自行处理的事项

- 全量 check:fast、最终全部 PG data/migration 和最新表格真实浏览器由主线程统一收尾，不以本报告代替最终验收。
- `order_fee.proto` 中 Create/List 补录注释仍描述旧锁准入，已交其所有者同步真相源并通知主线程统一生成；不可手改生成物。
- 其它已有模块的 7 条 Biome info 与本任务无关，未作无关格式化。
- 现有开发库旧待审补录按迁移预检拒绝自动处理，不删除、撤回、批准或重置；当前测试数据说明不等于这些动作授权。本项不是代码回退分支，环境处理由主线程决定。

截至本轮审查，未发现仍需产品决策的高风险代码缺陷；上述收尾验证完成后方可宣告整任务验收。

## 最终门禁相邻测试补查

主线程完整门禁其余通过后发现两处旧夹具问题，按主线程指派仅改对应测试文件：

- `orders-breadcrumbs.test.tsx`：完整服务 mock 缺新审核组件模块初始化所需的动作导出。改为保留生成服务的 actual 导出，并显式 mock 五个审核动作及未提交费用包预览；保留未知路由禁止请求、加载/不存在/面包屑原断言。
- `components/order-commission-summary.test.tsx`：单独及两文件并跑均通过，完整高并发时摘要查找发生在官方 ProTable request 渲染目标行前。依实际请求生命周期先等待查询调用、订单行出现，再断言原金额和下钻；不增加 sleep、不扩大 timeout、不改产品行为。
- 修改后两文件共 17 测试 PASS，9.19 秒；日志 `/tmp/roncin-fee-review-reviewer-final-adjacent.log`。Bi​​ome 定向检查通过；完整 web lint（含 tsc）另存 `/tmp/roncin-fee-review-reviewer-adjacent-lint.log`。完整门禁仍由主线程统一复验。

### 最终完整前端复验

第二轮完整门禁仅在同文件缺省摘要场景仍暴露列表请求尚未渲染的时序。将该文件所有列表接线用例统一为请求启动后等待目标行；beforeEach 统一 mockReset 请求实现、恢复权限初值并清理本地/会话列表偏好，防止上一项请求和偏好残留。全部原金额、空态、点击下钻断言保持不变。

在主线程停止同时跑浏览器后执行一次完整 `pnpm --dir web test`：**223 文件通过、1 文件既有跳过；1184 测试通过、12 测试跳过；49.64 秒，退出码 0**。日志 `/tmp/roncin-fee-review-reviewer-web-all.log`。跳过不计为验证通过；日志仍有测试环境 `ECONNREFUSED localhost:3000` 网络噪音，未出现失败，不将本次结果宣称为所有 stderr 均为零。
