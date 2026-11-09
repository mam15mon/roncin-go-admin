# 状态管理

## 服务端状态（React Query v5，2026-09 起唯一模式）

- 服务端数据一律用 `@tanstack/react-query`（直接 `import { useQuery, useMutation,
  useQueryClient } from '@tanstack/react-query'`，生产与 vitest 同源）；**禁止**
  新增手写 `useState` + `useEffect` + 竞态令牌（sequenceRef/cancelled/refreshToken）
  请求链，也禁止使用已废弃的 Umi `useRequest`（底层为 2021 年
  `@ahooksjs/use-request` v2，vitest 不可用、无卸载保护，选型记录见任务
  `09-20-data-layer-direction`）。
- `src/utils/queryClient.ts` 的 `createQueryClient()` 统一策略；`main.tsx` 的公开入口
  `queryClient` 仅用于登录配置预热和公开路由。全部认证业务由 `OrganizationWorkspace`
  内按用户 ID + 工作台 ID keyed 的 Provider 创建独立实例，业务用 `useQueryClient()`
  消费当前实例，禁止直接导入公开单例。默认 `retry: false`、
  `refetchOnWindowFocus: false`、`refetchOnReconnect: false`，不得在调用点覆盖 retry。
- QueryKey 规范：`[域前缀, 实体, 参数对象]`（如
  `['workbench', 'my-receivables', { page, pageSize }]`）；域前缀常量就近收敛，
  写操作后按前缀 `invalidateQueries` 刷新。
- 常用映射：抽屉/弹窗开合用 `enabled`；链式依赖用 `enabled: !!parent`；
  翻页/搜索保留旧内容用 `placeholderData: keepPreviousData`；防抖是
  「防抖后的关键词进 queryKey」，不是在 queryFn 里 setTimeout。
- v5 红线：`useQuery` 没有 `onSuccess/onError/onSettled` 回调；`isLoading`
  仅表首次加载，静默刷新看 `isFetching`。
- 错误提示约定：请求层 `errorHandler` 已统一弹 notification；组件历史上
  「空 catch 静默」的查询必须声明 `meta: { silent: true }`，避免全局
  onError 重复弹 message；历史上自带错误文案的查询把文案写入
  `meta: { errorMessage }`；动态文案（服务端明细需透出）在 queryFn 内
  catch 包装 Error 后抛出，由全局 onError 展示。
- 测试：用 `tests/queryClientTestUtils.tsx` 的 `renderWithClient`/
  `createTestQueryClient`（每用例独立 client，防缓存串味）；React Query
  本体不需要 mock。

### 多人协作表单的主动变动检查

- 适用于订单、账单、结算等多人编辑的已有业务单据；并发版本号只由系统内部维护，
  不要求业务人员填写或理解。新建、登录、搜索等无既有实体的表单不轮询；
  低频单人维护主数据按现有并发策略处理，不为本规则强行新增版本平台。
- 当前前台单据默认每 **15 秒**轻量查询一次，切回页面时立即检查；离开订单页面、
  浏览器隐藏或身份未就绪时停止轮询。局部使用 React Query 的轮询/聚焦能力，
  保留全局不聚焦重查与不自动 retry 的默认策略，不写手工请求 Effect/定时器链。
- 及时提示、保留输入和防止覆盖是体验契约，15 秒轮询是当前默认实现，不是永久限定
  技术方式。已有可靠实时变动通知时应复用而非重复轮询；字段级实时共编需要另外明确
  业务字段关联、同字段冲突处理与保存语义，不因接入通知而自动合并业务数据。
- 只读取必要详情或版本/身份标识；已有详情包含关联主单、航程、活动关联等变动标识时
  直接复用，不重复加载全量主数据，也不扩建全站 SSE/WebSocket 通知框架。
- 查询身份包含用户工作区、组织、单据类型与 ID；旧请求不得影响新身份。探测结果只用于
  判断变动，不能推进编辑基线或回填表单。本人保存成功、显式刷新或合法正式动作后，
  检查参考点按成功快照同步，禁止将任意更晚的查询版本借给旧草稿。
- 发现外部变动后持续提示「这个订单已经发生变动，请重新刷新。」其他单据替换实体名称，
  同时保留输入与草稿，不反复弹窗。检查失败不清除已有变动提醒，不伪装成功刷新。
- 用户点击刷新时如有未保存内容，必须先确认；取消保持内容，刷新成功才同步完整快照
  并清理提醒/草稿，失败保留内容。提交时仍由服务端做最终并发校验，不能依赖 15 秒检查
  保证无冲突，也不自动覆盖、合并或重试旧编辑。
- 必须验证停留检查、返回检查、隐藏停止、本人保存不误报、失败与迟到隔离、输入保留以及
  刷新取消/成功/失败。规则约束新建或正在修改的协作编辑页，既有全站页面迁移另行明确范围。

### 事件驱动型请求处置约定

- **豁免清单**（框架自管异步，不算手写数据链）：pro-components 表单字段的
  原生 `request` prop（如 `ProFormSelect request={...}`、
  `ProFormSearchableSelect` 透传形态），防抖/loading 由 ProForm 内部处理。
- 命令式搜索（Select onSearch 等事件触发拉取）一律收敛为「防抖（或规范化）
  关键词 + 依赖参数进 queryKey」的 useQuery；「已选项保活」在渲染侧
  useMemo 合并（必要时 `queryClient.getQueriesData` 回填历史名称），
  禁止再写 `xxxRequestRef` 序号令牌。
- 小型策略/布尔查询（如 `features/finance/credit-control` 的
  `useCreditLimitIntervention`）同样走 useQuery；
  原 catch 静默回退的声明 `meta: { silent: true }`。

## 请求客户端

- 需要就地展示错误和手动重试的查询，同时使用请求选项 `skipErrorHandler: true` 与
  Query `meta: { silent: true }`，分别避免请求层和 QueryCache 重复通知。共享箱与候选
  分别使用查询状态；候选刷新不得无条件重置人工分配草稿。
- 本仓库 ProtoJSON 会省略空 repeated 字段：明确 `success: true` 且缺省 `data` 是
  合法空列表，不能与请求层消费错误后返回的 `undefined` 混同；响应缺失、失败标识或
  非法数据类型仍显示失败，不显示零空态。参考共享箱与实际箱查询回归。

- 所有后端请求经过统一请求配置或 OpenAPI 生成客户端
  （`src/services/roncin/`）；禁止页面自行拼接后端主机地址。
- **错误被全局处理器消费后 `request()` 会 resolve `undefined`**（复刻
  umi-request 契约，见 `src/utils/requestClient.ts`）。列表类调用禁止把
  该返回值直接交给 `unwrapList` 读 `.data`：必须先用
  `ensureListResponse(response, '<中文失败文案>')`（`src/utils/api.ts`）
  转 reject，否则 403/5xx 之后会以
  `Cannot read properties of undefined (reading 'data')` 的 TypeError
  文案直达用户（案例：2026-09-21 总部用户新建订单）。
- 命令式写入必须验证实际成功报文才显示成功、关闭表单、导航、刷新或清 dirty/草稿。
  默认统一错误处理消费失败后返回 `undefined` 时立即返回失败，不追加重复错误提示；
  用 `isWriteResponse(response, requireEntityId?)` 做最薄统一守卫，携带实体的写接口
  应验证必要 `data.id` 且 `success !== false`；`useMutation.onSuccess` 也必须守卫，
  因为请求 resolve `undefined` 仍会进入 React Query 的成功回调。各接口按真实契约判定，
  禁止 `await api()` 后无条件执行成功逻辑。`ModalForm.onFinish` 失败返回 `false`；
  订单已成功落库后的后台刷新继续 best-effort，不把刷新失败改判为保存失败。
- 异步写确认使用 `@/components/ui` 的 `AsyncPopconfirm`；官方 Popconfirm 在 Promise
  resolve 后会关闭，`return false` 本身不能保留弹层。共享组件沿用官方 loading，
  在回调返回 `false` 或 reject 时保留弹层且不产生未处理拒绝；回调负责已有错误提示。
  调用方必须把无效写响应转换为 `false`，本地 `catch` 也须返回 `false`，禁止吞掉
  拒绝后以 `undefined` 被当作完成。仅本地同步确认继续使用原生 Popconfirm。
- `App.useApp().modal.confirm` 中通过拒绝 `onOk` Promise 保留输入时，必须消费该
  hook 确认结果，例如 `const confirmation = modal.confirm(...);` 后调用
  `void confirmation.then(() => {}, () => {});`，或显式 `await`。这会启用 Ant Design
  的 await/isSilent 模式；仅创建确认框会让 ActionButton 再次拒绝且无人接收，产生
  未处理拒绝。消费确认结果不会吞掉业务错误反馈，`onOk` 本地 catch 仍须保持失败，
  不能提示错误后 resolve 并关闭输入。静态 `Modal.confirm` 不具备同一 hook 契约，
  不得按此写法强制转换类型。
- 费用工作台已有详情时，后台失效重查不能切换为整页加载占位并卸载子表单；首次加载占位须同时满足无详情数据。撤回重提等会失效订单缓存的动作，须验证真实父页面刷新中及完成后保留预填和新输入。
- 审核/发放失败回归须覆盖全局消费后的 `undefined`，并断言弹窗、输入和幂等键保留；补录创建、驳回、撤回及批准成功均使费用审核查询失效，避免建账能力停留在旧状态。
- 开发期走 Vite 代理，生产同域（Go 服务同时提供 `/api/*` 与静态资源）；
  修改打包或路由时两种路径都要验证。

## 场景：认证工作区查询与在途响应隔离

### 1. 适用范围

所有业务 React Query 查询、命令式写入和身份切换，包括换公司、换用户、退出再登录与 A→B→A。

### 2. 签名

- `createQueryClient(): QueryClient`：策略工厂，缓存按实例独立。
- `OrganizationWorkspace`：身份完整时挂 keyed `WorkspaceQueryProvider`；缺用户或工作台 ID 返回空。
- `activateWorkspaceRequestScope(): () => void`：layout setup 激活业务响应作用域，cleanup 失效。

### 3. 契约

- 身份变化的首个渲染就使用空缓存，不能靠普通 effect 在已展示旧数据后清理。
- 工作区 layout cleanup 调用 `client.clear()` 清理旧缓存，忽略 AbortSignal 的迟到查询也不能回写新实例。
- 普通旧业务响应（成功或全局消费的错误）返回 `undefined`，禁止影响新身份的通知、导航和草稿。
  `skipErrorHandler: true` 的错误仍原样抛出，由当前消费方负责实例生命周期。
- 登录、钉钉配置和换码不参与业务工作区清理；不取消这些传输，不改变 Cookie 响应顺序。
- 同身份重渲染与页签返回复用当前客户端；A→B→A 与退出后重进均是新实例。

### 4. 校验与错误矩阵

| 场景 | 必须结果 |
| --- | --- |
| 缺少身份 | 不挂业务子树、不借公开缓存 |
| 切换用户或工作台 | 首帧无旧缓存，旧客户端清空 |
| 旧请求延迟成功、失败或 401 | 不回写新缓存，不提示或导航新身份 |
| 当前请求成功或失败 | 保持既有请求契约 |
| 公开钉钉预热跨工作区变化 | 返回结果并复用公开缓存 |

### 5. 正常、基准与错误场景

正常：A 抽屉加载后换 B，B 响应到达前为空；基准：同公司翻页保留旧内容。
错误：所有身份共用一个单例，或切换后的普通 effect 才执行清理。

### 6. 必需测试

真实 Provider 验证首帧、用户/公司变化、A→B→A、匿名、StrictMode、同身份缓存及旧在途结果；
真实请求层验证旧成功/401/失败和当前成功/失败，回归公开登录预热与换码。浏览器复核真实提成抽屉。

### 7. 错误与正确示例

```tsx
// 错误：业务缓存跨所有身份保留。
import { queryClient } from '@/utils/queryClient';
// 正确：消费当前工作区实例，写入结果先验证再刷新。
const queryClient = useQueryClient();
if (!isWriteResponse(response, true)) return false;
queryClient.invalidateQueries({ queryKey: ['orders'] });
```

## 本地状态

- 表单、弹窗开合等 UI 状态留在组件内；跨页面共享的 UI 偏好才考虑全局。
- 不把接口响应镜像进全局 store 再派生——直接消费服务端状态层。

## 场景：客商未保存离开与详情失败恢复

- 客商表单不保存持久草稿；同步 dirty 引用覆盖 Form 字段、联系人、别名及待添加别名。
  `useTabCloseGuard({ tabKey, isDirty })` 处理关闭/组织切换，
  `useRouteLeaveGuard(tabKey, isDirty)` 使用 Data Router 的 `useBlocker` 处理 pathname/search
  变化，含侧栏、其它页签、同菜单档案切换和浏览器 POP/replace。取消导航保留表单，
  保存结果成功才清 dirty；无改动及保存成功导航不提示。
- `confirmIfTabsDirty` / `confirmIfAnyTabDirty` 的确认动作接受 `void | Promise<void>`。
  已确认动作期间仅对本次关闭范围内的当前 dirty guard 对象协调路由放行（含批量关闭）；异步组织切换必须返回 Promise，
  动作结束或失败后撤销放行，不能把 dirty 清零或永久允许后续导航。
- 客商完整 pathname+search 为表单资源身份；身份变化用 React key 重建 Form store，并在
  layout effect 清外部联系人/别名/错误及首次回填标记。用户/组织继续由工作区 key 重建。
- 主档 GET 显式 `skipErrorHandler` 外抛错误，由 React Query 的中文 meta 提示；空成功响应
  也抛加载失败。首次 loading/失败不挂空表单，失败用 Result 提供重试和返回。
  同一资源只首次成功回填，背景 refetch 成功或失败均不覆盖用户输入。
- 客商保存发起时捕获身份代际。资源 layout effect 的 setup 与 cleanup 均递增同一代际，
  响应、catch 与 finally 仅当前代际可清 dirty、导航、提示或改变 saving；旧请求不能影响
  新档案，亦不能在 A→B→A 往返后作用于重新进入的 A。
- 必须验证 reject / 全局消费后的 undefined / 缺实体 ID / success:false；真实 router
  测试导航确认及取消、POP/replace、已确认关闭不重复提示和异步失败后再次保护。
  详情测试首次失败重试、有数据重查不覆盖输入，以及在途保存跨身份和往返的迟到结果；
  不能以 mock 关闭函数替代真实路由测试。

## 场景：订单详情离开保护与可选持久草稿

### 1. 适用范围

- 已有海运、空运订单主详情只维护当前挂载表单，不读写 `sessionStorage` 草稿。系统页签仅保存导航地址，不保活页面；切换、关闭或其他会丢输入的离开先确认，确认即放弃，重新打开读取服务端值。
- 新建订单（含带 `sourceOrderId` 的新建分单）和独立 HBL 制作仍保留既有持久草稿。草稿跨组件挂载存在，必须按用户、公司和完整表单路径隔离；不把此机制重新带回订单主详情。
- 不引入内存编辑会话、整页 KeepAlive、自动保存或旧详情草稿迁移。

### 2. 签名与职责

```ts
getFormDraftScope(userId?: string, organizationId?: string): string | undefined;
getFormDraftKey(tabKey?: string, pathname?: string, draftScope?: string): string;
hasTabDraft(tabKey: string, draftScope?: string): boolean;
clearTabDrafts(tabKey: string, draftScope?: string): void;

interface OrderFormTemplateActions<T> {
  resetTo: (values?: Partial<T>) => void;
  applyValues: (values: Partial<T>) => void;
}

useRouteLeaveGuard(tabKey: string, isDirty: () => boolean): void;
getTabGuardMessage(tabKey: string, action?: 'leave' | 'close'): string;
// confirmIfAnyTabDirty 的尾参 action?: 'organization' | 'logout' 默认 organization；
// 主动退出显式传 logout，确认后异步动作须返回 Promise。
```

- `OrderFormTemplate` 独占 dirty、保存与重置生命周期；页面不维护受控 dirty 镜像，类型扩展不读写草稿键或显式刷新令牌。
- 用户/组织由 `OrganizationWorkspace` key 隔离；业务类型/记录身份变化由模板外层 React key 重建。旧表单及迟到异步任务不得影响新记录或往返后重挂的同一记录。

### 3. 行为契约

#### 订单主详情

- 详情只提供稳定页签 key、用户公司 scope、动作引用和服务端初值，不提供开启持久草稿的完整身份，也不自行调用 `getFormDraft` 恢复输入、版本或单证基线。
- 当前表单初始化后，后台重读不覆盖用户输入、不替换原 `version`、主单/分单 expectedVersion；外部变动继续前台每 15 秒及回焦检查。确认刷新且成功重读后才 `resetTo`，失败保留原输入。
- 真实用户编辑及显式 `applyValues` 同步标脏；初始化、候选加载、只读浏览不标脏。guard 必须读取同步 ref，覆盖修改后立即关闭或导航的时序。
- 已有修改不能因外部锁单或权限变化成为 readonly 就丢失保护；readonly 禁止继续编辑，关闭或离开仍确认。
- 保存成功、明确重置或放弃才清 dirty；失败、校验未过、409 和未知结果继续保留。保存结果未知时不能声称请求已撤销；确认离开后再次进入核对服务端实际状态，不增加详情恢复平台。
- 站内离开复用 `useRouteLeaveGuard` 的 Data Router blocker，覆盖菜单、系统页签、返回列表、同菜单对象替换及浏览器 POP/replace。取消留在原页并保留输入，确认后卸载放弃。纯锚点滚动不新增确认。

#### 关闭、退出与浏览器离开

- 单关、关闭其他、关闭左/右和关闭全部均通过 `confirmIfTabsDirty` 一次确认真实关闭范围；取消不清页签、查询或输入，确认后才执行。固定工作台不可关闭。
- 关闭提示“有未保存的修改，是否放弃并关闭？”，操作为“继续编辑／放弃并关闭”；路由离开使用“离开”动作语义。
- 已确认动作仍用 `runConfirmedDeparture`/`isTabDepartureConfirmed` 临时放行本次全部 dirty guard，避免页签关闭和路由双重提示。异步动作结束或失败后撤回放行，不清 dirty 代替确认。
- 路由守卫保留源 `location.key` 检查，旧实例尚未注销时不得阻止新页面下一次导航。
- 主动退出在销毁身份及编辑前确认；接口失败保留当前内容和保护，强制鉴权失效不被旧表单阻止。
- 工作区统一监听 `beforeunload`，事件时检查实时 `getAnyDirtyTabKey()`，存在修改才 `preventDefault` 并设置 `returnValue`。刷新、关浏览器标签页和整页站外跳转使用浏览器原生提示，不开业务模态；工作区卸载删除监听。

#### 仍支持的持久草稿

- 只有 `tabKey + draftPathname + draftScope` 齐全才生成草稿键；不从 `window.location` 或当前路由猜身份，缺用户或公司不持久化。
- 仍有效的路径为订单新建和独立 HBL 编辑；来源 query 是新建身份的一部分。`hasTabDraft` 不把退役的普通订单详情键算作 dirty，不读取或迁移其内容，不扫删其他用户/公司的数据。
- 首次非 loading、非 readonly 且身份齐全时在 layout effect 恢复一次；后续锁态往返不覆盖内存编辑。新身份只恢复自己的草稿。
- `applyValues` 合并用户主动带入、标脏；启用持久化的表单同步存完整快照，已修改后的程序回填/清空也入草稿。初始化及 `resetTo` 不生成脏草稿。
- `resetTo(values)` 清当前草稿、重置旧 Form store、回填新值并清 dirty。保存成功才清草稿，失败保留；确认关闭只清当前身份、被关闭页签范围的有效草稿。
- 显式清空序列化为 `null`，不得省略 `undefined` 而让默认值复活。日期按已知字段及 `*Date/*Cutoff/*At` 恢复 Dayjs，普通 ISO 文本不凭值猜测。

#### 显式刷新与版本

- 保存结果由订单更新接口决定，后台重读失败不能把已落库结果改判成写入失败；已提交后只重试读取，结果未知继续沿用原键原载荷确认。
- 显式刷新令牌归通用详情页所有：每次刷新递增，身份 layout effect 的 setup/cleanup 也递增并清 pending。请求完成和回填两端都复核令牌与身份，覆盖 A→B→A 及“写入 pending 后、消费前”变化，旧任务不能重置新实例。
- 订单品类扩展只使用 `refreshOrderAndLock` 等普通命令，不取得模板 actions、dirty setter、草稿键或显式刷新令牌。

### 4. 验证与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 主详情存在旧 storage 草稿 | 忽略，不恢复、不产生假 dirty |
| 初始化完成，没有用户修改 | 关闭、导航和浏览器离开不提示 |
| 修改后点击工作台或另一页签，取消 | 路由不变，输入及原版本保留 |
| 修改后确认离开再返回 | 读取服务端值，不恢复放弃内容 |
| 批量关闭含多个 dirty 页签 | 一次确认，仅处理实际关闭范围 |
| 外部变 readonly / 保存失败 / 409 | 输入和离开保护继续保留 |
| 明确保存成功 / resetTo | 清 dirty，后续离开不误提示 |
| dirty 时刷新或关闭浏览器页 | 请求浏览器原生确认；取消留原页 |
| 有效新建/HBL草稿，实时 guard 已卸载 | 当前用户公司内仍判定该页签 dirty |
| 缺用户公司 / 存储受限 / JSON 非法 | 不串身份；持久读失败不使表单崩溃 |

### 5. 正常、基准与错误场景

- 正常：修改客户业务编号后切工作台，先确认；取消可继续编辑，确认后回来展示服务端编号。
- 基准：只读打开、初始化、候选搜索或保存成功后直接离开，不出现误提示。
- 错误：订单主详情继续恢复旧草稿，或为了保留跨页输入另加匿名缓存；另一类错误是变 readonly 后注销 guard，静默丢失原修改。

### 6. 必需测试

- 真实 data router + Form 验证海运、空运编辑→离开取消/确认→重开；预埋旧草稿不恢复、不误报，未填写配箱计划不崩溃。
- 保存/重置成功清 dirty，失败与外部只读后仍保护；用户编辑后同 tick 关闭，程序带入，初始化不误标脏。
- 单关及批量确认取消/确认、关闭全部只提示一次、未关闭页签不误清、已确认动作复用及失败后继续保护。
- 原生 `beforeunload` 的 clean/dirty/保存成功、工作区卸载及 StrictMode 清理；主动退出取消不发请求、失败保留、成功正常离开。
- 原并发及刷新回归不得删除：A/B 与 A→B→A 隔离、迟到刷新、原版本不偷换、已写入后重读失败只读重试。
- 新建来源 query、独立 HBL 草稿、日期复活、显式清空及独立创建幂等记录继续通过；重型用例按功能分文件，不能放大单文件长尾。

### 7. 错误与正确示例

```tsx
// 错误：订单主详情重新接入本地恢复。
<OrderFormTemplate draftPathname={`/orders/${kind}/${orderId}`} />;

// 正确：主详情只接即时关闭保护，并由详情编排复用路由离开 guard。
<OrderFormTemplate
  tabKey={resolveTabKey(`/orders/${kind}/${orderId}`)}
  draftScope={getFormDraftScope(user.id, currentOrganization.id)}
  actionsRef={templateActionsRef}
/>;

// 正确：仍支持的表单由模板独占持久草稿键，来源 query 保留在规范身份中。
<OrderFormTemplate
  tabKey={resolveTabKey(`/orders/${kind}/new`)}
  draftPathname={sourceRequest.draftPathname}
  draftScope={getFormDraftScope(user.id, currentOrganization.id)}
  actionsRef={templateActionsRef}
/>;

// 明确重置和用户主动带入均通过模板动作，不另造dirty或存储链路。
templateActionsRef.current?.resetTo(initialValues);
templateActionsRef.current?.applyValues({ printDetails: { containersText } });
```

## 场景：业务页签列表查询恢复

### 1. 范围

客商、共享订单、费用/账单/资金流水列表的已提交查询通过
`components/ui/list-query-state` 公开入口恢复。状态属于当前已认证用户/组织的
`OrganizationWorkspace` 实例；工作区 key 重建销毁全部快照。缺少用户或组织时禁用共享，
完整刷新浏览器回默认。快照只存查询条件、已选候选名称、页码和 pageSize。

### 2. 签名

```ts
type ListQuerySnapshot<T> = {
  filters: T;
  labels: Record<string, string>;
  current: number;
  pageSize: number;
};
ListQueryStateProvider({ enabled, children });
useListQueryState<T>({ tabKey, listId, defaultPageSize, defaultFilters });
// 返回 initial / snapshotRef / commit(filters, labels?, resetPage?) / pagination
useClearListQueries(); // 返回 clear(tabKeys)
```

### 3. 契约

- `tabKey + listId` 显式固定到挂载身份；客商角色、订单业务类型属于 listId，角色/类型变化
  重建消费实例。无身份不共享；禁止全局 Map、匿名命名空间、结果缓存或 KeepAlive。
- 提交、重置、即时筛选、分页事件同步写快照，卸载不回写；列表仍完全通过官方
  ProTable `request` / `actionRef` 请求数据，不存 data/loading。请求条件同步引用更新后，
  新查询用 `reload(true)` 回第一页，普通刷新 `reload()` 保留页码。
- 页签真正关闭的已确认回调清理对应快照并递增 tab 代际。旧租约禁止再写，关闭重开
  获得新代际；拒绝关闭不清理。覆盖叉号及当前/其他/左/右/全部菜单。
- 通用 SearchFilterTemplate 的 `initialValues`、`initialLabels` 只首次初始化；
  `onCommit(values, labels)` 固化已提交口径。重置明确清到真实默认值，不能回到恢复初值。
  删除单个 chip 只改变已提交条件，不把其他未提交编辑混入请求。
- 日期按字段定义恢复 Dayjs，不猜普通字符串；已选远程候选保活轻量 value/label，
  查询仍提交原始 ID。财务公司来自认证工作区，不属于可恢复筛选；业务标签已退役，
  完整边界见[财务当前公司工作区](./finance-current-workspace.md)。客商黑名单仍为真实查询，不丢 false/0 值。
- 订单收起筛选用 hidden 保持 Form 挂载，保留尚未提交输入；跨页签恢复只恢复已提交值。
  自动刷新开关不跨页签保存：60 秒调用官方 actionRef，仅开启且窗口聚焦、页面可见时运行，
  busy 时跳过；关闭、blur、hidden、卸载清理，恢复前台只建立一个计时器。

### 4. 验证矩阵

| 场景 | 行为 |
| --- | --- |
| 切页签后返回 | 恢复已提交值、中文名称、分页及实际请求参数 |
| 收起后展开 | 未提交编辑也保留，刷新仍按已提交条件 |
| 恢复后重置 | 默认查询/第一页，旧 initialValues 不复活 |
| 关闭后旧事件迟到 | 旧租约写入被拒绝，重开默认值且新查询可正常保存 |
| 拒绝关闭 | 原快照保留 |
| 用户/组织变化或身份缺失 | 不读取原实例共享快照 |
| 自动刷新失败 | 不清查询，不覆盖未提交输入 |

### 5. 正常与错误场景

正常：客户列表查关键词，切工作台再点客户页签，关键词和请求一致；关闭客户页签后重新进入，
回默认查询。错误：组件卸载时写筛选，关闭清理之后又把旧查询存回来；用全局匿名键让不同身份串用。

### 6. 必须测试

Provider 的工作区隔离/身份缺失、关闭代际与重开；真实路由卸载返回；各消费者已提交查询/分页；
客商角色、财务当前公司工作区与退役公司/标签筛选不可恢复、日期/候选名称；所有关闭菜单确认/取消；
假计时器的 tick/busy/blur/hidden/恢复/卸载及浏览器真实 65 秒请求观察。

### 7. 错误与正确示例

```ts
// 错误：卸载 cleanup 保存，让已关闭页签复活。
useEffect(() => () => saveQuery(form.getFieldsValue()), []);
// 正确：提交事件同步记录，再由官方表格刷新。
queryState.commit(filters, labels);
currentFilterRef.current = filters;
actionRef.current?.reload(true);
```

## 场景：列表筛选驱动的前端 CSV 导出

### 1. 使用范围

- 独立全量导出接口适用于列表与导出共用搜索条件的场景；接口必须使用 OpenAPI
  生成客户端，不得自行拼接地址或重新推导业务金额。
- 六类财务台账 `FinanceLedgerTemplate` 仅导出当前页或选中行；按钮和成功提示明确范围，
  不循环翻页获取全量。导出快照只能来自 ProTable 采纳的数据，禁止在请求 await 后
  无条件保存迟到或失败响应，导致屏幕与导出不一致。

### 2. 数据签名

- `CsvColumn<T>`（`header`、`kind?: 'text' | 'number'`、`value(item)`）由各领域
  `ledgerExport.ts` 定义；模板 `exportColumns` 必传，与界面列显隐/顺序独立。
  `serializeCsv(items, columns): string | null` 与 `downloadCsv(content, fileName)`
  位于 `utils/csv.ts`，页面不再从 ProTable `dataIndex` 推断导出。
- 搜索表单值与请求筛选对象必须使用不同类型；日期控件值只能停留在表单类型中。
- 提交搜索时，通过单一纯函数把表单值转换为后端请求字段，例如月份开始值转为
  当月首日、月份结束值转为当月末日，单边月份允许另一端缺省。
- 列表请求与导出请求必须读取同一个已提交筛选对象；分页字段只在列表请求处追加。

### 3. 正常契约

- 搜索提交顺序固定为：规范化表单值、同步写入已提交筛选引用、触发列表刷新。
  禁止先异步更新 React state 后立即刷新，并期待请求回调读到新值。
- 导出按钮读取已提交筛选，不读取尚未提交的表单编辑值；因此屏幕列表与导出数据
  始终对应同一组查询条件。
- CSV 列必须与导出 DTO 显式逐列映射并固定顺序；空结果不创建 Blob 或下载链接。
- 金额使用 `csvAmount` 两位无千分位，汇率/单价/数量/比例用 `csvDecimal` 保留有效精度；
  原币和本位币分别导出币种。时间用 `formatDate` 本地格式，状态复用现有领域中文映射，
  关联明细显式展开，禁止 `[object Object]` 或原始业务枚举；业务标签已退役，不恢复标签导出列。
- CSV 使用 UTF-8 BOM 与 CRLF 行分隔，包含逗号、双引号、回车或换行的单元格按
  RFC 4180 形式加引号并把双引号写成两个双引号。

### 4. 校验与错误矩阵

| 输入或结果 | 处理方式 |
| --- | --- |
| 关键字只有首尾空格 | 去除首尾空格；结果为空时不发送该筛选字段 |
| 仅选择开始月份 | 只发送开始月首日 |
| 仅选择结束月份 | 只发送结束月末日 |
| 文本字段以 `= + - @` 开头，或前导空白后出现这些字符 | 在原值前加单引号，阻止表格软件执行公式 |
| 受控数值字段 | `kind: 'number'` 且值为合法十进制数时保留负数；其他值仍做公式防护 |
| 台账金额与时间 | 按两位金额/本地时间契约格式化；纯日期保持日期值 |
| 导出结果为空 | 提示用户无可导出数据，不创建下载对象 |
| 导出请求失败 | 由统一请求错误处理呈现失败，不生成部分文件 |
| 导出成功 | 点击临时下载链接后立即移除链接并回收 Blob URL |

### 5. 基础、正常与边界行为

- 基础：无筛选时，列表和导出均不携带可选筛选字段，文件名使用导出当天日期。
- 正常：双边月份转为首日和末日，列表与导出收到完全相同的业务筛选字段。
- 边界：覆盖同月、跨月、跨年、闰年、仅开始月、仅结束月、中文、双引号、逗号、
  CR/LF、公式前缀、前导空白公式前缀、合法负数与空结果。

### 6. 测试要求

- 纯函数测试必须逐列验证 DTO 到 CSV 的映射、转义、公式保护和文件名规则。
- 页面测试必须实际提交搜索并触发列表与导出，断言两者复用同一规范化筛选；只测
  按钮显隐不能证明该契约。
- 下载测试必须断言空结果不调用 `URL.createObjectURL`，成功结果会移除临时链接并
  调用 `URL.revokeObjectURL`。

### 7. 错误与正确示例

```tsx
// 错误：state 更新异步，reload 可能读取旧筛选；导出又从表单单独计算一遍。
setFilters(normalize(values));
actionRef.current?.reload();
exportCommissions(normalize(form.getFieldsValue()));

// 正确：提交时同步固化一次，列表与导出共同读取。
committedFiltersRef.current = normalize(values);
actionRef.current?.reload();
exportCommissions(committedFiltersRef.current);
```

```ts
// 错误：对所有单元格做公式保护，会把受控 decimal 的合法负数改成文本。
escapeCsvCell(protectFormula(value));

// 正确：显式声明受控金额列，公共序列化验证数字后保留合法负号。
{ header: '金额', kind: 'number', value: (row) => csvAmount(row.totalAmount) }
```

## 异步业务核对与表单校验同步

- 校验器依赖外部异步状态时，React 状态变化不会自动清除 Form 已保存的错误。核对结束、失败或关联确认变化后，应对相关字段执行完整 `validateFields`；可使用 `{ dirty: true }` 避免未交互字段提前显示必填错误。禁止直接清空错误而掩盖格式或业务冲突。
- 通过 `setFieldValue` 保存但没有 Form.Item 注册的业务字段，使用 `Form.useWatch(name, { form, preserve: true })` 监听；否则自动关联 ID、航次 ID 等可能无法回显。
- 回归测试必须使用真实字段注册结构，不能额外添加生产中不存在的隐藏 Form.Item 来让监听生效。范本：`SeaMasterBillFields.validation.test.tsx`。

## 组织切换与编辑保护

切换组织的安全落点、异步身份/路由边界、多表单未保存守卫和业务确认框所有权见
[组织切换规范](./organization-switch.md)。新增编辑入口应按该文档绑定真实用户事件，并验证失败保留输入。


## 海运录单默认值、草稿与创建结果确认

### 适用入口与签名

- `SeaCustomerField`：客户岗位仅作为选择客户时的默认值；显式重载恢复草稿人员时不能清除已有人工值。
- `OrderFormTemplate`：启用持久草稿时通过独立观察器保存完整 Form store；订单主详情不挂持久快照观察器；`actionsRef.resetTo(values?)` 清除当前草稿与脏状态。
- `useSeaOrderCreate(draftKey)`：返回 `status/frozen/submit/clear`，`submit(payload?)` 返回服务端确认的订单 ID；待确认时不再读取当前输入。

### 行为契约

- 海运付款方式和主分单条款默认值由表单初始化入口提供；同一个字段不得同时声明整表 `initialValues` 与 `Form.Item/ProForm` 的 `initialValue`，否则会触发覆盖告警。移除字段默认值前必须核查新建、详情和主分单切换的全部入口，先补齐初始化；编辑、草稿恢复及用户显式清空不得被重复默认值覆盖。回归需检查实际 Form store 和控制台告警，不能仅断言常量。
- 客户默认值仅应用到本次选择后尚未手动修改的字段，包括主动清空也算修改；客户切换、A→B→A、组件重挂和请求迟到均隔离，后台重查不得覆盖表单。
- `setFieldsValue/setFieldValue/setFields` 不触发 `onValuesChange`。启用持久草稿且用户已开始编辑后，程序回填/清空也必须保存完整快照；只初始化、候选加载、resetTo 和保存成功不生成脏草稿。
- 草稿显式清空字段序列化为 `null`；不能省略 `undefined` 后让恢复时的 `initialValues` 重新带入旧值。日期继续通过 `reviveFormDraft` 恢复。
- 自定义集合控件必须接受草稿恢复传入的 `null`：计划箱型箱量将 `null` 与 `undefined` 均作为空计划渲染，新增、修改、删除统一基于该空集合语义；不能只用参数默认值 `value = []`，该写法不处理 `null`。回归须经过真实草稿序列化、卸载与重挂，覆盖只改其他字段和已有计划显式清空，并保留输入与版本基线。
- 创建前保存 JSON 规范化载荷与幂等键，待确认记录使用独立 `roncin:pending-order-create:` 前缀并包含用户/组织/来源路径草稿身份。普通页签草稿清理不得删除未确认提交意图。
- 请求在途或结果未知时冻结业务字段；确认按钮直接重发原键原载荷，不再走表单校验或依赖来源/候选加载。明确成功取得 ID 后清理草稿、解除脏状态再导航。
- 复合输入传递 `disabled` 时，用 `disabled || undefined` 继承父 Form 的禁用状态；显式 `false` 会覆盖父表单冻结。金额、件数、客商和单证字段都要实际验证。仅用于跳转创建另一票的详情「从此主单新增分单」继续显式 `disabled={false}`，它不修改当前锁定单据，不能被编辑冻结误伤。

### 错误与恢复矩阵

| 场景 | 处理 |
| --- | --- |
| 客户人员加载失败 | 在客户旁显示失败与手动重试，不当作客户配置缺失 |
| 人员加载未完成或失败后刷新 | 恢复客户且业务/客服缺失时提供显式重新加载，只补空岗位并保留已有手动值 |
| 客户缺业务/客服配置 | 明确提示维护档案；保持只读必填与权限规则 |
| 创建前无法保存会话记录 | 不发送请求，提示存储问题 |
| 超时、5xx、无 ID、裸 4xx、防重复拦截、幂等冲突 | 保留原键和载荷，显示结果待确认 |
| 第一次提交得到明确业务拒绝 | 保留输入，解除冻结，修正后形成新意图 |
| 已有未知历史，后续确认又被拒绝 | 不据此推断原请求未落库，继续保留原提交意图 |
| 待确认刷新、来源或候选失败 | 仍可手动确认原请求；不自动换键重建 |
| 会话记录损坏或无法读取 | 阻断新建并提示核对，不能当作没有待确认记录 |

### 正反例与验证

正确：客户加载期间手动改操作员，响应仅带入未改岗位；未知结果刷新后继续同键同 JSON 载荷确认。
错误：迟到默认值无条件 `setFieldsValue`；仅在用户事件保存草稿；超时后修改载荷却沿用旧键，或直接换键重建。

必需回归：手动填写/清空、A→B→A、失败重试、程序回填后立即刷新、显式清空恢复、resetTo 不重建草稿、所有复合输入继承禁用、存储受限、请求未知刷新恢复、成功 ID 导航与相邻空运入口。船期 ETD 后移保留 ETA，靠依赖校验显示冲突，不静默删除输入。

## 海运拆票与改配预览基线

- `SeaOrderReassignmentModal` 的母单号输入必须直接绑定带 `name` 的内层 `Form.Item`；帮助文案和复合布局放在外层。测试必须实际输入并断言请求，不能通过 `setFieldsValue` 掩盖字段未注册。
- 改配预览和执行共同调用 `buildReassignmentTarget`；NEW 的 ETD/ETA 使用 RFC3339，CANDIDATE 的 ID/版本/航程取当前明确选中的候选。输入或会话身份变化后旧预览不可提交，迟到响应不得关闭新会话或覆盖新订单。
- 只有当前成功且可执行的预览才能启用确认；请求失败保留业务输入，执行结果未知时保留原幂等键与载荷。关闭重开、换单、失败后重新预览、乱序响应必须有回归。
- 拆票守恒来源为实际货物明细，委托量与实际箱独立展示，不能自动生成货物替代。缺明细时通过现有 `CargoItemDrawer` 及货物权限直接录入。
- 货物保存只提示更新，不自动覆盖已有分配草稿；显式刷新基线前确认，确认后才重置分配与预览。外部变动检查复用 `usePageForeground` 的前台恢复机制，15秒及回焦重查拆票上下文（不加载主数据），覆盖订单版本未变的货物/箱/费用版本变化；隐藏或后台页签暂停，变化持续提示且保留提交基线。

## 财务列表采纳快照

六类财务台账使用官方 request/onLoad 的同一采纳结果更新表格、顶部、底栏与当前页导出。
请求完成时不得先更新外部指标；迟到、取消和失败结果不能推进任何汇总。
完整契约与选择范围见 [财务列表快照与汇总](./finance-ledger-snapshots.md)。
