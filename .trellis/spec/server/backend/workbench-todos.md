# 本人作业待办与订单直达

## 1. 适用范围

修改工作台费用/异常数量、分页明细或订单异常入口时适用。待办从现有订单费用与异常事实查询，
不另建待办表、定时扫描或历史快照；只读查询不新增 Schema、事务及锁。

## 2. 接口与入口

- `WorkbenchService.ListMyUnbilledFees`：`GET /api/v1/workbench/my-unbilled-fees`。
- `WorkbenchService.ListMyOpenAbnormalCases`：`GET /api/v1/workbench/my-open-abnormal-cases`。
- `MasterDataService.SearchOptions`：`GET /api/v1/master-data/options/search`，登录用户可读，
  异常登记使用该候选接口；`ListItems` 仍要求主数据管理读取权限。
- biz：`WorkbenchUsecase.ListMyUnbilledFees/ListMyOpenAbnormalCases`，接收
  `WorkbenchScope` 和 `WorkbenchTodoFilter`，返回 `PagedList`。
- data：`workbench_todo.go`；摘要 `workbench.go` 必须复用同一查询构建函数。
- web：`pages/workbench/MyTodosDrawer.tsx` 走官方 ProTable `request/actionRef`；
  费用直达 `/orders/{sea-export|air-export}/{orderId}/fees`，异常直达订单详情的 `?panel=abnormal`。

## 3. 请求、响应与身份契约

请求只有 `page/pageSize`；默认 1/20，最大 200。身份取会话，不接受改查同事或公司的参数。
基础范围为当前公司、当前用户真实 `OrderPersonnel` 协作、SE/AE 订单 `read`；
费用另需同品类 `fee.read`，异常另需 `abnormal_case.read`。`lock` 不替代费用读取，
管理员也不扩大为全公司订单。无分类读取能力时返回空列表、零数量，前端隐藏该分类入口。

封套为 `data/total/page/pageSize`。两类都携带订单 ID/单号、业务类型、客户、审核锁、
退关及结案状态；费用携带方向、名称、结算方、币种、原币金额、录入时间、待补审状态及
阻塞原因；异常携带记录/类型 ID、类型名称、标记时间及业务只读原因。金额传 decimal 字符串，
时间传 RFC3339；界面复用 `formatAmount` 和 `formatDate`，不混加币种。

`SearchOptions` 请求为有效 `kind`、`keyword`、可选 `page/pageSize`（默认 1/20，上限 200），
返回启用候选的分页封套；沿用原 `ListOptions` 公共字典读取边界，固定 `Enabled=true`，
复用已有中文/英文/代码/拼音检索。异常登记只有打开表单后才通过 ProFormSelect `request`
搜索，不要求额外授予 `system.master_data.item.read`，不加载全量选项。

费用谓词为 `accountingOrderFeePredicate()` 与无 `effectiveBillLinePredicate()` 关联；
有效草稿账单占用费用，已被更正替代或无需记账的行排除。冲回行方向用
`entFeeSettlementDirection`。异常仅取 ACTIVE，名称批量读取 A 型共享主数据，停用类型
仍有可读历史名称。时间相同按主键升序，费用按录入时间、异常按标记时间升序分页。

锁定、退关、结案不消除待办。异常写门禁复用 `ensureOrderBusinessContentEditable`；
待补审只阻塞源费用建账；审核锁本身不阻止符合原规则的建账。界面提供状态解释并允许进入原订单。

## 4. 校验与错误矩阵

| 场景 | 结果 |
| --- | --- |
| 未登录 | 沿用 Principal 登录错误 |
| `page < 1`、`pageSize < 1` 或 `pageSize > 200` | `ErrWorkbenchInvalid`，不查仓储 |
| 无分类权限、无协作订单或其他公司订单 | 空列表、零数量，不返回金额/内容 |
| 查询失败 | 原错误外传；抽屉显示失败与重试，不解释为零条 |
| 锁定、退关、结案 | 仍返回待办与只读状态，写入口遵守原门禁 |
| 处理成功后返回/重开/刷新 | 重查摘要与明细，跨请求数量差异通过刷新收敛 |
| 切公司或用户，旧请求迟到 | 工作区重建，首帧无旧数据，迟到响应不能回写新身份 |

## 5. 正常、基准与错误场景

正常：同票两笔未建账费用展示两行及「2 笔」，有效草稿占用后相应行消失。
基准：只有异常 read 者能打开锁定订单的异常面板，看到名称与只读说明。
错误：只有 lock 无 fee.read 仍展示逐笔费用；用订单票数代替费用笔数；停用类型显示 UUID。

## 6. 必需验证

- `biz/workbench_test.go`、`service/workbench_todo_test.go`：分页默认值、边界及权限交集。
- `data/workbench_todo_integration_test.go`：正式迁移下的本人/组织/品类隔离、稳定分页、
  草稿/取消/更正/冲回、源费待补审、锁定/退关/结案、异常解决/移除及摘要明细一致。
- `service/order_abnormal_case_test.go`：Mark/Resolve 的 `organizationID, actorID, orderID`
  参数顺序与领域用例一致，不把操作者当订单 ID。
- web：真实 ProTable 分页与重试，返回/重开刷新，真实 OrganizationWorkspace 首帧和迟到隔离；
  真实 Router 下 SE/AE、StrictMode、重复直达和普通重查；异常四项独立权限与写失败反馈。

## 7. 错误与正确示例

错误：`没有有效账单关联 = 未建账`；或把 `abnormal_case.create` 当整个异常面板权限。

正确：

```go
client.OrderFee.Query().Where(
    accountingOrderFeePredicate(),
    fee.Not(fee.HasFinanceBillLinesWith(effectiveBillLinePredicate())),
    // 同时应用本人、公司与同品类 order.read + fee.read 的范围。
)
```

异常入口按 read 展示，create/resolve/delete 各自校验；路由意图只在目标订单已加载且可读时
打开面板，随后 replace 删除 `panel` 并保留其他参数及 hash，普通数据重查不重复弹出。
详情聚合中的人员列表仅在 `personnel.read` 时请求，人员候选仅在 `create` 时请求，
两项能力进入查询键。缺权限显式不请求并隐藏对应内容；实际有权请求失败仍报错，禁止 catch 吞掉 403。
