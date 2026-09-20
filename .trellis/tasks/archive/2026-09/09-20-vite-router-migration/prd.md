# 前端构建与路由迁移：Umi → Vite + React Router v8

## Goal

将 web 前端从 @umijs/max（Umi 4）整体迁移到 Vite 7 + React Router v8（library mode、
集中式路由），重建入口、布局、权限、初始状态、请求客户端与测试基建；业务代码与
路由表保持最小改动。动机：开发体验优先（用户 2026-09-20 拍板），消除测试基建对
Umi 虚拟模块的依赖这一结构性痛点。

## 背景与动机

- 158 个测试文件中有 54 个在重复 `vi.mock('@umijs/max')`：根因是 Umi 的运行时
  （路由、model、access 等）由 `.umi/` 临时产物与虚拟模块注入，Vitest 下解析不到，
  只能逐文件手写 Mock，AI 与人写测试都在持续交税。
- 数据层已全量迁移到 React Query v5（`useRequest` 已清零），去 Umi 的最后一块
  硬骨头只剩构建器、路由与运行时插件。
- Umi 底层即 react-router v6，`useLocation/useParams/useSearchParams/Link` 切到
  React Router v8（实测解析为 `^8.4.0`，library mode 核心契约与 v7 严格等价）
  语法不变，迁移面可控。

## Requirements

1. 构建器从 `@umijs/max`（max dev/build/openapi/setup）切换到 Vite 7；路由切换到
   React Router v8 library mode（`createBrowserRouter`），**保持 `web/config/routes.ts`
   集中式配置为唯一真相源**，不引入文件约定路由。
2. 重建 Umi 运行时插件的等价物：入口（index.html + main.tsx）、ProLayout 布局壳
   （现有 app.tsx 各插槽原样平移）、初始状态（getInitialState 逻辑原样）、权限
   （access.ts 零改动 + 自建 useAccess）、请求客户端（复刻 umi-request 契约）、
   Helmet、antd 主题 token 与 zh-CN locale。
3. 业务代码改动最小化：`history` 155 处、`useAccess` 142 处零逻辑改动（只换
   import 源）；`useModel('@@initialState')` 15 处一次性替换为新 hook；
   Helmet 12 处换直连 `react-helmet-async`。
4. OpenAPI 生成链路脱离 `max openapi`（生成物仍禁止手改，仅通过生成命令更新），
   `requestLibPath` 指向自建请求客户端。
5. 测试基建重建：删除 54 个 `vi.mock('@umijs/max')`，提供共享测试 Provider 包装；
   恢复当前被排除的 `src/pages/user/login/login.test.tsx`；移除 vitest 的 `@@/`
   指向 `src/.umi` 的痕迹别名。
6. 对外命令面不变：`web/package.json` 脚本名（dev/build/test/lint/tsc/openapi 等）
   与 dev 端口 8001 保持，root 脚本与 `scripts/check-parallel.mjs` 无感。
7. 遵循 AGENTS.md：仅 pnpm、不做双构建/兼容层/自动回退（一次性切换）、每完成一组
   可验证修改即提交。

## Acceptance Criteria

- [x] `pnpm --dir web dev`（Vite）正常启动、代理到本地 Go 服务；登录、菜单、
      权限按钮、页面跳转全链路可用。
- [x] 全站 42 个路由可达；路由级代码分割生效；dev 与 `vite preview` 下深链刷新
      不 404；生产构建产物可由 Go 同域服务正确伺服。
- [x] `git grep "@umijs/max" web/src` 与 `git grep "vi.mock('@umijs/max" web` 均为 0。
- [x] 请求契约复刻验证：X-Request-ID 头、401 跳登录、403 提示、超时通知、写操作
      防重守卫、Sentry 上报行为与迁移前一致（requestErrorConfig 相关单测全绿）。
- [x] `login.test.tsx` 恢复执行且通过；`pnpm --dir web test` 全量、`tsc`、
      `biome lint` 通过。
- [x] `pnpm run check:fast` 通过；`pnpm --dir web build` 成功。
- [x] 权限链路不回归：access.ts 逻辑不变，`generate:permission-keys` /
      `generate:web-client` 链路正常。
- [x] playwright e2e 按现状基线跑通，或明确记录差异与原因。
      （记录：e2e 三套用例在本会话沙箱中因浏览器进程被环境立即终止（launch ESRCH）
      无法执行；前置链路 vite dev 服务、Go 后端、代理与深链回退已分别冒烟验证。
      e2e 需在有浏览器执行权限的环境补跑，非迁移回归。）

## 范围外

- 不改任何业务功能与路由表语义；不引入 TanStack Router / 文件约定路由 / Next.js。
- 明确放弃（无替代，记录于 design.md 放弃清单）：requestRecord 录制面板、
  routePrefetch/manifest（Vite 构建自带 contenthash，懒加载保留）、Umi mock 插件
  （仓库内 0 个 _mock.ts）、moment2dayjs（0 处 moment）、i18n 多语言（0 处使用）。
- 不动 server 侧与后端契约。
