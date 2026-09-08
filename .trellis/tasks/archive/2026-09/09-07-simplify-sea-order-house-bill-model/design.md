# 简化海运出口主分单模型技术设计

## 设计目标

以订单作为操作和结算边界，以 MBL 作为主单身份，以 TransportExecution 作为实际航次事实，
形成以下稳定关系：

```text
SeaMasterBill（船公司 + MBL 身份/正文）
       │ 1
       │
       │ N
SeaMasterBillOrderLink（当前/历史订单关系，同时指向实际执行）
       ├──────── Order（客户业务号、Booking No.、独立账单）
       └──────── SeaTransportExecution（船公司、港口、船名航次、ETD/ETA）
                         │
                         └── N 个活动订单关系

HOUSE Order ── 1 个当前 SeaHouseBill；DIRECT Order ── 0 个当前 SeaHouseBill
历史作废 HBL/版本继续保留，不计入“当前”数量。
```

客户业务号和 Booking No. 是查询键，不是聚合根，也不触发共享写入。共享字段只由明确外键关系
决定，避免相同文本号码导致误更新。

## 数据模型

### MBL 与运输执行解耦

- 从 `SeaMasterBill` 删除 `transport_execution_id` 及对应边/索引；保留
  `organization_id + shipping_line_id + normalized_master_no` 唯一身份。
- 在 `SeaMasterBillOrderLink` 增加必填 `transport_execution_id`，Link 表达某张订单当前/历史
  同时使用的 MBL 与实际航次。活动订单仍由 `order_id WHERE status='ACTIVE'` 条件唯一约束兜底。
- MBL 与 Link 指向的 TransportExecution 必须属于同一组织且 `shipping_line_id` 相同；Ent 外键
  不能表达跨表相等，所有创建/改配/共享修改路径在事务锁内重验，集成测试覆盖。
- 同一 MBL 可被不同 Link 配到不同 TransportExecution，从而表达同号 MBL 下部分订单甩柜到后续
  航次。一个 TransportExecution 也可服务多张 MBL 和多张订单。
- SE 订单不再把 `Order.shipping_line_id`、港口、船名航次、ETD/ETA 作为写入真相；响应和列表
  通过活动 Link → TransportExecution 投影这些字段。通用 Order 字段因其他运输类型仍使用而
  保留，但 SE 创建/更新不得双写。最终目的地及四类截点仍是订单自身事实。

### 运输执行不可变版本

新增 `SeaTransportExecutionVersion`，保存运输执行身份、来源实体版本、ShippingLine、港口、
船名、航次、ETD/ETA、内容散列、形成原因、原因、操作人和时间。`SeaTransportExecution` 增加
可空 `current_version_id`。

形成原因至少包括：

- `ORDER_LOCK`：订单锁定时创建或复用当前执行快照；
- `SHARED_UPDATE`：确认后统一修改共享航次；
- `REASSIGNMENT`：新建或选定目标执行并完成单票改配。

`OrderLockRecord` 增加 `transport_execution_id` 与 `transport_execution_version_id`；SE 锁定时
MBL 版本、运输执行版本均必填，非 SE 均为空。`SeaMasterBillVersion` 只保存 MBL 身份与正文，
删除单一 TransportExecution 及航次快照，防止共享 MBL 版本代表错误的单票航次。

### HOUSE/DIRECT 与唯一当前 HBL

- `SeaMasterBillOrderLink.document_structure` 只保留 `HOUSE | DIRECT`，无默认
  `UNDETERMINED`；创建 Link 时必须显式给值。
- HOUSE 建单请求必须包含一张 HBL 且 HBL 号非空；Order、Link、HBL、初始日志在同一共享事务
  中写入。DIRECT 请求不得携带 HBL。
- `SeaHouseBill` 允许保留同订单历史身份，但建立
  `order_id WHERE status IN ('DRAFT','CONFIRMED','RELEASED')` 条件唯一索引，保证至多一张当前
  HBL。HOUSE 事务提交时必须恰有一张当前 HBL，DIRECT 必须为零。
- 删除 `REPLACED` 状态和所有 Switch 来源；保留 `DRAFT | CONFIRMED | RELEASED | VOIDED`。
- 普通页面和 API 使用单值 `house_bill`，不再使用集合、Add/Remove 或“删除最后一张”语义。
  HBL 号、签发来源、本公司/同行主体、正文、附件、当前版本和历史版本继续保留。
- HOUSE→DIRECT：锁定 Order → MBL → Link → 当前 HBL，生成 VOID 版本/事件，将 HBL 置
  `VOIDED`，更新 Link 模式并追加模式变更事件。
- DIRECT→HOUSE：请求必须携带已取得的 HBL 号、签发主体及可选正文；事务内创建新当前 HBL，
  形成初始 `MODE_CHANGE` 不可变版本，更新 Link 并追加模式变更事件。
- 新增不可变 `SeaDocumentModeChangeEvent`，保存前后模式、前后 HBL/版本引用、原因、幂等键、
  请求指纹、外部确认和操作人。时间、ETD 与流程状态只进入提示/影响摘要，不做硬阻断。

### 外部确认事实

共享运输修改、正式改单、换配、HOUSE/DIRECT 切换统一接收：

- `confirmed_by_party`：非空文本，表示船代/船司/承运方；
- `confirmed_at`：有效时间；
- `confirmation_note`：去空白后非空的简短说明；
- `confirmation_attachment_id`：可空，必须是当前组织、当前订单的真实附件引用。

确认字段直接固化在对应不可变版本或事件中，不增加审批状态：

- MBL/HBL 单改：写入新 DocumentVersion；
- 单证作废：写入 `SeaDocumentVoidEvent`；
- 换配：写入 `SeaOrderReassignmentEvent`；
- 模式切换：写入 `SeaDocumentModeChangeEvent`；
- 共享航次修改：写入 `SeaTransportExecutionVersion`。

附件外键使用 `NO ACTION`，已成为确认凭证的附件引用不得被普通解除引用命令删除。API 历史响应
输出确认摘要和附件元数据，不输出文件正文。该模型没有待审批、批准人、拒绝或回退状态。

### Booking No. 与同批订单查询

- `Order` 新增可空 `booking_no`（最长 100），保存原值并以组织组合索引支持查询。
- `CreateOrderRequest`、`UpdateOrderRequest`、Order DTO 和页面增加 Booking No.。
- 列表复合号码类型增加 `CUSTOMER_REFERENCE` 与 `BOOKING`；`MASTER` 继续通过活动 MBL Link
  查询。详情页“同批订单”区域按当前订单已有的三个键分别查询并合并去重，展示每条命中的来源。
- 客户业务号关联使用 `organization_id + customer_id + customer_reference_no`，允许同一客户多票
  重复，现有“重复校验”改为“查看同业务号订单”而不是错误警告；Booking No. 使用
  `organization_id + booking_no`，MBL 使用活动 Link 的真实 `master_bill_id`。
- 相同客户业务号或 Booking No. 不设置 `is_shared`，不创建关系表，不触发批量更新；MBL 共享
  状态仅由当前活动 Link 数量计算。

### 常态箱归属与共享箱例外

- 删除旧 `SeaCargoAllocation` 及 Link 上的 cargo-allocation 状态/版本。常态箱仍使用
  `OrderContainer` 直接归订单，无需额外分配。
- 新增 `SeaSharedContainer`，字段包含组织、TransportExecution、箱号、箱型、封号、总件数、
  总重量、总体积、备注和版本；同一执行内箱号唯一。它不属于单个 Order。
- 新增 `SeaSharedContainerAllocation`，指向 SharedContainer、Order、当前 HBL 与 OrderCargoItem，
  保存件数、重量、体积。只允许 HOUSE、同一组织、且各订单活动 Link 指向该 SharedContainer
  的 TransportExecution；每张 HBL/货物可有多行，但组合键防重复。
- 保存草稿时拒绝任一来源货物或共享箱超分；确认时要求箱维度与各订单货物维度严格守恒。
  件数用整数，件重尺沿用精确 numeric/decimal，不用 float epsilon。
- 普通订单只显示独占箱。用户显式选择“共享箱/客户拼货”时打开跨订单工作台，通过活动
  TransportExecution 选择其他 HOUSE 订单；DIRECT 和不同执行的订单不能加入。
- SharedContainer 自身保存 `DRAFT | CONFIRMED`、聚合版本、确认人和确认时间。涉及的 Order、
  HBL、CargoItem、SharedContainer/Allocation 按 UUID 排序锁定并重验，失败整体回滚。

### 拆票与换配

- 拆票结果是一张保留原订单加一张或多张新订单；每张 HOUSE 结果都必须提交唯一 HBL 号及签发
  主体。输入改为按结果显式分配来源货物数量、独占箱、草稿费用和附件引用，不再通过“把多张
  HBL 分组”推导结果。
- 原订单和新订单均非空，所有货物件重尺严格守恒；独占箱必须完整归一张结果订单。已存在的共享
  箱只调整跨订单 allocation，不复制物理箱。
- 已确认的账单、发票、核销和提成事实不自动分配；若拆票无法保持原订单归属则 Preview 明确
  阻断，由业务先处理。草稿费用只有在用户逐行指定目标订单时才能移动。
- 改配目标明确包含 `master_bill_id + transport_execution_id`：
  - ShippingLine 不变时，可以保留当前 MBL，只把订单 Link 切到新/已有执行；
  - ShippingLine 改变时，必须同时选择/创建目标 MBL 与目标执行，两者船公司一致；
  - 不允许只更换 MBL 号而船公司与执行均不变化。
- 旧 Link 结束、新 Link 创建、HBL 当前 `master_bill_id` 调整、执行版本、改配事件和审计同事务
  完成；原 Link、原执行及其他订单不变。

## API 与页面收敛

- 以 `.proto` 为唯一契约源，删除 Switch Preview/Execute、Add/Remove HBL、Mark/Cancel Direct
  的旧路径，替换为单值 HBL 更新和显式 `ChangeSeaDocumentMode` Preview/Execute。
- 被删除的消息字段和枚举编号/名称全部 `reserved`；移除 RPC 后同步删除 Service/Biz/Data、
  HTTP 注册、生成客户端、权限依赖和前端调用，不留返回 200 的兼容空壳。
- `SeaOrderDocuments.house_bills` 改为可空单值 `house_bill`，摘要返回 `house_no`；历史 API 仍能
  按订单读取全部失效身份和不可变版本。
- 订单创建页必须先选 HOUSE/DIRECT；HOUSE 紧邻展示唯一 HBL 号和签发主体，DIRECT 显示
  “直接向客户交付 MBL”。详情页只有一个“分单信息”区块。
- 共享运输修改、改单、作废、模式切换和改配均保留 Preview→Execute：Preview 展示影响订单、
  历史/财务/放货事实和风险提示；Execute 重新锁定和校验外部确认，不信任缓存 Preview。
- “同批订单”视图替代容易误解的号码派生聚合，分别标明“同客户业务号”“同 Booking No.”
  “同 MBL”；只提供跳转和只读汇总，不提供按文本分组批量修改。

## 下游事实与历史

- 单改、换配和模式切换不得因为 ETD、实际离港、已放单或存在账单而一概返回 409；这些事实作为
  Preview 影响项展示。
- 携带完整外部确认并通过专用权限的共享运输修改、改单、换配和模式切换，是普通业务锁的显式
  例外：即使 `Order.locked_at` 非空也可执行，但不得自动解锁或覆盖 `OrderLockRecord`。旧锁定
  快照继续代表当时版本，新命令追加当前版本/事件；普通订单编辑仍受业务锁阻断。
- 已有费用、账单、发票、核销、提成和放货记录保持原 Order、原 Document ID/Version ID 关联，
  命令不得自动改写、删除、移动或红冲。
- 只有请求会违反外键、唯一性、组织隔离、当前 HBL/Link 不变量，或无法保留历史归属时才阻断。
- ReleasePod 和锁定快照继续引用历史 HBL/MBL/版本；HOUSE→DIRECT 不级联删除放货事实。
- MBL 共享内容变更锁定所有活动成员 Order；TransportExecution 共享变更锁定所有活动 Link 对应
  Order，均先按 Order UUID 排序，锁后重验成员集合。

## 事务与固定锁序

所有组合写入由 `biz.Transactor.WithinTransaction` 编排，仓储只使用 `Data.client(ctx)`。统一类别
锁序为：

```text
Order（UUID 升序）
→ SeaMasterBill（UUID 升序）
→ SeaMasterBillOrderLink（UUID 升序）
→ SeaTransportExecution（UUID 升序）
→ SeaHouseBill（UUID 升序）
→ OrderCargoItem / OrderContainer（各自 UUID 升序）
→ SeaSharedContainer / Allocation（各自 UUID 升序）
→ OrderFee 与只读下游事实
→ 不可变版本、事件、审计
```

首次需要修改的行直接 `FOR UPDATE`。所有请求携带对应 `expected_version` 和幂等键；版本冲突返回
409“已被更新，请刷新后重试”，参数/模式不一致返回 400。唯一性继续由 PostgreSQL 索引兜底，
`ent.IsConstraintError` 映射为稳定领域错误。

## 迁移、发布与回滚

- 新迁移第一段在任何 DDL 前检查 SE Order、MBL、Link、TE、HBL、旧 Allocation、版本、锁快照、
  Split/Reassignment/Void/Switch 事件及相关 Sea ReleasePod；任一非空则抛错并保持 Schema 不变。
- 空库前提下直接建立目标结构，不双写、不兼容读取、不猜测模式或航次归属。
- 修改 Ent Schema 后生成 Ent；修改 Proto 后生成 PB/OpenAPI，再运行 Web Client、枚举和必要权限
  生成。所有源文件与生成物同组提交。
- 代码回滚以阶段提交为单位；已执行目标迁移且产生新 SE 数据后，不支持自动降级到旧的一 MBL
  一航次/一订单多 HBL 模型。发布前用独立 PostgreSQL Schema 演练全迁移。
