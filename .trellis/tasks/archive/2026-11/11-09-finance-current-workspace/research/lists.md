# 研究：财务列表的当前公司范围

- 问题：移除财务列表与提成方案列表的所属公司筛选后，如何保持查询、导出、候选和批量操作都限定当前公司？
- 范围：仓库内部，只读研究六个待改财务列表、已完成的账单范本及提成方案抽屉；创建弹窗内部由其他专题研究。
- 日期：2026-11-09
- 基线：`497d3209`，未修改产品代码、未运行产品测试。

## 已有规范与模式

- `.trellis/spec/web/frontend/organization-switch.md`：公司切换成功后工作区重建，查询、分页、勾选、弹窗和未保存输入隔离；失败保留原输入，不能用清 dirty 绕过确认。
- `.trellis/spec/web/frontend/state-management.md:112`：工作区查询与在途响应隔离；React Query 使用工作区 QueryClient，不再建设序号令牌或全局可变数据层。
- `.trellis/spec/web/frontend/finance-ledger-snapshots.md:15`：六类财务列表的指标、合计、导出使用官方 onLoad 采纳快照；查询完成不能直接更新指标。公司变化清选择，分页、排序保留选择。
- `.trellis/spec/web/frontend/capability-navigation.md`：复用已有 `features/finance/ledger-summary` 和公共 `FinanceLedgerTemplate`，不另造页面状态平台。
- `web/src/app/AppLayout.tsx:233` 按用户+组织 key 重建 `OrganizationWorkspace`。
- `web/src/components/layout/OrganizationWorkspace.tsx:24` 身份缺用户/组织时不挂业务树；`:27` 隔离 QueryClient；`:44` 卸载时取消在途请求、清旧缓存。直接挂页面的测试仍需验证页面自身无组织时不请求，不能只依赖外层布局。
- 已完成范本 `web/src/pages/finance/bills/index.tsx:64` 直接读取 `useInitialState().initialState.currentUser.currentOrganization.id`；`:66` 查询恢复只保存业务筛选；`:446` 缺组织直接返回失败空结果；`:481` 最终载荷显式写当前组织。`bill-query-state.test.tsx:168`、`:196` 已覆盖未就绪和切换。

## 文件地图和待改锚点

| 页面 | 当前公司来源与入口 | 查询、快照和相邻路径 |
| --- | --- | --- |
| 费用台账 `web/src/pages/finance/fees/index.tsx` | `:82` 已读取认证态用于行颜色；`:106` 查询恢复含 organizationId，`:119` 从恢复值初始化手选公司；`:124` 拉 FEE_READ 公司候选；`:248` topBar 公司选择 | `:387` 从 queryState 再取公司，`:436` 进入 ListFeeLedger；`:382` 选择scope、`:383` 采纳快照保持；`:314`、`:347` 批量建账从行中推导公司；`:455`、`:457` 将推导公司和候选名称传工作台 |
| 收付 `web/src/pages/finance/cashflows/index.tsx` | `:96` 恢复筛选含公司，`:115` 手选公司；`:152` CASHFLOW_READ 公司候选；`:514` topBar 公司选择 | `:488` 搜索提交、`:503` 重置继续保留公司筛选；`:557` 从queryState取公司、`:568`进ListCashflows；`:548` scope及`:549`采纳指标保持。`:131`创建公司候选、`:596`创建载荷及`:624`弹窗公司字段由弹窗专题处理 |
| 发票 `web/src/pages/finance/invoices/index.tsx` | `:118` 可空手选公司；`:120` INVOICE_READ 公司候选；`:555` topBar | `:599` ListInvoices，`:607`公司参数；`:590` scope、`:591`采纳快照保持。`:623` InvoiceCreateModal 父级目前不传公司，仅传表单、选中账单和回调 |
| 核销 `web/src/pages/finance/verifications/index.tsx` | `:61` 可空手选公司；`:66` effect取VERIFICATION_READ公司候选；`:379` topBar | `:417` ListVerifications，`:424`公司参数；`:409`scope、`:410`采纳指标保持。`:439` VerificationWorkbench目前只有open/close/onCreated，无公司props |
| 对冲 `web/src/pages/finance/nettings/index.tsx` | `:76` 可空手选公司；`:85` effect取NETTING_READ公司候选；`:426` 独立Card中公司选择（不是topBar） | `:483` ListNettings，`:491`公司参数；`:475`scope、`:476`采纳指标保持。需删除整张仅承载公司选择的Card，避免遗留空白区 |
| 提成 `web/src/pages/finance/commissions/index.tsx` | `:98` COMMISSION_READ公司候选；`:713` 搜索配置中的organizationId选项 | `:87` searchFiltersRef保存查询；`:125` ExportCommissions直接传该ref；`:724`normalize、`:728`reset；`:781`ListCommissions使用spread筛选。`:800`CommissionCreateModal和`:835`RulesDrawer目前不传组织 |
| 提成方案 `web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx` | `:56`管理公司、`:59`只读公司候选及`:62`手选公司；`:76`、`:93`effect分别加载；`:432`公司筛选 | `:492`ListCommissionRules、`:499`公司参数；`:112`通用员工搜索按调用参数可省略公司；`:135`列表适用员工搜索完全未传组织；`:511`给FormModal传候选。员工筛选保留，查询固定当前公司 |

## 最薄实施路径

1. 六列表及方案抽屉按账单范本读取当前公司ID；不新增通用平台或用效应把当前ID再镜像进useState。公司ID是请求上下文，不属于用户可恢复的业务筛选。
2. 移除公司Select、候选查询、对应state/导入和只为该选择提供的外壳；费用、收付查询恢复类型移除organizationId。对业务筛选提交不再使用带旧公司字段的恢复对象作为请求组织来源；不添加旧缓存迁移/兼容分支。
3. 所有List请求在调用API前校验当前组织存在；继续官方request协议，缺身份返回失败而非成功空快照。请求组装最后显式写当前组织，防止后面的spread覆盖。
4. 提成纯业务过滤类型与normalize去公司：`web/src/pages/finance/commissions/commissionExport.ts:13`、`:20`、`:38`。List使用`{page,pageSize,...filters,organizationId}`；Export使用`{...filters,organizationId}`，且无ID不导出。不改变按筛选导出的既有行为、CSV公司展示列、金额格式或文件名。
5. `FinanceLedgerTemplate`内当前页导出继续消费accepted rows，不另发全量List；其他六列表`onAcceptedSnapshot`和summary转换保持。`useLedgerSelectionScope`（`web/src/features/finance/ledger-summary/index.ts:159`）只负责筛选提交，工作区公司变化依赖既有key销毁，足够满足当前模型。
6. 费用选行建账不能仅以“全部行同公司”决定写入目标：`resolveSingleBillCreationOrganization`（fees/index.tsx:51）目前不校验该公司是否等于workspace。应显式拒绝任一缺公司/外公司记录，建账目标恒取当前公司，不能把外公司行悄悄改挂当前公司。移除通过公司候选补名称的依赖；若工作台仍展示当前公司名称，应直接用认证态当前公司名称。
7. 提成方案的List和两类员工候选统一带当前公司，未就绪不请求。创建目前从表单读取公司（RulesDrawer.tsx:207），与移除FormModal公司字段必须同组修改；只删列表Select会让新建无目标。编辑/复制/名单仍保留记录版本和后端校验，异常外公司记录应明确拒绝。

## 提成相邻子视图

`commissions/index.tsx:648`、`:654` 提供待执行调整和月度申请。它们没有公司选择，但List请求目前没有显式组织参数：

- `components/PendingDecreasePanel.tsx:301` 调 ListCommissionAdjustments；生成DTO `web/src/services/roncin/typings.d.ts:8217`支持organizationId。
- `components/CommissionApplicationsPanel.tsx:337` 调 ListCommissionApplications；生成DTO`:8227`支持organizationId。

这两条是否列入本次显式公司载荷同步，应由主会话和后端研究按完整范围定案；不应为了新增一个前端参数改造整个月度申请流程。即使不改，它们仍由现有工作区重建和后端权限范围约束，不可声称仅因参数省略就已证实越权。

## 测试地图与建议

- 费用：`fee-query-state.test.tsx:89`注入旧target-org，`:206`、`:232`当前期望仍使用旧公司，需改成工作区；当前mock已有workspace-org（`:42`）。`billCreationOrganization.test.ts`验证缺公司/混公司，需覆盖单一外公司也拒绝；`nettingEntry.test.ts`保留双方向对冲建账边界。
- 收付：`cashflow-query-state.test.tsx:77`注入target-org，`:157`、`:177`期望旧公司；改为业务筛选与分页恢复但公司恒workspace。保留`cashflow-save.test.tsx`、`cashflow-rate-permission.test.tsx`，创建form及mock须同步工作区来源。
- 提成：`commissionExport.test.ts`同步纯过滤类型；`index.test.tsx`覆盖List与Export共同当前公司、搜索/重置不清公司；`components/CommissionRulesDrawer.test.tsx`和`.form.test.tsx`同步创建与员工过滤候选。
- 发票：`components/InvoiceCreateModal.test.tsx`、`InvoiceIssueAndRedFlushModals.test.tsx`、`InvoiceCancelModal.test.tsx`保护已有流程。对列表新增/调整轻量request协议测试，不只断言Select消失。
- 核销：`VerificationWorkbench.test.tsx`、`verification-allocation.test.ts`保护预览和金额分配；补列表request当前公司与未就绪验证。
- 对冲：`index.test.tsx`、`write-feedback.request.test.tsx`保留版本动作、错误反馈；补列表当前公司参数和移除独立Card断言。
- 公共：`components/layout/OrganizationWorkspace.query.test.tsx`、`.modal.test.tsx`及`workspaceMenus.test.tsx`覆盖切换销毁/请求隔离。已有`bills/bill-query-state.test.tsx`可参考未就绪/组织切换三个用例，避免只在stub里手工传新参数。
- 各页`ledgerExport.test.ts`继续确认“所属公司”文本列保留；新需求移除的是选择器，不是归属展示。
- 实施后先定向Vitest、改动文件Biome与tsc，最后主会话统一完整风险门禁；此专题未执行测试，不把现有用例推测为PASS。

## 外部参考与版本

无外部查询；本次仅消费仓库既有React Query/ProTable模式，不涉及新库或API版本选型。

## 注意事项与未发现项

- 未发现六列表里需要保留公司手选的真实业务差异；当前后端是否还允许跨公司读写由后端专题最终核对，前端研究不替后端断言授权结论。
- 未读取创建弹窗全部实现，不宣称弹窗候选、预览和写入已完整枚举。
- 本专题不改权限真相、SQL、版本和事务，不删除API DTO上的organizationId；显式载荷固定workspace与服务端最终校验可共存。
