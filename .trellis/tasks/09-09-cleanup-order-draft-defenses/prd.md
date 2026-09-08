# 清理订单草稿冗余防御代码 PRD

## 1. 背景与问题

在任务 `09-08-multi-org-foundation-architecture` 完成后，系统已建立全局统一的
`<OrganizationWorkspace key={`${userId}:${organizationId}`}>` 容器：

- 当用户切换组织成功后，旧工作区的 `TagsView` 与业务页面会作为整体被 React 卸载销毁；
- 路由统一 `history.replace('/welcome')`，新组织工作区挂载全新的空白初始状态；
- 在当前 `app.tsx` 挂载结构下，工作区内组件不会在存活状态下经历用户或
  `currentOrganization` 变化；该结论由 `app.test.tsx` 的挂载/卸载测试约束。

在此之前，为了防止组织切换时旧组织的输入泄漏到新组织，订单模块编写了部分防御性
胶水代码：

1. `web/src/pages/orders/detail.tsx` 使用
   `useEffect(() => { setIsFormDirty(false); }, [draftScope, orderId]);` 监听 `draftScope`；
2. `web/src/components/ui/order-template/OrderFormTemplate.tsx` 维护
   `previousDraftKeyRef`，在 effect 中比对 `draftContextChanged` 并强制清脏状态；
3. `OrderFormTemplate.tsx` 给 `<ProForm key={draftKey}>` 绑定动态 key，制造第二层重挂载；
4. `detail.tsx` 同时让模板和页面 `onReset` 回调清理同一个草稿，原生重置路径存在重复职责。

其中只有与工作区/表单身份重建等价的保护可以删除。保存成功、显式刷新、底部
“重置修改”、只读拦截和草稿命名空间仍承担独立业务语义，不能机械收缩。

## 2. 目标

- **G1（清理组织级防御样板）**：删除纯用于防止组织切换污染的组件内部监听、比对和
  二次重载代码。
- **G2（保留业务级核心草稿能力）**：完整保留同组织内的编辑自动保存、首次进入回填、
  成功提交/显式刷新/重置时清理，以及多页签关闭守卫 `useTabCloseGuard`。
- **G3（保留只读态防线）**：完整保留 `effectiveReadonly`，无编辑权限或业务写入关闭时
  不得从持久草稿回填，也不得保存草稿。
- **G4（零业务回归）**：受影响的草稿、只读、记录切换、多页签和工作区生命周期测试
  必须通过；任务最终验收运行一次 `check:web`，不以删除断言代替行为证明。

## 3. 需求范围

### 3.1 包含范围

- `web/src/pages/orders/detail.tsx`：删除对 `draftScope` 的无用监听，只删除已确认重复的
  原生重置清理，保留保存成功、显式刷新和底部“重置修改”的必要清理时序；
- `web/src/components/ui/order-template/OrderFormTemplate.tsx`：删除
  `previousDraftKeyRef`、`draftContextChanged` 与内部 `<ProForm key={draftKey}>`；
- `web/src/pages/orders/detail.tsx`：为 `OrderFormTemplate` 补充
  `key={`${config.kind}:${orderId}`}`，把“业务类型 + 记录”身份在挂载期内不变从隐式前提
  变成结构保证；
- `.trellis/spec/web/frontend/state-management.md`：将草稿身份切换契约改为工作区和调用方
  身份 key 共同保证，删除“同一个模板实例切换 `draftScope`”的过时测试要求；
- 更新相应单元测试，并执行类型与前端门禁验证。

### 3.2 不包含范围

- 不删除或重构 `formDraft.ts`；
- 不重做 `tabCloseGuard.ts`；
- 不新增当前尚未支持的空运、海运进口等订单类型，也不因泛化路由提前改造 `new.tsx`；
- 不改动后端接口、权限或数据模型；
- 不清理订单模块中与本任务不等价的异步竞态、业务资源身份或缓存保护。

## 4. 详细需求

### R1：详情记录身份与 effect 精简

- 移除详情 effect 对 `draftScope` 的依赖；
- 以 `${config.kind}:${orderId}` 作为详情表单记录身份；业务类型或记录变化时重置脏标记、
  关闭共享箱工作台并清理旧运输执行上下文；
- 将同一身份用于 `OrderFormTemplate` 的 React `key`，确保详情 A 原地导航到详情 B 时
  Form store 和模板本地 dirty 状态整体重建。

### R2：`OrderFormTemplate.tsx` 重复防线精简

- 移除 `previousDraftKeyRef` 和 `draftContextChanged` 探测逻辑；
- 移除内部 `<ProForm>` 的 `key={draftKey}`，避免模板和表单两层重复重挂载；
- 保持草稿恢复逻辑：仅在非只读、有 `draftKey`、非加载中且存在有效草稿时回填；
- `OrderFormTemplate` 将 `(draftScope, resolvedTabKey, pathname)` 视为挂载期内恒定；调用方
  若允许这些身份在同一页面实例内变化，必须在模板边界提供覆盖完整资源身份的 React key。

### R3：草稿清理与只读安全

- `effectiveReadonly` 的“无编辑动作权限或业务写入关闭”完整判定保持不变；
- 初次进入或重新挂载只读详情时禁止从 `sessionStorage` 读取或合并草稿；同一订单后台
  锁状态刷新不得覆盖用户已经开始编辑的内存表单值；
- 模板继续负责原生重置与 `onFinish` 返回成功后的草稿清理，详情页删除重复的
  `onReset` 清理回调；
- 详情页必须保留“订单更新接口已成功”后的立即清理：即使后续 `loadData` 或锁状态刷新
  失败，也不得留下会在下次进入时覆盖已保存服务端数据的旧草稿；
- 详情页继续在显式刷新成功和底部“重置修改”时清理当前草稿；更新接口失败或显式刷新
  失败时保留草稿；
- 所有清理只作用于当前 `draftKey`，不得清理其他页签、用户或组织的草稿。

### R4：规范与不变量同步

- 在 design、组件注释和 `state-management.md` 中记录双层身份边界：用户/组织切换由
  `OrganizationWorkspace` 卸载，详情业务类型/记录切换由页面传给模板的 key 卸载；
- 将“模板在同一组件实例切换 `draftScope`”改为“以新 identity key 挂载新模板实例，先用
  新身份 `initialValues`，再恢复新身份草稿”；
- 保留草稿键必须包含用户、组织、页签与 pathname 的原有契约。

### R5：当前订单类型事实与未来扩展约束

- 当前 `ORDER_KIND_CONFIGS` 只支持 `sea-export`；通用路由 `/orders/:kind/new` 不代表空运
  新建页已经可用，验收不得写成海运/空运均可创建草稿；
- 当前 `NewOrderPage` 的有效挂载期内 `kind` 恒为 `sea-export`，组织变化又会卸载工作区，
  本任务无需修改 `new.tsx`；
- 未来新增第二种有效订单类型时，新增类型任务必须为新建表单建立业务类型级父组件 key，
  或以测试证明路由会卸载旧模板；不得重新把身份切换防御塞回通用模板。

## 5. 验收标准

- [ ] **AC-1**：同分公司内新建当前已支持的海运出口订单，输入内容未保存时切换到其他
  页签再切回，草稿仍能正常恢复。
- [ ] **AC-2**：初次进入或重新挂载只读详情（无编辑权限或业务写入关闭）时，表单展示
  服务端原始数据，不应用持久草稿；同订单后台锁状态刷新不覆盖已开始编辑的内存值。
- [ ] **AC-3**：订单详情在点击底部“重置修改”、原生“重置表单”、显式刷新成功或订单
  更新接口成功后，当前草稿被正确销毁；更新接口失败或显式刷新失败时草稿保留。
- [ ] **AC-4**：组织切换时，页面与页签由 `OrganizationWorkspace` 整体卸载重挂载，
  新组织不残留旧输入。
- [ ] **AC-5**：详情 A 原地导航到详情 B 时，模板组件重挂载，B 展示自己的服务端初始值
  或草稿，不保留 A 的 Form store 与 dirty 状态。
- [ ] **AC-6**：`detail.tsx` 与 `OrderFormTemplate.tsx` 中无残留的
  `previousDraftKeyRef`、内部 `key={draftKey}` 或多余 `draftScope` effect；详情页不再向模板
  传入重复清草稿的 `onReset`。
- [ ] **AC-7**：受影响的定向测试、修改文件 Biome、TypeScript 和最终一次
  `pnpm run check:web` 全绿。
- [ ] **AC-8（规范与不变量）**：`state-management.md` 草稿契约已改写为工作区/调用方
  身份 key 机制，无组件内切换 `draftScope` 的过时条款；详情业务类型 + 记录 key 与
  `OrderFormTemplate` 草稿恢复处“draftKey 挂载期恒定”的注释就位。
