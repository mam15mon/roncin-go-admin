# 保存保护实施结论

- 目标写接口保留默认统一请求错误处理，不改变 `request` 契约。全局已消费的 HTTP/业务失败返回 `undefined` 时立即返回失败；不再追加局部提示，保留统一中文报文、401、超时与防重语义。
- 非空响应还须有实体 ID 且 `success !== false`，才允许成功提示、关闭、导航、刷新和清 dirty；mock reject 或非法响应走已有 catch。订单幂等键仅真正成功后轮换，后台刷新依旧 best-effort。
- 主档 GET 使用 `skipErrorHandler` 由 React Query 既有中文 meta 提示失败。首次加载/失败不挂载空白编辑表单，失败页提供返回和重试；有数据后的重查失败保留输入，单个路由身份仅首次回填。
- 客商 dirty 覆盖 Form 字段、联系人、别名和待提交别名输入。路由完整身份变化重建 Form store 并清外部编辑状态；用户/组织继续由既有 OrganizationWorkspace key 重建隔离。
- 复用 `useTabCloseGuard` 并增加真实 React Router `useBlocker`。取消离开保留输入，确认后放行；页签关闭/组织切换已有确认在该动作期间复用，异步失败自动撤销，不消除 dirty。
- 未新增客商持久草稿、自动保存、后端契约或业务数据写入。

## 验证

实现后补充客商创建/编辑失败矩阵及失败恢复、账单编辑、流水 ModalForm 返回值、真实 createMemoryRouter 的同菜单身份切换/POP/replace、关闭确认复用和异步失败后重新保护。既有订单草稿用例扩展 undefined、缺 ID、success:false；已有成功 mock 同步正式实体响应。

真实浏览器验收仍由主会话完成，尤其实际页签关闭、组织切换拒绝/失败/成功、点击侧栏和浏览器返回。全部写请求应拦截模拟，禁止真实业务写入。
