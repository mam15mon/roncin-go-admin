# 状态管理

## 服务端状态（React Query v5，2026-09 起唯一模式）

- 服务端数据一律用 `@tanstack/react-query`（直接 `import { useQuery, useMutation,
  useQueryClient } from '@tanstack/react-query'`，生产与 vitest 同源）；**禁止**
  新增手写 `useState` + `useEffect` + 竞态令牌（sequenceRef/cancelled/refreshToken）
  请求链，也禁止使用已废弃的 Umi `useRequest`（底层为 2021 年
  `@ahooksjs/use-request` v2，vitest 不可用、无卸载保护，选型记录见任务
  `09-20-data-layer-direction`）。
- 单例在 `src/utils/queryClient.ts`，Provider 挂 `app.tsx` childrenRender。
  默认策略：`retry: false`、`refetchOnWindowFocus: false`、
  `refetchOnReconnect: false`；不得在调用点覆盖 retry。
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
  携带实体的写接口还应验证必要 `data.id` 且 `success !== false`。各接口按真实契约判定，
  禁止 `await api()` 后无条件执行成功逻辑。`ModalForm.onFinish` 失败返回 `false`；
  订单已成功落库后的后台刷新继续 best-effort，不把刷新失败改判为保存失败。
- 开发期走 Vite 代理，生产同域（Go 服务同时提供 `/api/*` 与静态资源）；
  修改打包或路由时两种路径都要验证。

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

## 场景：跨页签表单草稿

### 1. 适用范围

- 适用于 `OrderFormTemplate` 等在 `sessionStorage` 暂存未提交表单、并由
  `TagsView` 在后台页签关闭时继续判断脏状态的场景。
- 该存储跨组件挂载存在，必须视为用户与组织级业务数据，不能只按路由命名。

### 2. 签名

```ts
getFormDraftScope(userId?: string, organizationId?: string): string | undefined;
// pathname 必传：tabKey、pathname、draftScope 任一缺失时返回空串，
// 不再回退 window.location，也不生成有效草稿键。
getFormDraftKey(tabKey?: string, pathname?: string, draftScope?: string): string;
hasTabDraft(tabKey: string, draftScope?: string): boolean;
clearTabDrafts(tabKey: string, draftScope?: string): void;

// OrderFormTemplate 动作接口：页面外部清理草稿与脏状态只能经由该接口。
interface OrderFormTemplateActions<T> {
  resetTo: (values?: Partial<T>) => void;
}
```

### 3. 契约

- 草稿键必须同时包含用户 ID、当前组织 ID、稳定页签 key 与完整 pathname；缺少用户或组织时
  禁止持久化草稿。
- 草稿身份由页面显式提供：`OrderFormTemplate` 只接收调用方传入的
  `tabKey + draftPathname + draftScope`（如 `/orders/:kind/new`、`/orders/:kind/:id`），
  不读取 `window.location`，也不从当前路由猜测 tabKey；任一身份输入缺失时不生成草稿键、
  不读写持久草稿。
- 完整 `draftKey` 只允许在 `OrderFormTemplate` 内通过 `getFormDraftKey` 计算；页面不得
  自行计算草稿键或直接调用草稿清理函数。
- 脏状态与草稿生命周期由模板独占：模板维护 `internalDirty` 并注册 `useTabCloseGuard`，
  不再提供受控 `dirty/onDirtyChange` props。页面级显式刷新成功、底部「重置修改」等外部
  动作只能调用 `actionsRef.resetTo(values)`（清当前草稿 → 重置 Form store → 回填最新值 →
  清 dirty）。
- 草稿恢复由模板在首次同时满足「非 loading、非 readonly、草稿身份完整」时通过浏览器绘制
  前的 layout effect 执行一次；初始只读或加载中的页面不读取草稿，首次变为可编辑时仍可恢复，
  同身份后续锁状态往返不得覆盖内存中的表单值。
- 身份重建由双层边界保证：用户与组织身份由 `OrganizationWorkspace` key 管理（切换时整体卸载
  页签与页面），页面内资源身份（如业务类型或记录 ID）由调用方在模板边界提供对应 React `key`
  （如 `key={`${config.kind}:${orderId}`}`）。
- 模板挂载期间 `(draftScope, tabKey, draftPathname)` 必须保持同一身份，模板内部不再进行
  多重身份探测；只要页面内资源身份可能变化，调用方就必须在模板边界提供对应 key。
- 新 identity key 挂载新表单实例时，先使用新身份的 `initialValues`，再恢复新身份自己的
  草稿；不得保留旧组织的 Form store 与 dirty 状态。
- 页签脏状态取“实时 guard 或当前身份持久草稿”，防止 React 状态尚未提交时漏掉已同步写入
  的草稿。
- 用户确认关闭后，只清除当前身份、目标页签下的草稿；其他用户或组织的草稿保持不变。
- 订单保存的成功/失败只由订单更新接口决定；更新成功由模板统一清草稿与 dirty，详情与锁状态
  的后台刷新是 best-effort 任务，不得把已落库的保存改判为失败。显式刷新以统一单调令牌
  门禁：发起时递增令牌；订单身份发生变化时在 `useLayoutEffect` 的 setup 与 cleanup
  中均递增同一令牌并清空 pending（不需要额外的 previousRef，亦不在 render 阶段修改 ref），
  使旧身份的全部在途刷新立即失效（覆盖 A→B 与 A→B→回 A 往返）。pending 同时携带令牌与订单身份，
  请求完成与消费 effect 两端都必须复核，覆盖「完成写入 pending 后、消费前令牌或身份已变化」的
  窗口。字符串身份只能区分订单、区分不了同一订单的实例代际（A→B→回 A 往返）；只有最后一次发起、
  且从未跨越身份切换的刷新完成才允许回填，A 的迟到刷新不得清 B 的草稿或重置 B 的表单，旧实例的
  迟到刷新不得覆盖重新挂载实例中的内容。
- 显式刷新令牌只属于通用订单详情页编排层；订单类型详情扩展（见
  component-guidelines.md 的「订单类型三类真相边界」）不得读写该令牌或模板
  `resetTo`，其业务写成功后的刷新只走页面提供的 `refreshOrderAndLock` 普通命令。
- 日期反序列化覆盖订单表单使用的 `*Date`、`*Cutoff`、`*At` 以及 `etd`、`eta` 等字段，普通
  ISO 格式文本不得仅凭值形态被转换。

### 4. 校验与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 用户 ID 或组织 ID 缺失 | 不生成草稿键，不读写草稿；实时关闭 guard 仍工作 |
| 同用户切换组织 | 重建表单，只加载新组织命名空间 |
| 同组织切换用户 | 不读取原用户草稿 |
| 存储不可用、超限或 JSON 非法 | 捕获存储异常；读取返回 `null`，不阻断表单 |
| 实时 guard 为 false、当前身份已有草稿 | 仍判定为 dirty |
| 用户取消关闭 | 保留页签与草稿 |
| 用户确认关闭 | 仅清理当前身份与目标页签草稿 |

### 5. Good / Base / Bad

- Good：用户 A 在组织 1 输入后切到组织 2，页面展示组织 2 默认值或其自身草稿；组织 1 草稿
  留存且不会显示。
- Base：没有草稿时按当前表单 `initialValues` 渲染，关闭页签不提示。
- Bad：使用 `roncin:form-draft:${tabKey}:${pathname}`，导致不同用户或组织共享同一个键。

### 6. 必需测试

- 工具测试：同页签、同路径的两个身份命名空间可以独立保存、判断和清理。
- 模板测试：缺少任一草稿身份输入（tabKey / draftPathname / draftScope）时不持久化，完整
  输入只命中唯一规范路径键；只读挂载不恢复草稿、初始只读在首次解除后只恢复一次且后续
  readonly 往返不覆盖内存编辑值；提交成功清草稿、提交失败保留草稿、原生重置清草稿；
  `resetTo` 清当前草稿、清旧 Form store、回填新值并清 internal dirty。
- 详情生命周期测试：详情 A 原地导航到 B 后模板实例被重建，B 不显示 A 的 Form store 与
  dirty 状态；A 的迟到显式刷新不得重置 B 的表单与 dirty；保存成功立即清草稿，后台刷新
  失败不改判保存结果；底部重置修改与显式刷新成功经由 `actionsRef.resetTo` 生效。
- 新建页测试：页面向模板传入规范 `tabKey + draftPathname + draftScope`，并从规范路径键
  恢复新建草稿。
- 关闭保护测试：实时 guard 暂为 false 但草稿已写入时仍提示；确认后仅清理当前 scope。
- 日期测试：嵌套对象中的 `cargoReadyAt`、截止时间及 `etd`/`eta` 恢复为 Dayjs，普通文本保持
  字符串。

### 7. 错误与正确示例

```ts
// 错误：组织切换后仍命中同一个草稿。
getFormDraftKey(tabKey, pathname);

// 错误：页面自行计算草稿键并直接清理，绕过模板生命周期所有权。
const draftKey = getFormDraftKey(tabKey, pathname, draftScope);
clearFormDraft(draftKey);

// 正确：页面只提供规范身份，模板独占草稿键与清理动作。
<OrderFormTemplate
  tabKey={resolveTabKey(`/orders/${kind}/${orderId}`)}
  draftPathname={`/orders/${kind}/${orderId}`}
  draftScope={getFormDraftScope(user.id, currentOrganization.id)}
  actionsRef={templateActionsRef}
/>;

// 需要外部重置表单时，经由模板动作接口执行。
templateActionsRef.current?.resetTo(initialValues);
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
  查询仍提交原始 ID。财务顶部所属公司、标签及标签名均属于查询条件，切所属公司清旧标签；
  顶部目标公司不是工作区身份。客商黑名单同样为真实查询，不丢 false/0 值。
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
客商角色、财务顶部公司与标签、日期/候选名称；所有关闭菜单确认/取消；
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

- 适用于同一页面中的服务端分页列表与全量导出共用搜索条件的场景。
- 导出接口必须使用 OpenAPI 生成客户端；页面只负责筛选规范化、CSV 序列化与
  浏览器下载，不得自行拼接后端地址或重新推导业务金额。

### 2. 数据签名

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
- CSV 使用 UTF-8 BOM 与 CRLF 行分隔，包含逗号、双引号、回车或换行的单元格按
  RFC 4180 形式加引号并把双引号写成两个双引号。

### 4. 校验与错误矩阵

| 输入或结果 | 处理方式 |
| --- | --- |
| 关键字只有首尾空格 | 去除首尾空格；结果为空时不发送该筛选字段 |
| 仅选择开始月份 | 只发送开始月首日 |
| 仅选择结束月份 | 只发送结束月末日 |
| 文本字段以 `= + - @` 开头，或前导空白后出现这些字符 | 在原值前加单引号，阻止表格软件执行公式 |
| 后端控制的日期、时间、decimal 字段 | 保持后端字符串原值，不做公式保护，避免改写合法负数 |
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

// 正确：仅自由文本字段保护公式；受控日期和 decimal 保持后端原值。
escapeCsvCell(column.kind === 'text' ? protectFormula(value) : value);
```

## 异步业务核对与表单校验同步

- 校验器依赖外部异步状态时，React 状态变化不会自动清除 Form 已保存的错误。核对结束、失败或关联确认变化后，应对相关字段执行完整 `validateFields`；可使用 `{ dirty: true }` 避免未交互字段提前显示必填错误。禁止直接清空错误而掩盖格式或业务冲突。
- 通过 `setFieldValue` 保存但没有 Form.Item 注册的业务字段，使用 `Form.useWatch(name, { form, preserve: true })` 监听；否则自动关联 ID、航次 ID 等可能无法回显。
- 回归测试必须使用真实字段注册结构，不能额外添加生产中不存在的隐藏 Form.Item 来让监听生效。范本：`SeaMasterBillFields.validation.test.tsx`。
