# 清理订单草稿冗余防御代码 PRD

## 1. 背景与问题

在任务 `09-08-multi-org-foundation-architecture` 完成后，系统已建立全局统一的 `<OrganizationWorkspace key={`${userId}:${organizationId}`}>` 容器：
- 当用户切换组织成功后，旧工作区的 `TagsView` 与业务页面会作为整体被 React 卸载销毁；
- 路由统一 `history.replace('/welcome')`，新组织工作区挂载的是全新的空白初始状态；
- 因此，组件不会在存活状态下经历 `currentOrganization` 变化。

在此之前，为了防止组织切换时旧组织的输入泄漏到新组织，在订单模块中编写了部分防御性胶水代码：
1. `web/src/pages/orders/detail.tsx` 中使用 `useEffect(() => { setIsFormDirty(false); }, [draftScope, orderId]);` 监听 `draftScope`；
2. `web/src/components/ui/order-template/OrderFormTemplate.tsx` 中维护 `previousDraftKeyRef`，在 `useEffect` 中比对 `draftContextChanged` 并在变化时强制清脏状态；
3. `OrderFormTemplate.tsx` 中给 `<ProForm key={draftKey}>` 绑定了动态 key 制造二次强制重挂载。

这些代码在工作区物理卸载机制建立后，已经变成了多余且容易产生心智负担的重复防御样板，需要做精准收敛与清理。

## 2. 目标

- **G1（清理组织级防御样板）**：删除纯用于“防组织切换污染”的组件内部监听、比对及二次重载代码。
- **G2（保留业务级核心草稿能力）**：完整保留同组织内的正常草稿防手抖能力（编辑自动保存、首次进入回填、提交成功/显式重置/放弃修改时清理）与多页签关闭守卫（`useTabCloseGuard`）。
- **G3（保留只读态防线）**：完整保留 `effectiveReadonly` 逻辑，在无编辑权限或业务写入关闭时绝对不回填草稿、不保存草稿。
- **G4（零业务回归）**：保持前端测试 100% 通过，不引入任何多页签或表单交互回归。

## 3. 需求范围

### 3.1 包含范围
- `web/src/pages/orders/detail.tsx`：删除对 `draftScope` 的无用监听，收敛重复的手动清理逻辑；
- `web/src/components/ui/order-template/OrderFormTemplate.tsx`：删除 `previousDraftKeyRef` 与 `draftContextChanged` 比对，移除 `<ProForm key={draftKey}>` 上的多余动态 key；
- `web/src/pages/orders/detail.tsx`：为 `OrderFormTemplate` 补充记录级 `key={orderId}`，把「表单身份在挂载期内不变」从隐式前提变成结构保证；
- 同步更新 `.trellis/spec/web/frontend/state-management.md` 的草稿契约（身份重挂载机制由组件级上移为工作区级，删除「同一组件实例切换 `draftScope`」的过时测试要求）；
- 相应单元测试与类型校验更新。

### 3.2 不包含范围
- 不删除草稿功能本身（`formDraft.ts` 继续保留）；
- 不重做多页签关闭守卫（`tabCloseGuard.ts` 继续保留）；
- 不改动后端接口或数据模型。

## 4. 详细需求

### R1：`detail.tsx` 依赖精简
- 移除 `useEffect([draftScope, orderId])` 对 `draftScope` 的依赖；
- 仅在资源切换（`orderId` 变化）时重置脏标记；
- 保持保存、刷新、重置、撤销时的草稿清理时机。

### R2：`OrderFormTemplate.tsx` 重复防线精简
- 移除 `previousDraftKeyRef` 探测逻辑；
- 移除 `<ProForm>` 的 `key={draftKey}` 属性，避免表单无谓二次挂载；
- 挂载时的草稿回填逻辑保持：仅在非只读、有 `draftKey`、非加载中且表单未填充时安全回填。

### R3：只读与业务安全保持
- `effectiveReadonly` 判定逻辑完整保留；
- 只读态下严格禁止从 `sessionStorage` 读取或合并草稿；
- 用户提交成功后仅清理当前记录对应的草稿。

### R4：规范与不变量同步
- 「draftKey 在组件挂载期内不变」的前提写入 design 与组件注释：组织切换由 `OrganizationWorkspace` 整体卸载兜底，记录切换由路由重挂载与 `detail.tsx` 的 `key={orderId}` 兜底；
- `.trellis/spec/web/frontend/state-management.md` 中「身份命名空间变化时表单必须重新挂载」与「模板测试：在同一个组件实例切换 draftScope」两条契约改写为工作区级机制描述，避免规范与实现相反。

## 5. 验收标准

- [ ] **AC-1**：同分公司内新建海运/空运订单输入内容未保存，切换到其他页签再切回，草稿仍能正常恢复。
- [ ] **AC-2**：订单详情在只读（无编辑权限或加锁关闭写入）时，表单展示服务端原始数据，不应用脏草稿。
- [ ] **AC-3**：订单详情在点击“放弃修改”、表单“重置”或“保存成功”后，本地草稿被正确销毁。
- [ ] **AC-4**：组织切换时，页面与页签由 `<OrganizationWorkspace>` 整体卸载重挂载，新组织绝对不残留旧输入。
- [ ] **AC-5**：`detail.tsx` 与 `OrderFormTemplate.tsx` 中无残留的 `previousDraftKeyRef`、`key={draftKey}` 或多余 `draftScope` effect。
- [ ] **AC-6**：运行定向单测与类型检查全绿，无 Biome 报错。
- [ ] **AC-7（规范与不变量）**：`state-management.md` 草稿契约已改写为工作区级机制，无组件内切换 `draftScope` 的过时条款；`detail.tsx` 的 `key={orderId}` 与 `OrderFormTemplate` 草稿恢复处「draftKey 挂载期恒定」的注释就位。
