# 技术设计：数据层统一迁移至 React Query v5

## 一、依赖与接入方式

- 直接依赖 `@tanstack/react-query@^5`（web/package.json，pnpm 安装；v5 官方
  支持 React 19）。
- **不启用** umi 的 request/react-query 插件注入路径：源码统一
  `import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'`，
  生产与 vitest 同源可用（选型依据见 prd.md 决策记录）。
- 不引入 devtools（上线前最小交付；后续调试需要时再评估）。

## 二、QueryClient 基建（阶段 0）

新增 `web/src/utils/queryClient.ts`：

- 导出单例 `queryClient = new QueryClient({...})`，默认配置：
  - `queries: { retry: false, refetchOnWindowFocus: false }`——后台管理系统
    错误即时呈现、不自动重试、不因切窗口突发请求（与现行为一致：手写层
    从不重试）。
  - `mutations: { retry: false }`。
  - `queryCache: new QueryCache({ onError })` 与 `mutationCache: new
    MutationCache({ onError })`：经 `utils/appFeedback` 的 `showErrorMessage`
    弹错（非 React 上下文可用的既有桥接）；错误文案取
    `query.meta.errorMessage` / `mutation.meta.errorMessage`（string），
    未设置时回退 `showErrorMessage` 的通用文案或仅 console——迁移时逐处
    用 `meta` 保留原提示文案，保证行为等价。
- Provider 挂载：`src/app.tsx` 的 `childrenRender` 外层包
  `<QueryClientProvider client={queryClient}>`（包裹 AppFeedbackBridge 与
  OrganizationWorkspace）。

## 三、迁移模式映射（唯一模式，禁止再手写竞态模板）

| 现状手写形态 | React Query v5 写法 |
|--------------|---------------------|
| `useState` 列表 + `useEffect` 拉取 + 序号竞态比对 | `useQuery({ queryKey, queryFn, meta })` |
| 抽屉/弹窗 `enabled/open` 才拉 | `enabled: open`（不 `manual`） |
| 参数变化重查（page/pageSize/keyword/orderId） | 参数进 `queryKey`，库自动重查 |
| 链式依赖（先组织后候选） | `enabled: !!parentValue` |
| 手写 `setTimeout` 防抖搜索 | `useDeferredValue`/`useState` + 防抖后的 keyword 进 queryKey；需要保留中间态时 `placeholderData: keepPreviousData` |
| 提交/启停后 `reload()+reloadStats()` | `useMutation` + `onSuccess` 里 `queryClient.invalidateQueries({ queryKey: [域前缀] })` |
| 错误 `message.error(文案)` | 全局 cache onError + `meta: { errorMessage: '原文案' }`；提交类已有局部 catch 的保持局部处理 |
| 卸载后迟到 setState（act 噪音源） | 库内置卸载保护，无需处理 |

- QueryKey 规范：`[域前缀, 实体, 参数对象]`，如
  `['workbench', 'my-receivables', { page, pageSize, keyword }]`；域前缀常量
  就近收敛在对应 hook/组件内，跨页共享失效的用域前缀失效。
- 保留 `unwrapPage`/`unwrapList` 等响应解包工具；queryFn 返回**已映射的
  业务数据**（mapItem 在 queryFn 内完成）。

## 四、测试策略

- 新增 `web/tests/queryClientTestUtils.tsx`：
  `createTestQueryClient()`（retry=false、gcTime=Infinity）与
  `renderWithClient(ui)`（每测试独立 client，防缓存串味）。
- 迁移组件的测试改用该工具包裹渲染；**无需 mock '@umijs/max' 或 React
  Query 本体**（真实实现直接跑）。
- 断言意图不变：原 mock service 函数继续作为 queryFn 依赖注入（props 或
  vi.mock services 层），loading/data 断言等价替换。
- 每个迁移文件迁移后：定向 vitest 全绿且 stderr 零 act/deprecated 警告。

## 五、v5 API 注意事项（迁移红线）

- v5 的 `useQuery` **没有** `onSuccess/onError/onSettled` 回调——错误提示
  一律走全局 cache onError + meta；成功副作用放组件 `useEffect` 或 mutation
  `onSuccess`。
- `isLoading` 仅表首次无数据加载；判断"翻页/换参期间的静默刷新"用
  `isPending && isFetching` 或 `isFetching` + `placeholderData`。
- 禁用 `retry` 不在调用点单独设，统一走默认配置。

## 六、风险与回滚

| 风险 | 缓解 |
|------|------|
| 行为差异（缓存导致页面数据陈旧） | 默认 `staleTime: 0` + 挂载即重查（v5 默认），与现行为一致；写操作后显式 invalidate |
| 测试缓存串味 | 每测试独立 QueryClient（工具函数强制） |
| 全局错误提示重复/丢失 | cache onError 幂等弹错；迁移时逐处核对原文案进 meta |
| 大面积迁移引入回归 | 阶段划分 + 每阶段独立提交 + 定向测试 + 收尾 check:fast |

回滚：按阶段 revert 对应 commit。

## 七、迁移阶梯与范围

- 阶段 0（基建）：依赖、queryClient.ts、app.tsx Provider、测试工具、冒烟探针。
- 阶段 1（高痛点抽屉/弹窗）：`MyReceivablesDrawer`、`MyCommissionDrawer`、
  `MyApplicationHistoryDrawer`、`MyApplicationCandidatesDrawer`、
  `CommissionCreateModal`（含 2 处手写防抖）。
- 阶段 2（详情聚合）：`orders/use-order-detail-data`（8 state + 五重竞态收敛）。
- 阶段 3（规范沉淀）：spec 写入唯一模式与禁令；AGENTS.md 口径核对（已写
  React Query，无需改）。
- 后续批次（本任务收尾时另立任务）：`use-order-create-options`、
  `use-order-fee-options`、`use-order-lock-state`、`VerificationWorkbench`、
  `BillCreationWorkbench`、`partner-detail`、`fees`、`split`、`cashflows`、
  `invoices` 等审计清单剩余项。

## 八、兼容性

- 不改 `.proto`/OpenAPI 生成物；不动用户可能在工作区的
  `master-data-template` 文件集。
- 上线前阶段无历史兼容负担，迁移一次性删净旧竞态令牌，不留双轨。
