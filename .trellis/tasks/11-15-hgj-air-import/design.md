# 空运进口业务闭环：设计

## 负责范围

- `AI适用的航空biz/data/service和类型/方向门禁`
- `权威航空时间/地点查询和AI费用数量来源`
- `web订单AI注册/模板/详情/主单入口及服务进度复用`

## 架构与合同

父设计 `.trellis/tasks/11-15-hgj-business-alignment/design.md` 为统一模型/事务/权限合同；本任务实现上述职责，不复制另一套事实真相。

- 注册并实现AI+IMPORT建票/详情/修改，复用航空DIRECT/HOUSE真实实体与固定进口承接进度。
- AI共享主单、货物、危险品、重量确认、费用来源、审核快照及终止恢复按真实类型授权。
- 经营日期为权威ETA，机场筛选/主单时刻/订单投影一致；不复制一套进口费用或重量算法。

## 事务、迁移与验证

写操作经Data.WithTx/WithinTransaction和Data.client，固定Order先行锁序、锁内完整集合/类型/公司/预期版本复验，提交后重读；不混用原生事务。新增Schema走正式有序SQL，保留旧文本/历史，异常存量显式拒绝，不自动纠错。Proto→绑定/OpenAPI→web客户端、Manifest→权限键同组。

字段/验收详见父研究field-adoption-proposal.md、validation-matrix.md；本设计尚待父最终复核批准。
