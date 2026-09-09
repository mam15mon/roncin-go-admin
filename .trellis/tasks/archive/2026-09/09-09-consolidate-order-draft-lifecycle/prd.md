# 收拢订单模板草稿生命周期 PRD

## 1. 目标与用户价值

订单表单的草稿键、脏状态、持久化、清理和关闭守卫应由
`OrderFormTemplate` 在一个稳定业务身份内统一管理。订单详情页只提供当前业务身份、
提交函数和显式重置命令，不再重复计算存储键或受控维护 dirty。

本任务完成后，应减少详情页草稿样板代码，消除父子 effect 争抢 dirty 的竞态，确保保存
成功、显式刷新、重置、只读恢复和详情 A → B 切换仍保持可验证的一致语义。

## 2. 已确认事实

- 当前只有 `new.tsx` 和 `detail.tsx` 两个产品调用方使用 `OrderFormTemplate`，两者均已显式
  传入稳定 `tabKey`；只有详情页使用受控 `dirty + onDirtyChange`。
- 模板当前用 `window.location.pathname` 计算草稿键，详情页另用
  `config.kind + orderId` 拼接路径计算同一草稿键，确有双重来源；但
  `window.location.pathname` 不包含 query 或 hash，当前隐患属于所有权和未来路径规范漂移，
  不是已证实的 query/hash 缺陷。
- `useOrderDetailData.loadData()` 与 `useOrderLockState.refresh()` 当前都会在 Hook 内吸收普通
  请求异常并更新错误状态，不会因普通超时直接 reject；详情页与模板的保存成功双重清草稿
  在当前契约下属于重复职责。
- `initialValues` 不是持续受控值；用 `useMemo` 计算
  `{ ...initialValues, ...draft }` 不能可靠覆盖 readonly 解除后的恢复，而且没有定义嵌套
  对象、数组和 `undefined` 的合并语义。
- 用户/组织切换继续由 `OrganizationWorkspace` key 卸载工作区，详情业务类型/记录切换继续
  由 `OrderFormTemplate key={orderFormIdentity}` 卸载模板。
- 项目正在快速上线阶段，本任务不保留旧的模板草稿 API 兼容层，不迁移或清理历史
  `sessionStorage` 数据。

## 3. 包含范围

### R1：显式草稿身份输入与单一键所有者

- `OrderFormTemplate` 新增规范化的 `draftPathname` 输入；当启用持久草稿时，调用方必须显式
  提供 `tabKey + draftPathname + draftScope`。
- 模板不再读取 `window.location.pathname`，不再自动调用 `resolveTabKey` 猜测草稿身份；
  缺少任一草稿身份输入时不读写持久草稿。
- 完整 `draftKey` 只允许在模板中通过 `getFormDraftKey` 计算；详情页删除自己的 `draftKey`
  计算和直接 `clearFormDraft` 调用。
- 新建页传入 `/orders/${config.kind}/new`，详情页传入
  `/orders/${config.kind}/${orderId}`；两者继续使用现有稳定菜单 `tabKey`。

### R2：模板独占 dirty 与草稿清理

- 删除 `OrderFormTemplateProps` 的受控 `dirty` 和 `onDirtyChange`；模板只维护
  `internalDirty`，并用它注册 `useTabCloseGuard`。
- 模板新增独立动作接口 `OrderFormTemplateActions<T>`，最小只暴露
  `resetTo(values?: Partial<T>): void`；不得向第三方 `ProFormInstance` 动态挂自定义方法。
- `resetTo` 必须在模板内部按固定顺序完成：清当前草稿、重置 Form store、可选回填调用方
  指定值、将 internal dirty 置为 false。
- 详情页的显式刷新成功与底部“重置修改”只调用 `actionsRef.resetTo(initialValues)`；详情页
  删除 `formDirtyState`、dirty 身份 ref、dirty setter 及相应 props。
- 原生“重置表单”和提交成功继续由模板清当前草稿并重置 internal dirty。

### R3：保存完成边界与后台刷新解耦

- `handleSaveEdit` 的成功/失败只由订单更新接口决定：更新失败返回 `false` 并保留草稿；
  更新成功提示成功并返回 `true`，由模板统一清草稿和 dirty。
- 更新成功后的 `loadData()` 与 `refreshLockState()` 作为 best-effort 后台刷新启动，不阻塞
  `onFinish` 的成功返回，也不得把已落库的保存改判为失败。
- 两个 Hook 当前已经呈现普通请求错误，禁止再无条件重复提示；仅对未来意外泄漏的 reject
  兜底提示“订单已保存，但最新数据刷新失败，请手动刷新”。
- 不引入通用任务队列、重试器或新的全局状态。

### R4：首次可编辑恢复且无视觉跳闪

- 草稿恢复从普通 effect 改为浏览器绘制前执行的 layout effect；不得通过渲染期浅合并
  `initialValues` 与 draft 实现。
- 一个模板身份只在第一次同时满足“非 loading、非 readonly、存在完整 draft identity”时
  尝试恢复一次；初始 loading 或 readonly 时继续等待首次合法时机。
- 初次或始终只读的详情不得读取并展示持久草稿；同订单后续锁状态同步不得再次恢复草稿并
  覆盖当前 Form store。
- 有效草稿恢复后必须把 internal dirty 标为 true，使页签和组织切换关闭守卫立即生效。

### R5：显式刷新身份隔离

- 显式刷新标记必须记录发起时的 `orderFormIdentity`，不能只使用无身份布尔值。
- A 的显式刷新若在导航到 B 后才结束，不得清 B 草稿、回填 A 或 B 的初始值，也不得重置
  B 的 dirty；只有请求身份仍等于当前身份且刷新后取得当前 order 时，才调用当前模板的
  `resetTo(initialValues)`。
- 后台锁状态同步仍不得被当成显式刷新，不触发 Form store 重置。

### R6：规范与测试同步

- 更新 `state-management.md`：明确页面提供规范草稿身份，模板独占草稿键和生命周期；页面
  外部动作只能通过模板动作接口清理草稿与 dirty。
- 更新组件 props、模板测试、详情生命周期测试和新建页测试；删除只服务于旧受控 dirty API
  的断言，不得降低行为覆盖。

## 4. 验收标准

- [x] **AC-1**：产品代码中 `getFormDraftKey` 的订单模板运行时调用只存在于
  `OrderFormTemplate`；模板不读取 `window.location`，新建页和详情页均传入规范化
  `draftPathname`。
- [x] **AC-2**：详情页不再包含 `formDirtyState`、`activeOrderFormIdentityRef`、
  `setIsFormDirty`、受控 `dirty/onDirtyChange` 或直接清当前草稿的代码；模板内部 dirty
  继续驱动关闭守卫。
- [x] **AC-3**：底部“重置修改”和显式刷新成功通过 `actionsRef.resetTo` 清当前草稿、回填
  最新服务端初始值并清 dirty；显式刷新失败保留当前值、草稿和 dirty。
- [x] **AC-4**：订单更新失败返回 `false` 且保留草稿；更新成功后模板只清一次当前草稿，
  后台数据或锁状态刷新失败不改变保存成功结果。
- [x] **AC-5**：A dirty 后原地导航到 B，B 不显示 A 的 Form store/dirty；B 有自己的草稿时
  在首次绘制前恢复 B 草稿并注册 dirty；A 的迟到显式刷新不能重置 B。
- [x] **AC-6**：初次或始终 readonly 时不恢复持久草稿；首次解除 readonly 后只恢复一次；
  同订单后续锁状态同步不覆盖已编辑的内存值。
- [x] **AC-7**：原生重置、页签关闭确认、用户/组织切换隔离、新建订单草稿恢复和日期复原
  语义无回归。
- [x] **AC-8**：相关定向测试、修改文件 Biome、`pnpm --dir web tsc`、`git diff --check`
  通过；任务最终只运行一次 `pnpm run check:web` 且通过。
- [x] **AC-9**：实现、测试与 `.trellis/spec/web/frontend/state-management.md` 描述同一所有权
  边界，不存在为了旧 props 或历史 sessionStorage 数据增加的兼容分支。

## 5. 不包含范围

- 不修改 `formDraft.ts` 的存储前缀、序列化格式、日期复原规则或异常吞吐策略；
- 不扫描、迁移或清空历史 `sessionStorage` 草稿；
- 不重构 `tabCloseGuard.ts`、`OrganizationWorkspace` 或 `TagsView`；
- 不修改后端、权限、OpenAPI 契约、数据库或生成物；
- 不新增空运、海运进口等尚未生效的订单类型；
- 不顺带处理详情页其他 loading 状态、所有异步请求或通用竞态 Hook；
- 不引入草稿深合并规则、自动重试、静默回退或旧 API 兼容层；
- 不运行生产构建，除非实施期间实际触及构建配置或用户另行要求。

## 6. 风险与约束

- `resetTo` 同时改变 Form store、草稿和 dirty，属于公共模板动作，必须用模板级和详情级
  行为测试共同锁定，不能只断言函数被调用。
- `useLayoutEffect` 必须在 ProForm ref 可用后执行；实现若发现 ref 时序与假设不符，应先用
  定向测试验证，再选择“首帧暂缓挂载”方案，不得回退到未经定义的浅合并。
- 保存后的后台刷新会晚于提交成功结束；现有 Hook 的订单/组织/请求序号保护必须保留，
  不得在本任务中删除。
