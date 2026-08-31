# Web 前端开发规范（React + Ant Design Pro + Umi）

> 唯一真相源是根目录 `AGENTS.md`；本目录是面向 AI 任务执行的浓缩版。

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | 页面组织与生成物边界 | ✅ |
| [Component Guidelines](./component-guidelines.md) | UI 模板规范与公共组件 | ✅ |
| [State Management](./state-management.md) | 服务端状态与请求客户端 | ✅ |
| [Hook Guidelines](./hook-guidelines.md) | 数据获取与自定义 Hook | ✅ |
| [Type Safety](./type-safety.md) | 生成物类型与权限键对齐 | ✅ |
| [Quality Guidelines](./quality-guidelines.md) | pnpm、校验命令、禁令 | ✅ |

## Pre-Development Checklist

1. 新页面放 `web/src/pages/<领域>/`，页面专属请求、类型、样式就近存放。
2. 需要的接口是否已有 OpenAPI 客户端方法；没有则先改服务端契约再生成。
3. 权限判断复用 `access`（路由级）与同一判断结果（按钮级），不硬编码第二套。
4. 页面结构是否应复用 `@/components/ui` 公共模板（见 Component Guidelines）。

## Quality Check

- `pnpm --dir web lint`、`pnpm --dir web tsc`、`pnpm --dir web biome:lint`、
  `pnpm --dir web test` 按风险选取最小集通过。
- 未手改 `web/src/services/roncin/`、`web/types/`、`permissions.generated.ts`。
- 全部命令使用 `pnpm`，没有引入 npm/npx/Yarn 入口。

**语言**：界面文案与开发者文档使用中文。
