# 收拢订单模板草稿生命周期实施计划

## 1. 实施前检查

1. 检查 `git status`，确认没有来源不明或与订单模板重叠的未提交改动；
2. 完整读取本任务 `prd.md`、`design.md`、本文件、根 `AGENTS.md` 及
   `implement.jsonl` 引用；
3. 只修改前端公共订单模板、订单新建/详情调用方、定向测试和状态管理规范；
4. 功能实现后补测试，禁止 TDD；开发中只跑定向检查。

## 2. Step 1：调整模板接口并收拢生命周期

目标文件：

- `web/src/components/ui/order-template/types.ts`
- `web/src/components/ui/order-template/OrderFormTemplate.tsx`
- `web/src/components/ui/order-template/OrderFormTemplate.test.tsx`

实施内容：

1. 新增 `OrderFormTemplateActions<T>`、`draftPathname` 和 `actionsRef`；
2. 删除 `dirty`、`onDirtyChange` 以及 `window.location` / `resolveTabKey` fallback；
3. 仅用显式 `tabKey + draftPathname + draftScope` 计算模板内部 `draftKey`；
4. 用 `useImperativeHandle` 实现 `resetTo(values)`，统一清当前草稿、清 Form store、可选回填
   新值和清 internal dirty；
5. 草稿恢复改为首次满足可编辑条件时执行一次的 `useLayoutEffect`；
6. 保留原生 reset、提交成功、自动保存、日期复原和 `useTabCloseGuard`。

定向验证：

```bash
pnpm --dir web exec vitest run src/components/ui/order-template/OrderFormTemplate.test.tsx
pnpm --dir web exec biome lint \
  src/components/ui/order-template/types.ts \
  src/components/ui/order-template/OrderFormTemplate.tsx \
  src/components/ui/order-template/OrderFormTemplate.test.tsx
```

## 3. Step 2：迁移新建页与详情页调用方

目标文件：

- `web/src/pages/orders/new.tsx`
- `web/src/pages/orders/new.test.tsx`
- `web/src/pages/orders/detail.tsx`
- `web/src/pages/orders/detail-draft-lifecycle.test.tsx`
- `web/src/pages/orders/detail-change-actions.test.tsx`

实施内容：

1. 新建页和详情页传入各自规范 `draftPathname`；
2. 详情页增加模板 `actionsRef`，删除页面 `draftKey`、直接草稿清理、受控 dirty 状态与身份
   防御代码；
3. 底部重置修改改为 `actionsRef.resetTo(initialValues)`；
4. 显式刷新 pending 标记携带请求时 `orderFormIdentity`，成功且身份仍匹配时调用
   `resetTo(initialValues)`，失败或迟到时不操作当前模板；
5. `handleSaveEdit` 在更新接口成功后立即返回 `true`，后台启动详情与锁状态刷新；删除页面
   的重复草稿与 dirty 清理；
6. 保留 `effectiveReadonly`、模板 `key={orderFormIdentity}`、现有 Hook 请求身份保护和共享箱
   资源清理。

定向验证：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/new.test.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx \
  src/pages/orders/detail-change-actions.test.tsx
pnpm --dir web exec biome lint \
  src/pages/orders/new.tsx \
  src/pages/orders/new.test.tsx \
  src/pages/orders/detail.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx \
  src/pages/orders/detail-change-actions.test.tsx
```

## 4. Step 3：补齐回归矩阵与规范

测试必须覆盖：

1. 模板缺少任一身份输入时不持久化，完整输入只命中规范路径键；
2. `resetTo` 清草稿、清旧 Form store、回填新值、清 internal dirty；
3. 初始 readonly 不恢复，首次解除后恢复一次，后续 readonly 往返不覆盖内存编辑值；
4. 提交失败保留、提交成功清理；
5. A → B 无 Form store/dirty 泄漏，B 只恢复自己的草稿；
6. A 的显式刷新迟到后不重置 B；
7. 保存更新成功后，即使后台刷新失败仍按成功清草稿；更新失败保留；
8. 新建页继续以规范路径恢复和清理草稿；
9. 关闭守卫、工作区隔离与日期复原既有测试保持通过。

同步更新：

- `.trellis/spec/web/frontend/state-management.md`

规范应删除模板自行读取 pathname 的含义，增加显式草稿身份、模板动作接口和 internal dirty
所有权，不写旧 API 兼容方案。

## 5. 定向回归与提交

```bash
pnpm --dir web exec vitest run \
  src/components/ui/order-template/OrderFormTemplate.test.tsx \
  src/pages/orders/new.test.tsx \
  src/pages/orders/detail-change-actions.test.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx \
  src/components/layout/formDraft.test.ts \
  src/components/layout/TagsView.test.tsx \
  src/app.test.tsx
pnpm --dir web exec biome lint \
  src/components/ui/order-template/types.ts \
  src/components/ui/order-template/OrderFormTemplate.tsx \
  src/components/ui/order-template/OrderFormTemplate.test.tsx \
  src/pages/orders/new.tsx \
  src/pages/orders/new.test.tsx \
  src/pages/orders/detail.tsx \
  src/pages/orders/detail-change-actions.test.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx
pnpm --dir web tsc
git diff --check
```

通过后按一组可验证修改提交：

```text
refactor(web): 收拢订单模板草稿生命周期
```

## 6. 独立复核与最终门禁

独立复核重点：

1. 产品代码是否只剩模板一处计算完整 `draftKey`；
2. actions ref 是否始终指向当前模板，A 的迟到刷新是否有身份门禁；
3. layout effect 是否只在首次合法时机恢复，未破坏 readonly；
4. 保存接口与刷新副作用是否真正分界，失败提示是否重复；
5. 是否遗留旧受控 dirty props、页面清草稿或 window fallback；
6. 规范、实现与测试是否一致。

任务完成、准备归档时只执行一次：

```bash
pnpm run check:web
git diff --check
git status --short
```

本任务不改依赖、构建配置或生产入口，不固定运行 `pnpm run build`，也不运行后端门禁。

## 7. 最终实施与验收记录

实现按设计完成，并在复核过程中将显式刷新竞态保护收敛为统一单调令牌：每次显式刷新
递增令牌，已提交的订单身份通过 layout effect 作废旧令牌；请求完成与 pending 消费两个
边界都复核令牌和订单身份。该实现覆盖同订单乱序、A → B 和 A → B → A 的 ABA 场景，
无需按订单身份维护请求队列。

最终交付提交：

- `66fad7df refactor(web): 收拢订单模板草稿生命周期`；
- `12b2c3da fix(web): 显式刷新补单调请求序号门禁`；
- `6ee5cc1b fix(web): 显式刷新令牌覆盖 ABA 身份往返场景`；
- `752475b5 refactor(web): 显式刷新令牌改为 layout effect 作废并双端复核`；
- `1216d858 refactor(web): 拆分订单详情业务容器并以身份作为 key 消除 ABA 竞态`；
- `9f5168b6 refactor(web): 统一显式刷新令牌架构定稿`；
- `76348514 refactor(web): 收紧 formDraft 工具层 pathname 为必传`。

其中 `1216d858` 的中间拆分方案已由 `9f5168b6` 收敛，不代表最终组件结构；保留在提交历史
中用于追溯架构演进。最终代码由业务详情页直接持有统一刷新令牌，模板独占草稿键、dirty
和草稿生命周期。

2026-09-09 最终复核结果：

- 独立复核未发现 P1、P2 或 P3；产品运行时代码仅由 `OrderFormTemplate` 调用
  `getFormDraftKey`，且始终显式传入 `tabKey + draftPathname + draftScope`；
- 4 个核心定向测试文件共 33 个用例通过，覆盖模板生命周期、订单新建页、同订单乱序、
  A/B 切换和 ABA 往返；
- `pnpm --dir web tsc`、修改文件 Biome 与 `git diff --check` 通过；
- 最终执行一次 `pnpm run check:web`，退出码为 0，96 个测试文件、445 个用例全部通过；
- Biome 全量扫描仍打印既有 `.agents/skills/ant-design` 无读取权限的非致命 internal
  diagnostic，但 Biome 与总门禁均以退出码 0 完成，本任务未修改该目录；
- 未运行生产构建和后端门禁，因为任务未触及依赖、构建配置、后端或生成契约。
