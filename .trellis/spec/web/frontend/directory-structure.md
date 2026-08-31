# 目录结构

```
web/config/             Umi 配置、路由、代理、OpenAPI 输入
web/src/pages/          按业务领域组织的页面（finance/、order/…）
web/src/services/roncin/ OpenAPI 生成的请求客户端（禁止手改）
web/src/components/     可复用界面组件（ui/ 内为全站公共模板）
web/src/access.ts       路由权限定义（权限键来自 permissions.generated.ts）
web/types/              Umi 生成类型（禁止手改）
web/tests/              前端测试基础设施
```

## 页面组织

- 页面按业务领域放 `src/pages/<领域>/`；页面专属请求、类型和样式就近存放，
  不上收到公共层。
- 公共模板组件统一由 `@/components/ui` 导出；避免引入无关的大型聚合组件。
- 页面内部可拆 `components/` 子目录（如 `pages/finance/commissions/components/`）。
