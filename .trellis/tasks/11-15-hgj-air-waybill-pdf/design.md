# 航空主分运单制作与PDF：设计

## 负责范围

- `航空打印Proto/biz/data当前内容/历史/输出元数据（必要时）`
- `web/orders航空制作/模型/PDF与真实共享features`
- `路由/标题/权限入口及静态字体资源复用`

## 架构与合同

父设计 `.trellis/tasks/11-15-hgj-business-alignment/design.md` 为统一模型/事务/权限合同；本任务实现上述职责，不复制另一套事实真相。

- 主/分航空制作页绑定明确单证ID/版本，结构化字段、费率/金额与声明文本分别编辑，保存/签发事实/打印分开。
- 一套A4航空版式覆盖取证分区，MAWB/HAWB身份差异明确；长文/多行续页，同域字体及当前真实主体印字。
- 预览/下载/打印同Blob与保存快照，指纹含内容/公司/模板；15秒前台及回焦检查、未保存输入保护。
- 只提取与现有HBL真实共用的PDF生命周期能力，保持模块边界和既有海运输出。

## 事务、迁移与验证

写操作经Data.WithTx/WithinTransaction和Data.client，固定Order先行锁序、锁内完整集合/类型/公司/预期版本复验，提交后重读；不混用原生事务。新增Schema走正式有序SQL，保留旧文本/历史，异常存量显式拒绝，不自动纠错。Proto→绑定/OpenAPI→web客户端、Manifest→权限键同组。

字段/验收详见父研究field-adoption-proposal.md、validation-matrix.md；本设计尚待父最终复核批准。
