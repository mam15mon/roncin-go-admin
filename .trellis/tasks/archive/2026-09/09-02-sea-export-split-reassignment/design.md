# 海运出口拆票与改配技术设计

## 设计目标与边界

阶段 4 在阶段 1～3 已有的 `Order`、共享 MBL、真实 HBL 和箱货分配之上增加两类
原子命令：HOUSE 部分拆票和整票改配。实现必须做到结果守恒、历史可重现、失败无
半成品，同时保持少人使用场景的操作简洁：没有审批、审核、解锁申请和补录流程，
只有提交前预览确认、权限、版本检查与自动审计。

本阶段不实现 SI/VGM/舱单外部提交、不可变单证版本、Switch B/L、已确认财务事实的
自动冲正或共享成本分摊；这些分别属于阶段 5、6。禁止用占位字段或兼容分支提前伪造。

## 现有模型复用

- `Order` 继续是操作票；拆出的新票仍是普通 `Order`，不增加子票类型或父 ID。
- `SeaMasterBillOrderLink` 继续表达当前/历史 MBL 关系；一个操作票只有一条 `ACTIVE`。
- `SeaHouseBill`、`OrderContainer` 作为不可跨结果票的完整实体移动归属。
- `SeaCargoAllocation` 是拆票数量的唯一依据；部分拆票只消费 `CONFIRMED` 分配。
- `OrderFee` 只在 `DRAFT` 时允许整行移动，全部财税和汇率快照原样保留。
- `OrderCommissionAttribution` 的现有快照被复制到新票；任何提成事实均不复制。
- 订单号继续使用组织级号码规则和既有 `allocateNumberInTx` 原语。
- MBL 目标继续使用现有“签发主体 + 规范化号码 + 候选版本 + 权威航程”匹配规则。

## 新增持久化模型

### SeaOrderSplitEvent

不可变拆分事实，字段至少包括：

- `organization_id`、`source_order_id`、`source_order_no`；
- `idempotency_key`、`request_fingerprint`，组织内同动作幂等；
- 可空 `note`（最多 500 字）；
- `source_order_version`、`source_link_id`、`source_link_version`、
  `source_allocation_version`；
- `before_snapshot` JSONB：提交锁定后读取的原票、当前 MBL、HBL、货物、实际箱、
  分配、箱计划、费用、附件引用和提成归属摘要；
- `conservation_snapshot` JSONB：原始基准、各结果汇总和零差额断言；
- `created_by`、`created_at`。

事件只追加、不更新、不删除；JSON 快照使用版本化结构，并存原始 UUID、号码、版本和
精确十进制字符串，不能只存显示文案。

### SeaOrderSplitResult

每个事件包含一条 `ORIGINAL` 和一到多条 `CREATED` 结果：

- `split_event_id`、`organization_id`、`order_id`、`order_no`；
- `result_role`（`ORIGINAL|CREATED`）、`sequence`、`client_result_key`；
- `initial_master_bill_id`、`final_master_bill_id`；
- `result_snapshot` JSONB：结果订单、货物旧新 ID 映射、HBL、实际箱、分配、箱计划、
  费用旧新 ID 映射、附件引用和件重尺汇总；
- `created_at`。

`(split_event_id, sequence)`、`(split_event_id, client_result_key)` 唯一。谱系通过事件与
结果递归查询，不在 `Order` 上维护一个会限制再次拆票的单父节点。

### SeaOrderReassignmentEvent

不可变整票改配事实，既可独立创建，也可由拆票结果触发：

- `organization_id`、`order_id`、`order_no`；
- 可空 `split_event_id` 和 `split_result_id`；
- `idempotency_key`、`request_fingerprint`；
- 改配前/目标 `master_bill_id`、`transport_execution_id`、`link_id` 及对应版本；
- `reason`、`responsibility_type`，可空 `responsible_partner_id` 和不可变名称快照；
- `before_snapshot`、`after_snapshot` JSONB，保存航程差异、关系、单证结构和箱货分配；
- `created_by`、`created_at`。

责任类型通过数据库 CHECK/Ent enum 限定为 `CARRIER|CUSTOMER|CUSTOMS|OWN_COMPANY|`
`FORCE_MAJEURE|OTHER`。事件只追加，不承担审批状态。

### 附件资产与引用

把现有单表拆为：

1. `OrderAttachmentAsset`：组织、对象键、文件名、MIME、大小、校验和、上传人和时间；
   `object_key` 全局唯一，一个物理文件只有一条资产。
2. `OrderAttachment`：订单、资产、单证类型、幂等键、创建人和时间；它成为附件引用，
   `(order_id, asset_id)` 和 `(order_id, idempotency_key)` 唯一。

API 继续返回合并后的附件视图，因此现有列表/登记交互不需要理解两个表。新增解除引用
命令，按资产 ID 加锁后删除当前订单引用；若仍有其他引用，物理资产不动；若是最后一条，
在同一事务登记既有对象存储删除任务并删除资产记录。登记、引用、解除和最终删除写审计。

开发库无历史业务数据，正式迁移开头检查旧订单附件以及本阶段涉及的海运业务表为空；
非空立即失败，不做自动回填、双写或旧结构读取。

## 命令契约

新增 `sea_order_change.proto` 与 `SeaOrderChangeService`：

1. `GetSeaOrderChangeActions`
   - 订单 read 权限；只返回是否可拆票/改配及逐项阻断，不返回费用、附件等操作明细，供详情页展示入口。
2. `GetSeaOrderSplitContext`
   - split 权限；返回当前关系、单证结构、箱货聚合、DRAFT 费用、附件、继承字段和版本集合。
3. `PreviewSeaOrderSplit`
   - split 权限；接收结果分配与目标定义，返回精确守恒、箱计划重算、备注、附件和目标航程预览；
     只读且不保留服务器草稿。
4. `ExecuteSeaOrderSplit`
   - split 权限；若任一结果选择不同 MBL，service 再要求 reassign 权限；事务内重验全部事实。
5. `PreviewSeaOrderReassignment`
   - reassign 权限；返回改配前与目标航程逐字段差异、目标成员数和兼容性结果。
6. `ExecuteSeaOrderReassignment`
   - reassign 权限；整票切换当前关系并返回事件和最新订单摘要。
7. `ListSeaOrderChangeEvents`、`GetSeaOrderChangeEvent`
   - 订单 read 权限；分页最大 200，按发生时间、ID 倒序稳定查询拆分/改配历史。

附件契约增加 `RemoveAttachmentReference`，仍使用 `attachment.register` 权限。请求包含
订单 ID 和引用 ID；事务中锁定资产与引用并严格验证归属，不能误删其他订单的引用或资产。

所有写命令必须携带客户端生成的幂等键和请求指纹。相同键、相同指纹重放返回既有事件；
相同键、不同指纹返回 409。预览结果不是提交令牌，提交始终锁后重算。

## 拆票输入与守恒

请求由顶层目标 MBL 定义和结果列表组成：

- 顶层目标用 `client_target_key` 标识，可指向当前 MBL、一个已确认候选，或声明创建新
  MBL；多个结果可引用同一目标键，避免重复创建主单。
- 结果用唯一 `client_result_key` 标识；必须恰有一个原票结果和至少一个新票结果。
- 每个结果提交完整 HBL ID 集合、目标键、草稿费用 ID 集合、附件引用 ID 集合、可选
  企业内部编号和三类备注；不允许客户端直接提交货物拆分数字或箱计划数字。
- 原票与每张新票的结果必须非零；原票不能被清空。

服务端根据锁定后的 `SeaCargoAllocation` 计算：

1. 一张 HBL 必须且只能属于一个结果。
2. 同一实际箱关联的全部分配必须落在同一结果，否则返回具体箱号和冲突 HBL。
3. 每个来源货物行按结果汇总件数、毛重、体积；所有结果之和必须精确等于来源基准。
4. 每条非取消费用必须且只能属于一个结果；取消费用作为原票历史保留，不出现在可分配项。
5. 每个结果至少包含正件数、正重量或正体积的真实货物，不能创建空票。

件数使用整数；重量、体积和费用使用 `decimal.Decimal` 与数据库 numeric 尺度，禁止
epsilon、float 求和、自动舍入或静默纠正。错误携带稳定 reason 以及 `result_key`、
`house_bill_id/house_no`、`container_id/container_no`、`cargo_item_id`、`fee_id`、
`baseline/allocated/difference` 等可定位 metadata。

## 拆票写入算法

锁后校验通过后，在同一事务按以下顺序写入：

1. 创建拆分事件，保存锁后前置快照。
2. 为每个新结果分配新订单号并创建普通 Order：
   - 继承 PRD 白名单和来源流程状态；终止状态 `ACTIVE`、结案状态 `OPEN`、版本 1；
   - 创建人为当前操作人，其他人员继承；服务类型、货物类别和提成归属快照复制；
   - 客户业务编号继承，内部编号使用本次可选输入；
   - 订舱/操作备注默认继承但使用页面最终值；换目标 MBL 时配舱备注默认空；
   - 航程平铺投影取最终 MBL 的权威运输执行，最终目的地仍取原票业务事实。
3. 先为所有新票建立指向来源当前 MBL 的活动关系，使拆分事实具有明确初始关系。
4. 按来源货物行和结果汇总：原票有剩余时更新原行数量和版本，无剩余时删除原行；每个
   新票创建独立货物行。记录全部来源 ID 到结果 ID 的映射。
5. 把 HBL、实际箱整实体移动到结果订单；HBL 此时仍指向来源当前 MBL。
6. 删除来源旧分配行并为原票/新票按新货物 ID 和各自关系重建全部分配，所有结果关系的
   分配状态保持 `CONFIRMED`、聚合版本递增或从 1 建立，并记录拆分确认人/时间。
7. FCL 按真实箱归属重建箱计划：新票等于各箱型真实箱数；未落实箱号的原计划余量只留
   原票；任一结果计划不得小于实际箱数。LCL/散杂不生成箱计划或虚拟箱。
8. 原票费用留存；移到新票的 DRAFT 费用按原字段和标签创建新记录与新幂等键，再删除旧
   DRAFT 记录，事件记录旧新 ID。取消费用和任何下游财务事实不移动。
9. 按用户选择为新票创建附件引用，物理资产不复制；原票引用默认保留。
10. 对目标不是来源当前 MBL 的结果，逐票执行下述改配原语，产生独立改配事件。
11. 写拆分结果、守恒快照、订单“由拆票创建”生命周期事件和业务审计。现有
    `OrderLifecycleEvent` 增加 `ORIGIN` 维度以及可空的 `reference_type/reference_id`，
    新票事件使用 `CREATED_BY_SPLIT` 动作并引用拆分事件，不伪造业务状态流转。

任一步失败，包括事件、订单、号码序列、全部业务写入和审计在内一起回滚。提交后使用
普通上下文重读响应。

## 整体改配算法

独立改配和拆票内组合改配复用同一个 biz 规则与 data 写入原语：

1. 目标可以是已确认候选或新 MBL；目标不得等于当前 MBL。
2. 对已有目标锁后重验组织、签发主体、规范化号码、MBL/运输执行版本和目标权威航程。
   目标已有其他操作票时仍执行相同共享规则；源航程与目标航程不同是预览差异，不是冲突。
3. 保存改配前快照；把旧活动关系更新为 `ENDED`，原因明确标记为整体改配并递增版本。
4. 创建目标活动关系，保持原单证结构、箱货分配状态和确认人/时间。
5. HOUSE 的所有 HBL 切换到目标 MBL；全部分配行切换到新关系。DIRECT 保持无 HBL。
6. 将 Order 的承运人、起运港、卸货港、中转港、船名航次、ETD/ETA 当前投影更新为目标
   运输执行；不改变最终目的地、流程、终止、结案、人员、费用或业务编号。
7. 创建不可变改配事件及审计，提交后重读。

不得用源订单输入更新目标 MBL 或目标运输执行，不修改旧关系/旧航程事件快照。DRAFT
箱货分配保持 DRAFT 及现有合法行；CONFIRMED 保持 CONFIRMED。

## 门禁与错误

共享门禁要求 SE、`ACTIVE`、`OPEN`、存在活动 MBL、MBL/HBL 均为 DRAFT、所有非取消费用
均为 DRAFT，并且没有活动账单行、核销分配、提成行或提成调整。部分拆票额外要求 HOUSE
和 CONFIRMED 箱货分配；整票改配允许 HOUSE 的 DRAFT/CONFIRMED，DIRECT 不要求分配。

服务端返回动作摘要和全部阻断项；提交端仍执行同一门禁。主要错误 reason：

- `SEA_ORDER_SPLIT_INVALID_ARGUMENT`
- `SEA_ORDER_SPLIT_BLOCKED`
- `SEA_ORDER_SPLIT_CONSERVATION_FAILED`
- `SEA_ORDER_SPLIT_ENTITY_CROSSES_RESULTS`
- `SEA_ORDER_SPLIT_VERSION_CONFLICT`
- `SEA_ORDER_SPLIT_IDEMPOTENCY_CONFLICT`
- `SEA_ORDER_REASSIGNMENT_INVALID_ARGUMENT`
- `SEA_ORDER_REASSIGNMENT_BLOCKED`
- `SEA_ORDER_REASSIGNMENT_TARGET_CONFLICT`
- `SEA_ORDER_REASSIGNMENT_VERSION_CONFLICT`
- `SEA_ORDER_REASSIGNMENT_IDEMPOTENCY_CONFLICT`

409 统一提示“数据已被更新，请刷新后重试”，同时用 metadata 指出过期实体。禁止自动重试、
自动选择其他 MBL、静默创建重复 MBL 或跳过冲突项。

## 权限

新增仅适用于海运出口的订单操作：

- `business.order.se.split`，依赖 `business.order.se.read` 和 `business.order.se.update`；
- `business.order.se.reassign`，依赖 `business.order.se.read` 和 `business.order.se.update`。

订单权限定义增加“适用业务类型”过滤，不能顺带生成 SI/AE/AI 的 split/reassign 权限。
拆票接口由中间件校验 split；组合目标存在不同 MBL 时，service 从当前 Principal 再校验
reassign。前端用同一权限集控制入口和目标选择，但后端始终复验。

现有 `attachment.register` 键不变，权限目录名称改为“管理附件”，同时控制登记和解除引用；
这项名称调整适用于现有订单业务类型，不新增 delete 键。修改 Manifest 后重新生成前端权限键。

## 事务与固定锁序

全部写命令由 `biz.Transactor.WithinTransaction` 编排，仓储只经 `Data.client(ctx)`。
首次需要修改的行直接 `FOR UPDATE`，禁止先 SHARE 再升级。固定类别顺序：

```text
来源 Order
→ 组织订单号码序列
→ 涉及的 SeaMasterBill（UUID 升序）
→ 涉及的 Active/Historical Link（UUID 升序）
→ 对应 SeaTransportExecution（UUID 升序）
→ OrderCargoItem（UUID 升序）
→ SeaHouseBill（UUID 升序）
→ OrderContainer（UUID 升序）
→ SeaCargoAllocation（UUID 升序）
→ OrderFee（UUID 升序）
→ AttachmentAsset（UUID 升序）及其 Reference（UUID 升序）
→ 下游门禁事实（各表 UUID 升序）
→ 事件、审计与业务写入
```

同类多行先排序再加锁。锁住 MBL 后、写入前再次确认其版本；锁住 Link 后重验仍是来源
Order 的唯一 ACTIVE 关系。现有会触碰这些实体的写入口必须保持同一类别顺序；若发现存量
顺序不一致，应在本阶段涉及的入口中统一，而不能以重试兜底死锁。

请求至少携带来源订单、当前 Link、分配聚合、全部 HBL/货物/实际箱/费用/附件引用及目标
MBL/运输执行的预期版本或集合指纹。数据库唯一索引继续兜底单活动关系、事件幂等和结果键。

## 页面设计

### 拆票页

新增 `/orders/sea-export/:id/split` 全页路由，并置于通用订单详情动态路由之前。使用
`PageHeaderShell`、默认展开的 `SectionCard` 和 `StickyFooterBar`：

1. 顶部：来源操作票、当前 MBL、流程状态、门禁摘要和可选拆票说明。
2. 分配区：左侧 HBL/货物/真实箱，右侧原票与可添加的新票结果；按完整 HBL 分配，箱跨票
   时立即红色阻断。
3. 每个结果：目标 MBL、内部编号、三类备注、件重尺、箱量计划和航程摘要。
4. 费用区：每条 DRAFT 费用单选归属结果，不提供金额拆分。
5. 附件区：默认仅原票，每个资产可多选新票引用。
6. 即时汇总：蓝色进行中、绿色守恒、红色超出/冲突；错误显示具体号码和差额。

前端使用十进制库计算即时反馈，但提交按钮还要调用服务端 Preview；Preview 通过且用户查看
最终摘要后才能确认 Execute。没有审批人、审批状态或服务器草稿。

### 改配抽屉与历史

订单详情动作区增加“整体改配”，使用较短宽抽屉，展示目标候选、成员数、责任方、必填原因
和逐字段航程差异。确认文案明确“保留当前操作票号和流程状态；当前航程切换为目标 MBL”。

详情页增加“拆票/改配记录”区块，默认展开，按时间展示事件并互链原票、新票、来源 MBL、
目标 MBL。附件抽屉增加“移除当前操作票引用”，显示最后引用时的物理删除提示。

## 生成、迁移与验证

- 修改 Proto 后运行 `make -C server api` 和 `pnpm run generate:web-client`。
- 新增/修改 Ent Schema 后运行 `go -C server generate ./...` 并生成正式 SQL 迁移。
- 修改权限 Manifest 后运行 `pnpm run generate:permission-keys`。
- 所有生成命令重跑必须无新增差异，不手改任何生成文件。
- 迁移只在已获授权的空开发库应用，并核对 enum/CHECK、JSONB、FK、部分唯一索引和幂等索引。

测试至少覆盖：

- biz：多结果精确守恒、HBL/箱跨票、空结果、备注/原因、箱计划重算、权限组合和稳定错误。
- data：字段白名单、人员/归属复制、货物重建、HBL/箱移动、费用快照与标签、附件共享引用、
  当前关系切换、航程投影、审计失败和中途失败完整回滚。
- service/API：UUID、枚举、十进制字符串、metadata、路由静态优先和组合权限不可绕过。
- web：即时剩余量与颜色、精确错误、目标差异、费用单选、附件多选、无审批 UI、成功互链。
- PostgreSQL：一次多结果、组合多目标、DIRECT/HOUSE 改配、相同命令重放、不同指纹冲突、
  两个并发拆票/改配无死锁且只有一个成功、失败无孤儿订单/双活动关系/重复费用。

## 回滚点

阶段 4 作为一个产品提交交付。失败时同时回滚 Schema、迁移、Proto、生成物、后端、页面和
测试；不能只撤页面保留半套命令。迁移空库检查失败时停止，不清理未知数据、不增加兼容兜底。
