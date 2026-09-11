# 清理订单草稿冗余防御代码实施计划

## 1. 实施前提

- 本任务是前端公共订单模板与详情页的精简重构，不改后端、草稿存储格式或数据库；
- 当前唯一有效订单类型为 `sea-export`，不得把通用 `:kind` 路由误判成空运已上线；
- 只删除能被上层挂载身份严格替代的保护，不删除保存成功、显式刷新、底部重置、
  `effectiveReadonly`、缓存分键或多页签关闭守卫；
- 功能实现后按风险补测试，禁止先删测试再以缺少断言证明“无回归”。

## 2. 实施步骤

### Step 1：精简 `OrderFormTemplate`

目标文件：

- `web/src/components/ui/order-template/OrderFormTemplate.tsx`
- `web/src/components/ui/order-template/OrderFormTemplate.test.tsx`

实施内容：

1. 移除 `previousDraftKeyRef`；
2. 移除 `draftContextChanged` 比对与由它触发的 dirty 重置；
3. 移除内部 `<ProForm key={draftKey}>` 的 key；
4. 在草稿恢复 effect 旁注明 draft identity 在模板挂载期内恒定，身份变化由上层 key 负责；
5. 保留 loading/readonly 变化时重新判断草稿恢复的现有 effect 语义；
6. 将“同一模板实例切换 `draftScope`”测试替换成以下真实职责测试：
   - 只读挂载不读取持久草稿，展示服务端 initialValues；
   - 成功提交清当前草稿，提交返回 `false` 时保留；
   - 原生重置清当前草稿；
   - 重新挂载恢复当前身份草稿并注册 dirty。

定向验证：

```bash
pnpm --dir web exec vitest run src/components/ui/order-template/OrderFormTemplate.test.tsx
pnpm --dir web exec biome lint src/components/ui/order-template/OrderFormTemplate.tsx src/components/ui/order-template/OrderFormTemplate.test.tsx
```

### Step 2：建立详情记录身份边界并删除唯一重复清理

目标文件：

- `web/src/pages/orders/detail.tsx`
- `web/src/pages/orders/detail-change-actions.test.tsx`
- `web/src/pages/orders/detail-draft-lifecycle.test.tsx`（新增，避免把草稿职责继续塞进拆票/改配测试）

实施内容：

1. 定义 `orderFormIdentity = config && orderId ? `${config.kind}:${orderId}` : undefined`；
2. 资源切换 effect 从 `[draftScope, orderId]` 改为 `[orderFormIdentity]`；
3. `OrderFormTemplate` 增加 `key={orderFormIdentity}`；
4. 删除详情页传给模板的重复 `onReset` 回调；
5. 保留并在测试中锁定三个页面级清理点：
   - 更新接口成功后、刷新详情前立即清草稿；
   - 显式刷新成功并取得当前 order 后清草稿；
   - 底部“重置修改”清草稿并回填 initialValues；
6. 不改 `pendingExplicitFormRefreshRef` 时序，不把后台锁状态同步视为显式刷新；
7. 不创建清理 helper 合并上述不同生命周期。

定向测试必须证明：

- 详情 A 原地导航到 B 后模板实例被重建，B 不显示 A 的 Form store/dirty；测试必须记录
  mock/真实模板的 mount、unmount 或实例 ID 变化，不能只检查 B 最终字段值，因为 React
  `key` 不会作为普通 prop 传给 mock；
- 更新接口失败时保留草稿；
- 更新接口成功后，即使后续 `loadData` 或锁状态刷新失败也已清草稿；
- 显式刷新失败保留草稿，成功才清理并回填；
- 底部重置只清当前 `draftKey`；
- 缺少编辑权限或业务锁单时，模板与分节仍同时 readonly；
- 同订单后台锁状态同步仍不覆盖未保存内存值。

定向验证：

```bash
pnpm --dir web exec vitest run src/pages/orders/detail-change-actions.test.tsx src/pages/orders/detail-draft-lifecycle.test.tsx
pnpm --dir web exec biome lint src/pages/orders/detail.tsx src/pages/orders/detail-change-actions.test.tsx src/pages/orders/detail-draft-lifecycle.test.tsx
```

### Step 3：同步草稿规范

目标文件：

- `.trellis/spec/web/frontend/state-management.md`

实施内容：

1. 第 3 节明确身份重建的两层责任：用户/组织由 `OrganizationWorkspace` key 管理，页面内
   资源身份由调用方在模板边界提供 key；
2. 明确模板挂载期间 `(draftScope, resolvedTabKey, pathname)` 必须保持同一身份；
3. 第 6 节删除“同一模板实例切换 `draftScope`”要求，替换为新 identity key 触发新实例，
   新实例先使用自己的 initialValues，再恢复自己的草稿；
4. 保留用户、组织、页签、pathname 都必须进入草稿键的契约；
5. 补充通用调用方约束：只要页面内资源身份可能变化，就必须在模板边界提供对应 key。

### Step 4：定向回归

```bash
pnpm --dir web exec vitest run \
  src/components/ui/order-template/OrderFormTemplate.test.tsx \
  src/pages/orders/detail-change-actions.test.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx \
  src/components/layout/formDraft.test.ts \
  src/components/layout/TagsView.test.tsx \
  src/app.test.tsx
pnpm --dir web exec biome lint \
  src/components/ui/order-template/OrderFormTemplate.tsx \
  src/components/ui/order-template/OrderFormTemplate.test.tsx \
  src/pages/orders/detail.tsx \
  src/pages/orders/detail-change-actions.test.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx
pnpm --dir web tsc
git diff --check
```

### Step 5：独立审查与最终门禁

独立审查重点：

1. 删除项是否都能由 workspace 或页面 identity key 严格替代；
2. 是否误删保存成功但刷新失败时的提前清草稿；
3. readonly 初次挂载与同订单后台锁同步语义是否同时保留；
4. `new.tsx` 未修改的前提是否仍为“只有一个有效 kind”，而不是错误的 pathname 恒定；
5. 规范、测试与实现是否描述同一不变量。

任务完成时执行一次：

```bash
pnpm run check:web
git diff --check
git status --short
```

不要求 `pnpm run build`：本任务不改依赖、构建配置或生产入口。

## 3. 实施后独立复核

2026-09-09 在 agy 实施完成后进行了独立复核，并补齐以下问题：

1. 原 A → B 用例只证明模板实例重建，没有先制造 A 的脏状态，也没有断言 B 的
   服务端初始值与父级 dirty 状态；复核测试已补齐这条完整数据流；
2. 原显式刷新失败用例让 `loadData` 抛错，但真实 `useOrderDetailData` 会吸收请求错误、
   清空当前 `order` 后正常 resolve；复核测试已改为按真实 Hook 契约模拟；
3. 补充“B 存在自己的持久草稿”场景后发现真实竞态：B 的模板恢复草稿并上报
   `dirty=true` 后，父页面的资源切换 effect 会再次写入 `false`。详情页现将脏状态
   与 `orderFormIdentity` 绑定，旧订单的迟到异步回调也不能覆盖当前订单状态；资源
   切换 effect 不再执行无身份归属的 dirty 重置。

复核验证结果：

- 本任务 6 个定向测试文件共 68 个用例通过；
- 修改文件 Biome 检查通过；
- `pnpm --dir web tsc` 通过；
- `pnpm run check:web` 通过，96 个测试文件共 435 个用例通过；
- Biome 全量扫描仍打印既有 `.agents/skills/ant-design` 目录无读取权限的内部诊断，
  但未使门禁失败，本次未修改该无关目录。

## 4. 提交与回滚

建议把产品代码、测试和对应规范作为同一可验证组提交：

```text
refactor(web): 清理订单草稿冗余防御代码
```

本任务无数据迁移。若验收发现草稿或表单身份回归，整体回退该提交；不得清空用户
`sessionStorage`、恢复旧字段兼容或增加静默 fallback。
