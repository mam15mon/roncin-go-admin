# 领域编辑输入覆盖矩阵

工作区：`/tmp/roncin-org-switch-20261020`。领域实施只登记真实用户编辑事件，未改请求、权限或业务提交语义。

## 共用接入与生命周期

- 下列 ModalForm 明确改为公共出口 `@/components/ui` 的 `GuardedModalForm`；它透传原 form/formRef、onValuesChange、官方 onFinish Promise，只有用户 onValuesChange 标脏，open/编辑身份变化、onFinish 返回 true 或关闭后清理。原有业务联动回调保留。
- Form + Modal 直接使用 `useUnsavedChangesGuard`，enabled 来自实际 open，编辑记录 ID 作为 identity；程序 setFieldsValue / setFieldValue、候选请求与默认值均不标脏。成功原流程关闭弹窗后清理，失败不清理。记录可在同一弹窗切换的 ModalForm 显式传 guardIdentity。
- 完全使用 React state 的编辑，只在真实待提交载荷的用户回调内 markDirty；不以 useEffect 监听 state、不扫描 DOM、不把默认回填当编辑。
- 每页多个守卫独立注册，父表单不代替尚未提交的子表单；联系人和共享 CRUD/QuickCreateModal 由守卫代理处理，主客商与 OrderFormTemplate 既有守卫保留。

## 财务

| 真实入口 | 接入与待提交数据 | 排除或生命周期说明 |
| --- | --- | --- |
| bills/components/BillEditModal | Form onValuesChange；账期、日期、结算账户、预计开票配置、备注 | 异步默认账户回填不脏；按账单 id 清理 |
| cashflows/index 登记资金流水 | GuardedModalForm；金额、交易日、账户、备注等 | 列表搜索/筛选不脏 |
| invoices/components/InvoiceCreateModal | Form onValuesChange + 用户关联账单 rowSelection 标脏 | 公司仅过滤候选不脏；账单关联是实际提交载荷，选择即保护；默认开票抬头回填不脏 |
| InvoiceCancelModal | Form onValuesChange；取消/作废原因 | target.id 身份清理 |
| InvoiceIssueAndRedFlushModals 两表单 | Form onValuesChange；税票号、开票日期、红字票号、红冲日期、原因 | 两守卫互不覆盖，分别按 target.id/open 清理 |
| exchange-rates/components/ExchangeRatesPanel | GuardedModalForm；币种、有效期、汇率 | editing.id 清理，列表条件不脏 |
| CommissionAdjustmentModal | GuardedModalForm；订单、方向、金额、原因 | detail.id 身份清理 |
| CommissionCreateModal | GuardedModalForm + 独立外部 state guard；计提人、归属、备注、实际对冲来源选择 | 关键词、候选浏览、纯来源 Tab 切换不脏；默认/预览联动不脏；创建成功两 guard 清理 |
| CommissionPaymentModal | GuardedModalForm；实际抵扣确认字段 | record.id 身份清理，预览请求不脏 |
| CommissionRuleCopyModal / CommissionRuleFormModal | GuardedModalForm；方案、日期区间、适用员工等 | copySourceRule.id 或表单 key 控制身份 |
| CommissionRuleRosterModal.RosterChangeForms | 两个独立 Form guard；加入/移除员工、变更日期 | 加入成功只清加入表单，移除成功只清移除表单；搜索不脏 |
| verifications/VerificationWorkbench | 外部 state guard；手工核销金额、核销日期、备注，生成实际分配记录的自动分配动作 | 公司/方向/结算单位/币种用作候选过滤及普通勾选资金/账单不标脏；打开/默认日期不脏；提交成功清理 |
| features/finance/bill-creation/BillCreationWorkbench + BillGroupConfigurationStep | 父工作台 guard；用户拆单策略配置、账单组 Form 的抬头/日期/账期/账户/汇率/备注 | guard 在父工作台跨步骤保留；默认预览、默认账户回填、候选费用勾选、组导航不脏；建单成功清理 |

## 往来单位与企业资源

| 真实入口 | 接入与待提交数据 | 排除或生命周期说明 |
| --- | --- | --- |
| ContractCardList / ShippingPresetSection | GuardedModalForm；合同与出货预设字段 | editingContract/editingPreset.id 身份清理；纯分类 Tab 不脏 |
| RoleSwitchModal | Form onValuesChange + handleQuickSelect 真实用户快捷身份选择 | 默认角色 setFieldsValue 不脏；保存成功显式 resetDirty |
| secondary/AccountsPanel | GuardedModalForm；账户资料 | editingAccount.id 身份清理 |
| secondary/AttachmentsPanel | GuardedModalForm；附件信息 | 仅表单输入，列表浏览不脏 |
| secondary/ContractsPanel | GuardedModalForm；合同信息/日期 | editingContract.id 身份清理 |
| secondary/InvoiceProfilesPanel | GuardedModalForm；开票抬头、税务资料 | editingProfile.id 身份清理 |
| secondary/SettlementRulesPanel | GuardedModalForm；结算规则、账期、额度 | editingRule.id 身份清理 |
| partners/index 黑名单 | GuardedModalForm；黑名单角色、原因 | 原 onValuesChange 联动保留，列表候选不脏 |
| enterprise-resources/components/ResourceEditorModal | Form onValuesChange + 图片 Upload.onChange/onRemove 的外部 imageFiles state | 身份为资源类型 + editing.id；父级初始化图片不脏；实际上传/删除保护 |
| enterprise-resources/components/TagGroupModal | Form onValuesChange；标签组名称、颜色、排序 | editingGroup.id/open 清理；点击选择已有标签组仅初始化不脏 |

## 订单、审核与海运

| 真实入口 | 接入与待提交数据 | 排除或生命周期说明 |
| --- | --- | --- |
| abnormal-case-panel | GuardedModalForm；异常记录字段 | 只读详情无编辑守卫 |
| release-pod-panel | GuardedModalForm；放货资料 | editingRecord.id 身份清理 |
| components/drawers/ShippingDocumentDrawer | GuardedModalForm；HBL 单证字段 | editingShippingDocument.id 清理；纯 Drawer 列表浏览不脏 |
| components/detail/OrderLockControl | Form onValuesChange；解锁原因 | unlockModalVisible 门控，纯锁历史展示不脏 |
| FeeFormModal | GuardedModalForm + 手动汇率开关用户动作 guard | 原费用联动不变；editingFee.id 清理；onSubmit true 清两 guard |
| FeeSupplementModal | GuardedModalForm + 手动汇率/系统汇率确认用户动作 guard | 原补录更正联动不变；程序默认汇率不脏；提交 true 清两 guard |
| FeeSupplementSection 审核 | GuardedModalForm；审核意见 | 审核预览和展开不脏 |
| fee-review/OrderFeeReview、BatchFeeReviewModal | GuardedModalForm；人工审核意见、完整性确认、逐票批量意见 | 原预览/列表勾选不脏 |
| features/orders/fee-correction/FeeCorrectionSection | GuardedModalForm；更正审核意见 | 原更正预览不脏；驳回原因共用有 owner signal 的 confirmWithReason |
| FeeCorrectionCreate | GuardedModalForm 保护第一步更正方式/原费用/作废原因；后续现有 FeeSupplementModal 保护费用编辑 | 更正第一步完成后外部 session guard 保留待提交配置保护，按原交互进入完整费用表单，不误当服务端成功；费用提交仍走原契约 |
| OrderFeeTableTabs | 三个 GuardedModalForm；批量结算单位、费用日期、删除原因；应收/应付行内编辑各自独立 guard | 普通表格多选与进入编辑不脏；editable.formProps.onFieldsChange 仅 touched 用户字段同步标脏（绕过框架 onValuesChange 的 64ms 防抖）；取消/成功保存仅清对应方向，失败保留另一侧独立输入 |
| SeaSharedContainerCreateModal | Form onValuesChange；共享物理箱号、规格、件重尺、说明 | 默认数量/重量/体积不脏 |
| SeaTransportExecutionUpdateModal | Form onValuesChange；共享航次修订、外部确认字段 | order.id/open 清理，原预览逻辑保留 |
| SeaOrderReassignmentModal | Form onValuesChange + 外部 targetType/candidateTe 用户事件 | 母单提供方式和实际关联航次属于提交配置；自动回填不脏，原预览/匹配请求不变 |
| SeaDocumentHistoryActions | Form onValuesChange；改单/作废原因及外部确认 | mode/documentId 身份清理；版本/事件历史翻页不脏 |
| SeaDocumentSection | 独立模式切换 Form.onValuesChange；模式变更原因、外部确认 | modeModalOpen 门控，主订单表单仍由 OrderFormTemplate 独占 |
| orders/split + SplitAttachmentsAndNotesSection | 父 guard + 子组件真实用户 setter；结果票资料、箱货件重尺/费用/附件分配、说明、外部确认 Form | 初始上下文分配和默认值不脏；候选查询/预览不脏；添加/删除结果票是实际规划变更；成功执行清理 |

## App.modal 临时输入所有权

- `confirmWithReason`（守卫代理维护）登记真实 TextArea 变化，`useModalOwnerSignal` 的所属组件卸载 signal 销毁 App modal，Promise.race 显式完成 abort，避免 antd destroy 不结算 thenable 导致守卫残留。
- 领域直接调用覆盖：CommissionApplicationsPanel、order-fee-panel、FeeSupplementSection、FeeCorrectionSection；commissions/index 的两类 makeVersionActions，以及 bills/cashflows/nettings/verifications 四个台账的全部 makeVersionActions，均传当前组件 signal getter。后四页由独立审查补齐；嵌套 App 销毁界面不保证结算 thenable，仍需该 signal 注销手工注册的原因守卫。
- `orders/order-detail-transitions.tsx` 三类主流程/退关/结案 App.modal 使用本地 showTransitionConfirm 收敛同一 owner 生命周期与输入守卫，detail.tsx 提供 signal；原因或退关类型用户编辑标脏。owner abort 立即注销/销毁并在 onOk 复查 aborted，关闭动画期间也不能提交旧订单。
- `FeeSupplementSection` 的撤销补录费用特殊说明弹窗保留原内容，单独用户 cancelReason 标脏，owner signal 销毁且 onOk 复查 aborted。无字段的普通确认弹窗不作为未保存编辑保护入口。

## 明确排除

BillDetailDrawer、纯费用/提成详情、历史版本、列表搜索筛选、分页/排序、展开、候选关键字、普通表格勾选不标脏。主客商/订单主体既有守卫未重写。发票关联账单与拆票归属分配虽是多选，却直接进入待提交业务载荷，因此纳入编辑，不与普通候选勾选混同。

## 验证记录

- 实际 BillEditModal 用户备注 → 取消离开不执行动作且保留输入；确认动作失败保留 dirty；切换账单 id 重置；默认账户回填不脏。
- 核销真实 state：候选勾选不脏，自动生成实际分配标脏；提交成功清理后再次修改金额重新标脏；备注单独输入与卸载清理。
- 企业资源真实 Upload 文件输入驱动外部 imageFiles state 标脏，Form 名称单独编辑标脏；关闭/卸载清理；默认状态不脏。
- 真实 App.modal 原因输入标脏，所属订单页卸载销毁旧弹窗；关闭动画期间点击确认不调用旧订单 API。
- 定向旧业务回归覆盖账单/现金流/提成/账户/拆票/订单状态与解锁；最终命令结果由实施报告与主会话 review 记录。

## 独立审查补充验证

- OrderFeeTableTabs.unsaved.test.tsx 使用真实 EditableProTable 同时编辑应收/应付备注；断言初始不脏、用户输入立即脏、取消应收不清应付、切换取消保留应付、保存失败继续保护、成功保存清理。单文件 1 项通过，业务执行 3.07 秒。
- AppLayout.organization-switch.test.tsx 补真实 router.navigate(-1) 模拟浏览器 POP；懒加载被取消后落安全工作台，目标迟到不会覆盖，且新组织没有旧订单请求。单文件 7 项通过。

- 原因生命周期补充回归：confirmWithReason.unsaved.test.tsx 通过真实 StrictMode + makeVersionActions 打开原因框，输入后卸载所属组件，验证守卫移除且不提交旧实体。与账单/对冲真实请求写入失败和资金流水相邻路径定向共 5 文件 11 项通过（6.23 秒，stderr 无输出）。


## 独立 review 后补齐及非 Form 复扫

2026-10-20 再次沿真实 `Input/InputNumber/Upload/Checkbox/Switch/Select` 及其父调用方复扫 56 个领域源文件，补齐初审遗漏：

| 入口 | 实际待提交载荷与保护 | 初始化/失败/完成 |
| --- | --- | --- |
| ExchangeRateSyncModal | updateDraft 的应收/应付/基准汇率三个受控 Input；精度、套用上次微调、还原原价均直接修改发布 rows，因此用户动作标脏 | 默认抓取和 applyPreview 不脏；发布失败保留用户行和dirty；成功reset，open/baseCurrency变化清理 |
| ExchangeRateImportModal | 有效previewToken且可确认批次返回后标脏，待预检文件/previewToken/batch会随组织切换丢失 | 仅打开/模板下载不脏；从未形成有效payload的预检失败不脏，已有dirty的后续失败不清理，确认失败保留dirty；reset/重新上传/取消/成功清理；更换实际文件重新标脏 |
| enterprise-resources/ResourceImportModal | CSV解析得到非空rows时markDirty，真正待提交payload即受保护 | 默认空弹窗不脏；父请求失败不清理；移除文件明确reset；open/资源类型变化清理；成功按父流程关闭 |
| partners/PartnerExcelImportModal | 实际Excel解析得到非空items及导入模式Radio属于items/mode待提交配置，真实事件标脏 | 模板下载/默认值不脏；API失败保留；取消/reset/文件移除（同步恢复默认模式）/成功清理 |
| SeaSharedContainerDrawer | handleUpdateDraft与handleFillAllCargo的件重尺分配state；identity=业务contextKey+selectedContainerId，enabled=open且context存在 | 默认已有分配初始化、候选搜索/分页不脏；保存/确认成功reset；失败删除原强制reload，防止覆盖未保存分配；选换箱/关闭/换上下文重置 |

复扫逐类依据：

- 财务各 index 的 Input/Select 为列表搜索与筛选；CommissionRulesDrawer的组织/员工选择是列表过滤，不进入待提交载荷，不标脏。
- `CustomSettingsPanel` 的 Switch/Checkbox 点击立即调用更新API并以服务端响应更新state，没有本地待点击保存的配置草稿，排除。
- `BillCandidateSelectionStep` 的公司和费用选择为候选导航；真正待提交拆单策略及账单组输入已由父工作台guard保护。
- 发票关联账单属于实际创建载荷，已显式标脏；核销金额/日期/备注及自动生成实际分配已有显式标脏，纯资金/账单勾选仍排除。
- `SplitResultsSection/SplitAllocationSection/SplitFeesSection/SplitAttachmentsAndNotesSection` 均由拆票父用户setter/外部确认onValuesChange保护。
- 海运 `SeaBillContentFormFields/SeaCargoSection/SeaBasicInfoSection/SeaTransportSection/SeaExternalConfirmationFields` 由OrderFormTemplate或对应独立Modal的Form保护；其中生产使用的 containerRequests、散杂货主动清空、选择既有共享 MBL 实际航次已由主会话改为标准 Form.Item 受控值事件，接回模板原有 dirty/草稿生命周期；初始化程序回填不触发。OrderShippingDocumentFields 当前无生产消费者，仅旧单测使用，未扩大改造。
- 三个企业资源 Batch 控件存在真实外部state选择，已由共享守卫代理补齐，真实用户编辑三项回归通过，纯搜索不脏。
- QuickAddFee/QuickAddPartner/PartnerQuickAddSelect 的实际新增字段由共享QuickCreateModal的Form保护；候选搜索/浏览不脏。

新增真实回归覆盖汇率默认抓取清洁、三个价格编辑、发布失败/成功；Excel实际FileReader预检失败/成功/取消；客商Excel实际载入及导入模式；企业CSV实际载入；共享箱真实件数/全部填入、保存成功、确认失败保留输入。

主会话追加真实订单模板回归：计划箱量初值不脏，数量编辑/新增进入原模板草稿及dirty；提交false保留、resetTo清理；散杂货主动清空也触发表单编辑。2项通过；相邻主单异步校验9项通过。
