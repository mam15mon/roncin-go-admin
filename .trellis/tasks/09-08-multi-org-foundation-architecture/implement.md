# 多组织快速上线薄底座实施计划

## 1. 实施前提

- 本任务当前只完成规划，不运行 `task.py start`，不修改产品代码；
- 开始实施前，先收尾或暂停当前进行中的
  `09-08-fix-order-partner-quick-add`，避免同时修改订单模板、草稿和快捷新增组件；
- 实施时再次检查 `git status`，保留其他任务的未提交改动；
- 按 Phase 逐组提交，使用 Conventional Commits；
- 不实现历史兼容、双读双写、灰度开关或全局 Ent 拦截器；
- 任何清空、重建本地数据库或删除开发数据的命令，执行前取得用户明确授权。

## 2. Phase 1：前端组织工作区

### 2.1 建立工作区 Context 与容器

目标文件：

- `web/src/app.tsx`
- `web/src/components/layout/OrganizationWorkspace.tsx`（新增，名称可按目录风格调整）
- `web/src/components/layout/OrganizationWorkspace.test.tsx`（新增）
- `web/src/components/layout/index.ts` 或现有导出入口

实施内容：

1. 定义只包含 workspace 身份的 Context；
2. 使用 `userId:organizationId` 生成 `workspaceKey`；
3. 让同一个 keyed 容器同时包裹 `TagsView` 与路由页面；
4. 处理未登录或组织缺失时的非持久化上下文；
5. 测试 key 变化会卸载旧页签和页面实例。

最小验证：

```bash
pnpm --dir web exec vitest run web/src/components/layout/OrganizationWorkspace.test.tsx
pnpm --dir web exec biome lint web/src/app.tsx web/src/components/layout/OrganizationWorkspace.tsx
git diff --check
```

建议提交：

```text
feat(web): 建立多组织工作区边界
```

### 2.2 接入切换守卫与成功后重定向

目标文件：

- `web/src/components/OrganizationSwitcher.tsx`
- `web/src/components/OrganizationSwitcher.test.tsx`
- `web/src/components/layout/tabCloseGuard.ts`
- `web/src/components/layout/tabCloseGuard.test.ts`
- `web/src/components/layout/TagsView.tsx`

实施内容：

1. 在现有 guard 模块增加 workspace 级脏状态查询与确认；
2. `OrganizationSwitcher` 在 API 前执行确认；
3. 切换期间禁用重复选择；
4. 失败时保留状态、页签和路由；
5. 成功后更新 `initialState` 并 `history.replace('/welcome')`；
6. 确认 keyed remount 清理旧组织页签，不再额外维护旧页签映射。

定向测试必须覆盖：

- 选择相同组织；
- 无脏状态直接切换；
- 有脏状态取消；
- 有脏状态确认；
- API 失败；
- API 成功；
- 快速重复选择只产生一次有效切换。

最小验证：

```bash
pnpm --dir web exec vitest run web/src/components/OrganizationSwitcher.test.tsx web/src/components/layout/tabCloseGuard.test.ts
pnpm --dir web exec biome lint web/src/components/OrganizationSwitcher.tsx web/src/components/layout/tabCloseGuard.ts web/src/components/layout/TagsView.tsx
pnpm --dir web tsc
git diff --check
```

建议提交：

```text
feat(web): 收口组织切换工作区生命周期
```

### 2.3 复核迟到响应与模块级缓存

重点文件：

- `web/src/utils/order-options-cache.ts`
- `web/src/pages/orders/components/PartnerQuickAddSelect.tsx`
- 订单选项 Hook 与直接持有模块级 Map 的文件

实施内容：

1. 列出受影响的模块级缓存，确认 key 包含组织 ID；
2. 验证旧组织的搜索响应和快捷创建响应不能回填新工作区；
3. 保留请求序列号、mutation 组织快照或 mounted/generation 防护；
4. 只删除由 keyed remount 明确取代的手动清理 effect/ref；
5. 不以批量删代码为目标，不碰与本任务无关的快捷新增行为。

最小验证：运行现有订单模板、快捷新增和选项缓存定向测试，并增加至少一个切换期间迟到
响应用例。

建议提交：

```text
refactor(web): 对齐组织工作区异步状态边界
```

## 3. Phase 2：草稿 Scope 内聚

### 3.1 提供 `useScopedDraft`

目标文件：

- `web/src/components/layout/formDraft.ts`
- `web/src/components/layout/useScopedDraft.ts`（是否独立文件按现有结构决定）
- 对应测试文件

实施内容：

1. 从 Workspace Context 获取用户和组织；
2. 使用 v2 key 生成规则；
3. 提供 load/save/clear/hasDraft/draftKey；
4. 无 scope 时不持久化；
5. 不读取或迁移旧 key；
6. workspace guard 与 TagsView 复用同一 key/scope 工具。

### 3.2 精简订单模板接口

目标文件：

- `web/src/components/ui/order-template/OrderFormTemplate.tsx`
- 订单新建、详情页面及模板测试
- `.trellis/spec/web/frontend/state-management.md`

实施内容：

1. 删除公开 `draftScope` prop；
2. 模板内部调用 scoped draft；
3. 一次性更新全部仓库内调用方；
4. 提交成功仅清当前草稿；
5. 更新状态管理规范，记录业务组件不得自行拼用户/组织 scope。

定向验证：

- 两组织草稿互不覆盖；
- 切回原组织恢复；
- 新建与详情草稿分离；
- 提交后只清当前 key；
- 旧版本 key 不恢复；
- 页签关闭和组织切换的确认一致。

最小验证：

```bash
pnpm --dir web exec vitest run web/src/components/layout/formDraft.test.ts web/src/components/ui/order-template/OrderFormTemplate.test.tsx web/src/components/layout/TagsView.test.tsx
pnpm --dir web exec biome lint web/src/components/layout/formDraft.ts web/src/components/ui/order-template/OrderFormTemplate.tsx web/src/components/layout/TagsView.tsx
pnpm --dir web tsc
git diff --check
```

建议提交：

```text
refactor(web): 内聚组织工作区草稿范围
```

## 4. Phase 3：通用角色组织范围

Phase 3 是原子契约阶段。不得只提交新 Proto 而保留旧前端调用方，也不得通过兼容字段
拆成长期双模型。可以在本地形成多个工作提交，但阶段验收时必须处于单一新模型。

### 4.1 替换 Ent 与 Proto 模型

重点文件：

- `server/internal/data/ent/schema/role.go`
- `server/internal/data/ent/schema/role_order_organization_access.go`
- `server/api/admin/v1/admin.proto`
- 角色 Biz、Data、Service 转换与测试
- Ent / Proto / OpenAPI / Web 生成物

实施内容：

1. 将订单专用 schema/type/edge 重命名为通用角色组织访问；
2. Proto 字段改为 `organization_accesses`；
3. 删除旧字段和旧类型的代码路径；
4. 同步角色提权校验和 DTO 转换；
5. 生成并审阅数据库迁移；
6. 更新全部生成物，禁止手改；
7. 不编写旧字段适配器或双表读取。

生成与定向验证：

```bash
go -C server generate
make -C server api
pnpm run generate:web-client
go -C server test ./internal/biz/... ./internal/data/... ./internal/service/...
git diff --check
```

具体生成目标以仓库现有 Makefile 为准，避免无关生成物漂移。若迁移验证需要清空或重建
当前本地数据库，先暂停并取得用户明确授权。

建议提交：

```text
refactor(server): 泛化角色组织访问模型
```

### 4.2 建立按权限解析组织范围的 Biz 能力

重点文件：

- `server/internal/biz/auth.go`
- `server/internal/biz/admin_role.go`
- 新增 resolver 文件及测试
- `server/internal/data/auth.go`
- 组织目录仓储接口与最小实现

实施内容：

1. Principal 保留 role grant 来源；
2. ResolvePrincipal 加载角色权限、data scope 和通用组织访问项；
3. 建立读取全部启用组织、当前组织树的最小仓储能力；
4. 实现按权限解析 readable/writable 集合；
5. 结果去重排序；
6. 测试角色不串联、四种 scope、停用数据、显式 writable 和无权限场景。

不得做的事：

- 不把 `all` 实现成跳过后续仓储过滤；应解析成明确的启用组织 ID 集合；
- 不注册 Ent 全局 interceptor/hook；
- 不引入 `SystemScope`；
- 不让 Service 复制范围算法。

建议提交：

```text
feat(server): 按权限解析角色组织范围
```

### 4.3 接入订单聚合

实施内容：

1. 将订单现有专用组织访问读取迁移到通用 resolver；
2. 列表、详情和写操作使用显式 allowed organization IDs；
3. 保留业务类型权限、悲观锁、expectedVersion 和状态机；
4. 直接访问订单的认证路径与新的权限组织范围一致；
5. 覆盖跨角色不串联、只读追加组织和范围外组织。

建议提交：

```text
refactor(server): 接入订单通用组织范围
```

### 4.4 接入往来单位聚合

实施内容：

1. 列表默认查询 readable 组织集合；
2. 详情在主键查询中附带组织集合；
3. 创建和编辑校验 writable 集合；
4. 保留组织内代码唯一性、企业角色和税号等现有规则；
5. 更新相关选择器请求，只传业务筛选，不在前端推导授权组织。

建议提交：

```text
feat(server): 接入往来单位通用组织范围
```

### 4.5 接入财务账单聚合

实施内容：

1. 账单列表、详情使用 readable 集合；
2. 创建、编辑和状态流转使用 writable 集合；
3. 保持 `biz.Transactor` 事务边界；
4. Data 层继续通过 `Data.client(ctx)` 获取客户端；
5. 多行锁维持固定顺序，版本冲突维持 HTTP 409 语义；
6. 覆盖只读跨组织查看和禁止跨组织写入场景。

建议提交：

```text
feat(server): 接入账单通用组织范围
```

### 4.6 更新角色管理页面

目标文件：

- `web/src/pages/admin/roles.tsx`
- 角色表单组件与测试
- OpenAPI 生成客户端调用方

实施内容：

1. 使用 `organization_accesses`；
2. 文案从“订单组织”改为通用“可访问组织”；
3. 显示与保存 `writable`；
4. 删除旧字段 fallback；
5. 验证加载、编辑、校验、提交和权限只读态。

建议提交：

```text
feat(web): 更新通用角色组织范围配置
```

## 5. Phase 4：跨层验收与文档收口

### 5.1 代码与数据流审查

逐条检查：

- Workspace 是否同时包住 Tabs 与页面；
- 切换失败是否可能提前清状态；
- 草稿 key 是否只由基础设施生成；
- async mutation 是否可能在卸载后回填；
- Principal 是否保留权限与角色范围关联；
- read/write 是否使用各自动作权限解析；
- 订单业务类型权限是否仍与组织范围求交集；
- 三个仓储是否在数据库谓词中应用组织集合；
- 生成文件是否完全来自生成命令；
- 是否意外加入旧字段、双读或全局拦截器。

### 5.2 最终风险匹配验证

定向检查全部通过后，因本任务包含契约、生成物和跨层权限模型，执行一次完整验收：

```bash
pnpm run check:server
pnpm run check:web
git diff --check
git status --short
```

只有在实现改动涉及生产构建入口、依赖或构建配置时才增加：

```bash
pnpm run build
```

记录每条命令、结果和无法执行原因。不得为通过门禁关闭规则或忽略真实错误。

### 5.3 浏览器抽验

使用两个组织和至少两个角色组合验证：

1. 组织 A 新建订单输入内容并形成草稿；
2. 切换组织，分别验证取消、接口失败和成功；
3. 成功后确认 `/welcome`、页签清空和组织 B 干净页面；
4. 切回 A，重新打开页面并恢复草稿；
5. 总部角色查看多个组织的订单、往来单位和账单；
6. 只读追加组织可以查看但不能写；
7. 组合两个权限范围不同的角色，确认权限不串联。

### 5.4 规范与任务收尾

- 若实施中形成可复用的新约束，更新对应 `.trellis/spec/`；
- 更新任务 PRD 偏差记录，所有偏差须说明事实依据；
- 按 Trellis finish-work 流程归档；
- 不在此任务顺手扩展其他业务聚合或全局 Ent 框架。

## 6. 阶段退出条件

| 阶段 | 退出条件 |
| --- | --- |
| Phase 1 | 切换成功/失败/取消与页签页面重建均有测试，旧响应不污染新工作区 |
| Phase 2 | 业务调用方不再传 `draftScope`，组织草稿隔离与恢复通过 |
| Phase 3.1-3.2 | 单一通用契约生成完整，范围解析不跨角色串联 |
| Phase 3.3-3.5 | 三个聚合的读写路径全部显式应用组织范围 |
| Phase 3.6 | 角色页面只使用新字段且可配置 writable |
| Phase 4 | 定向验证、完整门禁和浏览器抽验均有记录，无未解释失败 |

任何阶段未满足退出条件时，不进入下一阶段的扩大接入。
