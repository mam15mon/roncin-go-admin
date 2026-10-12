# UI-UX-Pro-Max 交互审查报告

日期：2026-10-12；代码基线：`0e883c21`（已包含统一列宽调整）；交付为审查与证据记录，无产品代码修改。

## 结论与优先级

当前高密度后台的公共模板基础完整，主要问题集中在**保存结果判定、工作保护、实际查询与界面回显的一致性，以及键盘与窄屏入口**。本轮记录 **3 项 P1、9 项 P2**，另有一项列宽操作增强建议。P1 表示容易造成工作丢失或业务结果误判；P2 表示交互一致性、恢复入口或可访问性缺口，不等同于线上事故级别。

| 编号 | 级别 | 问题 | 证据 |
| --- | --- | --- | --- |
| A01 | P1 | 保存失败仍报成功，关闭或清理输入 | 客户更新浏览器复现；订单/账单/资金流水为代码证据 |
| A02 | P1 | 客商长表单取消/关页签没有未保存确认 | 客户新建浏览器复现 |
| A03 | P1 | 利息规则宣称已更新和自动计息，但仅保存在本地状态 | 代码确认 |
| A04 | P2 | 订单筛选收起后丢回显，实际仍使用旧条件 | 浏览器复现 |
| A05 | P2 | 订单自动刷新开关没有实际刷新行为 | 浏览器 65 秒观察 + 代码 |
| A06 | P2 | 客户详情加载失败显示空白禁用表单，没有重试入口 | 浏览器模拟 GET 503 |
| A07 | P2 | 顶栏下拉不能通过键盘展开 | 浏览器键盘/鼠标对照 |
| A08 | P2 | 公共分节折叠标题无法键盘操作、缺展开状态语义 | 浏览器 DOM + 代码 |
| A09 | P2 | 业务页签及关闭按钮缺可见键盘焦点 | 浏览器计算样式 + 代码 |
| A10 | P2 | 窄屏隐藏顶部菜单，抽屉缺部分常规业务入口 | 390px 浏览器 + 路由/工作台代码 |
| A11 | P2 | 新建海运出口订单在窄屏发生整页横向溢出 | 390px 浏览器稳定复测 |
| A12 | P2 | 切换页签回来，列表查询状态丢失 | 客户列表浏览器复现；相邻列表代码 |

建议先修 A01–A03，再修 A04–A06/A12，随后处理共享键盘能力与窄屏问题。利息业务是否本期交付需要业务范围决定；审查没有替用户制定计息规则。

## 范围、方法与边界

应用用户指定的本地 UI-UX-Pro-Max 技能：`/home/vince/.claude/plugins/marketplaces/nextlevelbuilder-ui-ux-pro-max-skill/.claude/skills/ui-ux-pro-max/SKILL.md`，完整读取并使用其本地 design-system、UX、React 搜索。遵循本仓库既有纯白、高密度、流式全宽和 Antd 模板规范；营销 Hero/字体建议、手机所有按钮 44px、全站 16px、自动保存等没有被当作本项目强制要求。

公共层覆盖导航、页签、组织工作区、搜索、列设置、调宽、长表单、分节、错误定位和 ModalForm 生命周期。浏览器以公司工作区为主，系统管理切换到系统工作区，访问以下 10 条代表路由，在 1440px、390px 两档记录结构：`/welcome`、`/partners/customers`、`/partners/customers/create`、`/orders/sea-export`、`/orders/sea-export/new`、`/finance/fees`、`/finance/bills`、`/finance/cashflows`、`/master-data`、`/admin?tab=users`。

专项验证包括未保存取消/关页签、订单筛选、自动刷新、页签切换查询恢复、顶栏键盘操作、分节语义、焦点样式、移动抽屉、整页溢出、客户详情 GET 失败及编辑 PUT 失败。使用独立 Playwright Chromium 上下文、本地开发服务和环境中的开发登录信息。基线采样仅读取页面结构；专项验证将页面所有非 GET/HEAD/OPTIONS 请求在浏览器本地拦截并返回模拟 503，写失败验证没有向后端提交业务改动；认证会话登录/切组织/退出单独完成。证据没有凭据、组织身份、客户名称、业务行内容或原始响应。

覆盖代表路径和公共消费者，不宣称逐一实测所有路由、权限组合、手机浏览器或财务完整生命周期。文字对比度/reduced-motion 未完成系统化测量，本轮不对其给出通过结论。

## 确认问题与改进建议

### A01 · P1 · 请求失败仍执行成功流程

**复现**：打开有编辑权限的客户档案，修改公司抬头；在浏览器将该更新请求完全拦截并返回 503；点击保存。页面同时出现「客户档案已成功更新」与模拟失败通知，随后返回客户列表。提示在导航后消失，因此复测通过 MutationObserver 捕获跳转前新增提示节点。

**原因与影响**：统一请求层消费错误后 resolve 为 `undefined`，多个调用方只 await，不校验成功结果，就执行 success/关闭/清 dirty。用户无法判断实际是否保存，还可能失去已输入内容。代码证据：`web/src/utils/requestClient.ts:61`、`:98`；`web/src/pages/partners/partner-detail.tsx:586`、`:628`。相邻路径 `orders/detail.tsx:382` 返回 true 后，由 `OrderFormTemplate.tsx:177` 清草稿和 dirty；`finance/bills/index.tsx:262` 关闭编辑；`finance/cashflows/index.tsx:498` 关闭登记。客户创建分支在 success 后访问 `createRes.data`，属于失败后还可能追加异常的相邻问题，不能说与更新分支完全同样关闭。

**建议**：写调用只有取得契约定义的成功结果才能成功提示、关闭和清草稿；在相关路径显式选择错误外抛/处理或校验必要返回结果，避免增加静默回退。复用新建订单已有非空 createdId 成功判定。修复后验证客户更新、订单保存及资金流水/账单失败保持输入；本轮仅客户更新完成浏览器复现。

### A02 · P1 · 客商长表单未保存工作没有保护

**复现**：新建客户输入公司抬头，分别点击底部取消和当前页签关闭。均直接离开，无确认对话框。代码中该档案表单没有 dirty/关闭 guard；订单模板则已有草稿和 guard。

**影响**：录入联系人、责任人员、结算规则及备注后容易误操作丢失。证据：`web/src/pages/partners/partner-detail.tsx:708`、`:727`、`:746`、`:899`；对照 `web/src/components/ui/order-template/OrderFormTemplate.tsx:123`。

**建议**：复用现有未保存关闭守卫，并覆盖取消、返回、页签关闭和组织切换；dirty 同时包含表单与联系人/别名等外部状态。先完成确认保护，不需要新建自动保存平台。编辑/供应商/代理是共享代码影响范围，浏览器实测样本为客户新建。

### A03 · P1 · 利息规则只有界面状态，文案却承诺生效

**触发**：客户财务结算规则打开「编辑规则」，启用逾期计息并确认。弹窗宣称自动计算滞纳金与违约金；确认回调只 `setInterestRule` 并提示「利息规则已更新」。档案保存载荷及详情回填没有这组字段。

**影响**：用户可能据此认为计息配置已经持久化并参与财务计算。证据：`web/src/pages/partners/components/InterestRuleModal.tsx:75`、`SettlementSection.tsx:261`；`partner-detail.tsx:86`、`:556`、`:916`。这是这条配置链路缺闭环的代码结论，不声称验证过整个系统不存在其它计息能力。

**建议**：若本期未交付，收起未完成入口及生效承诺；若业务已批准，则补齐契约、持久化、回填及计算后再显示成功。只改 toast 无法解决主问题。本轮未提交这项配置。

### A04 · P2 · 收起筛选造成显示条件与实际查询不一致

**复现**：海运出口订单展开筛选，单号输入 `UXAUDIT_NO_MATCH` 并查询，收起再展开。输入变为空；手动刷新仍请求 `numberKeyword=UXAUDIT_NO_MATCH`。

**原因/影响**：筛选组件卸载，但父级 `currentFilterRef` 保留已提交条件。用户看到无条件却只有过滤后的数据。证据：`web/src/components/ui/order-list-template/OrderListTemplate.tsx:64`、`:567`、`:633`；`OrderListSearchFilter.tsx:172`、`:256`。

**建议**：收起只改变显示，保持表单实例；或者父级保留完整原始条件并回填。验证未提交输入、已提交单号、日期和状态多选的收起/展开，不必引入全局状态。

### A05 · P2 · 自动刷新是无功能开关

**复现**：订单列表开启开关，保持页面前台 65 秒，没有订单列表 GET；手动刷新产生 1 次 GET。tooltip 承诺「每 60 秒自动静默刷新」。

**原因/影响**：开关只改变 `autoRefresh` 本地 state，没有计时器或父级回调；用户会误以为正在看到最新状态。证据：`web/src/components/ui/order-list-template/OrderListToolbar.tsx:64`、`:219`、`:223`。

**建议**：未交付则移除入口/承诺；保留则接入现有 `actionRef.reload()`，卸载清理，并明确编辑期间是否暂停。无需额外轮询平台。

### A06 · P2 · 客户初次加载失败缺恢复入口

**复现**：打开有效客户详情，拦截档案 GET 返回 503。加载结束后仍显示普通档案结构和空白禁用字段，保存按钮不显示，也没有重新加载按钮。

**影响**：失败通知消失后，页面缺明确失败状态；用户只能刷新整页或离开。不是空白表单可以直接保存的缺陷。证据：`web/src/pages/partners/partner-detail.tsx:162`、`:196`、`:675`、`:744`；`web/src/utils/queryClient.ts:14`。订单已提供 Result/重试。

**建议**：首次失败且无数据时显示失败状态、重试和返回列表；重查失败有旧数据时保留当前输入。直接复用现有失败页模式。

### A07 · P2 · 顶栏下拉缺键盘打开方式

**复现**：聚焦企业资源/设置中心按钮，按 Enter、Space、ArrowDown 均不打开；鼠标 hover 正常打开。组织切换按钮按 Enter/ArrowDown 也不打开，Space 未复测该按钮。

**影响**：键盘用户无法正常选择这些入口。证据：`web/src/components/layout/HeaderMenus.tsx:173`、`:195`、`web/src/components/OrganizationSwitcher.tsx:133`、`web/src/components/HeaderDropdown/index.tsx:44` 及浏览器对照。

**建议**：基于原生按钮 click trigger 支持键盘激活，同时保留既有 hover，并补展开状态关联。复测菜单项焦点、Esc、返回触发按钮；不重写 Antd 菜单键盘模型。

### A08 · P2 · 可折叠分节标题缺键盘语义

**证据**：客户新建的基础信息、财务结算规则、联系方式、客户备注标题均为 `DIV`，`tabIndex=-1`，无 role/aria-expanded；`SectionCard.tsx:38` 仅 click 切换。没有其它键盘等价折叠入口。

**影响/建议**：键盘用户无法操作共享折叠能力，也无法得知展开状态。可折叠时使用保持原样式的原生 button，提供 `aria-expanded` 和内容关联；展示标题不增加交互。源码：`web/src/components/ui/page-shell/SectionCard.tsx:38`、`:103`。

### A09 · P2 · 页签焦点不可见

**复现**：键盘模式聚焦业务页签，`:focus-visible=true`，计算样式 outline-style 为 none、box-shadow 为 none、border-width 为 0。页签与关闭按钮样式都主动移除 outline，没有 focus-visible 补偿。

**影响/建议**：页签可通过 Enter/Space 激活，关闭有名称，但键盘位置不清楚。给页签/关闭补统一焦点样式即可；左右/Home/End 导航可后续增强。证据：`web/src/global.less:653`、`:693`；`web/src/components/layout/TagsView.tsx:378`、`:401`。关闭按钮的计算样式没有单独浏览器采样，其问题依据共享源码。

### A10 · P2 · 窄屏部分入口从主导航消失

**复现/限定**：390px 公司工作区顶部企业资源/设置中心隐藏，移动抽屉仅工作台、订单管理、费用管理。供应商列表、国外代理列表、企业资源配置缺常规主导航入口；公司工作区设置中心入口亦缺失。

**排除误报**：公司工作台仍有客户管理/主数据快捷按钮；系统工作台可经公司管理/人员分配进入系统管理，再切内部页签。订单中存在供应商/代理快捷新建，对应详情可返回列表。因此不是所有隐藏路由完全不可达，也不是权限缺陷。

**建议**：将同一权限过滤后的资源/设置入口在窄屏保留，或加入抽屉，复用既有菜单数据。证据：`web/src/global.less:1393`、`web/config/routes.ts:38`、`:74`、`:102`、`web/src/router/adaptRoutes.tsx:128`、`web/src/pages/workbench/WorkbenchSideCards.tsx:240`。375/767/768/769px 断点边界尚未实测，修复时补检查。

### A11 · P2 · 新建订单发生整页横向溢出

**复现**：390px 新建海运出口订单，等待响应式稳定后 `document.documentElement.scrollWidth=537`，超出视口 147px；工作台同条件为 390px。非表格区域的表单字段向右溢出，宽表已有容器内滚动不能解释该整页溢出。

**影响/建议**：窄屏需横向拖动整页，阅读与录入错位。检查订单布局、字段默认/最小宽度及跨列包装；用表单内真实容器降列和 min-width 约束修复，保留全宽规范。定位入口：`web/src/components/ui/order-template/OrderFormTemplate.tsx:282`、`web/src/components/ui/form-row/FormRow.tsx:20`、`web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx:354`。这些是排查入口，尚未锁定唯一根因，不把已有单列媒体查询直接当作错误。

### A12 · P2 · 列表页签没有恢复查询上下文

**复现**：客户列表输入并提交 `UXAUDIT_NO_MATCH`，点击侧栏工作台，再点击已有客户页签。返回客户路由后关键字为空。页签保留，列表本地状态重新初始化。

**影响/建议**：反复核对客户/订单/费用时需重复输入；返回列表也可能失去核对位置。先保存已提交的筛选/页码等轻量查询上下文，可同步 URL 或复用用户/组织/页签 scope 快照；不要缓存整份结果或引入全站 KeepAlive。证据：`web/src/app/AppLayout.tsx:90`、`web/src/components/layout/TagsView.tsx:108`、`web/src/pages/partners/index.tsx:478`、`web/src/components/ui/order-list-template/OrderListTemplate.tsx:64`。页码及其它列表仅代码范围推断，未逐一浏览器复测；组织隔离仍需保持。

## 已有良好交互与后续候选

已做好：统一列宽鼠标拖动/双击、拖动 Escape/失焦清理；列设置即时生效、冻结和权限必显限制、用户/组织/表格隔离；列排序有带名称的键盘上下按钮；订单草稿/未保存守卫和校验错误展开/滚动/聚焦；新建订单成功结果校验与加载失败重试；长表单页头及吸底保存；组织切换 loading、防重和 dirty guard；公共 ModalForm/快捷建档的异步提交生命周期。

P3 增强：列宽手柄当前是鼠标 div，键盘/触屏缺等价调整。可在列设置补宽度控件或给手柄 separator/键盘语义；触屏是否需要由实际使用场景决定。`web/src/components/ui/responsive-table/ResizableHeaderCell.tsx:138`、`:156`。主要桌面鼠标路径已统一，不将此升级为业务阻断。

待复核而未计入问题：联系人图标按钮可能由图标 `aria-label=edit/delete` 获得名称，不能称无名；可进一步核对业务名称是否明确。重新查询是否应回第 1 页、列表 GET 失败与真实空态的区别、页签右键键盘入口/大量页签自动滚入视野、账单/发票弹窗窄屏、对比度与 reduced-motion 留给定向验证。没有据静态代码直接给它们定性。

## 证据与验收记录

- `research/skill-method.md`：技能取舍与项目适用口径。
- `research/forms-audit.md`、`research/navigation-tables-audit.md`：两条代码研究链路与相邻路径。
- `research/browser-baseline.json`：10 条代表路由、两档视口结构记录，共 20 条；初始只等两帧，最终响应式结论取稳定复测。
- `research/browser-confirmed.json`：未保存离开、订单筛选、分节语义、页签焦点、390px 布局/菜单、客户详情失败。
- `research/browser-followup.json`：自动刷新 65 秒、顶栏鼠标/键盘对照、写失败提示捕获。
- `research/browser-list-return.json`：修正列表导航定位后的真实页签返回、写失败复现。

浏览器首次定位部分按钮未计入 icon 的可访问名称、将侧栏误作 link，造成脚本 locator 超时；后续使用明确按钮文本及 menuitem 修正并复测。失败的脚本定位没有作为产品缺陷。通知因导航卸载而漏采样的问题也通过跳转前观察器修正。原始采样中“客户/主数据不存在入口”的推断已排除。

验收只验证审查交付：检查文档/JSON、代码位置及证据对应、无敏感数据、范围与严重度一致、`git diff --check`；产品代码、依赖与构建配置未变，本轮不重复产品全量测试/构建。修复任务应按上述具体失败场景补验证。
