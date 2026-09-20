# 执行计划：Umi → Vite + React Router v8

> 前置门禁：**batch3（09-20-data-layer-migration-batch3）收尾合入后**才从最新
> 主干切出 `refactor/vite-router-migration` 开工。每步一组可验证修改、一个 commit、
> 一个回滚点。

## 阶段 0：开工准备

- [x] 0.1 确认 batch3 已归档合入；`git status` 干净；从主干切出
      `refactor/vite-router-migration`。
- [x] 0.2 安装依赖：生产 `axios`（requestClient 基座）、`react-helmet-async`
      （12 处 Helmet）；dev `less`（4 个 .less 编译）、`react-router`（^8.4.0）、
      `@vitejs/plugin-react`、`@tailwindcss/vite`、`@umijs/openapi`（独立代码
      生成器，非 Umi 运行时）。三者实测均不在 package.json（现由 Umi 内嵌携带，
      pnpm 隔离下不可达）。移除项在阶段 3 处理。验证：`pnpm --dir web install`。

## 阶段 1：Vite 基建并存（不碰现有文件，主线仍是 Umi）

- [x] 1.1 `vite.config.ts`：吸收 `vitest.config.ts` 全部内容，追加 react 插件、
      tailwind 插件、`server.port=8001`、`server.proxy`（消费 `config/proxy.ts`）、
      `define` 宏（UMI_ENV/SENTRY_DSN/CI/COMMIT_HASH/__APP_VERSION__）、
      `build` 基本项。暂不删除 `vitest.config.ts`（vitest 优先级会合并，阶段 3 删）。
- [x] 1.2 `config/theme.ts`：antd token 从 `config/config.ts` 平移；`index.html`
      与 `src/main.tsx`：loading 脚本、字体、Sentry、**顶部显式
      `import './global.less'` 与 `import '../tailwind.css'`（src/global.tsx
      退役）**、Provider 组合（HelmetProvider 最外层之一 + ConfigProvider
      zh-CN + dayjs locale + AppProvider + QueryClientProvider +
      AppFeedbackBridge + OrganizationWorkspace 上移）。
- [x] 1.3 `src/router/`：`history.ts` shim、`adaptRoutes.ts`（全量审计 routes.ts
      267 行的 path/name/icon/component/access/hideInMenu/layout/redirect 字段，
      `import.meta.glob` 懒加载映射）、`guard.tsx`（403）。
- [x] 1.4 `src/app/`：`AppProvider.tsx`（getInitialState 逻辑平移 + devMock +
      401 redirect + publicAuthPaths）、`access.tsx`（useAccess 消费现有
      `src/access.ts`）、`AppLayout.tsx`（ProLayout 插槽从 app.tsx 平移：
      menuHeaderRender/collapsedButtonRender/menuItemRender/headerContentRender/
      actionsRender/avatarProps + 未登录 effect 守卫 + TagsView 挂载）。
- [x] 1.5 `src/utils/requestClient.ts`：axios 复刻 umi-request 契约；把
      `requestErrorConfig.ts` 的拦截器/错误语义挂载过来（该文件逻辑不动，仅换
      类型与挂载点）。验证：`pnpm --dir web exec vitest run
      src/requestErrorConfig.test.ts`（先适配该测试为直接消费 requestClient）。
      验证：`pnpm --dir web tsc` 通过（并存期两套共存）。

## 阶段 2：一次性切换（原子提交）

- [x] 2.1 全局 import 切换：`@umijs/max` → 新模块源（history→@/router/history；
      useAccess→@/app/access；useLocation/useParams/useSearchParams/Link→react-router；
      Helmet→react-helmet-async）；`useModel('@@initialState')` 15 处 →
      `useInitialState()`；app.tsx 退役（逻辑已平移）。批量 sed + 人工核对。
- [x] 2.2 services 重新生成：新增 `web/scripts/generate-services.mjs` 直调
      `@umijs/openapi` 的 `generateService`（参数平移 config.ts openAPI 段，
      `requestLibPath` 指向 `@/utils/requestClient`，保证与现有生成物风格
      完全一致）；`openapi` 脚本改为 `generate-openapi-input.mjs &&
      generate-services.mjs`；重跑 `pnpm run generate:web-client` 核对 diff
      仅 import 行变化。验证：`git grep "@umijs/max" web/src` 为 0。
- [x] 2.3 切换入口与脚本：`web/package.json` scripts（dev/build/preview/start*
      /prepare 去掉 `max setup`，保持脚本名与端口）；`dev` 走 vite；
      root 脚本核对无感。提交点：`pnpm --dir web dev` 手工冒烟（登录、菜单、
      跳转、任一业务页 CRUD）。

## 阶段 3：删除 Umi

- [x] 3.1 删 `@umijs/max`、`@umijs/max-plugin-openapi`、`@umijs/request-record`
      依赖；删 `config/config.ts`、`vitest.config.ts`（已并入 vite.config）、
      `src/.umi` 痕迹、`typings.d.ts` 中 `*.md` 声明（0 消费）、tsconfig 中
      Umi 注入项（核对 paths 与 types）。
- [x] 3.2 vitest 配置：删 `@@/` 别名；恢复 exclude 中的 `login.test.tsx`。
      验证：`git grep -r "umijs\|\.umi" web/src web/config` 仅剩合理历史注释。

## 阶段 4：测试基建批量替换

- [x] 4.1 新增 `tests/renderWithApp.tsx` 共享包装（MemoryRouter + HelmetProvider
      + AppProvider 注入 currentUser + QueryClientProvider + ConfigProvider）。
- [x] 4.2 54 个文件的 `vi.mock('@umijs/max')` 批量替换（可先 codemod 后人工），
      恢复 `login.test.tsx` 执行。验证：`git grep -l "vi.mock('@umijs/max" web` 为 0。
- [x] 4.3 `pnpm --dir web test` 全量绿。注意长尾约束（AGENTS.md：单文件重型用例
      ≤10、>20s 拆文件），若恢复的测试超时按规范拆分。

## 阶段 5：全量验收与收尾

- [x] 5.1 深链刷新与分包：`vite preview` 下抽测 42 路由中的关键页（登录后直开
      详情页、刷新、TagsView）；`pnpm --dir web build` 成功且产物结构可由 Go
      同域伺服（核对 publicPath 与静态资源路径）。
- [x] 5.2 `pnpm run check:fast` 全绿；playwright `test:e2e` 跑通或记录差异。
- [x] 5.3 spec 更新：AGENTS.md / .trellis/spec/web 中涉及 umi 命令、路由、请求
      封装约定的段落改为 Vite + RR v8 表述（trellis-update-spec）。
- [x] 5.4 提交、归档任务、记录 journal；PR 目标分支 main。

## 回滚点

每个编号步骤一个 commit；阶段 2 是原子切换点，之前任意回退不影响主线；
阶段 2 之后回滚 = 整体切回 Umi 分支（无生产契约，无需兼容）。
