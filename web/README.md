# Roncin 管理后台前端

Roncin 货代后台的 React 应用。当前使用 React 19、Vite 7、React Router 8、React Query 5、Ant Design 6 和 ProComponents；依赖版本以 [package.json](./package.json) 与根目录锁文件为准。当前订单入口包含海运出口和空运出口；业务能力与验收边界见[项目说明](../README.md)。

## 开发入口

在仓库根目录安装和启动，Node.js 与 pnpm 版本遵循根 [package.json](../package.json)：

```bash
pnpm install
pnpm dev
```

`pnpm dev` 检查数据库并执行开发迁移，随后启动前后端；数据库准备与初始化步骤见[本地开发](../docs/local-development.md)。单独启动前端可执行 `pnpm run dev:web`，默认访问 `http://localhost:8001`，API 经开发代理访问本地 Go 服务。

## 目录与职责

| 入口 | 职责 |
| --- | --- |
| [vite.config.ts](./vite.config.ts) | Vite 构建、开发代理和 Vitest 测试入口 |
| [config/routes.ts](./config/routes.ts) | 路由、菜单、稳定标题和组织切换落点元数据 |
| [src/router/](./src/router/) | React Router 适配、路由权限与导航 |
| [src/app/](./src/app/) | 身份、应用布局与 access 接入 |
| [src/pages/](./src/pages/) | 页面及其专属请求、类型、样式和测试 |
| [src/features/](./src/features/) | 真实跨页面共享的领域能力，通过各领域 `index.ts` 公开 |
| [src/components/ui/](./src/components/ui/) | 页面骨架、表格、表单等公共模板 |
| [src/services/roncin/](./src/services/roncin/) | OpenAPI 生成客户端，禁止手工修改 |

列表优先使用 ProTable 官方 `request`、`actionRef` 和已采纳快照；其他服务端状态使用 React Query。认证业务查询按用户及公司工作区隔离，权限消费 `/api/v1/auth/me` 的 `permissionCapabilities`，不在页面复制规则。财务经营请求绑定当前公司工作区。

订单主详情只保护当前内存输入，关闭或离开时确认未保存修改，重新进入读取服务端数据；订单新建与独立 HBL 编辑仍按身份保存持久草稿。详细规则见[状态管理](../.trellis/spec/web/frontend/state-management.md)。

## 代码生成

接口真相源为 `server/api/**/*.proto`。契约变化先执行服务端生成，再在仓库根目录更新前端派生物：

```bash
make -C server api
pnpm run generate:web-client
```

前端命令读取 `server/openapi.yaml`，更新 OpenAPI JSON 输入、`src/services/roncin/` 和 Proto 枚举/错误常量；后端权限清单变化另执行 `pnpm run generate:permission-keys`。源文件与生成物须一起审阅，禁止手工修补生成结果。

## 验证与交付

下列命令均从仓库根目录执行，定向测试路径相对 `web/`：

```bash
pnpm --dir web exec vitest run src/pages/orders/detail-leave-guard.test.tsx
pnpm --dir web test:changed
pnpm --dir web tsc
pnpm run check:architecture
```

开发过程中按改动运行定向测试及修改文件的 Biome 检查；普通提交增加 `git diff --check`，按类型影响决定是否执行 tsc。整个前端任务验收执行 `pnpm run check:web`；全栈任务可用 `pnpm run check:fast`。构建配置、依赖、生产入口变化或发布前验收才执行 `pnpm run build`，具体阶段要求见[协作规范](../AGENTS.md)。实现功能后按风险补验证，禁止 TDD。

生产由 Go 同域提供 API 和内嵌前端资源，使用仓库根目录构建流程。应用壳源自 Ant Design Pro，现行工程已采用 Vite 与 React Router；上游模板只作参考，不作为框架覆盖或部署依据。

## 规范导航

- [全仓文档入口](../docs/README.md)
- [前端开发规范](../.trellis/spec/web/frontend/index.md)
- [复用能力导航](../.trellis/spec/web/frontend/capability-navigation.md)
- [类型与权限契约](../.trellis/spec/web/frontend/type-safety.md)
- [财务当前公司工作区](../.trellis/spec/web/frontend/finance-current-workspace.md)
