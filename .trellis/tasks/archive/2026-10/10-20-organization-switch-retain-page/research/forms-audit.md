# Research: 切换组织的未保存输入保护覆盖

- Query: 审计现有页签/路由守卫与共享表单接入，重点角色、财务弹窗和同页多守卫。
- Scope: internal
- Date: 2026-10-20
- 工作区：`/tmp/roncin-org-switch-20261020`。本文只读研究产品代码，没有实现产品变更。

## Findings

### 事实：当前保护入口与覆盖

- `web/src/components/OrganizationSwitcher.tsx:72`：切换前调用 `confirmIfAnyTabDirty`；切换 API 成功才更新身份并跳转。失败不主动清空 UI。未注册 guard 的输入无法被此函数发现。
- `web/src/components/layout/tabCloseGuard.ts:34`：当前注册表是 `Map<string, TabCloseGuard>`；`registerTabCloseGuard`（74）执行 `guards.set(tabKey, guard)`，一个菜单页签最多保留最后注册者。已实现异步确认放行，Promise 完成/失败后撤回，不应以清 dirty 的方式绕过组织切换确认。
- `web/src/components/layout/tabCloseGuard.ts:110`：`getAnyDirtyTabKey` 只找挂载表单，不扫描其他组织的持久草稿；这是当前约定，不应扩大为全 sessionStorage 草稿扫描。
- `web/src/components/layout/useRouteLeaveGuard.ts:7`：Data Router `useBlocker` 保护 pathname/search 变化，只在客商页面生产调用。它无法发现没有注册或没有提供 dirty 的弹窗。
- 全仓生产调用 `useTabCloseGuard` 的位置只有 `OrderFormTemplate.tsx:132` 和 `partner-detail.tsx:145`；生产直接注册位置只有 hook 本身。其余命中为测试。
- `web/src/components/ui/order-template/OrderFormTemplate.tsx:130`：模板维护 `internalDirty`，用户修改触发 `onValuesChange`（300）并写组织/用户隔离草稿；resetTo/成功提交清理。其调用方可禁用 guard，且 tabKey 必须显式提供。现有能力应保持所有权。
- `web/src/pages/partners/partner-detail.tsx:138`：同步 `dirtyRef` 同时接入页签与路由守卫（145–146）；`onValuesChange`（825）及联系人/别名等外部主表状态接入 `markDirty`。但联系人子弹窗在点击“添加/保存”前的输入没有进入父 contacts 状态，因此父 dirty 不能代表其尚未提交输入。
- `web/src/components/layout/OrganizationWorkspace.tsx:26`：身份改变按用户+组织 key 整体重建业务子树，QueryClient、列表查询上下文和弹窗会卸载。成功切换后无需挨个关闭弹窗；失败时必须不改变该身份 key。

### 事实：角色弹窗的实质缺口

- `web/src/pages/admin/components/roles/RoleFormModal.tsx:302` 是直接 `ModalForm`，没有 `onValuesChange` 或关闭守卫；输入角色名称后切组织不会提示。
- `RoleFormModal.tsx:122` 的 `setSelectedPermissionKeys` 统一包装了矩阵业务勾选（全选、只读、清空、单项/资源变更都调用它）。权限集合由父层 React state 管理，既不是可见 Form 字段，也不能指望 `formRef.current.isFieldsTouched()` 检测；只勾选权限再切组织是独立必须验证的真实场景。
- 搜索、模块导航、资源展开仅用于浏览权限，不应单独构成待保存变更。

### 事实：共享表单层尚未保护

以下不是未来抽象，而是当前已经承载真实编辑弹窗的共享入口：

| 文件 | 现有消费者或场景 | 缺口 |
| --- | --- | --- |
| `web/src/components/ui/master-data-template/MasterDataTemplate.tsx:789` | 主数据 CRUD | `ModalForm open={modalOpen} form={form}`，无 dirty 接入 |
| `web/src/components/ui/parameter-setting-template/SettingTableTemplate.tsx:205` | `finance/fee-settings/components/{BillingUnitsPanel,FeeItemsPanel,TaxableServicesPanel,FeeTemplatesPanel}`，管理员异常分类 | `ModalForm` 无 dirty 接入 |
| `web/src/components/ui/sub-entity-card-grid/SubEntityCardGrid.tsx:174` | `pages/partners/components/ContactCardList.tsx:85` 的联系人 | 子弹窗未保存前父档案 dirty 仍可为 false |
| `web/src/components/ui/sub-entity-drawer/SubEntityDrawerTemplate.tsx:258` | 子实体新增/编辑 | `ModalForm` 无 dirty 接入 |

当前未发现项目自建 `Form.Provider`、`onFormChange` 或全局表单脏状态机制。不能假定加一层 Provider 就自动覆盖全部表单：项目真实存在 React state 控件、权限矩阵及表格选中行不经过 Form store。

### 事实：财务页面与弹窗

- `web/src/pages/finance/bills/components/BillDetailDrawer.tsx:85` 只展示详情，没有可编辑 Form；它自身不需要“未保存”提示，成功切换后的工作区重建应关闭它。
- `web/src/pages/finance/bills/components/BillEditModal.tsx:173` 的真实编辑弹窗使用 `Modal + Form`（183），有账期、结算账户、备注等字段，未接入 dirty。异步候选还会调用 `setFieldValue` 自动填默认账户（156）；以“任何 store 变化”为 dirty 会把初始化当用户编辑。
- `web/src/pages/finance/cashflows/index.tsx:546` 的登记资金流水直接 `ModalForm`，没有关闭保护。
- `web/src/pages/finance/invoices/components/InvoiceCreateModal.tsx:162` 的新增发票为 `Modal + Form`，除此之外有 `selectedIDs/selectedBills` 外部状态（49–51）；只检测 Form 输入会漏掉用户挑选账单。`InvoiceIssueAndRedFlushModals.tsx:27,72`、`InvoiceCancelModal.tsx:35` 同样有待提交字段且无保护。
- `web/src/pages/finance/verifications/VerificationWorkbench.tsx:78–93` 的核销工作台完全使用 React state，包含 scope、资金流水/账单选中行、allocations；用户输入分配金额（361）、选择结算单位（493）、核销日期（522）、备注（656）不经过 Form。这里需要显式 dirty，不能套用 `Form.isFieldsTouched`。
- `web/src/pages/finance/fees/detail.tsx:138` 虽然主体为只读费用详情，嵌入了 `features/orders/fee-correction/FeeCorrectionSection.tsx:336` 的审核 `ModalForm`，不能因父页面叫 detail 就一概视作无编辑状态。
- `features/finance/bill-creation/BillGroupConfigurationStep.tsx:84` 已有 `onValuesChange`，但这只是配置联动，没有注册关闭守卫。若本期要求覆盖所有正在录入的财务工作台，应把建账配置也列入明确验收；不能宣称现有订单/客商守卫已覆盖它。

### 推荐：本期最薄实现

以下是技术建议，不替用户扩张已批准范围：

1. 继续复用同一 `tabCloseGuard` 注册与组织切换确认入口，先将存储调整为每 tabKey 多个 guard（例如 Set），按任一 dirty 判定；注销只删除当前对象，空集合再删除 tabKey。`confirmedDepartures` 应覆盖本次动作涉及的所有 dirty guard，对象身份保持隔离；`confirmIfAnyTabDirty` 当前只放行找到的首个 tabKey，未来同时有多个 dirty guard 时要避免漏放行或重复确认。
2. 增加轻量 hook（适合放 `components/layout/`），接收 `tabKey`、`enabled/open` 并返回同步 `markDirty/resetDirty`，复用 `useTabCloseGuard`。如果要让模板从当前路由取稳定菜单 key，解析复用 `resolveTabKey`，不要复制菜单规则。只需要切组织保护时不要为每个弹窗额外安装一个 `useBlocker`；多个路由 blocker 有争抢风险。
3. 在上面四个真实共享模板的 `ModalForm.onValuesChange` 接入 markDirty，关闭、重开另一记录或验证成功保存后清 dirty。父表单与联系人弹窗共用菜单 key，必须验证互不覆盖。保留模板现有 `onFinish` Promise 生命周期。
4. 角色在 `ModalForm.onValuesChange` 标记字段修改，并在统一的权限集合 setter 标记业务勾选；打开/关闭和成功保存按弹窗身份清理。父层加载已有权限不走用户 setter，避免初始化误提示。权限搜索与展开不标脏。
5. 财务 `Modal + Form`（优先 BillEditModal）只在用户 `onValuesChange` 时标记，并以 open 门控。现金流等直接 ModalForm 同理。核销 allocations、发票选择账单与建账工作台通过现有用户动作回调补一行 markDirty；不要用全局 DOM input 扫描或自动拦所有点击。
6. 失败响应或请求异常不能清 dirty、关闭弹窗或修改工作区身份；确认只是授权这次离开，不能提前重置旧表单。这样切组织失败后输入自然保留，下次切换依然提示。
7. 若本期产品范围是“所有未保存新增/编辑页面与弹窗”，仅修角色不满足该范围；实际财务 state 工作台与联系人子表单必须有明确覆盖或明确收敛后的验收。不要把“已注册 guard 的表单”写成全部表单已经保护。

### 必要验证建议

- 角色仅改名称、仅勾权限分别提示；只搜索不提示；取消切换原输入保持；确认后 API 失败仍保持，下次重试仍提示。
- BillEditModal 仅自动填充默认账户不提示，用户改备注/账期提示；只读 BillDetailDrawer 不提示并在成功切换后关闭。
- 客商父表单+联系人未提交弹窗同时 dirty；关闭/卸载其中一个 guard 后另一个仍有效；确认成功只确认一次。
- 核销仅修改分配金额、仅改日期/备注、仅勾选账单都能保护；Form.Provider 方案的测试不能代替此回归。
- 共享模板覆盖一次真实用户输入到组织切换取消/失败的链路，不能只调用 mock `isDirty: true` 证明表单已接入。

## Related Specs

- `.trellis/spec/web/frontend/state-management.md:138`：客商页签/路由守卫协调、异步失败不永久放行。
- `.trellis/spec/web/frontend/state-management.md:161`：订单草稿生命周期模板独占，用户/组织隔离。
- `.trellis/spec/web/frontend/component-guidelines.md:197`：页签路径和工作区状态身份。
- `.trellis/spec/web/frontend/index.md`：共享能力、权限来源、按风险验证。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：组织角色与财务业务工作台关系。

## External References

本次为仓库现状与最小接入研究，未依赖外部资料。`web/package.json` 声明 `antd ^6`、`@ant-design/pro-components ^3.1.14-2`；具体 hook 行为建议以项目已使用 API 和实现阶段的本地类型检查为准。

## Caveats / Not Found

- 未修改产品、规范、提交或任务方案；实施文件数量由主会话依据批准范围确定。
- 财务还存在提成、汇率等独立 ModalForm；它们同样未在 guard 生产调用检索中出现，但本文不以关键词命中作为逐一完成的数据流审计。若要求全站保证，应继续以实际编辑入口补充覆盖清单，不可用本研究替代全站验收。
- 未运行测试或操作数据库；这里的“存在缺口”依据真实事件绑定与守卫调用链静态证据，不冒充运行验证。

## 补充：本地验证 Provider 方案与当前真实入口清单

### Provider 对比结论（有本地实现证据）

本地安装 `@rc-component/form@1.8.6`：

- `node_modules/.pnpm/@rc-component+form@1.8.6_react-dom@19.2.8_react@19.2.8__react@19.2.8/node_modules/@rc-component/form/es/Form.js:48` 注册/注销时携带 `name`；63 行即使无 name 也调用 `triggerFormChange(name, changedFields)`。
- 同目录 `FormContext.js:44` 只有 `name` 非空才存入 `formsRef.current[name]`；53 行注销不暴露应用侧通知。
- 因而 Provider **能收到无 name 的变化事件，但无法通过公开事件区分/定位多个无 name Form 或获得其销毁通知**。账单编辑的 Form（183）确实没 name。强行统一收集将引入身份分配、闭合生命周期和表单改名，最终仍需多数调用方接入。不能推荐用匿名 name 的全局 boolean 保留脏状态，那会在已关闭弹窗后持续误提示。
- Provider 也不会捕获角色矩阵或完全使用 state 的核销输入。故本期优先“现有 guard + 薄 hook + 真实表单显式绑定”，不建议 Provider 自动兜底。

### 当前实际独立表单清单（均有待提交字段、未注册 guard）

下表明确列出当前会随工作区销毁的输入入口，以便主会话分配实施；这些不是新功能要求。每组的最薄改法是对已有 `ModalForm/Form` 的用户变化加同一 hook，处理 open 生命周期，并保留其已有 onValuesChange 业务逻辑。

| 领域 | 文件与真实输入位置 |
| --- | --- |
| 管理 | `pages/admin/components/number-rules/NumberRuleEditModal.tsx:34` 编号规则；`org/OrgCreateModal.tsx:38`、`org/OrgEditModal.tsx:34` 组织；`users/DingTalkAssignmentModal.tsx:58` 开通分配、`users/ResetPasswordModal.tsx:23` 密码、`users/UserFormModal.tsx:221` 用户、`users/UserMembershipModal.tsx:54` 组织成员；`roles/RoleFormModal.tsx:302` 角色及外部矩阵 |
| 财务 | `pages/finance/bills/components/BillEditModal.tsx:183`；`cashflows/index.tsx:546`；`invoices/components/InvoiceCreateModal.tsx:170`、`InvoiceCancelModal.tsx:68`、`InvoiceIssueAndRedFlushModals.tsx:34,81`；`exchange-rates/components/ExchangeRatesPanel.tsx:354`；`commissions/components/CommissionAdjustmentModal.tsx:34`、`CommissionCreateModal.tsx:307`、`CommissionPaymentModal.tsx:48`、`CommissionRuleCopyModal.tsx:52`、`CommissionRuleFormModal.tsx:54` |
| 客商 | `pages/partners/components/ContractCardList.tsx:406` 合同；`ShippingPresetSection.tsx:548` 出货预设；`RoleSwitchModal.tsx:161` 角色变更；`secondary/AccountsPanel.tsx:200` 账户、`AttachmentsPanel.tsx:140` 附件、`ContractsPanel.tsx:181` 合同、`InvoiceProfilesPanel.tsx:162` 开票资料、`SettlementRulesPanel.tsx:338` 结算；`pages/partners/index.tsx:653` 黑名单填写 |
| 企业资源 | `pages/enterprise-resources/components/ResourceEditorModal.tsx:90` 资源表单及 imageFiles state（357）；`TagGroupModal.tsx:47` 标签组 |
| 订单 | `pages/orders/abnormal-case-panel.tsx:290` 异常记录；`release-pod-panel.tsx:448` 放货记录；`components/drawers/ShippingDocumentDrawer.tsx:349` 单证；`components/detail/OrderLockControl.tsx:405` 解锁原因；`components/fees/FeeFormModal.tsx:102` 费用新增编辑、`FeeSupplementModal.tsx:156` 补录、`FeeSupplementSection.tsx:833` 审核意见、`fee-review/OrderFeeReview.tsx:130` 审核意见、`fee-review/BatchFeeReviewModal.tsx:250` 批量意见；`features/orders/fee-correction/FeeCorrectionSection.tsx:336` 更正意见；`components/fees/FeeCorrectionCreate.tsx:106` 更正方式后续待提交表单 |

不要将现有 `FeeFormModal.onValuesChange`（112）或 `CommissionCreateModal.onValuesChange`（326）误认为关闭守卫，它们调用的是业务联动回调。批量审核“意见”是输入，要保护；单纯列表勾选不是未保存编辑。

### 对最小文件集合的实质判断

- 能证明角色这个原始场景修好：`tabCloseGuard.ts`、一个薄 dirty hook、`RoleFormModal.tsx`、组织切换主体与其测试。这个集合**不能**证明“未保存新增/编辑页面或弹窗先提醒”全局承诺成立。
- 能满足已批准的全局表单承诺：上述基础设施 + 四个共享模板 + 表内独立编辑 Form/ModalForm 入口。独立入口只是几行事件绑定，属于必要覆盖成本，不需要复杂框架。
- 非 Form 的财务工作台应单列核验：创建核销的备注/分配金额/手工日期确实是待提交输入；只打开工作台、筛选候选、浏览和单纯批量选中不应直接标脏。建账应只从用户编辑配置事件标脏，不以预览请求或候选选中作为通用“编辑”判断。
- 上表只列有真实表单的输入，不把所有 `Input` 关键词、列表搜索、只读详情、登录页面算入范围。还有 `modal.confirm` 的输入原因（例如 `FeeSupplementSection.tsx:562`），这是确切的临时待提交文本；如其反馈弹窗允许组织切换交互，应在该具体 action 的弹窗生命周期绑定 guard，不能假设工作区卸载会自动处理 App modal 的闭包。
