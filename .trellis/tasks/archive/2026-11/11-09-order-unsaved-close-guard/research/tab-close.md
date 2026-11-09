# 调研：订单详情未保存修改的最简离开保护

- 问题：取消已有订单详情的本地草稿，在切换、关闭或其他会丢失输入的离开前确认；用户确认后放弃修改。
- 范围：仓库内部；只读产品代码，输出仅写本任务 `research/`。
- 日期：2026-11-09。
- 最新产品决定：用户明确选择最简路线。切换系统页签也确认放弃，不再要求后台保留订单输入；不引入内存表单会话或整页 KeepAlive。

## 核心结论

当前没有页面 KeepAlive。`TagsView` 只记录菜单页签最后 URL，实际页面由 `AppLayout` 的 `<Outlet />` 渲染，跨菜单切换会卸载订单详情。原来的「切回恢复」依赖 sessionStorage 草稿。因此最薄路线是：取消详情草稿读写，订单详情复用已有 `useRouteLeaveGuard` 阻断所有会替换输入的真实导航，关闭继续复用 tab guard；再由工作区统一补浏览器原生刷新/关窗保护。

新建订单仍保留既有草稿契约。详情输入只在当前已挂载表单内存在；切换至工作台、列表或另一单据前用户确认一次，确认后卸载并放弃，回来读取服务端值。

## 文件与现状

| 文件 | 现状与证据 |
| --- | --- |
| `web/src/app/AppLayout.tsx:232` | 用户+组织 key 的 `OrganizationWorkspace` 包住单一 `<Outlet />`（:237），没有路由页面缓存。组织切换在 :117 先卸载旧业务，再更新身份并导航。 |
| `web/src/components/layout/OrganizationWorkspace.tsx:27` | QueryClient、Antd App、列表查询 Provider、TagsView 与正文共享工作区生命周期；适合放唯一浏览器离开监听。 |
| `web/src/components/layout/TagsView.tsx:112` | 页签状态是 `{key,path,title,closable}[]`；:150 同 key 更新最后 URL；:215 点击直接 `history.push(tag.path)`，会经过 Router blocker。 |
| `web/src/components/layout/routeUtils.ts:62` | 海运订单列表、新建、详情、费用、拆票、HBL 共用 `/orders/sea-export`；:98 `resolveTabKey` 为现有身份入口。空运未列入该归组表，回退实际 pathname。 |
| `web/src/components/layout/tabCloseGuard.ts:35` | `Map<tabKey,Set<guard>>` 支持同一页签多个独立表单；卸载只注销自身。 |
| `web/src/components/layout/tabCloseGuard.ts:97` | `isTabDirty` 同时读实时 guard 与本身份的 `hasTabDraft`；详情取消持久化后应依赖实时 guard，新建草稿仍需保留后半段功能。 |
| `web/src/components/layout/tabCloseGuard.ts:164` | `confirmIfTabsDirty` 对本次关闭范围只弹一次；:39 `runConfirmedDeparture` 临时放行本次全部 dirty guard，异步结束/失败撤回，供路由避开第二次提示。 |
| `web/src/components/layout/tabCloseGuard.ts:119` | `getAnyDirtyTabKey` 只查实时已注册 guard，不扫描 sessionStorage；适合作为当前工作区原生离开监听的判断。 |
| `web/src/components/layout/tabCloseGuard.ts:196` | `useTabCloseGuard` 通过 ref 读取当前 isDirty；组件卸载注销。在详情没有后台输入的最简路线下，这是正确的生命周期。 |
| `web/src/components/layout/useRouteLeaveGuard.ts:7` | 已有 `useBlocker`，绑定源 `location.key`，pathname/search 变化时确认，识别 `isTabDepartureConfirmed` 复用关闭确认；目前订单主详情未接入。 |
| `web/src/pages/partners/partner-detail.tsx:145` | 客商同时登记 tab guard 与 route leave guard，没有自动本地草稿，正是本期详情可复用的模式。 |
| `web/src/pages/orders/hbl/HblEditor.tsx:148` | HBL 已按完整表单指纹登记 route/tab guard，保留现有独立行为。 |
| `web/src/components/layout/useUnsavedChangesGuard.ts:17` | 当前组件 dirty ref，用户事件 markDirty，初始化不标脏，identity/保存成功 resetDirty。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:170` | 完整 draft 身份决定存储；:189 恢复，:204 主动带入，:212 resetTo，:229 保存成功，:351 用户修改，:403 后续程序回填集中在模板。宜显式关闭详情持久化而保留即时 dirty，不另造事件扫描。 |
| `web/src/pages/orders/detail.tsx:308` | 页面另读海运草稿保护原版本；取消详情草稿时必须一并移除此读取及 restoredDraft 初值分支。 |
| `web/src/pages/orders/detail.tsx:517` | 手工刷新与 :1244 类型扩展显式刷新通过 `isTabDirty` 提示覆盖输入，保留这些保护和原并发版本校验。 |
| `web/src/components/RightContent/AvatarDropdown.tsx:38` | 退出登录直接调用 API、清身份、导航，没有未保存确认。如果本期覆盖主动退出，须在 API 前确认，不能等身份销毁后才提示。 |

## 关闭链路与最薄改法

### 单个与批量关闭

- 关闭按钮和右键「关闭标签页」均先执行 `confirmIfTabsDirty([key], action)`，再清查询/草稿、移除页签、必要时导航（`TagsView.tsx:220-242`、:268-280）。保留这个入口，只调整关闭专用文案。
- 「关闭其他」「关闭右侧」「关闭左侧」先计算本次待关闭 keys，再统一确认，只有确认后才清理与导航（:284-363）。不能按每个 dirty 页签循环弹框。
- 当前没有显式「关闭全部标签页」。对工作台执行「关闭其他」已有全部关闭效果（:299-301）。最薄新增是在同一右键菜单加「关闭全部标签页」，取 `tags.filter(t => t.closable)`，一次 `confirmIfTabsDirty(keys, action)`，确认后统一清查询/保留入口草稿、setTags([FIXED_TAB])，导航 `/welcome`。
- `computeNextActivePath` 只在关闭当前页签时选择前一页签。关闭后台页签不应改变当前路由。
- 关闭专用提示使用「有未保存的修改，是否放弃并关闭？」及「继续编辑／放弃并关闭」。站内导航使用「有未保存的修改，是否放弃并离开？」及「继续编辑／放弃并离开」。共享 dirty 判定，动作词由确认入口决定，不让订单每页复制第二套模态逻辑。

### 避免关闭后路由重复提示

- 现有 `runConfirmedDeparture`（`tabCloseGuard.ts:39-61`）会把本次涉及的所有 dirty guard 放入临时确认集合；route blocker 用 `isTabDepartureConfirmed(tabKey)` 放行（`useRouteLeaveGuard.ts:17`）。继续沿用，不能删去。
- 关闭全部或关闭其他的确认范围必须包含实际关闭的全部 keys；否则当前 dirty 订单在后台草稿排第一时可能再弹一次。
- 执行动作如返回 Router navigate Promise，应返回该 Promise，使放行持续到导航结束；不能在导航异步加载前就提前撤回放行。
- 动作失败后临时放行应撤销，保留输入与 dirty，下次离开仍提示；不要为了省一次弹窗先清 dirty。
- 不把确认放行改成永久路径白名单，不让本次关闭误放行另一个表单或下一次导航。

## 真实路由离开接入

1. 订单主详情读取模板实时 dirty（同步 ref 或已有 guard），接入现有 `useRouteLeaveGuard`。已确认关闭/切组织复用 `isTabDepartureConfirmed`。
2. 所有真正 pathname 或 search 导航都经过 blocker：点击工作台、切换系统页签、返回列表、另一订单、费用/HBL 子页、history.push/replace、浏览器前进后退均按「当前有未保存输入」确认。取消保留路由和输入；确认后才导航并放弃。
3. 保留 `currentLocation.key === location.key` 条件。Router 提交与旧组件 effect 注销不同步，缺此条件会让已离开的旧表单阻断下一次立即前进。
4. 当前同 pathname+search 的纯 hash 变化不拦截；一般只是锚点，不替换输入，不需扩大阻断。初始化、候选查询、只读浏览不标脏。
5. 保存成功、明确放弃/重置后清 dirty；失败、校验未通过、被全局消费后的 undefined 响应仍保护输入。不要只因为 Promise resolve 就清 dirty。
6. 避免全局给每个 guard 登记独立 blocker：React Router 同时多个 blocker 会引发冲突。当前只有活跃路由的单个详情 blocker，独立 HBL/客商继续原接入；模板如通用接入应有明确模式，不能让 HBL 重复接入。
7. 详情关闭持久化后，已有旧详情 sessionStorage 记录不能继续影响 isTabDirty，造成「打开未编辑就提示」。最薄做法是在已确认属于当前详情键的入口一次删除该详情旧草稿；不迁移、不恢复、不扫描新建及其他身份草稿。需由主任务确定具体清理所有权。

## 原生刷新与关窗保护

- 全仓 `web/src` 未发现 `beforeunload`、`useBeforeUnload` 或 `pagehide` 保护，需要补充。
- 工作区内放一个 `beforeunload` 监听，事件触发时查实时 `getAnyDirtyTabKey()`，有已注册未保存输入则 `event.preventDefault()` 并设置 `event.returnValue`，请求浏览器原生确认；干净时不阻断。
- 事件时读取回调，不依赖字段变化后的 effect 重新绑监听，保证首次修改后马上刷新也有保护。
- 原生浏览器刷新、关标签页、关窗口、站外整页跳转均走此机制；浏览器可能仅展示原生固定文案，不要求显示业务自定义文本或 Antd 模态。
- 右键页签「重新加载」当前直接 `window.location.reload()`（`TagsView.tsx:254-264`），会被同一 beforeunload 自动覆盖；不用先弹 Antd 再弹浏览器形成重复确认。
- 监听只属于当前工作区，卸载删除。组织切换是站内动作，继续既有 `confirmIfAnyTabDirty`，不额外浏览器确认。
- 主动退出登录若进入本期范围，在发 logout API 前用离开专用统一确认；接口失败保持输入。强制鉴权失效不能让旧表单阻止身份隔离。

## 必需回归与现有范本

| 文件 | 已有覆盖与本期用途 |
| --- | --- |
| `web/src/components/layout/tabCloseGuard.test.ts:27` | 多 guard 注销、批量确认、异步失败撤回；保留 :84/:99 新建持久草稿相关测试，新增关闭/离开动作文案及即时 dirty。 |
| `web/src/components/layout/TagsView.test.tsx:819` | 取消保留、:849 确认关闭、:878 关闭其他一次确认、:923 清草稿；现有文件已重，不继续堆重用例，新增独立关闭全部/路由集成文件。 |
| `web/src/components/layout/TagsView.query-state.test.tsx:104` | 单个/其他/左右/工作台关闭其他的实际 keys；:123 取消不清、确认清、切换不清查询。新增全部关闭的同类矩阵。 |
| `web/src/components/layout/useRouteLeaveGuard.test.tsx:88` | 路由提交后旧 effect 仍在时立即前进；:121 后台草稿+当前客商批量只确认一次；:154 对象切换；:175 query；:200 前进后退；:249 关闭确认复用及失败后仍保护。保持这些真实 Router 断言。 |
| `web/src/components/layout/OrganizationWorkspace.query.test.tsx:154` | 工作区 A→B→A、:188 退出再登录、:269 StrictMode；原生监听需验证工作区卸载与 StrictMode 不残留。 |
| `web/src/pages/orders/detail-draft-lifecycle.test.tsx` | 原详情草稿恢复用例需改成详情不读写 storage、切换确认放弃、保存/刷新清 dirty；保留失败保护安全语义。 |
| `web/src/pages/orders/detail-port-save.test.tsx` | 主单原 expectedVersion、保存成功推进、缺版本拒绝、显式刷新保护，不能为删草稿同时删并发校验。 |

最低真实路由集成流程：编辑客户业务编号→点击工作台/另一页签显示离开确认→继续编辑留在原页且原输入仍在→再次离开并确认→返回订单只见服务端值且无详情草稿；另测关闭按钮同样保留/放弃、关闭全部只确认一次且无 Router 二次提示、保存成功不提示、保存失败继续提示、beforeunload 在 dirty/clean 时分别阻断/放行。

## 已排除取舍

- 工作区内存表单会话：可以做到切换不提示且保留，但需要快照、后台 guard、关闭代际与版本恢复，仍增加生命周期复杂度。用户已选择简单路线，本期不实现。
- 整页 KeepAlive：会引入后台轮询、弹层和路由上下文问题，本期不实现。
- 本地自动草稿兼容/迁移：详情不再恢复，旧键只按本次范围清理；不为历史数据建立双读或 fallback。
- 不顺便修改空运菜单归组、全站客商/财务表单缓存、新建订单草稿或后端并发契约。

## 相关规范与外部参考

- `.trellis/spec/web/frontend/component-guidelines.md:269`：稳定页签 key、完整 URL；:328 模板拥有表单生命周期。
- `.trellis/spec/web/frontend/state-management.md`：保存成功判定、原版本基线、程序带入、前台变动检测；更新详情持久化边界。
- `.trellis/spec/web/frontend/organization-switch.md:18-27`：工作区隔离、多 guard、源 location.key、失败保留；同步详情不保留持久草稿的边界。
- `.trellis/spec/web/frontend/capability-navigation.md:19-21`：未保存保护、表单生命周期入口。
- 本次为内部源代码调研，未依赖外部资料、未新增依赖。原生提示最终展示受浏览器环境约束，自动测试验证阻断请求，真实浏览器补验交互。

## 限制

未启动应用或执行测试；以上结论基于真实代码与已有用例。未修改产品代码、规范或其他任务，无 Git 写操作。

## 补充：历史详情键不再参与 dirty 的最薄边界

- 取消普通订单详情持久化后，剩余调用并非只有新建订单：`pages/orders/new.tsx:483-485` 和独立 `pages/orders/hbl/HblEditor.tsx:453-455` 都仍向模板提供完整持久化身份。HBL 还在 :148-150 自己登记 route/tab guard。
- 新建来源草稿键可带 query：`sea-house-order-context.ts:25` 生成 `/orders/sea-export/new?sourceOrderId=...`，不能只精确判断路径字符串末尾 `/new` 而漏掉来源新建。
- 推荐把「哪些持久草稿仍属于有效关闭保护」的判断集中在 `formDraft.ts` 的 `hasTabDraft` 边界：命中既有用户/组织/tab 前缀后，提取剩余 draft pathname，剥离 query/hash，仅计入当前仍支持持久化的两个路径形态 `/orders/:kind/new`、`/orders/sea-export/:id/hbl`。二者对应真实调用及 `web/config/routes.ts:156`、:196；普通详情键不计入，不删除、不迁移。
- 此处只需一个纯路径匹配判断，不建持久在线 registry，不让 `tabCloseGuard` 知道订单字段或复制存储扫描，不在详情挂载时清理历史键；新建/HBL 的未挂载草稿仍能保护关闭。若本期决定 HBL 也取消持久化，再从有效路径集合排除 HBL，同时必须改其调用；不能误称只有新建在用。
- 最小测试矩阵：仅历史详情键→false；同 tab 历史详情+新建→true；带 sourceOrderId 新建→true；HBL 草稿→true；其他用户/组织→false。新键无需变化，无写入格式兼容分支。
