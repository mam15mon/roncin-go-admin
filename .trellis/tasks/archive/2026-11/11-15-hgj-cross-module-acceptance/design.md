# 四类订单横切接通与完整筛选导出：设计

## 负责范围

- `server/api/order/v1/order.proto专用导出与订单read投影`
- `workbench/notification/fee_ledger/report等横切服务`
- `web/order列表导出、当前公司与统一导航/权限入口`

## 架构与合同

父设计 `.trellis/tasks/11-15-hgj-business-alignment/design.md` 为统一模型/事务/权限合同；本任务实现上述职责，不复制另一套事实真相。

- 同步SI/AI工作台/待办/异常/通知深链、财务主单搜索/投影、经营日期/汇率、报表和所有列表跳转；禁止新建另一套名称表。
- 专用全筛选导出接口复用过滤排序及当前公司授权，返回完整DTO/行数/生成时间，withReadSnapshot读取；前端复用CSV工具，不分页拼接。
- 按父任务字段采纳表及V1–V20执行四类业务和财务全流程、正式迁移、权限并发、实际单证输出验收。

## 事务、迁移与验证

写操作经Data.WithTx/WithinTransaction和Data.client，固定Order先行锁序、锁内完整集合/类型/公司/预期版本复验，提交后重读；不混用原生事务。新增Schema走正式有序SQL，保留旧文本/历史，异常存量显式拒绝，不自动纠错。Proto→绑定/OpenAPI→web客户端、Manifest→权限键同组。

字段/验收详见父研究field-adoption-proposal.md、validation-matrix.md；本设计随父最终方案获批，按前置交付验收顺序实现。

实施授权同步：父最终方案已获用户明确批准，按前置依赖执行；具体完成状态以实际验证结果为准。
