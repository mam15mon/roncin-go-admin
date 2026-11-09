# Research: 路由标题单一来源与海运改配零值提示

- Query: 如何从集中路由生成顶栏/页签标题并保持现有业务差异；改配 false 省略的提示与门控影响。
- Scope: internal（仓库源代码、Proto、现有测试和 Node 直接解析实验）
- Date: 2026-11-09

## Findings

### A1. 两处标题错误已直接执行确认

执行 Node 24 的原生 TS 读取（只读导入 routeUtils.ts），实际输出：

| 路径 | 当前 resolveRouteTitle | 配置声明/应保留业务语义 |
| --- | --- | --- |
| /orders/air-export/masters/EXAMPLE | 空运出口订单列表 | config/routes.ts:145–150 定义“航空共享主单” |
| /commission-adjustments/EXAMPLE/my-supplement-source | my-supplement-source | config/routes.ts:280–284 定义“我的冲减来源” |
| /orders/sea-export/new | SE 新建订单 | 保留既有明确业务标题 |
| /orders/air-export/new | AE 新建订单 | 保留既有明确业务标题 |
| /orders/sea-import/new | 订单管理 | 未开放 kind 不显示虚构业务名 |

- `components/layout/routeUtils.ts:5` 的 ROUTE_TITLE_MAP 与 `:35` DYNAMIC_ROUTE_PATTERNS 是另一套人工标题目录。
- `routeUtils.ts:141–171` 先查表、查正则、按已知前缀回退，再取最后路径段，所以遗漏路由会退成父列表或英文。
- `HeaderTitle.tsx:9–10` 每次只由 pathname 调 resolveRouteTitle；`TagsView.tsx:128/:158/:174` 同样消费它。
- config/routes.ts 是实际路由唯一真相源。routeTypes.ts:9–20 当前只有 name，没有单独展示标题字段。

### A2. 有意差异不是缺陷，必须保留

- 菜单名“海运出口/空运出口”（routes.ts:133/:140）与页签“海运出口订单列表/空运出口订单列表”并非机械相等要求。
- 企业资源路由 name 为“企业资源配置管理”（routes.ts:40），快捷菜单 label 为“配置管理”（workspaceMenus.tsx:118），顶栏/页签当前也明确采用“配置管理”。不能把它简单改成长名。
- `component-guidelines.md:272–277/:335` 明确：动态订单详情、费用、拆票使用中性占位，数据加载后由页面发事件回填；通用 layout 不维护订单 kind 字典。
- 动态单号事件生产者：detail.tsx:287–304、fees.tsx:318–335、split.tsx:435–447；消费者 `TagsView.tsx:180–212` 只修改与事件 pathname 完全相同的当前标签，防旧订单事件污染新路径。
- 顶栏 HeaderTitle 不消费单号事件，继续显示静态路由标题即可；本任务不需要将动态单号强塞顶栏。
- `TAB_KEY_RULES` 是页签身份归组，独立于标题。既有海运/客商/费用归组应保持；目前空运新建仍独立 key，此次只修标题，不顺带扩大空运页签归组。

### A3. 建议最小统一方案

1. 在 UmiRoute 增加可选显示标题元数据（例如 `pageTitle`），route `name` 继续用于菜单。默认标题用 name，只有上述明确差异才声明 pageTitle。所有具体差异都落 config/routes.ts，删除 layout 手写另一份标题目录。
2. 从 routes.ts 扁平索引/匹配生成标题，匹配静态更具体路由优先于动态，再使用具体业务父路由的中性回退。可复用 React Router 已在工程使用的 `matchPath`；organizationSwitchDestination.ts:6–14 已有扁平化先例，但无需为几十行逻辑建设通用路由平台。
3. 泛型 `/orders/:kind/new` 需要保留 SE/AE 标题并对未知 kind 回落“订单管理”。技术上可在集中配置允许一个轻量标题 resolver（接收匹配 params），或在集中配置登记已知具体新建展示元数据；由实施选最低成本形态。关键是 layout 不再复制字典、不导入 pages 注册表。
4. 动态详情/费用/拆票在配置声明中性 pageTitle（订单详情/订单费用录入/订单拆票），保留动态事件和稳定 tabKey。航空主单的更具体路由不能被泛型订单详情/空运列表吞掉。
5. 重定向取配置的最终业务目标，不再复制 REDIRECT_ROUTES。当前 /orders、/partners 是已有页签合并语义；/settings 在 routes.ts:274 已重定向费用设置，旧 routeUtils 仍写“参数设置”，应测试直接入口最终只留实际业务页签。重定向循环防护按有限配置处理，不增加历史回退层。
6. hideInMenu 只控制菜单可见性，不能用于过滤标题索引；本次两个缺陷路由恰好都是 hideInMenu。layout:false 路由及其继承子路由不进入业务页签，保留 TagsView 的登录排除，不把 login/callback 的英文回退列为用户可见缺陷。
7. workspaceMenus 中组织架构/用户人员等 label 指向 /admin?tab=...，是配置中心内部子页签；HeaderTitle 当前仅 pathname 显示“系统管理”，本任务不改变为 query 级标题。可仅将重复的“具体业务路由”快捷菜单标签接到配置元数据，不借此重构配置中心菜单或权限。

### A4. 最小标题回归矩阵

| 用例 | 断言 |
| --- | --- |
| 直接进入两处遗漏路由 | HeaderTitle 和 TagsView 分别显示配置中的中文标题，不出现英文段，不退父列表 |
| SE/AE 列表、新建 | 保留列表完整标题及 SE/AE 前缀，不改变菜单短名 |
| 动态详情/费用/拆票 | 初始中性标题，真实单号事件更新标签；顶栏静态标题不变；过期 pathname 事件不覆盖新子页 |
| /orders、/partners、/settings | 按真实配置跳到最终业务入口，不建立重定向中间标签 |
| hideInMenu 与 layout:false | 隐藏业务路由仍能解析中文；登录及回调不新增业务标签 |
| 末尾斜杠、query/hash | 标题按规范化 pathname；标签完整 path 与稳定 key 保持既有语义 |
| 未开放订单 kind/未知子路径 | 维持中性回退与既有身份规则，不虚构新业务，不把未知扩展过度合并 |
| 集中路由增加一条普通 named 页面 | 解析器自动消费 name，避免再次需要手填 layout 名称表 |

现有基础测试：TagsView.test.tsx:89–143 覆盖解析与业务差异，:144 起覆盖稳定 key，组件测试覆盖标题事件；可定向扩充而非推倒重写。

### B1. 改配无效原因缺失已确认

- `sea_order_change.proto:475–483`：SeaOrderReassignmentPreviewData.is_valid 是非 optional bool，errors 是 repeated string。
- `server/internal/service/sea_order_change.go:353–355` 直接投影 IsValid/Errors；当前 ProtoJSON 会省略 false。
- `SeaOrderReassignmentModal.tsx:228–244` 验证响应实体和可执行预览的版本，但原样 return response.data，未对 isValid 做归一化。
- `:250–263` 只有 `previewData?.isValid === false` 才拼出错误原因，实际 wire `{ errors: ['...'] }` 使该分支不成立，`:486–493` 因 previewError 为空不显示错误 Alert。
- `:264–268` 可执行条件要求 isValid===true；`:314–330` 和 `:359–360` 还会检查当前预览/disabled。因此这是“用户看不到阻断原因”的确定缺陷，不能据此声称前端放行非法改配，更不能声称服务端权限可绕过。
- 现有 `SeaOrderReassignmentModal.preview.test.tsx` 第二项模拟 isValid:false（显式字段），因此没有覆盖真实省略 key；它可直接增加/调整真实 wire 缺 key、空 errors 的场景。

### B2. matched 已正确归一化，应排除

- `order.proto:962` matched 同为非 optional bool，但弹窗 `:148–165` 已验证成功信封，并 return `{ ...response, matched: response.matched === true }`。
- 因此 `:182` 和 `:768` 后续比较 matched===false 是读取稳定模型，不是死分支；不要机械改成 `!==true` 后把加载中也解释为未匹配。

### B3. canReassign 菜单/弹窗存在确定的不同步路径

- `SeaExportDetailFeatures.tsx:262` 菜单 disabled 为 `!changeActions?.canReassign || businessWritesDisabled`，因此加载中、失败、wire 省略时不能新打开弹窗。
- 同文件 `:374–379` 给已经打开的弹窗传 `disabled={changeActions?.canReassign === false}`。动作资格请求 `:134–145` 会把真实 wire `{ reassignBlockedReasons:[...] }` 或失败后的 null 放入 state；此时菜单已禁用，但弹窗 disabled 仍为 false。
- 具体触发序列：先用 canReassign:true 打开弹窗 → refreshTypeState / order.version 变化重查 → 新动作返回不允许（省略 canReassign）或失败清为 null → 外层菜单停用，已打开弹窗没有收到 disabled，不触发 `SeaOrderReassignmentModal.tsx:128–130` 的自动关闭，也不会因为动作变化停止后续预览/确认。
- 当前业务锁变化有单独防护：父组件 `:168–174` 会关闭未同步中的弹窗；这不能覆盖“权限/资格改变但 businessWritesDisabled 未变化”或资格请求失败。
- 建议本包统一“明确允许才可操作”的派生条件，菜单与弹窗消费同一结果，缺资格数据不得继续编辑/执行；不要顺带将现有动作加载 Hook 改造为新架构。
- 必须保留写成功的同步阶段：父组件 reassignSynchronizingRef（:381–390）和子组件 synchronizing（:374–395）用于保护“写已成功但刷新未完成”的会话，不能因一次临时资格清空而提前卸载成功反馈。

### B4. 最小改配修复及回归

- 最薄修复可在成功 queryFn 返回时局部归一化 isValid（与 matched 先例一致），或仅在 `previewQuery.isSuccess && previewData` 时使用 `isValid !== true`。不要无上下文使用 `previewData?.isValid !== true`，否则未加载也显示失败。
- 有效预览仍必须验证订单/关联版本，目标改变仍立即阻止提交，迟到数据仍由 QueryKey 隔离；不变更服务端状态机、并发版本或权限规则。

| 用例 | 断言 |
| --- | --- |
| 成功返回省略 isValid、带 errors | 显示具体阻断原因，确认禁用，保留表单输入 |
| 成功返回省略 isValid、errors 也省略 | 显示“当前目标不满足改配条件”，不空白、不允许确认 |
| 首次加载/请求失败 | 加载不冒业务否定提示；请求失败保留现有具体错误和重试 |
| 显式 false / true / true但缺版本 | false 同样提示；合法 true 可确认；缺版本仍明确失败 |
| 旧无效预览迟到 | 不污染新目标，不替换当前版本、不显示旧原因 |
| matched 省略且 success:true | 仍正常进入新建母单匹配结果，保留已有归一化 |
| 已打开弹窗后 canReassign 省略或资格刷新失败 | 菜单与弹窗同样拒绝继续操作，旧确认不得提交 |
| 写成功后的刷新同步中 | 保留当前成功会话直至刷新结束，成功/刷新失败提示和关闭行为不回退 |

现有邻近测试：SeaExportDetailFeatures.test.tsx:306–349 有同步与业务锁关闭保护，:442–466 有 refreshTypeState；各 ReassignmentModal.preview/lifecycle/write-lifecycle/conflict 测试分文件组织，可按职责加最少用例避免重型 suite 长尾。

### B5. 相邻拆票路径核对

- split.tsx:742–751 以 truthy isValid/conservationPassed 判定可执行，省略 false 仍正确阻断。
- SplitConservationSection.tsx:25–35 用通过/未通过分支，:38–55 直接展示 validationErrors，不依赖 false 字段存在；当前未确认与改配相同的错误原因死分支。
- split.tsx:713–717 对 resp?.data??null 的失败响应语义另有可审计空间，但它不是本轮确认的 false 比较问题，不建议顺带扩大。

## Files Found

- `web/config/routes.ts`：实际集中路由与业务 name、redirect、layout/hideInMenu。
- `web/src/router/routeTypes.ts`：可扩展轻量 pageTitle 元数据的类型。
- `web/src/components/layout/routeUtils.ts`：独立标题表、动态匹配、重定向与页签归组。
- `web/src/components/layout/HeaderTitle.tsx` / `TagsView.tsx`：标题消费者与动态单号事件。
- `web/src/components/layout/workspaceMenus.tsx`：快捷菜单 label 与 query 级内部配置入口。
- `web/src/router/organizationSwitchDestination.ts`：既有 routes 扁平化、matchPath、generatePath 用法。
- `web/src/pages/orders/components/drawers/SeaOrderReassignmentModal.tsx`：匹配、预览、执行及同步生命周期。
- `web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx`：动作资格数据与外层/弹窗门控。
- `web/src/pages/orders/split.tsx` / `components/split/SplitConservationSection.tsx`：相邻拆票校验。

## Related Specs / External References

- `.trellis/spec/web/frontend/component-guidelines.md:268–277/:335`：稳定页签身份、动态标题、中性占位与禁止 layout 维护 kind 字典。
- `.trellis/spec/web/frontend/type-safety.md`：Proto presence、业务成功上下文及展示映射。
- `.trellis/spec/web/frontend/state-management.md`：QueryKey 隔离、生命周期、写成功刷新失败区分。
- 只使用仓库源代码和已安装 Node/React Router 的既有调用先例；未引用外部网页或依赖新版本行为。

## Caveats / Not Found

- 本轮只读研究；未编辑产品代码、既有测试或规范，未运行浏览器，也未运行产品测试。Node 直接调用已确定证明两处解析错误；改配结论由 Proto、queryFn 与 UI 分支调用链确认。
- 没有声称全部路由 name 与标题必须相同；有意短名、动态单号及内部 tab 名已列为必须保留。
- 除最终方案评审外，本研究未发现必须由用户决定的产品问题：两处标题采用已声明业务名称，改配展示已由服务端返回的阻断原因，动作资格不重定义。
- 若实施要改变空运页签归组、让顶栏也显示动态单号、重构全部配置中心快捷菜单、改变同步期间关闭语义，这些都是超出最薄修复的范围，需回到规划明确，不可默认加入。
