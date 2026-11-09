# Roncin 前端协作入口

本目录遵循根 [AGENTS.md](../AGENTS.md)。本文仅记录项目定位，完整规则见[前端规范](../.trellis/spec/web/frontend/index.md)；文档、设计说明和开发者注释使用中文。

## 技术与代码入口

- React 19 + Vite 7 + React Router 8 + React Query 5 + Ant Design 6 / ProComponents；以 [package.json](./package.json) 和根锁文件为版本依据。
- `vite.config.ts` 为构建与测试入口；`config/routes.ts` 统一路由、菜单、标题与切换落点。`src/router/adaptRoutes.tsx` 将集中配置装配为 React Router 和 ProLayout 菜单。
- `src/app/AppProvider.tsx` 管理认证初始状态；`src/access.ts` 从 `/api/v1/auth/me` 的 `permissionCapabilities` 按具体权限及范围推导能力。`roleScopes` 和权限展示列表不能替代能力判定，按钮与路由复用同一 access。
- 页面专属能力就近放 `src/pages/`，真实跨模块复用才进入 `src/features/<领域>/<能力>/index.ts`；先查[能力导航](../.trellis/spec/web/frontend/capability-navigation.md)。
- 列表请求使用 ProTable 官方 `request` / `actionRef`，指标、当前页导出消费已采纳快照；其他服务端状态使用 React Query。认证业务缓存由 `OrganizationWorkspace` 按用户与工作台隔离，不直接使用公开登录预热单例。
- 后端请求使用生成客户端与 `src/utils/requestClient.ts`；错误已被统一处理后可能返回 `undefined`，写入必须验证成功响应再关闭、导航或清 dirty。
- 订单主详情仅保留当前内存编辑与未保存离开保护；新建订单和独立 HBL 编辑仍支持身份隔离的持久草稿，见[状态管理](../.trellis/spec/web/frontend/state-management.md)。

## 命令与生成物

统一使用 pnpm，在仓库根目录执行：

```bash
pnpm run dev:web
pnpm --dir web exec vitest run src/pages/orders/detail-leave-guard.test.tsx
pnpm --dir web test:changed
pnpm --dir web tsc
pnpm run check:architecture
```

接口变化先修改 Proto 并执行 `make -C server api`，再执行 `pnpm run generate:web-client`。权限 Manifest 变化另执行 `pnpm run generate:permission-keys`。`src/services/roncin/`、`config/openapi.generated.json`、`src/enums.generated.ts`、`src/errorReasons.generated.ts` 与 `src/permissions.generated.ts` 均由对应生成流程维护，不手改。

Ant Design API 优先使用本地 `pnpm --dir web exec antd info <组件> --format json` 查询，必要时查询 demo、semantic 和 token。仓库内 [antd 技能](./.claude/skills/antd/SKILL.md)提供本地查询方式；[升级技能](./.claude/skills/pro-upgrade/SKILL.md)仅用于已授权的依赖升级，上游模板不能覆盖本项目路由、工作区和权限实现。

## 验证与提交

先实现功能，再按风险补充针对性验证；缺陷至少覆盖触发场景和相邻路径，禁止 TDD。开发期只运行定向测试与修改文件的 Biome 检查；普通本地提交运行相关测试和 `git diff --check`，按类型影响选择 tsc。任务最终验收运行风险匹配的 `check:web` 或 `check:fast`，构建仅按根规范的变更风险触发。

修改前检查 Git 状态，保留他人的未提交内容；每组可验证变更用准确的中文 Conventional Commit 提交。不要为通过检查关闭规则，也不要引入历史兼容、静默回退或无需求的抽象。
