# Research: 财务业务弹窗的当前公司归属

- Query: 移除业务弹窗内的所属公司选择后，如何让候选、预览、写入统一使用当前认证公司，并保留会话与未保存输入保护。
- Scope: internal；只读产品源码，只写本研究文件。
- Date: 2026-11-09

## Findings

### 1. 共享身份及隔离边界

- `web/src/access.ts:164` 定义系统工作台，`:168` 定义 `canOperateBusiness`，`:171` 的 `canOperateOrganization(id)` 同时校验公司 kind、有效 ID 和当前公司等值。前端已有判断入口，不需要复制新的一套 kind/权限判断。
- `web/src/components/layout/OrganizationWorkspace.tsx:21` 从 `useInitialState` 取当前用户，身份不完整时 `:24` 不挂业务子树；`:27` 通过用户/组织 key 建立独立工作区。`:44` 卸载时取消工作区请求并清 QueryCache。真实全局切公司会卸载旧弹窗、候选和表单，不能另建全站手工竞态状态。
- 当前账单列表的参照实现为 `web/src/pages/finance/bills/index.tsx:64`；读取 `initialState.currentUser.currentOrganization.id`，缺身份不请求。弹窗同样必须在每个业务动作入口保证存在公司身份；公司 ID 来源只从认证态读取，不能从表单或旧 props 选择。
- 公司切换未保存保护沿用 `GuardedModalForm` / `useUnsavedChangesGuard`，不因自动设定公司而标脏。非公司工作台不得因为“有一个组织 ID”就加载经营候选或提交，优先复用既有 access 结果。

### 2. 建账工作台：固定带入路径必须同时处理

文件：`web/src/features/finance/bill-creation/BillCreationWorkbench.tsx`、`BillCandidateSelectionStep.tsx`。

- Workbench `:46` 定义 props，`initialFeeIds/initialOrganizationId/initialOrganizationName` 是来自单票或费用台账的来源；`:104` 另有可选公司 state，`:128` 有 ref，`:140` 打开即请求可建账公司候选。
- `:229` 的 `runPreview` 允许第三参数覆盖公司，`:240` 优先该覆盖参数，再读 ref，`:271` 写入预览请求。创建最终读取公司 state（`:442` 检查，`:500` 写载荷）。仅删候选 Select 会保留多个公司真相源。
- `:346` 初始化 key 包含来源公司；`:351` 有费用时直接采用来源公司，`:363` 固定带入费用直接自动预览，并于 `:373` 把来源公司作为覆盖参数。必须改为认证当前公司；来源公司与当前公司不等时显式阻断预览，不能悄悄把这些费用重新解释成当前公司的费用。
- `:390` 的 `handleOrganizationChange` 专供本地选择器，删除后同步清理 setter/ref 覆盖路径。保留现有预览会话、group 配置、快照令牌失效、幂等键和账户候选隔离；`:298` clearPreviewSnapshot、`:579` 预览组表单归位均不应泛化重写。
- SelectionStep `:63` 是可自由选费用模式的所属公司 Select；`:107` 是官方 ProTable request，`:114` 传公司，缺公司现在返回成功空列表，目标应是不发送请求且不表现为已成功取得业务数据。固定带入模式 `:52` 只显示来源公司，这个展示可以保留，但不再请求公司候选为它补名称；优先用可信来源名称或当前公司名称，不用新兼容查找。
- 父入口：账单 `web/src/pages/finance/bills/index.tsx:507` 无固定费用；费用台账 `web/src/pages/finance/fees/index.tsx:313` 验证费用只来自一个公司，`:455` 传来源公司；订单费用页 `web/src/pages/orders/fees.tsx:712` 仅保证订单公司非空，`:806` 传来源公司。台账/订单入口都应确保来源等于当前公司，不能只保证“单一公司”。
- 最薄设计：Workbench 从认证态取唯一公司；来源组织 props 仅承担来源一致性断言或在同步调用方后收窄，不再能覆盖请求公司；删除组织候选 Query、selector 和变更回调，候选/预览/创建使用同一认证 ID。保留 NORMAL/NETTING 两模式。

### 3. 收付登记与开票

**收付登记**：`web/src/pages/finance/cashflows/index.tsx`。

- `:74` 的表单 Values 含必填 organizationId，`:131` 取得创建公司候选；`:581` GuardedModalForm 为登记弹窗，`:596` 直接使用 `v.organizationId` 创建。
- `:623` 所属公司字段及其切换清空客商逻辑都应删除；`:648` 结算单位 request 目前从 `form.getFieldValue('organizationId')` 读目标，必须改为同一当前公司，否则删字段后客商候选永远为空。
- `:654` 候选 payload 带公司；缓存 `partyCasualMap`（`:660`）属于客商关联状态，需随既有工作区销毁而隔离。保留现金流方向、付款方式、汇率权限和账户校验；不写隐藏 organizationId 表单补丁，不允许表单残留覆盖当前公司。

**开票**：`web/src/pages/finance/invoices/components/InvoiceCreateModal.tsx`。

- `:58` 本地 organizationID state，`:63` 组织候选 Query；`:178` 是公司 Select，`:189` 切换清账单选择；`:264` 账单 ProTable 按公司 key 重建，`:300` request 按公司过滤。
- `:81` 抬头 Query 按第一张已选账单 ID 驱动，`:87` 现在只检查 open 和 firstBillId。去 selector 后应先确保已选账单属于当前公司，再允许抬头读取；不是把旧公司账单 ID 直接挂到新公司弹窗。
- `:121` 关闭弹窗重置组织/抬头并 removeQueries；可以删“重置当前公司 state”，但抬头与账单选择关闭/重开隔离须保留。
- `:288` 行选择限制方向、公司、结算单位、币种一致；当前只和第一张账单比较，没有比较认证当前公司。选中集合必须属于当前公司，外公司或缺公司行显式不可提交。
- 最终提交在父页 `web/src/pages/finance/invoices/index.tsx:164`，`:172` 创建载荷只有 billIds/profile/type/note/幂等键，没有 organizationId。不能为本次 UI 改动凭空发明后端字段；应在候选与父提交前校验所选账单归属，服务端继续按实体鉴权。

### 4. 核销工作台

文件：`web/src/pages/finance/verifications/VerificationWorkbench.tsx`。

- `:58` Scope 同时保存组织、方向、结算单位、币种/日期/备注；`:101` bootstrap 并行拉公司和币种。移除公司候选后保留币种查询，不把已存在的币种/日期/分配业务一起删掉。
- `:120` 客商 Query、`:162` 资金和账单 Query 都以 `scope.organizationId` 作为依赖；`:196` 作用域改变会清勾选和分配。可从 Scope 删除组织，queryKey/request/enabled 统一引用认证公司；方向、单位、币种变更的原清理逻辑保留。
- `:430` 公司 Select 是上方第一列，移除后其他列可自然扩展；`:479` 与 `:545` 的“先选公司”占位需改为身份未就绪/当前工作区不支持的明确反馈，不保留无法满足的交互提示。
- `:298` submit 先检查组织、单位和分配有效性；`:309` 创建核销载荷由 allocations/date/note/幂等键组成，没有 organizationId。须确保 cashflows/bills 都属于当前公司，不加入无契约字段；既有余额与同币种约束保留。

### 5. 提成生成：删除 Form 字段后必须调整依赖

文件：`web/src/pages/finance/commissions/components/CommissionCreateModal.tsx`。

- `:103` 公司 state 与 `:370` 的 Form.organizationId 双源并存，`:123` 打开查询公司候选。核销/对冲/计提候选 Query 分别在 `:138/:166/:188`，都已把组织纳入 key/enabled，可替换唯一当前公司，删除公司候选。
- `:344` 创建和 `:532` 预览均显式提交组织，现用 `organizationId ?? ''`。去选择后应缺认证公司直接阻断，不能继续发空字符串。
- **关键陷阱**：`:484` 的 `ProFormDependency` 依赖 `['verificationId','nettingId','organizationId']`，`:499` 根据 Form.organizationId 禁用候选。只删所属公司表单项后，即使 state 有当前公司，这里仍会永久 disabled。应去掉组织 Form 依赖，直接使用当前公司变量；不加隐藏字段维持双源。
- `:268` resetSourceSelection、`:277` resetCandidateState 与 `:316` 关闭清预览/来源逻辑应保留；自动绑定公司不算用户编辑。来源切换、候选切换须继续让旧预览失效。
- 预览是直接 await 后 setPreview（`:532`），并非观察 Query；全局切工作区通过卸载保护，但同一弹窗关闭重开/快速改来源仍需相邻测试，不能删组织回调时顺带删所有预览重置。

### 6. 提成方案：列表、表单、复制、员工名单一起收窄

文件：`web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx`、`CommissionRuleFormModal.tsx`、`CommissionRuleCopyModal.tsx`、`CommissionRuleRosterModal.tsx`。

- Drawer `:56/:59/:62` 保存两组组织候选和筛选公司；`:76/:93` 两条旧 useEffect 只服务公司选项，删除即可，不重建新的手写请求链。`:432` 筛选 Select 删除；`:492` 官方列表 request 必须有当前公司，身份未就绪不发请求。
- 员工搜索 `:112` 可接受任意 targetOrganizationId，缺失时省略组织；`:135` 列表员工筛选完全未传组织。目标是都显式绑定认证当前公司；缺公司直接不请求，不依赖后端“省略参数恰巧限制范围”。
- 新建 `:207` 从 ruleFormRef 读公司并写入 `:219`；需直接使用当前公司。FormModal `:94` 组织字段、`:79` 组织 initialValue 删除后，`:214/:224` 两处员工搜索不能继续从 Form.getFieldValue 读组织。
- 编辑 `:195` 按现有 rule ID/version 更新，复制 `:234` 按 source ID 操作；`:148` 名单管理、`:390` 复制员工候选都以 record.organizationId 调用。record 是业务来源，不应成为跨公司目标选择：打开/提交前校验 record.organizationId 等于当前公司，缺失/不等显式拒绝，禁止静默把另公司方案复制到当前公司。
- `CommissionRuleRosterModal.tsx:123` 还将 rosterRule.organizationId 传给员工搜索；CopyModal props `:30` 同样允许组织参数。最薄方案为父入口统一验证、传当前公司；若移除多余参数则一次性同步三个表单消费者，避免把旧可选组织参数留作隐藏能力。
- 提成方案已有只读旧规则、未来生效日期、名单区间与version校验，不属本次重构范围，保持原业务行为。

### 7. 验证清单与现有测试入口

- 建账：`web/src/features/finance/bill-creation/BillCreationWorkbench.test.tsx`、`.scale.test.tsx`、`BillGroupCard.test.tsx`；当前测试通过操作公司 Select 驱动 A/B 切换（如普通 suite `:162/:169`），须改成真实工作区/认证身份切换，保留迟到预览、固定带入、NORMAL/NETTING和大规模分组覆盖。费用来源工具测试为 `web/src/pages/finance/fees/billCreationOrganization.test.ts`。
- 收付：`cashflow-save.test.tsx`、`cashflow-rate-permission.test.tsx`；实际输入并保存，断言表单注入外公司值不能覆盖认证公司、候选参数正确、失败保留输入、付款方式与汇率权限正常。
- 发票：`components/InvoiceCreateModal.test.tsx`，相邻 `InvoiceCancelModal.test.tsx`、`InvoiceIssueAndRedFlushModals.test.tsx`；仓库当前没有发票页面query-state或write-feedback测试，不能假定已有覆盖。去掉选公司操作后断言候选立即请求当前公司、无组织不请求、关闭重开清选择、抬头迟到隔离、混入外公司账单不提交；父页createInvoice需补实际载荷测试。
- 核销：`VerificationWorkbench.test.tsx`；保留金额分配、余额与方向/结算单位变化清选择，补缺身份/非company和当前公司payload。
- 提成：`CommissionCreateModal.test.tsx`、`CommissionRulesDrawer.test.tsx`、`CommissionRulesDrawer.form.test.tsx`；注意旧测试 mock 只有label“所属公司”时才提供选项。需改成当前身份驱动，真实断言候选可用（防隐藏Form依赖）、预览/提交同公司、新建/复制/名单员工范围与非当前记录拒绝。
- 工作区相邻：`web/src/components/layout/OrganizationWorkspace.query.test.tsx`、`.modal.test.tsx`；已有用户+组织 QueryClient/弹窗销毁验证可复用，不新增跨页面组织状态平台。
- 每类覆盖：当前公司成功、认证未就绪、非公司工作区、来源record不等、旧表单/恢复值不覆盖、关闭重开、切工作区及旧响应迟到。重型suite按职责拆分，单文件不堆超过10个深层组件用例，不先写测试。

## Related specs

- 根 `AGENTS.md`：官方request、onFinish、权限、共享事务/版本、未保存保护、生成物约束。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：账单/核销/提成来源与企业经营主体。
- `.trellis/spec/web/frontend/organization-switch.md`：组织切换安全落点、dirty确认、工作区销毁。
- `.trellis/spec/web/frontend/state-management.md`：React Query唯一服务端状态、enabled/queryKey、关闭会话隔离。
- `.trellis/spec/web/frontend/finance-ledger-snapshots.md`：onLoad已采纳快照和筛选清选择。
- `.trellis/spec/server/backend/operating-company-commission-attribution.md`：经营动作只在启用公司工作台，目标必须等于当前公司。

## External references

本次研究只使用仓库当前代码与规范，不涉及第三方API选型或版本变更，无外部资料依赖。

## Caveats / Not Found

- 仅研究前端弹窗，不对后端省略organizationId行为作安全结论；由后端研究者验证服务端最终边界。
- 当前公司自动绑定不等于有写权限；继续使用服务端下发的access能力，缺权限不因移除候选查询而放行。
- 本研究未运行测试、未改产品代码，所有修改方案仍属规划；无真实数据库操作。
