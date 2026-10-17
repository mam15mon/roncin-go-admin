# R2：工作区查询与迟到请求隔离

## 设计证据与改动边界

- `main.tsx` 是生产代码中唯一直接导入公开 `queryClient` 实例的位置；业务组件使用 `useQueryClient` / `useQuery` 消费 Provider。入口客户端还承担 `preloadDingTalkLoginConfig`，其请求不能因业务身份切换而取消。
- `AppLayout` 已按 `userId:organizationId` 为 `OrganizationWorkspace` 设置 React key。新增的私有 `WorkspaceQueryProvider` 在工作区内再次以相同身份 key 挂载，确保工作区即使由其他入口直接消费，身份变化的**当次渲染**也立即取得新客户端，不依赖下一拍 effect 清旧数据。
- 客户端由 `createQueryClient()` 创建，工厂收敛原有 `retry: false`、不聚焦/重连重查及错误元数据策略。入口公开实例仍由同一工厂创建，`main.tsx`、钉钉配置预热及公开路由无需改变。
- 已认证用户 ID 或当前公司 ID 缺失时不挂业务工作区，因此匿名状态不会读业务缓存或启动业务查询。
- 每个工作区实例独占一个客户端，不建立按身份保留的全局客户端 Map。A→B→A 获得第三个实例，退出后以相同身份登录也不会复活旧缓存；同身份重新渲染、页签返回和分页仍复用当前客户端。
- 工作区 layout cleanup 调用 `client.clear()`。已安装 TanStack Query 5.103.1 源码中，`QueryClient.clear` 清 QueryCache/MutationCache，QueryCache.remove 调用 Query.destroy，后者以 silent 取消 retryer；因此即使 queryFn 忽略 AbortSignal，迟到 Promise 也不能把响应写回缓存。工厂错误回调同时检查 query/mutation 仍属于当前 cache，清理后的 mutation 迟到失败不会触发全局提示。

## 请求层相邻缺口与最薄保护

源代码核对发现：`requestClient.ts` 在 Axios Promise 失败后直接调用全局 `errorHandler`；`requestErrorConfig.ts` 的该回调会执行 401 导航、403 message、超时/5xx notification。取消 React Query retryer 无法阻止这条调用链。因此仅清查询缓存不足以阻止旧身份迟到网络失败打断新身份。

经主会话明确授权，统一请求层增加单个工作区生命周期 token：

1. 工作区 layout setup 激活 token，cleanup 同步使 token 失效，请求发起时捕获它。
2. 旧工作区失败维持全局消费失败后的 `undefined` 契约，但不执行全局提示、401 导航或重复上报。`skipErrorHandler: true` 继续原样外抛原始错误，由被取消的查询丢弃结果。
3. 旧工作区成功也返回 `undefined`，与 R1 各写调用的成功守卫协作，避免旧保存响应执行导航、成功提示或清草稿；当前工作区成功仍返回原始报文。
4. 不取消 Axios 传输，不改变请求 Cookie 响应顺序。公开 `/api/v1/auth/dingtalk/login-config`、`/api/v1/auth/dingtalk/login` 和 `/api/v1/auth/login` 显式不绑定业务 token，原预热与换码串行协议继续成立。
5. 退出后不保留活动 token，公开／初始化请求保持原错误契约；未引入新的请求命令框架或失败回退。

## 文件清单

- `web/src/utils/queryClient.ts`：统一工厂、独立实例与已清缓存错误门禁。
- `web/src/components/layout/OrganizationWorkspace.tsx`：完整身份门禁、keyed 业务 Provider、layout 清理及请求 token 生命周期。
- `web/src/utils/requestClient.ts`：最薄工作区请求 token、旧业务成功/失败丢弃、公开认证豁免。
- `web/src/components/layout/OrganizationWorkspace.query.test.tsx`：真实 React Query/Provider 工作区生命周期回归。
- `web/src/utils/requestClient.workspace.test.tsx`：真实请求层/拦截器及 Provider 接入回归，仅 Axios adapter 模拟网络。
- 本文。

没有修改生成客户端、查询键、`main.tsx` 或其他代理负责的写操作/CSV 文件。

## 验证记录

实现完成后补测试，没有先写失败测试或模拟 QueryClient：

- `OrganizationWorkspace.query.test.tsx`：7 个用例通过。同公司换用户、同用户换公司首个渲染无旧值；其他业务域缓存全隔离；A→B→A 新实例及迟到成功；匿名/缺公司/缺用户；退出重登；同身份缓存与翻页保留；旧查询/写入失败不提示；StrictMode 与卸载清理。
- `requestClient.workspace.test.tsx`：11 个用例通过。旧 401/500/超时不影响新身份；当前身份 403 正常提示；旧成功返回 undefined、当前成功返回原报文；真实工作区 Provider 切换与匿名清理；skipErrorHandler 原始 reject；三类公开认证响应保留；公开登录失败正常显示。
- 最终组合定向命令匹配 **10 个文件 / 102 个测试，全部 PASS，无 SKIP**：

  ```bash
  pnpm --dir web exec vitest run src/components/layout/OrganizationWorkspace.query.test.tsx src/utils/requestClient.workspace.test.tsx src/requestErrorConfig.test.ts src/app/AppLayout.menu.test.tsx src/app/dingtalkLoginConfig.test.tsx src/app/dingtalkLoginConfig.bootstrap.test.tsx src/pages/user/login/dingtalk-callback.test.tsx src/pages/user/login/components/dingtalk-qr-login.test.tsx src/pages/user/login/components/dingtalk-qr-login.preloaded.test.tsx src/pages/user/login/components/dingtalk-qr-login.switch.test.tsx
  ```

- 随后仅修正请求测试中的 AxiosResponse 类型夹具，重跑该文件 **1 文件 / 11 测试 PASS**。
- `pnpm --dir web tsc`：最终 PASS（中间并发编辑期间的类型错误已由对应所有者修复；本边界的 Mutation 泛型比较与测试夹具已修正）。
- 5 个改动 TS/TSX 文件定向 Biome：PASS。
- `git diff --check`：PASS。

## 交接与规范更新建议

- 主会话负责最终 `check:fast`、生产入口风险匹配的构建及浏览器身份切换复核。
- `.trellis/spec/web/frontend/state-management.md` 及 `roncin-web-stack` 仍写“业务使用全局单例”，应改为“统一工厂；公开入口实例保留钉钉配置，认证工作区独立 Provider；业务一律通过 useQueryClient 消费当前实例”。主会话统一更新规范。
- 工作区 token 只解决响应与展示生命周期，不中止已发出的业务写入；服务端事务及组织授权继续承担业务写入正确性。
