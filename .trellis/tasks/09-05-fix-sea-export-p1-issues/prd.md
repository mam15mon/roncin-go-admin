# 修复海运出口四项 P1 问题

## 目标与用户价值

消除海运出口正式使用报告中保留的四项真实软件故障，使单证、放货凭证、箱货删除和并发拆票不再依赖人工规避。修复后，操作人员从列表或详情进入对应功能时应命中真实海运模型；合法删除应携带并发版本；同一版本的并发拆票失败方应稳定得到可恢复的 409 冲突。

## 已确认事实

- 当前业务仍采用人工判断后点击 UI 流程按钮的简单模式；本任务不改变订单流程状态、结案门禁或外部渠道边界。
- SE 列表的“主分单据管理”仍打开旧 `OrderShippingDocument` 抽屉，而后端禁止 SE 使用该旧模型；真实 MBL/HBL 位于订单详情的海运单证区。
- `OrderReleasePod.shipping_document_id` 只关联旧 `OrderShippingDocument`；真实海运 MBL/HBL 分别由 `SeaMasterBill`、`SeaHouseBill` 表达。
- 每条 ReleasePod 当前最多关联一张单证，且允许不关联；本任务延续该交互，不引入一条凭证关联多张单证。
- 用户要求在海运单证页面直接展示每张真实 MBL/HBL 已关联的放货记录；删除可删除的分单时，不应只给出外键错误或静默清空关系，而应列出关联记录并询问是否连同删除。
- `RemoveContainerRequest` 与 `RemoveCargoItemRequest` 已要求非零 `expected_version`；两个前端 `removeItem` 请求目前只传 `orderId` 与 `id`。更新请求已正确传版本，不属于缺陷。
- `ExecuteSeaOrderSplit` 的无锁 Preview 发生在事务内版本检查之前；竞争胜方提交后，失败方可能先被 Preview 按当前可变状态判成“分单数量不足”，返回 400 `SEA_ORDER_SPLIT_INVALID_ARGUMENT`，而不是 409 `SEA_ORDER_SPLIT_VERSION_CONFLICT`。
- 当前并发拆票仍保证一成功一失败，没有双成功、重复订单或重复写入；本任务修复错误优先级和恢复语义，不重写拆票业务模型。
- 工作区存在其他窗口的未提交前端改动；本任务必须逐文件隔离，不覆盖或提交这些改动。

## 需求

### R1：修正 SE 列表单证入口

- SE 列表的单证动作不得再打开 `ShippingDocumentDrawer` 或调用旧 `OrderShippingDocument` 接口。
- SE 动作应以“海运单证”语义进入当前订单详情，由详情页真实 `SeaDocumentSection` 维护 MBL/HBL。
- 非 SE 订单继续使用现有主分单据抽屉，交互和接口不得被破坏。
- 公共列表模板不得硬编码一套仅适用于 SE 的第二业务判断；由调用方提供动作文案/行为或使用现有可复用扩展点。

### R2：ReleasePod 支持真实海运 MBL/HBL

- 每条 ReleasePod 允许以下三种互斥状态：不关联单证、非 SE 关联旧 `OrderShippingDocument`、SE 关联当前订单可见的真实 MBL 或真实 HBL。
- SE 关联 MBL 时，MBL 必须是该订单当前活动 `SeaMasterBillOrderLink` 指向的 MBL；共享 MBL 可以被其每个成员订单的 ReleasePod 合法引用。
- SE 关联 HBL 时，HBL 必须属于同组织、同订单，并属于该订单当前活动 MBL。
- 非 SE 请求不得提交海运单证引用；SE 请求不得继续提交旧 `shipping_document_id`。
- API 必须显式表达海运单证类型和 ID；数据库必须保留真实外键，并约束旧单证、Sea MBL、Sea HBL 三种引用最多存在一个。
- 新增和更新必须在订单锁定、组织隔离与状态门禁生效的同一事务中校验引用；禁止事务外先查后写或回落到 `data.db`。
- 前端根据订单业务类型加载正确候选：SE 使用 `GetSeaOrderDocuments` 返回的当前 MBL/HBL，非 SE 沿用旧分单查询；加载失败必须明确展示并阻止带错误引用保存。
- 列表、编辑回显和保存后展示必须能区分“MBL: 单号”与“HBL: 单号”。
- 海运订单详情的单证区必须按 MBL/HBL 展示关联放货记录，包括放货编号、回单编号和当前状态；没有记录时显示明确空状态。
- 删除一张允许硬删除的 HBL 时，如果存在关联放货记录，页面必须先列出这些记录并询问是否一并删除；用户取消时 HBL 和放货记录均保持不变。
- 用户确认关联删除后，HBL、关联放货记录和操作日志必须在同一事务中完成；任一步失败全部回滚，不允许只删掉其中一部分。
- 如果任一关联放货记录已进入“已回单”，该 HBL 和全部关联放货记录均禁止删除；页面必须展示阻断记录和原因，不再提供关联删除确认。
- 只有关联放货记录全部处于“待签收”或“已签收”时，才允许用户确认后一并删除。
- MBL 当前没有硬删除命令，作废 MBL/HBL 时保留已关联放货记录及关联关系，用于历史追溯；关联删除只进入现有 HBL 硬删除场景。
- 不自动猜测或迁移历史旧 SE 引用，不增加双写、静默兼容或无外键的通用 UUID 引用。

### R3：箱和货物删除携带真实版本

- `ContainerDrawer` 删除请求必须发送当前记录的 `version` 作为 `expectedVersion`。
- `CargoItemDrawer` 删除请求必须发送当前记录的 `version` 作为 `expectedVersion`。
- 版本缺失或为零时前端必须失败关闭，不得使用 `'1'` 等猜测值发起删除。
- 后端现有 409 冲突契约保持不变，由统一请求错误处理提示用户刷新；不增加自动重试。

### R4：并发拆票稳定返回版本冲突

- Execute 的静态输入校验仍可在事务前执行；依赖当前订单、Link、HBL、箱货分配和费用状态的校验不得抢在过期版本判定之前产生 400。
- Preview 作为独立只读功能仍应返回完整业务校验；Execute 必须以事务锁内重验作为最终权威。
- 两个不同幂等键、相同预期版本的并发拆票应稳定为一个成功、一个 409 `SEA_ORDER_SPLIT_VERSION_CONFLICT`。
- 修复不得把真正的静态非法输入、守恒失败或业务门禁一律伪装成版本冲突，也不得通过自动重试隐藏并发。

### R5：契约、迁移与生成物一致

- 先修改 `.proto` 与 Ent Schema 真相源，再运行既有生成命令；禁止手改 Protobuf、OpenAPI、Ent 或 Web Client 生成物。
- ReleasePod 新列、外键、索引与互斥 CHECK 必须通过正式 PostgreSQL 迁移交付，并与 Ent Schema/生成元数据保持一致。
- 不新增权限码；现有 `release_pod.*`、订单 read/update、箱货 delete 和 sea split 权限继续生效。
- 不修改已批准的简单人工流程、外部提交、共享费用分摊或外部文件库边界。

## 验收标准

- [ ] AC1：从 SE 列表点击单证动作会进入对应订单详情并使用真实海运单证区，不会请求旧 `OrderShippingDocument`；非 SE 原抽屉行为通过回归测试。
- [ ] AC2：SE ReleasePod 可选择并保存当前共享 MBL，列表和编辑回显显示正确 MBL 号码。
- [ ] AC3：HOUSE 订单的 ReleasePod 可选择并保存本订单真实 HBL；其他订单、其他组织、非当前 MBL 的 HBL 均被明确拒绝。
- [ ] AC4：同一 ReleasePod 同时提交旧单证、Sea MBL、Sea HBL，或提交类型与 ID 不完整时返回稳定 400，且事务无业务写入和操作日志残留。
- [ ] AC5：非 SE ReleasePod 旧分单关联保持可用；SE 不能再保存旧 `shipping_document_id`。
- [ ] AC6：海运单证页面在 MBL/HBL 下展示各自关联的放货记录及状态；新增、编辑、解除或关联删除后页面数据同步刷新。
- [ ] AC7：删除有关联放货记录的可删除 HBL 时，页面列出关联记录并二次确认；取消后零变更，确认后 HBL、处于待签收/已签收的关联记录和日志原子删除。
- [ ] AC8：任一关联记录为“已回单”时，前后端均阻止删除 HBL 及其所有关联记录，页面展示阻断记录；并发变成已回单时服务端仍能原子阻断。
- [ ] AC9：作废 MBL/HBL 不删除放货记录；记录仍能显示其历史关联。
- [ ] AC10：ReleasePod Schema、正式迁移、生成元数据、Proto/OpenAPI/Web Client 完整一致，重跑生成命令无新增差异。
- [ ] AC11：箱和货物删除请求分别断言发送记录真实 `expectedVersion`；缺失/零版本时不调用 API；版本冲突不自动重试。
- [ ] AC12：并发拆票真实 PostgreSQL 测试连续至少 3 次均为一成功一 409，完整 `TestSeaOrderSplitAndReassignment_PostgresIntegration` 父测试至少 `-count=3` 通过。
- [ ] AC13：静态非法拆票输入仍返回 400，守恒失败仍返回对应 400；过期 Order/Link/Allocation/HBL/货物/箱/费用版本返回 409。
- [ ] AC14：ReleasePod 前后端针对性测试、Go 全量测试与 vet、Web 测试与 TypeScript 检查按风险通过；无法运行的真实 PostgreSQL 检查必须明确记录原因。
- [ ] AC15：本任务仅提交计划内文件；其他窗口的未提交改动和 `clash-for-linux/` 均未被覆盖或纳入提交。

## 不在范围内

- 外部订舱、报关、SI、VGM、舱单提交或自动回执。
- 新增实际开航、到港、交付、操作完成等订单流程状态。
- 为每张 MBL/HBL 新增确认、签发、放单状态流。
- 共享费用自动分摊、系统内完整文件库、复杂节点/SLA/异常工单。
- 一条 ReleasePod 同时关联多张单证、ReleasePod 状态机重构或新增审批。
- 对未知历史 SE 旧单证引用做自动转换、双写或静默兜底。

## 技术约束

- 后端遵循 API → service → biz → data 分层；Ent 类型不得泄漏到 biz/service。
- 仓储读取使用 `Data.client(ctx)`；写入使用统一事务封装，跨仓储事务使用 `biz.Transactor`。
- ReleasePod 写入锁序从 Order 开始；真实 MBL/HBL 引用在锁内按当前活动关系重验。
- 生成文件只由生成器更新；前端仅调用 OpenAPI 生成客户端。
- 文档、设计、提交说明与开发者注释使用中文。
