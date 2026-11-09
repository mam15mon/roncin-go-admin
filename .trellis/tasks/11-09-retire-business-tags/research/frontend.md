# 研究： 全站业务标签前端退役与账单工作区绑定

- 研究问题： 全站业务标签的真实入口/调用链/缓存/导出/测试，普通 Tag 的保留边界，以及账单去所属公司选择的最薄范围。
- 范围： internal（只读现有源码、规范和生成类型；仅本研究文件可写）
- 日期： 2026-11-09

## 研究结果

### 1. 这是业务标签完整退役，不能只删筛选框

| 业务入口 | 已确认代码锚点 | 必须同步删除 |
| --- | --- | --- |
| 订单列表（海运/空运共同消费） | `web/src/pages/orders/list.tsx:8,17-20,130-134,161-175,223-242,313-357` | 标签候选首屏请求、标签状态、批量维护入口、BusinessTagModal、orderTagService 调用；不能保留空白/隐藏入口 |
| 订单列表查询/投影 | `pages/orders/list-query.ts:72,210` | 请求 tagIds、行 tags 投影；保留其他业务字段和现有状态/提成/审核列 |
| 订单列表公共模板 | `components/ui/order-list-template/OrderListTemplate.tsx:191-215,650`；`OrderListToolbar.tsx:8,23,73-80` | 标签列、onManageTags prop、工具栏按钮；保留其余正常状态 Tag |
| 订单搜索 | `OrderListSearchFilter.tsx:60,274,300,531-565`；`types.ts:22,60,85,178` | tagIds/options.tags/labels tagIds:*、标签输入和参数/行模型/回调类型 |
| 订单费用双表 | `pages/orders/components/fees/OrderFeeTableTabs.tsx:380,479-489,1495,1793-1834` | 添加/删除标签动作、标签上下文与共用弹窗、候选和批量写接口；其他批量费用动作保留 |
| 订单费用可选列与偏好 | `orderFeeOptionalColumns.tsx:345-350`；`feeColumnPreference.ts:36,55,135-138` | 费用标签列、合法列key及默认隐藏定义；不能只隐藏列让设置仍可重新打开 |
| 费用台账 | `pages/finance/fees/index.tsx:136-137,239-295,299-304,390-418,518,550,599,641-678` | 标签 queryKey/候选缓存名称保活、筛选/标签显示名状态、批量入口、弹窗与API；公司选择是否保留不在本次账单范围扩展内 |
| 费用列与导出 | `fees/components/feeLedgerColumns.tsx:84-88`；`fees/ledgerExport.ts:12-15` | 标签列和CSV标签字段 |
| 账单 | `pages/finance/bills/index.tsx:7,27-35,73-94,105-148,500-539,581-610,661,700-736` | 标签候选、筛选、关联已有标签及批量操作状态/回调/API；配合公司selector移除后整个topBar无需保留 |
| 账单列与导出 | `bills/components/billColumns.tsx:38-41`；`bills/ledgerExport.ts:6-10` | 标签列和CSV标签字段；其余状态/来源标记保留 |
| 企业资源类型页签 | `router/workspaceTabs.ts:9-17` | 删除 key=tags/type=4 项，保留其他type原值（5发货人/6收货人/7通知人，不重排复用数字） |
| 企业资源维护 | `pages/enterprise-resources/index.tsx:112-144,228,274,355-360,610-638,774-783,895,930-938` | 标签组状态、加载与创建更新、标签类型输入/详情列、标签组入口和TagGroupModal；其他地址/备注/图片/主体资源保留 |
| 企业资源编辑表单 | `components/ResourceEditorModal.tsx:49,78,295-304`；`resourceConstants.ts:EditorValues.groupId` | tagGroups prop、标签组字段、type=4分支、注释；共享编辑和校验提交流程保留 |
| 共享业务标签UI | `components/business-tag/BusinessTagList.tsx`、`BusinessTagModal.tsx:14-18,23-46,142-174` | 三类消费者退出后可删除整个目录及guard测试，快捷创建resourceType=4和EnterpriseTagGroup调用一并退役 |

模块路径前缀为 `web/src/`。自动生成的 `web/src/services/roncin/{orderTagService,orderFeeService,settlementService,enterpriseResourceService}.ts` 与 `typings.d.ts` 不能手改；由后端契约退役后统一生成，`orderTagService.ts` 若不再有RPC应由生成器流程清理。

### 2. 额外发现与保留范围

- `web/src/features/audit/audit-presentation.ts:651-685` 有 enterprise_tag 与 enterprise_tag_group 事件展示词典，测试 `audit-presentation.test.ts:70` 有标签组断言。这不是可写业务标签入口。建议现行代码同步移除已无生产者的专属映射，历史审计记录本身不删除；若要求历史仍保留中文描述，需由主会话在最终方案明确保留理由，不误把审计存量当业务标签数据清除。
- `web/src/access.ts:39,239-240` 消费 `system.finance.fee.tag`，`permissions.generated.ts:38` 是生成项。后端删除权限后运行 generate:permission-keys，手写 access 同步去掉 canManageFinanceFeeTags；不要用旧key的兼容分支继续放行。
- 普通 `antd.Tag` 保留：订单进程/操作进度/异常（OrderListTemplate.tsx:115-136,287,455,470）、账单状态/来源/更正标记（billColumns.tsx:58-85,163,237）、企业资源关联/启用状态（index.tsx:457-478）、`features/partners/PartnerSelectOptionTags`（信用超额等提示）、`features/personnel` 人员候选、`features/finance/exchange-gain-loss/ExchangeGainLossTag`、`components/layout/TagsView` 页签。这些不是自由维护的业务标签字典。
- 对 partners/workbench/财务详情的定向搜索未发现额外自由标签维护入口；它们的状态Tag和候选Tag不可按文件名批删。未声称全仓无其他动态引用，实施后须用精确标识再次扫残留。

### 3. 查询快照与列设置：无需历史兼容层

- `components/ui/list-query-state/index.tsx:16-30,58-69,86-97` 只在当前 Provider 内存 Map 保存查询，不写浏览器持久存储。`components/layout/OrganizationWorkspace.tsx:10-13,27-35` 按 userId/currentOrganization.id 重建 Provider；切公司即独立查询身份，旧工作区缓存不会自然跨过来。
- 费用/账单的 tagIds/tagLabels 从查询泛型、初始化、commit和request明确删除；订单SearchFilter删除tagIds及labels补齐读取。产品请求只构造仍有效字段，不做“发现旧tagIds则转换”的双逻辑。
- 账单 organizationId 也从可恢复 filters 模型删除，禁止将 snapshotRef.filters 里的旧 organizationId 当最终API目标。来源应唯一是 auth/me 的 currentOrganization.id。现有测试可注入外公司旧值验证请求仍走当前公司，但不新增生产历史兼容层。
- 费用列偏好 `feeColumnPreference.ts` 有 tags 合法key和默认隐藏项，应按新列schema删除；通用 `components/ui/column-settings/preference.ts:33-64` 只负责解析本地偏好，具体已不存在列继续沿现有列映射机制排除，不为业务标签发明新迁移或改整个通用列设置框架。
- 保留现有选择scope边界：跨页保留、提交现有筛选/公司变化清选、官方采纳快照/导出/汇总一致。删除tag筛选不应回退这些最近完成的公共契约。

### 4. 账单公司范围：当前事实与最薄改动

- 权限事实：`access.ts:165-175` 只有 currentOrganization.kind=COMPANY 才 canOperateBusiness；`canReadFinanceBills:241` 还要求bill.read，`canReadEnterpriseResources:283-284` 同样限定公司。`config/routes.ts:227-231` 用 canReadFinanceBills 保护账单页。因此系统管理工作区现已不能通过正常路由进入经营账单，不需要新增跨公司只读selector或放开系统workspace。
- 列表去掉 `bills/index.tsx:99-104,149-154,471-499` 的 organization state/options/request 与所属公司Select；删 FinanceOrganizationPurpose 只因该页候选使用而导入的代码，不能删后端给其他财务页/建账流程用的整个组织候选接口。
- 列表请求 `:628,662` 的 organizationId 改为 `useInitialState().initialState.currentUser.currentOrganization.id`，保留后端自己的授权/组织判定，前端不复制权限真相。搜索/重置只改有效查询，不能把公司从默认值写回旧快照。
- 标签全站退役后该页公司依赖的标签候选和提交自然删除。其余行级写入已走record.id+expectedVersion，`canOperateOrganization` 对row.organizationId与当前公司比对（access.ts:172-175），保留该判断。
- **已与主会话确认边界**：建账抽屉内部多组织选择本期先不动。事实是账单页 `index.tsx:688-692` 只传open/onClose/onCreated；`BillCreationWorkbench.tsx:351-354` 仅 initialFeeIds非空时采用initialOrganizationId；`BillCandidateSelectionStep.tsx:63-74` 有独立所属公司Select。删除账单列表selector不会自动删除它，这是现有共享写入流程，不能因本任务移除标签顺带改造；若用户之后也不希望这里选公司，另行明确修改。
- 工作区切换时现有 OrganizationWorkspace keyed 卸载账单页面及业务Modal，故列表/详情/编辑不会跨工作区保留；不需新增手写useEffect请求或全局组织缓存。

### 5. 退役 type=4 的导航行为

- 只从 `enterpriseResourceTabs` 删除tags项，其他整数不变；正式Proto枚举应由后端reserved旧值/名称，不将4重新代表其他类型。
- 页面 `enterprise-resources/index.tsx:976-978` 当前找不到query tab会选择 availableTabs[0]（地址）；组织切换落点 `organizationSwitchDestination.ts:51-62` 如果显式请求不存在子页签，会按现有规则回工作台并提示权限不足。
- 最薄方案：删除入口后沿现有未知tab机制处理直接 `?tab=tags`，不建标签兼容路由或自动映射成某种新类型。不得让ResourceTypePage收到active.type=4继续请求。
- `workspaceMenus` 的企业资源设置子项由共享tab清单构建，删除应通过唯一清单传播；仍需验证顶栏/移动菜单没有残留（而非只删页面Tab）。如果产品希望所有旧标签链接显式404而非现有回落，属额外产品行为，当前未获要求，不建议新增。

### 6. 最小验收矩阵

| 范围 | 用例与断言 |
| --- | --- |
| 订单列表海/空 | 不出现标签列、标签搜索、批量维护标签；不发送tagIds/标签候选请求；日期/客户/人员/状态筛选及审核/提成列仍正常 |
| 订单费用 | 费用标签可选列及添加/删除标签动作消失；税额/汇率/关联账单/其他批量动作保留；新列偏好不含tags |
| 费用与账单 | 无标签筛选/列/批量弹窗，CSV不含标签列，列表payload无tagIds；采纳快照、跨页选择与金额合计回归不变 |
| 企业资源 | 无标签管理/标签组/快捷创建；地址备注图片收发通知人各type原编号不变；最近修复的校验与类型隔离测试仍有效 |
| 菜单/旧链接 | 顶栏/移动菜单无tags项，?tab=tags不触发type4请求，直接入口按现有未知tab行为；工作区切换不保留无效标签页签 |
| 当前公司账单 | 不显示所属公司框、不发列表公司候选请求；初次/翻页/搜索/重置都传当前公司id；旧过滤里其他公司不能覆盖；换公司后的新页面请求新id、无旧记录/弹窗 |
| 权限 | COMPANY有bill.read可进账单，SYSTEM仍不可；已退役fee.tag从access与生成keys消失，其他企业资源CRUD权限保留 |
| 正常Tag保留 | 状态/进度、信用超额、汇差、人员候选、页签等视觉标记和测试不被误删 |

应调整/移除的现有测试（非穷举）：`BusinessTagModal.guard.test.tsx`（随组件删除），`OrderFeeTableTabs.bulkTags.test.tsx`（退役用例删除并补无入口断言），`OrderFeeTableTabs.bulkActions.fixtures.tsx`（删除遗留tag mock/导出），`feeColumnPreference.test.ts`，`OrderListSearchFilter.restore.test.tsx`，`orders/list-query.test.ts`，`orders/list.test.tsx`，`finance/{fees,bills}/*query-state.test.tsx`、`ledgerExport.test.ts`，`bills/bill-save.test.tsx`，`enterprise-resources/resource-types.test.tsx`（以保留类型替代标签用例），`ResourceEditorModal.guard.test.tsx`（去tagGroups prop），`resource-save.test.tsx`，`features/audit/audit-presentation.test.ts`，`access.test.ts`，workspaceMenus/organizationSwitchDestination相关用例。

### 7. 规范与生成物

- `.trellis/spec/web/frontend/capability-navigation.md` 删除业务标签能力导航；保留业务状态标签、人员/客商候选和汇兑损益能力。
- `.trellis/spec/web/frontend/directory-structure.md`、`state-management.md`、`finance-ledger-snapshots.md` 的既有边界与采纳快照约定继续适用；不是重新建立请求层的理由。
- `AGENTS.md` 本次无需给普通Tag加禁令；可最薄记录业务标签已退役及账单列表当前工作区来源，避免下次按旧生成文件恢复旧入口。
- 生成客户端/枚举/权限key由主会话或后端owner统一运行对应生成命令；前端owner不要一边删除类型一边手修生成代码。

## 文件清单

核心文件及用途已经在第1、3、4节给出（页面、公共模板、类型、缓存、CSV、测试、路由/权限）；本次不创建额外工具或新领域模块。跨层Proto/Ent/权限迁移详细清单应由后端研究合并，避免前端根据文件名猜底层归属。

## 外部依据

未使用外部网页或新增版本行为；结论直接依据仓库当前源代码与已有测试。框架/API改动实现时按项目现有安装版和技能要求查询，不在规划期升级依赖。

## 相关规范

- `.trellis/spec/guides/code-reuse-thinking-guide.md`：同类入口/契约一次清理；没有实际复用不新增框架。
- `.trellis/spec/web/frontend/directory-structure.md`：页面/features/通用UI依赖边界。
- `.trellis/spec/web/frontend/state-management.md`：服务端状态、错误、工作区隔离。
- `.trellis/spec/web/frontend/organization-switch.md`：组织切换确认与身份边界。
- `.trellis/spec/web/frontend/type-safety.md`：生成物、Proto枚举和presence。
- `.trellis/spec/web/frontend/finance-ledger-snapshots.md`：官方采纳快照、跨页选择和金额范围。

## 限制与未确认项

- 本研究没有编辑任何产品/测试/生成文件、没有执行数据库查询或Git操作，也没有浏览器或产品测试运行；文件行号针对研究时的当前checkout。
- 存量标签资产/关联是否允许物理删除属于主会话最终方案的数据处置决策，不能从“全站取消功能”自动推导可删除真实数据库数据；前端研究不作授权判断。
- 主会话已明确建账抽屉公司选择列后续、本期仅账单列表绑定当前公司，无其他阻塞前端产品决策。
- 在全面删契约后要再次扫描精确 BusinessTag/tagIds/TagGroup/fee.tag 标识；不能用匹配所有 Tag/tags 的脚本删除普通视觉状态、路由页签或测试临时变量。
