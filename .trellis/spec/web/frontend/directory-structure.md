# 目录结构

```
web/config/             Vite 应用配置：路由、代理、主题、OpenAPI 输入
web/vite.config.ts     Vite 构建/测试统一入口（react+tailwind 插件、define、代理）
web/src/app/            应用壳：AppProvider（初始状态）、useAccess、AppLayout
web/src/router/         路由适配：routes.ts→RouterConfig+菜单、history shim、403 守卫
web/src/pages/          按业务领域组织的页面（finance/、orders/…）
web/src/features/       跨页面共享的领域能力（仅真实共用时提取，公开入口见下）
web/src/hooks/          通用异步竞态 Hook（useLatestAsync/useAsyncGuard，无业务逻辑）
web/src/services/roncin/ OpenAPI 生成的请求客户端（禁止手改）
web/src/components/     可复用界面组件（ui/ 内为全站公共模板）
web/src/constants/      状态元数据等全局常量（statusMeta.ts 统一状态展示）
web/src/access.ts       路由权限定义（权限键来自 permissions.generated.ts）
web/tests/              前端测试基础设施
```

## 页面组织

- 页面按业务领域放 `src/pages/<领域>/`；页面专属请求、类型和样式就近存放，
  不上收到公共层。
- 公共模板组件统一由 `@/components/ui` 导出；避免引入无关的大型聚合组件。
- 页面内部可拆 `components/` 子目录（如 `pages/finance/commissions/components/`）。
- 订单类型领域注册层放 `pages/orders/order-kinds/`（注册表、类型定义、品类表单
  适配与有状态详情扩展），其所有权边界见 component-guidelines 的
  「订单类型三类真相边界」。

## 模块边界与 features 分层

- **页面模块**：普通领域为 `pages/<领域>/`；`pages/finance/<模块>/` 是独立
  财务工作区边界（bills、fees、cashflows、verifications 等互为跨模块）。
  同模块内部允许就近组合，不按每个 URL 强制建 feature。
- **features 只容纳真实跨模块共享的领域能力**：目录为
  `features/<领域>/<能力>/`，公开入口是 `features/<领域>/index.ts` 或
  `features/<领域>/<能力>/index.ts`；更深层目录的 index 不自动成为公共 API，
  不建立 features 全局总 barrel。feature 内部模块用相对路径互引。
- **提取时机**：先搜索[能力导航](./capability-navigation.md)，页面私有就近，
  出现真实跨模块使用才提取；不做猜测性抽象。
- **依赖边界由检查器强制**（`pnpm run check:architecture`，接入 `check:web`）：
  1. 不同页面模块禁止互相导入；
  2. features 禁止导入 pages；
  3. 外部只能经 features 公开入口消费，绕过入口即失败；
  4. `components/ui`、`hooks`、`utils`、`constants` 禁止导入 pages/features，
     其他非页面业务代码禁止直接导入 pages（路由装配走精确授权清单）；
  5. features 公开能力之间出现有向环报错。
  设计取舍见 [ADR 0016](../../adr/0016-frontend-module-boundaries.md)。
