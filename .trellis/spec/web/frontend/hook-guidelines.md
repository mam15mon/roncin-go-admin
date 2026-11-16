# Hook 与数据获取约定

`web/src/hooks/` 只保留与业务无关的异步竞态工具（`useLatestAsync`/
`useAsyncGuard`）。数据获取与业务 Hook 遵循以下约定：

- 数据获取通过 `src/services/roncin/` 生成客户端 + React Query 完成，逻辑
  就近放在页面文件或页面 `components/` 内（见 state-management.md）。
- 不为一次性页面逻辑新建公共 hook；确实跨页面复用时按领域提取到
  `src/features/<领域>/<能力>/` 并经 `index.ts` 公开（如
  `features/finance/credit-control/` 的信用控制查询），**不放全局
  `src/hooks/`**——全局 hooks 只收业务无关的通用工具。
- 领域 Hook 与其他能力的存在性先查[能力导航](./capability-navigation.md)，
  已有能力的直接消费公开入口，不重写。
- 列表页的筛选、分页状态与查询参数保持同步（URL 或查询键），避免组件内
  私有状态与服务端状态漂移。
- 下拉/联想候选项走服务端 `keyword` 过滤接口，不循环翻页或一次性拉全量。

## 组织级异步联想隔离

- 声明式候选由 `useQuery` 管理，完整资源身份（组织、运输方式、业务类型、父级、字段用途）和规范化/防抖关键词全部进入 key；不能只按组织缓存。
- `RemoteSearchSelect` 的 `queryKey` 必填，`SearchFilterTemplate` 用实例和字段名隔离，字段 `requestKey` 表达同次挂载内会改变的权限/来源依赖。回调更新只经 ref 稳定，不以函数引用作为重取依据。
- 已选名称从当前工作区的对应资源缓存与当前结果渲染合并，不把历史所有关键词结果混进当前候选；快捷新建保留必要的本地增量与创建会话保护。
- 仅为原生 ProForm `request` 或显式命令提供的 Promise 入口可以使用当前客户端 `fetchQuery`；返回前仍核对当前业务身份，过期时返回 `[]`，不能把旧结果交回框架组件。
- 空关键词复用当前身份首批候选，数据未就绪/错误时返回空；无身份或未开放运输方式不请求。
- 查询失败由 Query/meta 或就地错误统一处理；更换查询后取消旧读请求，不能仅禁止旧结果写状态，却仍让默认请求错误处理器通知新关键词/实体。

```ts
const resourceKey = ['orders', 'locations', { organizationId, transportMode, businessType }];
const query = useQuery({
  queryKey: [...resourceKey, { keyword: debouncedKeyword }],
  enabled: Boolean(organizationId && supported),
  queryFn: ({ signal }) => loadLocations(debouncedKeyword, { signal, skipErrorHandler: true }),
  meta: { errorMessage: '地点候选加载失败' },
});
// Promise 返回给 ProForm 前仍需拒绝旧身份。
const result = await client.fetchQuery(options);
if (requestedIdentity !== currentIdentityRef.current) return [];
return result;
```

错误做法是在 Effect 中直接请求、`setPorts` 并以请求序号筛结果；新写法让候选属于键对应的 Query，身份变化的首个渲染自然隔离结果。身份保护 ref 仅用于命令或框架 Promise 返回值，不承担远端数据存储。

### 验证要点

- 同组织 sea→air 首帧不暴露旧港口；旧搜索迟到时 Promise 返回空，不污染当前 Query。
- land/rail 或缺组织时主聚合、客户、地点、承运人、人员等入口均零请求、无错误提示。
- 同挂载 enabled/父级变化会变换 Query key，false→true 不因缓存空结果失去候选。
- 关键词乱序、卸载、工作区 A→B→A、已选名称保活、取消与真实失败分别验证。
