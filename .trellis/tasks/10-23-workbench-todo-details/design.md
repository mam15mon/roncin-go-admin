# 技术设计

## 边界与契约

数据流：工作台入口 → 分页明细 RPC → service 解析会话/权限 → biz 校验 → data 查询现有事实 → 只读 DTO → 抽屉 → 原订单办理页。

不建立待办实体。费用/异常仍是唯一业务事实，写操作继续走现有服务；查询只读，无新增事务、锁、版本字段和 Schema 迁移。

在 `server/api/workbench/v1/workbench.proto` 增加两个明确分页接口，沿用工作台列表封套：

- `ListMyUnbilledFees`：GET `/api/v1/workbench/my-unbilled-fees`。
- `ListMyOpenAbnormalCases`：GET `/api/v1/workbench/my-open-abnormal-cases`。

请求只含 page/page_size，复用公共分页默认值和校验。费用记录携带 fee_id、订单最小摘要、方向、名称、结算方、币种、decimal 金额和 created_at；异常记录携带记录 id、订单最小摘要、异常类型 id/名称和 marked_at。两类记录均提供订单 locked/termination_status/closure_status；费用提供待审阻塞原因，异常提供业务只读原因。时间传 RFC3339，金额保留现有精度，前端才格式化。

## 查询范围与投影

service 通过 Principal 生成 WorkbenchScope，保留本人协作、当前公司及订单 read 品类，补充按业务类型的费用/异常读取能力。费用明确要求业务 fee.read，不以 lock 审核入口权限代替逐笔读取；只有 lock 无 fee.read 者计数为零、明细为空且入口隐藏。异常要求 abnormal_case.read，无需新权限码。

data 使用 `Data.client(ctx)` 和 `workbenchMyOrderIDs`，与各类读取能力取交集。摘要和明细共享基础查询：费用 `accountingOrderFeePredicate()` + `Not(HasFinanceBillLinesWith(effectiveBillLinePredicate()))`，异常 ACTIVE；无权限类别计数为零、列表空且不查内容。订单/客户、结算方、异常名称批量装配，避免 N+1。异常类型沿用 A 型共享主数据契约，不能错误按订单组织过滤；停用不影响历史名称。

费用使用现有记账方向投影处理冲回。订单锁定/退关/结案不改变待办事实；批量装配待补审源费状态，复用现有门禁语义解释只读或阻塞：审核锁禁止普通修改但不禁止正常建账，待补审只阻源费用。异常按现行业务可编辑条件说明锁定/退关/结案只读原因。禁止为这次只读投影新增并行状态机。

按 PRD 时间 + 主键稳定升序分页，复用现有 paginate。跨请求并发变化通过刷新收敛，不引入快照机制。

## 页面与直达

- 待办抽屉放在 workbench 目录；ResponsiveProTable 使用 request/actionRef，不维护镜像 data/loading。切类别回第一页，重开重取，错误明确重试。
- 当前工作区控制挂载与缓存隔离，不允许组织切换暂存旧响应。
- 费用链接使用 `workbenchOrderPath(..., true)`，不修改费用表列设置或审核/建账规则。现有财务账单跟踪查询有独立权限，仅在有权时加载，不作为所有费用待办前提。
- 异常用 `?panel=abnormal` 表达入口意图。订单加载成功、身份匹配且可读后打开；消费后 replace 清理该参数，保留其他参数。验证首次进入、已有页签、关闭重开、StrictMode；普通数据重查不再打开，不生成重复订单页签。
- AbnormalCasePanel 拆分 read/create/resolve/delete，头部入口按 read 展示，锁只禁写。补充异常可读名称投影到既有响应供原面板与待办消费，修改 Proto 后生成，不额外获取全量主数据，也不以 UUID 充当名称。
- 异常登记使用登录可读的 `MasterDataService.SearchOptions`（`/api/v1/master-data/options/search`），
  明确 kind、keyword 与分页，固定仅启用项；沿用原公共候选授权，不能误用要求主数据管理读取权限的 ListItems。
- 只读异常直达不依赖人员列表或订单创建权限。详情聚合按 `personnel.read` 与 `create`
  分别请求人员列表和候选，能力值进入 queryKey；无权区块隐藏、无候选字段只读，不吞掉实际请求错误。
- 返回工作台/重开抽屉时按现有路由生命周期重取聚合和列表；协调 React Query、ProTable，不添加轮询或全局事件总线。处理页本身仍使用现有 actionRef 刷新。

## 交付与风险

一次性同步服务端、前端、生成物及桩实现，不写历史格式兼容分支。按本任务提交整组回退，无数据库回退事项。

摘要权限收紧可能使少数员工计数减少，应作为授权口径修正验证。异常面板拆权限时重点覆盖只读/仅解决/无删除/业务锁组合；页签缓存下的参数消费和返回刷新需要真实路由测试。实施前刷新 git 状态，在独立分支工作，保留其他任务修改。
