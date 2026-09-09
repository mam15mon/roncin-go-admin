# 订单类型注册薄底座实施计划

> 本任务为纯前端 SE 行为等价重构。任务激活前不得实施。

## 1. 实施前检查

- [x] 检查 `git status`，确认没有与订单页面、模板或当前任务重叠的并行改动；
- [x] 完整读取 `prd.md`、`design.md`、本文件、根 `AGENTS.md`、`implement.jsonl` 及引用；
- [x] 记录重构前 SE 创建/更新请求夹具、详情初始值和详情操作按钮顺序；
- [x] 确认最终范围不包含后端、契约、数据库、权限生成物和未来订单类型；
- [x] 开发阶段只运行受影响的定向检查，不重复运行全量门禁。

## 2. Step 1：建立唯一注册表并迁移静态消费者

目标文件：

- `web/src/pages/orders/order-kinds/types.ts`
- `web/src/pages/orders/order-kinds/registry.ts`
- `web/src/pages/orders/order-kinds/registry.test.ts`
- `web/src/pages/orders/order-kinds/sea-export/definition.ts`
- `web/src/pages/orders/common.ts`
- `web/src/pages/orders/list.tsx`
- `web/src/pages/orders/fees.tsx`
- `web/src/pages/orders/components/OrderPageHeader.tsx`
- `web/src/pages/orders/list-query.ts`
- `web/src/pages/orders/list-resources.ts`
- `web/src/pages/orders/use-order-create-options.ts`
- `web/src/pages/orders/use-order-detail-data.ts`
- 上述文件的既有测试

实施内容：

- [x] 定义 `OrderKindDefinition`、`OrderKindFormAdapter`、创建默认值上下文和详情扩展契约；
- [x] 只注册 `sea-export`，实现直接 kind/路径解析和未知类型 fail-closed；
- [x] 迁移列表、费用、Header、查询及资源 Hook 到注册定义；
- [x] 将 `category` 语义改名为 `transportMode`，删除 Sea/Air 默认分支；
- [x] 删除 `common.ts` 的旧类型、字典与解析函数，不保留 alias 或桥接；
- [x] 删除 `OrderListTemplate` 的重复 `OrderKind`/`kindMap` 和“未知即海运出口”兜底；
- [x] 清理相关 `as any`，未知类型不得进入 Hook 或发请求。

定向验证：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/order-kinds/registry.test.ts \
  src/pages/orders/orders.test.ts \
  src/pages/orders/list-query.test.ts \
  src/pages/orders/list-resources.test.ts \
  src/pages/orders/use-order-create-options.test.ts \
  src/pages/orders/use-order-detail-data.test.ts \
  src/pages/orders/fees.test.tsx
pnpm --dir web exec biome lint <本步骤修改文件>
git diff --check
```

完成后提交：

```text
refactor(web): 建立订单类型唯一注册入口
```

## 3. Step 2：迁移 SE 表单适配器

目标文件：

- `web/src/pages/orders/order-kinds/sea-export/form-adapter.ts`
- `web/src/pages/orders/order-kinds/sea-export/form-adapter.test.ts`
- `web/src/pages/orders/order-create-payload.ts`
- `web/src/pages/orders/components/detail/orderDetailHelpers.ts`
- `web/src/pages/orders/new.tsx`
- `web/src/pages/orders/detail.tsx`
- 相关测试

实施内容：

- [x] 将海运 Sections 选择接入 `definition.form.buildSections`；
- [x] 把订单日期、运输模式、FCL、CIF、推荐服务、普货和创建人默认值移入纯默认值函数；
- [x] 将创建请求转换改为明确 SE 函数，删除 config 参数、`businessType === 1` 与 `isSea`；
- [x] 将详情初始值和更新请求转换改为明确 SE 函数，删除字段存在性类型猜测；
- [x] 新建与详情页只调用当前注册定义的表单适配器；
- [x] 删除旧导出与旧测试入口，不保留包装兼容；
- [x] 用固定夹具断言创建请求、详情值和更新请求的完整等价结果。

定向验证：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/order-kinds/sea-export/form-adapter.test.ts \
  src/pages/orders/order-create-payload.test.ts \
  src/pages/orders/components/detail/orderDetailHelpers.test.ts \
  src/pages/orders/new.test.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx \
  src/pages/orders/detail-change-actions.test.tsx
pnpm --dir web exec biome lint <本步骤修改文件>
git diff --check
```

完成后提交：

```text
refactor(web): 由 SE 适配器统一订单表单转换
```

## 4. Step 3：抽离有状态 SE 详情扩展

目标文件：

- `web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx`
- `web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.test.tsx`
- `web/src/pages/orders/detail.tsx`
- `web/src/pages/orders/components/detail/OrderDetailHeader.tsx`
- `web/src/pages/orders/detail-change-actions.test.tsx`
- `web/src/pages/orders/detail-shared-container-workbench.test.tsx`

实施内容：

- [x] 将 change-actions 数据、请求序号、身份门禁和错误呈现移入 SE 扩展；
- [x] 将改配、共享航次、历史、共享箱的开关、ref、目标 ID 和订单切换清理移入扩展；
- [x] 通过 render-prop 贡献 header actions、more menu、append sections、overlays 和类型刷新命令；
- [x] `OrderDetailHeader` 用通用 `businessActions` 插槽替代 split/reassign 命名 props；
- [x] 保持拆票与改配按钮位置、顺序、样式、禁用原因和权限/锁单判断；
- [x] `detail.tsx` 删除 Sea 覆盖层、同批订单、历史组件和 sea change service 的直接 import；
- [x] 扩展普通刷新不调用模板 `resetTo`，显式刷新 token 仍完全留在通用详情页；
- [x] 以 `orderFormIdentity` 重挂载扩展并保留 change-actions 迟到响应门禁。

定向验证：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.test.tsx \
  src/pages/orders/detail-change-actions.test.tsx \
  src/pages/orders/detail-shared-container-workbench.test.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx \
  src/pages/orders/orders-breadcrumbs.test.tsx
pnpm --dir web exec biome lint <本步骤修改文件>
git diff --check
```

完成后提交：

```text
refactor(web): 抽离海运出口详情业务扩展
```

## 5. Step 4：清理旧真相并同步规范

- [x] 全仓搜索并确认以下符号在产品代码中归零：

```text
ORDER_KIND_CONFIGS
OrderKindConfig
parseOrderKind
category === 'sea'
category === 'air'
businessType === 1
```

- [x] 搜索 `SeaOrder`、`seaOrderChangeService` 等依赖，确认通用 `detail.tsx` 不再直接引用；
- [x] 搜索 `as any`，确认订单类型迁移没有残留强转；
- [x] 更新 `.trellis/spec/web/frontend/component-guidelines.md` 或
  `.trellis/spec/web/frontend/state-management.md`，记录注册、权限和生命周期三类真相边界；
- [x] 核对任务 PRD AC，无历史兼容分支、无未来类型占位注册。

定向验证：

```bash
pnpm --dir web exec vitest run \
  src/components/ui/order-template/OrderFormTemplate.test.tsx \
  src/components/ui/order-list-template/OrderListTemplate.test.tsx \
  src/pages/orders/order-kinds/registry.test.ts \
  src/pages/orders/order-kinds/sea-export/form-adapter.test.ts \
  src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.test.tsx \
  src/pages/orders/new.test.tsx \
  src/pages/orders/fees.test.tsx \
  src/pages/orders/detail-change-actions.test.tsx \
  src/pages/orders/detail-draft-lifecycle.test.tsx \
  src/pages/orders/detail-shared-container-workbench.test.tsx \
  src/pages/orders/orders-breadcrumbs.test.tsx \
  src/pages/orders/list-query.test.ts \
  src/pages/orders/list-resources.test.ts
pnpm --dir web tsc
pnpm --dir web exec biome lint <全部修改文件>
git diff --check
```

完成后提交：

```text
docs(spec): 记录订单类型注册与生命周期边界
```

## 6. Step 5：独立复核与最终门禁

独立复核重点：

1. 注册表是否真的成为全部运行时调用方的唯一配置源；
2. 未知类型是否在发请求前 fail-closed；
3. SE 请求和默认值是否行为等价，而非只完成函数搬家；
4. 公共列表模板和权限判断是否残留第二套类型/能力真相；
5. SE 扩展是否独立持有状态，通用详情是否仍泄漏 Sea 依赖；
6. change-actions、显式刷新 token、草稿和锁单的竞态边界是否互不越权；
7. 是否误注册或半开放 SI/AE/AI/LAND/RAIL。

整个任务完成准备归档时执行一次：

```bash
pnpm run check:web
git diff --check
git status --short
```

本任务不改后端、契约、依赖、构建配置或生产入口，因此不运行后端门禁或生产构建。

## 7. 回滚点

- Step 1 只改变配置来源，可独立回滚；
- Step 2 只改变 SE 表单适配入口，可独立回滚；
- Step 3 只改变详情专属 UI 所有权，可独立回滚；
- 不得以恢复旧兼容桥、增加 fallback、清空用户草稿或跳过断言作为回滚方案。
