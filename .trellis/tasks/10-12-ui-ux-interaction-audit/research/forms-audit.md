# Research: 长表单、弹窗与反馈交互审查

- Query: 按 UI-UX-Pro-Max 检查公共长表单/Modal/Drawer、订单新建与详情、客商档案和财务写弹窗的未保存离开、字段标签、校验定位、提交反馈、取消恢复与失败恢复。
- Scope: internal；本地技能方法与源码审查，浏览器实证由主会话执行并汇总。
- Date: 2026-10-12

## Findings

### 方法与相关规范

已读本任务 `prd.md`、`design.md`、`research/skill-method.md`、`.trellis/workflow.md`、领域 glossary/architecture-map 和 Web frontend index/component/state/hook 规范。使用用户指定的本地技能 `/home/vince/.claude/plugins/marketplaces/nextlevelbuilder-ui-ux-pro-max-skill/.claude/skills/ui-ux-pro-max/SKILL.md` 中 accessibility、loading-buttons、error-feedback、escape-routes、input-labels、submit-feedback、confirmation-dialogs 和 progressive-disclosure 检查项。

本报告只评价高密度桌面业务后台：不把所有按钮 44px、正文必须 16px、自动保存、max-width、换字体/主题等当作本次缺陷。框架提供的 label、loading 和 focus 行为先看实际代码；没有浏览器测量，不声称配色/焦点或窄屏裁切已经实证。严重度为 P1 重要（易造成工作丢失或业务结果判断错误）、P2 中等（关键恢复/可访问性）、P3 建议。

### F1 · P1 · 若干写请求失败后仍报成功并关闭输入界面

**状态：客户编辑 PUT 503 浏览器复现；订单、账单、资金流水路径仍是代码证据。**

共同触发：服务端返回 409/500、超时，统一请求层展示错误后将 Promise resolve 为 `undefined`。多个写调用仅 `await`、不检查结果，随后无条件执行成功反馈和关闭/清理。注意这不是“catch 缺失”，而是请求层消费错误和调用方成功判定的契约不一致。

证据：

- `web/src/utils/requestClient.ts:61-66` 明确记录错误消费返回 undefined；`:98-104` 执行该返回。
- `web/src/pages/orders/detail.tsx:382-394` 更新仅 await，随后“保存订单成功”、返回 true；`web/src/components/ui/order-template/OrderFormTemplate.tsx:177-183` 接受 true 后清草稿、清 dirty。
- `web/src/pages/finance/cashflows/index.tsx:498-516` 登记仅 await，随后“资金流水已登记”、关弹窗并返回 true。
- `web/src/pages/finance/bills/index.tsx:262-279` 更新账单仅 await，随后“账单已成功更新”、关弹窗。
- `web/src/pages/partners/partner-detail.tsx:586-605` 更新档案仅 await，随后成功 toast；`:628` 返回列表。创建分支 `:607-624` 先报成功再访问 `createRes.data`，失败时可能又追加技术异常。

用户影响：失败仍可能执行成功反馈、关闭输入界面、清理 dirty，用户容易错误判断保存结果。主会话拦截客户编辑 PUT 返回 503 后，通过 MutationObserver 在跳转前捕获“客户档案已成功更新”和“审查模拟服务暂不可用 请联系系统管理员查看服务日志。”同时出现，随后跳回客户列表。这条客商更新路径已浏览器复现，证据保存于任务浏览器输出 `browser-followup.json`、`browser-list-return.json`。最初450ms采样因跳转卸载提示而漏掉成功 toast，现已由复测修正。订单 dirty 与恢复草稿被清、资金流水和账单弹窗关闭目前是代码路径证据，需分别复核。

最薄建议：在已确认失败不能视为成功的写路径显式使用 `skipErrorHandler` 并统一展示业务错误，或检查非空响应/必要成功业务结果后才能反馈成功、关闭和清 dirty；遵循当前请求契约，避免再增加回退。首先覆盖订单保存、账单编辑和资金流水登记，不要求本次重写全请求框架。

浏览器复核：若主会话扩展审查授权，只用 Playwright 路由拦截对应 PUT/POST 并返回 409/500，确保请求绝不到真实后端；填合法表单后保存，检查失败通知、成功 toast、弹窗开合及订单页签 dirty。本研究代理没有执行写模拟；主会话已采用拦截客户编辑 PUT 返回 503 的方式补充证据，未向真实后端提交修改。

### F2 · P1 · 客商长表单没有未保存关闭守卫

**状态：代码确认结构缺口；主会话浏览器已证实客户新建取消与关闭页签均无确认。**

触发：新建客户输入公司抬头、别名或联系人，关闭当前页签或切换组织后重开。客商表单没有注册关闭守卫、草稿恢复，也没有 dirty。取消/返回可直接导航，而普通导航是否暂存页签由全局布局决定，不能仅凭返回代码断言所有返回都丢输入。

证据：`web/src/pages/partners/partner-detail.tsx:708` 返回列表，`:727-729` 和 `:899-901` 取消直接导航；`:746-757` ProForm 无 onValuesChange/draft/guard，组件没有 `useTabCloseGuard`。相对地 `OrderFormTemplate.tsx:123-152` 注册草稿与守卫，`:300-305` 编辑保存草稿。

用户影响：联系人、责任人员、结算和备注的大量输入缺少离开保护。主会话已证实客户新建取消及关闭页签都不弹确认；组织卸载、重开后的具体值恢复尚未在本代理报告中证明，不能扩大实证范围。行为与订单的既有关闭守卫不同。

最薄建议：客商编辑页复用现有页签关闭 guard，dirty 要包含表单值以及 contacts/aliases 等外部 state；取消/明确离开采用现有确认语义。先保护工作，不预建自动保存系统。需一致覆盖新建与现有档案。

浏览器复核：`/partners/customers/create` 输入明显未保存标记，关闭页签，再重开；查看是否弹确认、值是否保留。再试客户详情修改、切组织并取消；用订单新建作为正向对照。只输入不提交。

### F3 · P1 · “利息规则已更新”及自动计息承诺没有保存链路

**状态：代码确认；浏览器可只查看文案与局部配置状态。**

触发：客户档案的财务结算规则点击“编辑规则”，启用逾期计息并确认。该确认只更新 React 内存 state，没有服务器写入；主档保存载荷也没有利息字段。

证据：`web/src/pages/partners/components/SettlementSection.tsx:261-284` 暴露可编辑入口/“已启用”摘要；`web/src/pages/partners/components/InterestRuleModal.tsx:75-87` 明确承诺“自动计算滞纳金与违约金”“开启后…产生滞纳利息”；`web/src/pages/partners/partner-detail.tsx:86-92` 仅本地默认 state，`:916-924` setInterestRule 后 toast“利息规则已更新”；`:556-571` 主档保存 settlementRuleInput 没有任何利息参数，`:310-419` 详情回填也不恢复利息参数。搜索 server/api 未找到该局部规则的对应参数。

用户影响：用户会以为配置已生效，实际重开归默认且不会由这条配置链路影响财务计息，产生业务误导。

最薄建议：若逾期计息本期未交付，收起该配置入口并删除自动计息承诺；若已批准，补齐契约/持久化/回填闭环后才给成功反馈。不能用“已更新”掩盖仅局部 state。

浏览器复核：打开客户结算规则的弹窗读取说明，不保存主档。局部确认是否允许取决于主会话审查约束；它的代码没有网络写请求，但应明确这只是局部状态。不要提交真实客商档案。

### F4 · P2 · 客商详情初次加载失败后仍显示空表单，缺少页面恢复入口

**状态：代码确认；主会话已注入客户详情 GET 503 并浏览器复核。**

触发：直接打开有效客户 ID 详情，首次档案 GET 失败。loading 仅由 isPending 控制，失败后停止 Spin，但没有对 isError/null data 分支显示加载失败 Result 或重试。

证据：`web/src/pages/partners/partner-detail.tsx:162-175` 详情查询，`:196-197` loading 判断；`:675-694` 仅非法 ID 判断后直接正常页面；`:744-757` 仍渲染 ProForm。`web/src/utils/queryClient.ts:14-16` 不自动重试、不随窗口焦点/重连重查。订单新建 `new.tsx:186-232` 和详情 `detail.tsx:455-484` 已有明确失败与重新加载入口。

用户影响：通知消失后无法区分“档案为空”与“数据加载失败”，页面本身没有恢复入口。主会话实际观察到空白禁用表单，没有保存按钮，也没有重试按钮；因此这是失败恢复问题，不能报为“空白可编辑表单”或声称能直接误写。

最薄建议：首次无数据失败显示公共 Result/Alert、重试与返回列表；若已有数据后台重查失败，应保留当前表单输入。参考订单失败分支即可，不需要新增通用容错层。

浏览器复核：拦截该客户详情 GET 返回 500，观察通知结束后页面内容、按钮禁用及是否存在重新加载。恢复 GET 后检查在页面能否重试。

### F5 · P2 · 公共可折叠分节标题只能鼠标操作

**状态：代码确认自定义元素缺语义；键盘实证待主会话。**

触发：用 Tab 和 Enter/Space 尝试收起/展开客商“基础信息”“财务结算规则”等区块。

证据：`web/src/components/ui/page-shell/SectionCard.tsx:38-67` 标题为带 onClick 的 div，无 button、tabIndex、onKeyDown、aria-expanded/aria-controls；`:103` 折叠通过 display:none 执行。`:75` 作为普通 Card title 传入，Antd Card 不会把该自定义标题转成按钮。

用户影响：键盘用户无法操作相同的折叠控件，也不能得知展开状态；公共组件使多个长表单共享这一问题。

最薄建议：可折叠时用原生 button（保持现有样式），关联区块并提供 aria-expanded；纯展示时保持标题语义。继承原生 Enter/Space，不重复造键盘模型。

浏览器复核：在客户新建页逐个 Tab，确认分节标题不进入焦点；鼠标收起后再以键盘尝试展开。不能以所有 antd 自带 Collapse 都缺键盘支持来泛化。

### F6 · P3 待复核建议 · 联系人图标按钮可增加中文业务语义名称

**状态：待复核建议，不列正式 P2 缺陷。Antd 图标本身可能通过 role="img" 和 aria-label="edit"/"delete" 组成按钮的可访问名称。**

证据：`web/src/pages/partners/components/ContactCardList.tsx:147-153` 编辑 Button 只有 EditOutlined，调用方无中文文字/aria-label/title；`:160-165` 删除同样。不能据此断言最终按钮无名称，框架图标可能提供 edit/delete。需要检查 computed accessible name 后再判断。

潜在改善：即使框架图标提供英文 edit/delete 名称，加入中文业务动作与联系人姓名仍有利于区分多个重复控件；当前证据不足以认定屏幕阅读器读出无名称按钮。问题不涉及手机 44px 尺寸规则。

最薄建议：增加“编辑联系人［姓名］”“移除联系人［姓名］”的 aria-label，必要时 Tooltip 方便鼠标辨识；保留现有确认框。

浏览器复核：在既有联系人卡片读取两个 button 的 computed accessible name，记录图标语义是否参与按钮命名；只在结果确实为空或错误时升级缺陷。键盘聚焦后可确认删除 Popconfirm 的取消恢复，只打开/取消，不删除真实数据。

## 已有良好行为

- **订单草稿生命周期已有闭环**：`OrderFormTemplate.tsx:123-183` 以显式身份保存/恢复、关闭守卫、成功才清草稿；`orders/detail.tsx:205-239` 刷新校验身份与令牌，不让迟到响应覆盖其它单据。此优点受 F1 更新接口误判成功的例外影响，应修成功判定而不是重建草稿。
- **长表单校验能展开隐藏位置并聚焦**：`OrderFormTemplate.tsx:191-219` 先 onRevealError 展开隐藏区域，再定位；`form-navigator/formErrorUtils.ts:133-143` 查询真实 input 并 focus，`:220-252` 展开父分节、滚动、高亮。`partner-detail.tsx:630-640` 校验失败也复用定位/展开。不能声称全站表单不会自动聚焦。
- **新建订单明确加载/失败恢复**：`orders/new.tsx:186-232` 提供主数据加载失败重试；`:238-257` 必须取得非空 createdId 才成功，失败保留表单与幂等键。
- **快捷建档提交保护齐全**：`QuickCreateModal.tsx:61-87` 同步 savingRef 防重复；仅非空结果成功后 reset；`:90-94` 保存时不允许取消；`:110-113` close/mask/keyboard/cancel 同时门控。取消重置字段、失败保留输入，`preserve:false` 配合 destroyOnHidden 避免下次新建沿用旧字段。
- **财务 ModalForm 遵循 Promise 生命周期**：资金流水 `:496-521` 返回 true/false，框架接管 spinner；正常 reject 路径保留弹窗。但 F1 的 undefined 不是 reject，仍需修正。
- **客户页上下双操作入口可达**：`partner-detail.tsx:725-740` 页头，`:887-912` StickyFooterBar；已有 saving spinner 和取消 disabled，不要求另加第三个提交入口。
- **子实体取消/成功明确**：`SubEntityCardGrid.tsx:94-101` 编辑先重置再回填，`:185-191` 仅成功时关闭；空态 `:145-159` 提供“添加第一条”入口。联系人删除有 Popconfirm（ContactCardList.tsx:154-159）。
- **发票创建关闭时清候选身份**：`InvoiceCreateModal.tsx:120-126` 关闭清公司/抬头及弹窗查询缓存；选择字段有 visible labels、disabled 上下游约束和所属公司 aria-label（`:170-215`）。

## 文件导航与版本依据

| 文件 | 职责 |
|---|---|
| `web/src/components/ui/order-template/OrderFormTemplate.tsx` | 订单长表单、草稿、dirty、错误定位 |
| `web/src/components/ui/form-navigator/formErrorUtils.ts` | 校验展开、滚动、聚焦与分节错误计数 |
| `web/src/components/ui/page-shell/SectionCard.tsx` | 公共业务分节与折叠标题 |
| `web/src/components/ui/quick-create-modal/QuickCreateModal.tsx` | 快捷建档 Promise 反馈、取消和重置 |
| `web/src/components/ui/sub-entity-card-grid/SubEntityCardGrid.tsx` | 子实体局部编辑卡片及 ModalForm |
| `web/src/components/ui/sub-entity-drawer/SubEntityDrawerTemplate.tsx` | 子实体抽屉与快捷表单 |
| `web/src/pages/partners/partner-detail.tsx` | 客商长档案保存、回填、财务规则、联系人 |
| `web/src/pages/partners/components/InterestRuleModal.tsx` | 本地利息规则编辑与业务说明 |
| `web/src/pages/partners/components/ContactCardList.tsx` | 联系人本地增改删和图标按钮 |
| `web/src/pages/orders/new.tsx`、`detail.tsx` | 订单新建/详情业务保存与恢复 |
| `web/src/pages/finance/bills/components/BillEditModal.tsx`、`bills/index.tsx` | 账单编辑表单与写结果反馈 |
| `web/src/pages/finance/cashflows/index.tsx` | 资金流水登记弹窗 |
| `web/src/pages/finance/invoices/components/InvoiceCreateModal.tsx` | 发票候选、抬头与创建表单 |
| `web/src/utils/requestClient.ts`、`queryClient.ts` | 统一失败契约与查询恢复默认值 |

相关规范：`.trellis/spec/web/frontend/component-guidelines.md`（公共模板、ModalForm.onFinish、表单生命周期）、`state-management.md`（错误返回 undefined、草稿身份与清理）、`hook-guidelines.md`（身份/异步保护）；根 `AGENTS.md`。
外部参考采用技能本地 UX 规则库，未联网查询；关键判断直接依据仓库实现，框架依赖通过已安装包核对（`@ant-design/pro-form@2.32.0`），不将不存在的 aria/focus 问题归咎框架。

## Caveats / Not Found

- 本代理仅静态只读产品代码；本文已明确标注主会话转交的浏览器实证：客户新建取消/关页签无确认、客户详情 GET 503 为空白禁用表单且无重试、客户编辑 PUT 503 后同时成功/失败反馈并跳列表。其余浏览器步骤仍是建议，不能当成已经复现。
- 撤回“窗口聚焦会触发客商回填覆盖输入”候选：`queryClient.ts:15` 已关闭该行为；仅在将来主动 invalidate/refetch 时可能暴露，当前不列正式问题。
- 标准 Modal/ModalForm 的 focus trap、Esc、提交 spinner 不能因为调用方没有 aria props 就判定缺陷；自定义折叠 div 有明确代码证据。联系人图标按钮名称须结合 Antd 图标自身 aria-label 复核，已降为 P3 待复核建议。
- 账单编辑 `BillEditModal.tsx:222` 的 minWidth:620 和发票固定字段宽在小视口可能溢出，需真实测量。本代理没有加入已确认问题，避免重复导航/列表代理的响应式范围。
- 本研究代理未对财务写请求或真实业务数据执行修改、提交、删除。主会话仅对客户编辑 PUT 使用浏览器拦截返回 503，未提交真实业务数据。F1 各调用方证据与 F3 配置链路分别标注；具体修复另行批准实施。
- F2 的“取消返回是否丢输入”受全局页签缓存影响，最可靠复核是关闭页签/组织卸载。报告不能声称切到任意列表都丢工作。
