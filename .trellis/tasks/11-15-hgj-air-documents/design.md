# 航空直单与结构化主分运单：设计

## 负责范围

- `server/api/order/v1/air_export.proto、air_weight.proto、order.proto、order_lock.proto`
- `server/internal/biz/air_*、data/air_*及航空Schema/迁移`
- `web/src/pages/orders/order-kinds/air-export、templates/air-template.tsx及航空详情/主单页`

## 架构与合同

父设计 `.trellis/tasks/11-15-hgj-business-alignment/design.md` 为统一模型/事务/权限合同；本任务实现上述职责，不复制另一套事实真相。

- 航空Link增加DIRECT/HOUSE；DIRECT一MAWB一票且不混HOUSE，真实无HAWB；HOUSE一票一真实HAWB。
- 独立航空结构化主分内容/不可变版本/哈希/审核快照；旧content完整迁入备注，一次性同步源合同与所有调用方。
- 保留MAWB号规则、HAWB批次永久占号及重量确认来源；运输/单证/本票资料分别写入。
- 补AE公共字段UI/双向映射，制作页后续使用同一内容模型；普通保存不可切模式或换主单。

## 事务、迁移与验证

写操作经Data.WithTx/WithinTransaction和Data.client，固定Order先行锁序、锁内完整集合/类型/公司/预期版本复验，提交后重读；不混用原生事务。新增Schema走正式有序SQL，保留旧文本/历史，异常存量显式拒绝，不自动纠错。Proto→绑定/OpenAPI→web客户端、Manifest→权限键同组。

字段/验收详见父研究field-adoption-proposal.md、validation-matrix.md；本设计随父最终方案获批，按前置交付验收顺序实现。

实施授权同步：父最终方案已获用户明确批准，按前置依赖执行；具体完成状态以实际验证结果为准。
