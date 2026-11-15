# 海运进口及固定服务进度闭环：设计

## 负责范围

- `新增进口服务Proto/biz/data/Schema/迁移及Order flow派生`
- `海运公共单证/运输/箱货仓储和order_lock/release等适用性`
- `web订单SI类型注册/模板/详情、固定服务UI及HBL制作适配`

## 架构与合同

父设计 `.trellis/tasks/11-15-hgj-business-alignment/design.md` 为统一模型/事务/权限合同；本任务实现上述职责，不复制另一套事实真相。

- 实现SI合法IMPORT建票/详情/修改、真实MBL/航程/HBL/箱货/费用及同方向共享。
- 新增固定每票进口承接五环节、换单代理、实际时间、派生进度与不可变服务事件；无服务与未完成分别表达。
- SI审核快照修订CHECK、恢复/放货引用真实单证；按票适用性参与完结；已结案变更先重开。
- 接通SI适用的海运内容维护/版本/制作归档与HBL输出；不复制海运出口页面或通用历史单证。

## 事务、迁移与验证

写操作经Data.WithTx/WithinTransaction和Data.client，固定Order先行锁序、锁内完整集合/类型/公司/预期版本复验，提交后重读；不混用原生事务。新增Schema走正式有序SQL，保留旧文本/历史，异常存量显式拒绝，不自动纠错。Proto→绑定/OpenAPI→web客户端、Manifest→权限键同组。

字段/验收详见父研究field-adoption-proposal.md、validation-matrix.md；本设计尚待父最终复核批准。
