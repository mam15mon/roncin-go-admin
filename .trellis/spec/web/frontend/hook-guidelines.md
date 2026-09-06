# Hook 与数据获取约定

当前 `web/src/` 没有公共 hooks 目录；遵循以下约定：

- 数据获取通过 `src/services/roncin/` 生成客户端 + React Query（或项目已在用
  的服务端状态方案）完成，逻辑就近放在页面文件或页面 `components/` 内。
- 不为一次性页面逻辑新建公共 hook；确实跨页面复用时才上提到 `src/hooks/`，
  并以业务领域命名。
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
