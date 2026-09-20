# 目录结构

```
web/config/             Vite 应用配置：路由、代理、主题、OpenAPI 输入
web/vite.config.ts     Vite 构建/测试统一入口（react+tailwind 插件、define、代理）
web/src/app/            应用壳：AppProvider（初始状态）、useAccess、AppLayout
web/src/router/         路由适配：routes.ts→RouterConfig+菜单、history shim、403 守卫
web/src/pages/          按业务领域组织的页面（finance/、orders/…）
web/src/hooks/          通用异步竞态 Hook（useLatestAsync/useAsyncGuard）
web/src/services/roncin/ OpenAPI 生成的请求客户端（禁止手改）
web/src/components/     可复用界面组件（ui/ 内为全站公共模板）
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
