# 技术设计：Umi → Vite 7 + React Router v8

## 1. 目标架构

```text
web/
  index.html                 # 新增：Vite 入口（含 loading.js 白屏脚本）
  vite.config.ts             # 新增：合并现有 vitest.config.ts，单一配置源
  src/
    main.tsx                 # 新增：createRoot → Sentry → router → providers
    app/
      AppProvider.tsx        # 初始状态（getInitialState 逻辑平移）+ useInitialState
      access.tsx             # useAccess（消费 access.ts 纯函数）
      AppLayout.tsx          # ProLayout 壳（app.tsx layout 插槽平移）+ 路由守卫
    router/
      history.ts             # history shim：push/replace/location → router 实例
      adaptRoutes.ts         # routes.ts → RR RouteObject[] + ProLayout menuData
      guard.tsx              # 路由级 access 守卫（未命中渲染 403）
    utils/
      requestClient.ts       # axios 封装，复刻 umi-request 契约
  config/
    routes.ts                # 保持不变（唯一真相源）
    theme.ts                 # antd token 从 config.ts 平移
    proxy.ts                 # 保留，改由 vite server.proxy 消费
```

## 2. Umi 能力 → 替代方案映射（基于 2026-09-20 实测摸底）

| Umi 能力 | 现状实测 | 替代方案 | 业务面改动 |
|---|---|---|---|
| 路由（react-router v6 内核） | useLocation 33 / useParams 24 / useSearchParams 5 / Link 21 | react-router v8（^8.4.0）同名 API 直连 | 只换 import 源 |
| `history` 单例 | 155 处（push/replace/location） | `src/router/history.ts` shim 委托 `router.navigate` / `router.state.location` | 零逻辑改动 |
| access 插件 | useAccess 142 处；access.ts 为纯函数 | 自建 useAccess + 路由 guard；access.ts 不动 | 只换 import 源 |
| initialState/model 插件 | useModel('@@initialState') 15 处 | AppProvider + useInitialState | 15 处替换调用 |
| layout 插件 | app.tsx RunTimeLayoutConfig 全套插槽 | ProLayout 直接使用（pro-components） | 无业务面改动 |
| request 插件 | requestErrorConfig.ts（拦截器/防重/Sentry/401）+ services 生成物 `import { request } from "@umijs/max"` | axios 复刻契约；openapi requestLibPath 改指自建客户端 | 生成物重新生成 |
| Helmet | 12 处 | react-helmet-async 直连 | 只换 import 源 |
| antd 插件（token/appConfig） | config.ts 内 80+ 行 token | ConfigProvider + config/theme.ts | 无 |
| locale 插件（zh-CN） | 0 处 intl API 使用 | ConfigProvider locale + dayjs 'zh-cn' | 无 |
| moment2dayjs | 0 处 moment | 删除（antd v6 原生 dayjs） | 无 |
| mock 插件 | 0 个 _mock.ts | 删除 | 无 |
| tailwindcss 插件 | tailwind v4 + tailwind.config.js | @tailwindcss/vite 插件 | 无 |
| proxy / headScripts / hash | config.ts + public/scripts/loading.js | vite server.proxy / index.html / Vite 构建自带 hash | 无 |
| define 宏 | UMI_ENV/SENTRY_DSN/CI/COMMIT_HASH/__APP_VERSION__（sentry.ts 消费） | vite define 同名注入 | sentry.ts 零改动 |
| max setup / max record / routePrefetch / manifest | prepare 钩子、record 脚本 | 放弃（见 §6） | 无 |

## 3. 关键设计决策

### D1 路由：集中式 routes.ts 保持真相源，适配层双产出

- 依赖实测解析为 `react-router@^8.4.0`。本项目采用纯客户端 SPA（Library Mode），
  `createBrowserRouter`、`RouteObject`、原生 `lazy` 等核心契约与 v7 严格兼容，
  并原生支持 React 19，无需降级，直接基于 v8 实施。
- `config/routes.ts` 保持 Umi 形状（`path/name/icon/component/access/hideInMenu/
  layout:false/redirect`），不手改。
- 新增 `src/router/adaptRoutes.ts`：
  - **routerRoutes**：产出 `createBrowserRouter` 的 `RouteObject[]`；`layout: false`
    的路由（登录/注册/回调）挂在壳外，其余挂 `AppLayout` 下。转换规则（依据
    routes.ts 267 行实测）：
    - **组件路径两段式解析**：`component` 字符串先试 `pages/${path}.tsx`，未命中
      回退 `pages/${path}/index.tsx`（单文件与目录 index 两种形态并存，只拼一种
      会有一半路由找不到模块）；模块经 `import.meta.glob('../pages/**/*.tsx')`
      预收集。
    - **原生 route.lazy**：用 RR v8 的 `lazy: () => importer().then(m =>
      ({ Component: m.default }))`，不手写 `React.lazy` + `Suspense` 边界。
    - **父级无 component 的嵌套组**（`/user`、`/orders`、`/finance` 三处）显式
      `element = <Outlet />`。注：RR 对无 element 父级本就隐式渲染 Outlet，
      显式写出是可读性约定而非修 bug。
    - **redirect**：4 处重定向项映射 `<Navigate to={redirect} replace />`。
    - **通配符**：末项 `path: './*'` 规范化为 `'*'`（去掉 `./` 前缀）指向
      NotFound。
  - **menuData**：同一份配置产出 ProLayout `route` 树（name/隐藏项），按
    useAccess 结果过滤，等价于 Umi layout 插件的菜单权限过滤；`icon` 字符串经
    内置字典映射为 antd 图标组件（实测 8 个 key：contacts/dashboard/database/
    global/orderedList/setting/shop/transaction）——原由 Umi layout 插件编译期
    完成的字符串→图标转换由适配层接管。
- Router 实例在 `main.tsx` 用 `createBrowserRouter(routerRoutes)` 创建并模块级
  导出，供 history shim 与 TagsView 等消费。

### D2 history shim（最大耦合点，155 处调用）

```ts
// src/router/history.ts —— API 面实测只有 push/replace/location（无 listen/block/options）
export const history = {
  push: (to: string) => router.navigate(to),
  replace: (to: string) => router.navigate(to, { replace: true }),
  // router 尚未创建（模块顶层加载期）时回退 window.location：
  // pathname/search/hash 同构，业务恰好只消费这三者（无 state/key 依赖）
  get location() {
    return router ? router.state.location : window.location;
  },
};
```

- 业务文件只把 `import { history } from '@umijs/max'` 换成 `from '@/router/history'`，
  155 处调用体零改动（含 requestErrorConfig 的 401 重定向）。
- 测试中出现的 `history.replaceState` 是 `window.history`，不受影响。

### D3 初始状态：AppProvider 平移 getInitialState

- `app.tsx` 的 `getInitialState`（fetchUserInfo、DEV_MOCK 兜底、publicAuthPaths、
  401 redirect）**逻辑原样**搬入 `AppProvider`，启动时执行一次并下发 context。
- 导出 `useInitialState()`，15 处 `useModel('@@initialState')` 一次性替换为
  `useInitialState()`；不保留 useModel 兼容别名（无历史契约，不留暗桩）。
- `app.tsx` 中 `childrenRender` 的组合（QueryClientProvider + AppFeedbackBridge +
  OrganizationWorkspace）上移到 `main.tsx` 的 Provider 层级。

### D4 权限：access.ts 零改动 + 双层守卫

- `src/access.ts` 是 initialState 的纯函数，直接复用。
- `useAccess()`：从 AppProvider context 读 initialState，`useMemo(access)`。
- 路由级双层等价 Umi 行为：菜单过滤（menuData 按 access 过滤）+ `guard.tsx`
  （路由 access 未命中渲染现有 403 `Result`，平移 `unAccessible`）。
- `onPageChange` 未登录守卫平移为 `AppLayout` 内 `useLocation` effect。

### D5 请求客户端：复刻 umi-request 契约（次大耦合点）

- `src/utils/requestClient.ts` 基于 axios（Umi request 本就是 axios；axios
  当前由 Umi 内嵌携带，迁移时须升为直接生产依赖）：
  - `request<T>(url, options)` 返回 `response.data`（生成物只消费 data）；
  - `errorThrower` 语义：`success:false` 报文抛 `BusinessError`；
  - `errorHandler` 全语义：skipErrorHandler 直抛、401 跳登录、403 message、
    超时通知、防重守卫拦截不报 Sentry、其余上报并通知（含 traceId）；
  - 拦截器：X-Request-ID 头、写操作 in-flight 防重（method+URL+body 键）、
    完成/失败释放记录。
- `requestErrorConfig.ts` 保留（其单测 `requestErrorConfig.test.ts` 是契约回归
  资产），只把 `RequestConfig`/`@@/plugin-request/request` 类型换成自建
  `RequestOptions` 并改挂载到 requestClient。
- 生成物：生成引擎为独立的 `@umijs/openapi`（已核实 `@umijs/max-plugin-openapi@2`
  的依赖即 `@umijs/openapi@^1.8.3`）。注意它是 openapi-typescript-codegen 的
  umi 分支，输出为「每服务函数文件」风格，与现有 25 个 `*Service.ts` +
  `typings.d.ts` 完全一致；上游 openapi-typescript-codegen 输出 `OpenApi` 类
  命名空间，风格不同，**不可用**。新增 `web/scripts/generate-services.mjs`
  （十余行，参数平移 config.ts 的 openAPI 段）直调
  `generateService({ schemaPath, serversPath: './src/services', projectName:
  'roncin', requestLibPath: "import { request } from '@/utils/requestClient'" })`，
  `openapi` 脚本改为 `generate-openapi-input.mjs && generate-services.mjs`，
  重跑 `pnpm run generate:web-client` 核对 diff 仅 import 行变化。
  `@umijs/openapi` 是纯 dev 时码生成器，不会把 Umi 运行时带进产物。
- `withCredentials: true`、`DEFAULT_REQUEST_TIMEOUT`、`baseURL: ''` 保持。

### D6 入口、样式与环境

- `index.html`：引 `public/scripts/loading.js`、字体；`main.tsx` 组装
  Sentry → HelmetProvider → ConfigProvider（config/theme.ts token + zh-CN +
  dayjs locale）→ AppProvider → RouterProvider，Provider 栈中
  QueryClientProvider/AppFeedbackBridge/OrganizationWorkspace 从 app.tsx
  childrenRender 上移。
- **显式样式导入（Umi 隐式约定失效点）**：Umi 自动载入 `src/global.tsx` 与
  `src/global.less`，Vite 下必须在 `main.tsx` 顶部显式
  `import './global.less'`（39KB 全局高密度样式）与 `import '../tailwind.css'`
  （`src/global.tsx` 仅此一句 import，随迁移退役该文件）；tailwind v4 走
  `@tailwindcss/vite`，`tailwind.config.js` 保留；`*.module.less`（登录页等
  3 个）由 Vite 原生 CSS Modules 支持，仅需 `less` devDep。
- **HelmetProvider**：12 处 Helmet 依赖 react-helmet-async，其 Provider 当前由
  Umi 运行时隐式提供；迁移后 `main.tsx` 与 `tests/renderWithApp.tsx` 的
  Provider 栈都必须显式包裹 `<HelmetProvider>`，否则含 Helmet 的页面（登录页
  等）渲染即崩。
- vite `define`：实测读取面仅 4 键——`process.env.SENTRY_DSN`(8 处)/
  `UMI_ENV`(7)/`COMMIT_HASH`(4)/`NODE_ENV`(1)，**无裸 `process.env` 读取**。
  按键注入这 4 键 + `__APP_VERSION__` 即可（NODE_ENV Vite 本就内建替换，显式
  写出作冗余保障），`sentry.ts` 零改动；不做 `'process.env'` 整体对象替换
  （粗于实际需要，且可能干扰依赖库对 process.env 的特性探测）。
  `__UMI_VERSION__` 无消费方，不迁移。冒烟清单含全局复查无新增裸读。
- `vite.config.ts` 吸收 `vitest.config.ts`（别名去掉 `@@/`，test 段原样保留），
  `server.port = 8001`、`server.proxy` 消费 `config/proxy.ts`。

### D7 测试基建

- 新增 `tests/renderWithApp.tsx`：`MemoryRouter(initialEntries)` + HelmetProvider
  + AppProvider（可注入 currentUser）+ QueryClientProvider + ConfigProvider 的
  组合包装，替代 54 个文件里的 `vi.mock('@umijs/max')`。
- mock 替换为机械操作：删 `vi.mock` 块 → import 改新模块源 → 用包装渲染；
  可先写 codemod 脚本批量转换再人工修复。
- 恢复 `login.test.tsx`（当前因依赖 Umi 测试基建被 exclude）；删除 vitest
  exclude 中的该项与 `@@/` 别名。

## 4. 实施策略：单分支、按提交粒度分步、原子切换

- 独立分支 `refactor/vite-router-migration`（**等 batch3 收尾合入后从最新主干切出**）。
- 顺序（详见 implement.md）：先并存新增 Vite 基建（不碰现有文件，可编译）→
  一次性 import 切换 + services 重生成 → 删除 Umi（config/依赖/脚本/痕迹）→
  测试基建批量替换 → 全量验证。
- 不做双构建并存期：并存只存在于同一分支内的中间提交，主线一次性切换。

## 5. 风险与回滚

| 风险 | 缓解 |
|---|---|
| ProLayout 直接使用的插槽行为与 Umi layout 插件有差异（菜单过滤、折叠态、面包屑） | 插槽逐项平移对照；重点回归 AGENTS.md 侧边栏 48px 折叠规范与 TagsView |
| request 契约细节偏差（errorThrower 时机、skipErrorHandler 透传、防重键） | requestErrorConfig.test.ts 作为契约回归基线，先迁测试再换实现 |
| routes.ts 字段覆盖不全（redirect/wrappers 等未摸底字段） | adaptRoutes 编写时全量审计 routes.ts 267 行，未知字段编译期报错而非静默丢弃 |
| 42 路由懒加载与深链刷新回归 | 验收清单逐路由冒烟 + preview 验证 |
| 回滚 | 分支保留 Umi 主线，任何阶段失败直接切回；无生产历史契约，无需数据兼容 |

## 6. 明确放弃项（无替代，用户已知）

- `max record`（requestRecord 请求录制面板）——纯开发期玩具；
- `routePrefetch` / `manifest`——保留 route.lazy 分包，放弃预取；
- Umi mock 插件、moment2dayjs、i18n 插件——仓库内均为 0 使用；
- `analyze`（max build ANALYZE）——如需产物分析后续用 rollup-plugin-visualizer。
