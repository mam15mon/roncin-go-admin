# 多组织快速上线薄底座实施计划

## 1. 实施前提

- 当前只完成规划，不运行 `task.py start`，不修改产品代码；
- 开始前收尾或暂停进行中的 `09-08-fix-order-partner-quick-add`，避免重叠编辑订单组件；
- 保留其他未提交改动，按可验证组使用 Conventional Commits；
- 不实现草稿重构、历史兼容、全局 Ent 拦截器或系统上下文；
- 清空、重建数据库或删除开发数据前必须单独取得用户授权。

## 2. Phase 1：前端工作区

### 2.1 让页签和页面共享 keyed workspace

目标文件：

- `web/src/app.tsx`
- `web/src/components/layout/TagsView.tsx` 及测试

步骤：

1. 从 `initialState.currentUser` 取得用户 ID 与当前组织 ID；
2. 生成稳定 workspace key；
3. 用同一个 keyed 容器包住 `TagsView` 和路由页面；
4. 不新增草稿 Context；
5. 测试用户或组织变化会卸载旧页签与页面实例。

建议提交：

```text
feat(web): 建立多组织工作区边界
```

### 2.2 接入组织切换守卫

目标文件：

- `web/src/components/OrganizationSwitcher.tsx` 及测试
- `web/src/components/layout/tabCloseGuard.ts` 及测试

步骤：

1. 为现有 guard 增加任意已注册 dirty 表单检查；
2. 选择当前组织时直接结束；
3. dirty 时弹一次确认，取消不调用 API；
4. 请求期间锁定切换器；
5. API 失败时保留旧状态、路由和页签；
6. 成功后更新 `initialState` 并 `history.replace('/welcome')`；
7. 由 workspace key 重建页面和页签，不广播多套 reset 事件。

定向验证：

```bash
pnpm --dir web exec vitest run web/src/components/OrganizationSwitcher.test.tsx web/src/components/layout/tabCloseGuard.test.ts web/src/components/layout/TagsView.test.tsx
pnpm --dir web exec biome lint web/src/app.tsx web/src/components/OrganizationSwitcher.tsx web/src/components/layout/tabCloseGuard.ts web/src/components/layout/TagsView.tsx
pnpm --dir web tsc
git diff --check
```

建议提交：

```text
feat(web): 收口组织切换生命周期
```

## 3. Phase 2：通用异步守卫

### 3.1 实现 Hook

建议新增：

- `web/src/hooks/useLatestAsync.ts`
- `web/src/hooks/useLatestAsync.test.ts`
- `web/src/hooks/useAsyncGuard.ts`（若共享实现适合放在同一文件，可合并）

步骤：

1. 实现 mounted + sequence + AbortController 的共享内核；
2. `useLatestAsync` 新调用自动使旧调用失效；
3. `useAsyncGuard` 在卸载或 invalidate 后使结果失效；
4. `run(request, apply)` 只在 token 当前时调用 apply，统一返回
   current/stale/unmounted/aborted 判别；
5. 当前请求的 400/403/500 等真实错误继续抛出；
6. 测试乱序完成、卸载、显式取消、不可取消 Promise 和真实错误。

### 3.2 首批替换重复样板

先通过 `rg` 列出 organization ref、request sequence、mounted ref 和组织切换清理 effect，
逐个判断职责。优先改造：

- `PartnerQuickAddSelect` 的搜索和创建回填；
- 订单页面中同类远程选项查询；
- 本任务实际触及且存在等价样板的组件。

删除规则：

- 可删除：与通用 Hook 等价的 sequence/mounted ref；
- 可删除：workspace 重挂载已覆盖的“组织变化时清空本地 state”代码；
- 不删除：模块缓存组织 key、提交 loading、服务端组织校验、业务资源 ID 检查；
- 不做全站机械替换，不改与本次数据流无关的页面。

测试必须证明：

- 连续搜索只有最后一次结果进入 options；
- 组织切换或组件卸载后，旧创建结果不回填新表单；
- 当前请求错误仍展示；
- 创建按钮不能重复提交；
- 服务端已成功但页面已卸载时，不产生旧工作区 UI 副作用。

定向验证按实际改造文件执行 Vitest、Biome 与 `pnpm --dir web tsc`。

建议提交：

```text
refactor(web): 统一页面异步竞态保护
```

### 3.3 草稿只做回归验证

不修改 `formDraft.ts`、不删除 `draftScope`、不增加新 Hook。运行：

```bash
pnpm --dir web exec vitest run web/src/components/layout/formDraft.test.ts web/src/components/ui/order-template/OrderFormTemplate.test.tsx web/src/components/layout/TagsView.test.tsx
```

确认真实组织切换不串草稿，切回原组织仍可恢复。若现有行为失败，只修实际缺陷，不扩大为
草稿架构重构。

## 4. Phase 3：通用角色组织范围

本阶段涉及契约，必须原子切换到新模型，不保留长期双字段或双表路径。

### 4.1 泛化 Ent 与 Proto

重点文件：

- `server/internal/data/ent/schema/role.go`
- `server/internal/data/ent/schema/role_order_organization_access.go`
- `server/api/admin/v1/admin.proto`
- 角色 Biz、Data、Service 与测试
- Ent、Proto、OpenAPI、Web 客户端生成物

步骤：

1. 将订单专用组织 access 重命名为通用模型；
2. Proto 改用 `organization_accesses`；
3. 删除旧字段和旧类型调用；
4. 更新角色保存、读取和提权校验；
5. 生成并审阅迁移和所有生成物；
6. 若需要破坏性数据库操作，暂停并请求用户授权。

建议提交：

```text
refactor(server): 泛化角色组织访问模型
```

### 4.2 实现权限级组织范围 Resolver

重点文件：

- `server/internal/biz/auth.go`
- `server/internal/biz/admin_role.go`
- `server/internal/data/auth.go`
- 新增 resolver 与定向测试

步骤：

1. Principal 保留 role grant；
2. ResolvePrincipal 加载权限、data scope 和通用组织 access；
3. 提供全部启用组织与当前组织树的最小仓储查询；
4. 解析 readable/writable 组织集合；
5. 结果去重排序；
6. 测试不同角色不串联、四种 scope、停用组织和 writable。

不得注册 Ent 全局 interceptor/hook，不引入 `SystemScope`，不在 Service 复制算法。

建议提交：

```text
feat(server): 按权限解析角色组织范围
```

### 4.3 接入订单

1. 订单列表、详情与写操作使用通用 resolver；
2. 仓储显式应用 allowed organization IDs；
3. 保留业务类型权限、锁、版本和状态机；
4. 增加天津角色 + 北京只读 access 的完整用例；
5. 增加订单权限不能借用财务角色范围的反例；
6. `writable=false` 时北京订单写入返回 403。

建议提交：

```text
refactor(server): 接入订单通用组织范围
```

### 4.4 接入往来单位

1. 列表与详情使用 readable 组织集合；
2. 创建与编辑使用 writable 组织集合；
3. 保留组织内代码唯一性、企业角色和税号规则；
4. 前端选择器不自行推导授权组织。

建议提交：

```text
feat(server): 接入往来单位通用组织范围
```

### 4.5 接入财务账单

1. 列表、详情使用 readable 集合；
2. 创建、编辑和状态流转使用 writable 集合；
3. 保持 `biz.Transactor`、`Data.client(ctx)`、锁顺序和 expectedVersion；
4. 回归只读跨组织查看、禁止写入和版本冲突。

建议提交：

```text
feat(server): 接入账单通用组织范围
```

### 4.6 更新角色管理页面

1. 使用新生成的 `organization_accesses`；
2. 文案改为通用“可访问组织”；
3. 编辑并保存 `writable`；
4. 删除旧字段 fallback；
5. 测试加载、编辑、提交和权限只读态。

建议提交：

```text
feat(web): 更新通用角色组织范围配置
```

阶段生成与定向检查至少包括：

```bash
go -C server generate
make -C server api
pnpm run generate:web-client
go -C server test ./internal/biz/... ./internal/data/... ./internal/service/...
pnpm --dir web tsc
git diff --check
```

具体生成命令以仓库 Makefile 为准，避免无关生成漂移。

## 5. Phase 4：最终验收

### 5.1 跨层审查

- workspace 是否同时包住页签和页面；
- 切换失败是否会提前清状态；
- stale/abort 是否错误吞掉真实业务错误；
- mutation 是否在卸载后仍回填、跳转或写缓存；
- 是否误删缓存组织 key 和重复提交保护；
- Principal 是否保留权限与角色范围关联；
- 三个聚合是否在数据库谓词中应用组织集合；
- 订单业务类型权限是否仍生效；
- 是否意外加入草稿重构、旧字段兼容或全局拦截器。

### 5.2 浏览器抽验

1. 连续搜索制造乱序响应，只展示最后一次结果；
2. 创建请求进行中切换组织，旧结果不回填新页面；
3. 组织切换分别验证取消、失败和成功；
4. 组织 A 草稿切到 B 不出现，切回 A 能恢复；
5. 天津员工在天津工作区查看北京订单；
6. 北京只读订单不能编辑或流转；
7. 两个不同权限范围角色组合后不发生串联。

### 5.3 最终门禁

定向检查全部通过后，因包含契约、生成物和跨层权限变更，执行一次：

```bash
pnpm run check:server
pnpm run check:web
git diff --check
git status --short
```

只有涉及依赖、构建配置或生产入口时才增加 `pnpm run build`。记录所有结果，不通过关闭
规则或跳过类型错误来换取绿灯。

## 6. 阶段退出条件

| 阶段 | 退出条件 |
| --- | --- |
| Phase 1 | 成功/失败/取消与页签页面重建均有测试 |
| Phase 2 | latest、卸载失效、错误传播通过；草稿仅完成回归，无新抽象 |
| Phase 3.1-3.2 | 单一通用契约生成完整，权限范围不跨角色串联 |
| Phase 3.3-3.6 | 三个聚合和角色页面只使用新模型 |
| Phase 4 | 定向检查、最终门禁和浏览器抽验均有记录 |
