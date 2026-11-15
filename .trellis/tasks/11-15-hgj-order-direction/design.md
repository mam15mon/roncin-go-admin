# 四类订单身份与主单方向隔离：设计

## 负责范围

- `server/internal/data/ent/schema/{sea_master_bill,air_master_waybill}.go`
- `server/internal/data/*master*、订单候选/同步/类型校验`
- `server/api/order/v1/order.proto、air_export.proto、必要权限Manifest`
- `海空候选/主单类型DTO和对应前端调用方`

## 架构与合同

父设计 `.trellis/tasks/11-15-hgj-business-alignment/design.md` 为统一模型/事务/权限合同；本任务实现上述职责，不复制另一套事实真相。

- 海/空主单不可变business_type、方向唯一/查询索引，按实际现有链接检查并一次性回填；未知或混向数据停止迁移。
- 候选/ID关联/预览/恢复均同公司同类型；保持SE/AE已有能力，不仅改前端菜单。
- 补公共字段的源合同、适用性和完整映射入口；后续新增进口入口不得出现能保存但无真实单证的中间假闭环。

## 事务、迁移与验证

写操作经Data.WithTx/WithinTransaction和Data.client，固定Order先行锁序、锁内完整集合/类型/公司/预期版本复验，提交后重读；不混用原生事务。新增Schema走正式有序SQL，保留旧文本/历史，异常存量显式拒绝，不自动纠错。Proto→绑定/OpenAPI→web客户端、Manifest→权限键同组。

字段/验收详见父研究field-adoption-proposal.md、validation-matrix.md；本设计尚待父最终复核批准。
