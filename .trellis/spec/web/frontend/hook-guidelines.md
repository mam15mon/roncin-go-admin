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

- 发起客户、地点、承运人、人员等组织级联想请求前，必须捕获当前
  `organizationId`；没有有效组织时直接返回空数组，不发送请求。
- 请求完成后必须再次核对捕获的组织与当前活动组织。身份已变化时，不仅禁止写入
  Hook state，还必须向调用下拉组件的 Promise 返回空数组，避免旧组织候选项通过
  返回值泄漏到新组织页面。
- 空关键字用于展开下拉时，应复用已经按当前组织与当前业务类别校验过的首批缓存；
  缓存尚未就绪或身份不匹配时返回空数组。有真实非空关键字时仍走服务端搜索。
- 加载错误也必须绑定发起请求时的组织及页面实体身份；只有错误身份仍等于当前页面
  身份时才允许展示，不能依赖 `useEffect` 在组织切换后的下一拍清除旧错误。

```ts
// 错误：只阻止旧响应写 state，返回值仍会显示在新组织的下拉框中。
const options = await searchPartners(keyword);
if (requestedOrganizationId === activeOrganizationIdRef.current) {
  setOptions(options);
}
return options;

// 正确：state 和 Promise 返回值都受同一组织身份保护。
const options = await searchPartners(keyword);
if (requestedOrganizationId !== activeOrganizationIdRef.current) {
  return [];
}
setOptions(options);
return options;
```

### 组织级资源的多维身份

- 候选或资源若同时依赖组织、运输方式、业务类型等维度，加载身份和迟到响应门禁必须覆盖
  所有会影响结果的维度；不能只比较 `organizationId`。
- 身份任一维度变化时，渲染阶段必须同步隐藏旧 state；不能等下一拍 `useEffect` 清空，否则
  新身份首帧仍会暴露旧候选。
- 未注册或未开放运输方式的所有公开查询入口统一 fail-closed：不发请求、返回 `[]`、不设置
  UI 错误状态。

```ts
// 错误：只检查组织，sea 请求完成后可能把结果写入同组织的 air 页面。
const requestedIdentity = { organizationId };
const result = await searchPorts(keyword);
if (requestedIdentity.organizationId === activeOrganizationIdRef.current) {
  setPorts(result);
}

// 正确：结果依赖的每个身份维度都参与即时隐藏和迟到门禁。
const requestedResourceIdentity = `${organizationId}:${transportMode}`;
const currentResourceIdentity = `${organizationId ?? ''}:${transportMode ?? ''}`;
const visible =
  Boolean(organizationId && transportMode) &&
  loadedResourceIdentity === currentResourceIdentity;
const ports = visible ? loadedPorts : [];
const result = await searchPorts(keyword);
const activeResourceIdentity = [
  activeOrganizationIdRef.current ?? '',
  activeTransportModeRef.current ?? '',
].join(':');
if (requestedResourceIdentity !== activeResourceIdentity) {
  return [];
}
setPorts(result);
return toOptions(result);
```

对未开放类型，公开的每个入口都应走同一 fail-closed 语义：

```ts
if (!definition || definition.transportMode === 'land' || definition.transportMode === 'rail') {
  return [];
}
```

测试至少断言：

- 同组织 `sea → air` 后旧 ports/location options 在当前 render 立即为空；
- 旧 sea 请求最后完成时返回 `[]`，且不调用 `setPorts`、不污染当前候选；
- land/rail 对主数据 Effect、客户、港口、地点、承运人和人员等全部公开查询入口均为零请求、
  返回 `[]`，且不触发 UI 错误。
