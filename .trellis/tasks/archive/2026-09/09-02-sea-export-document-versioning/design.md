# 海运出口单证版本、订单锁与换单技术设计

## 设计目标与边界

阶段 5 在阶段 1～4 已有的共享 MBL、真实多 HBL、箱货定量分配、拆票和整体改配之上，
完成三件互相关联但职责不同的能力：

1. 用不可变版本固定每次订单封存时实际使用的 MBL/HBL 内容；
2. 用唯一的订单级业务编辑锁阻止封存后的所有业务修改，并提供角色直解、系统管理员
   紧急直解和普通编辑人钉钉审批三条清晰分流；
3. 在订单未锁定时提供带差异预览的改单、作废和 Switch B/L，永久保留旧版本与替代链。

本阶段不新增 SI、VGM、舱单人工状态，不伪造船公司、海关或港区回执，不实现共享费用
自动分摊、账单红冲、核销撤销或提成重算。订单解锁只解除业务编辑锁，绝不绕过费用、账单、
发票、核销和提成各自已有的财务不可变门禁。

## 现有事实与复用边界

- `Order` 就是操作票，已有 `locked_at` 和 `version`，但没有正式锁定命令、锁定代次、
  锁定人、解锁请求或批准事实。
- `SeaMasterBill` 是共享 MBL 身份和当前工作内容；`SeaHouseBill` 是单个操作票下的真实 HBL
  身份和当前工作内容。二者已有乐观版本，但尚无不可变内容版本实体。
- `SeaMasterBillOrderLink` 表达当前/历史 MBL 成员关系，一个活动操作票只有一个 `ACTIVE`
  关系；共享 MBL 修改会影响所有当前成员。
- 阶段 4 的拆票与改配已经使用 Order → MBL → Link → 子实体的固定锁序和不可变事件；
  本阶段必须复用该顺序，不能另建一套相反锁序。
- `AuditLog` 是追加式业务审计；锁定、解锁、改单、作废、Switch 和钉钉结果生效必须与对应
  业务写入同事务记录成功审计。
- 现有 DingTalk 代码只提供登录身份解析和一对一机器人文字通知；它不能承担审批。现有
  `BackgroundTask` 的租约、领取和重试框架可以复用，但审批请求、审批实例与回调事件使用
  独立模型、适配器和 Worker。
- `User` 已保存 `dingtalk_userid`，可用于钉钉审批人映射；系统初始化账号目前没有稳定标记。
- 开发库没有历史业务数据。本阶段使用正式迁移重建新增结构，不做用户名猜测、旧模型双写、
  隐式回填或兼容读取。

## 领域关系

```text
User(is_bootstrap_admin)
  ├─ Membership ─ RoleAssignment ─ 非 administrator 业务角色 ─ lock 权限
  ├─ OrderLockRecord.locked_by / unlocked_by
  └─ OrderUnlockRequest.requested_by / decided_by

Order
  ├─ current lock state: locked_at / locked_by / lock_generation
  ├─ 1 ── N OrderLockRecord（每个锁定代次一条）
  │          ├─ 1 ── 1 SeaMasterBillVersion（本代次使用的共享 MBL 版本）
  │          └─ 1 ── N OrderLockHouseBillSnapshot ── 1 SeaHouseBillVersion
  └─ 1 ── N OrderUnlockRequest ── N OrderUnlockApproverCandidate

SeaMasterBill(identity + working content)
  └─ 1 ── N SeaMasterBillVersion（不可变）

SeaHouseBill(identity + working content)
  ├─ 1 ── N SeaHouseBillVersion（不可变）
  └─ 1 ── 0..N SeaHouseBillSwitchEvent（old → new）

OrderUnlockRequest
  ├─ 0..1 BackgroundTask(kind=DINGTALK_APPROVAL_CREATE)
  ├─ 0..1 DingTalk approval process instance
  └─ 1 ── N DingTalkApprovalInboxEvent（验签后幂等入站）
```

`Order` 的三个当前字段只用于快速判断和列表展示；`OrderLockRecord`、不可变单证版本、解锁
请求和审计才是历史真相。删除或清空当前锁字段不得删除任何历史记录。

## 持久化模型

### User 初始化管理员标记

在 `User` 增加：

- `is_bootstrap_admin bool NOT NULL DEFAULT false IMMUTABLE`。

只有 `cmd/bootstrap-admin` 创建初始化账号时显式写 `true`。普通用户创建/更新 Proto、service、
biz 和 data 均不接受该字段，不能通过改用户名、显示名、角色或直接调用管理接口改变它。

初始化管理员的规则是：

- 不属于业务锁定/解锁角色，不进入该角色人数和钉钉候选人计算；
- 可以在自身数据范围内执行系统管理员紧急解锁，不创建钉钉审批；
- 紧急解锁审计必须记录实际操作人和时间，原因可选；
- 兜底只适用于解锁，不能借管理员全量权限被识别为日常锁单人或业务审批人。

真实员工即使同时拥有 `administrator` 角色，也只有在另行分配了包含锁定权限的非
`administrator` 启用业务角色时，才属于锁定/解锁角色。

### Order 当前锁状态

保留 `locked_at`，增加：

- `locked_by uuid NULL`：当前锁定实际操作人；
- `lock_generation uint64 NOT NULL DEFAULT 0`：成功锁定一次加一，解锁不减；
- User 边和 `(organization_id, locked_at)` 既有索引继续使用。

锁定和解锁都必须递增 `Order.version`。判断“同一轮锁”必须同时使用 `order_id +
lock_generation`，不能只比较时间或布尔值。

### OrderLockRecord

每次成功锁定追加一条锁定周期事实：

- `organization_id`、`order_id`、`order_no`；
- `generation`，与锁定后的 `Order.lock_generation` 相等；
- `locked_by`、`locked_at`、`order_version_at_lock`；
- `master_bill_id`、`master_bill_version_id`；
- 可空 `unlocked_by`、`unlocked_at`、`order_version_at_unlock`；
- 可空 `unlock_request_id`；
- 可空 `unlock_reason`；
- `unlock_mode`：`ROLE_DIRECT | ADMIN_EMERGENCY | DINGTALK_APPROVED`，未解锁时为空。

唯一约束为 `(order_id, generation)`。锁定字段创建后不可改；解锁字段只能从空写入一次。
不能删除旧周期，也不能在再次锁定时复用旧记录。

### OrderLockHouseBillSnapshot

一条锁定记录关联该操作票在锁定瞬间所有有效 HBL 版本：

- `organization_id`、`lock_record_id`、`house_bill_id`、`house_bill_version_id`；
- `house_no_snapshot`、`created_at`。

`(lock_record_id, house_bill_id)` 唯一。DIRECT 没有 HBL 时集合为空，不创建虚拟 HBL。

### OrderUnlockRequest

所有解锁路径都先形成一条本地请求事实，字段至少包括：

- `organization_id`、`order_id`、`order_no`、`lock_record_id`、`lock_generation`；
- `requested_by`、`requested_at`、可空 `reason`；
- `expected_order_version`；
- `idempotency_key`、`request_fingerprint`；
- `route`：`ROLE_DIRECT | ADMIN_EMERGENCY | DINGTALK_APPROVAL`；
- `status`：
  `PENDING_DISPATCH | PENDING_APPROVAL | APPROVED_PENDING_APPLY | APPROVED | REJECTED |
  CONFIGURATION_FAILED | DISPATCH_FAILED | DISPATCH_UNKNOWN | STALE`；
- 可空 `dingtalk_process_instance_id`、`dingtalk_process_code`；
- 可空 `decided_by`、`decided_at`、`decision_source`；
- 可空 `failure_code`、`failure_message`；
- 可空 `superseded_by_request_id`，角色直解或管理员紧急解锁取代既有活动审批时填写；
- 可空 `unlocked_at`、`result_order_version`。

数据库约束：

- `(organization_id, idempotency_key)` 唯一；
- 同一订单同一锁定代次最多一条活动请求，活动状态限定为 `PENDING_DISPATCH`、
  `PENDING_APPROVAL`、`APPROVED_PENDING_APPLY` 或 `DISPATCH_UNKNOWN`；
- 外部流程实例 ID 非空时全局唯一；
- 相同幂等键、相同指纹返回原请求；相同键、不同指纹返回 409；
- 终态不得回退到活动态。

角色直解和管理员紧急直解在创建请求的同一事务内完成批准、解锁、锁定周期关闭和审计，
不会短暂出现伪造的钉钉待审批状态。

### OrderUnlockApproverCandidate

普通编辑人申请时，保存本次提交时解析到的全部审批候选快照：

- `request_id`、`user_id`、`membership_id`、`role_id`；
- `display_name_snapshot`、`dingtalk_userid_snapshot`；
- `created_at`。

`(request_id, user_id)` 唯一。快照用于说明当时向谁发起，不替代审批生效时的实时资格复验。

业务角色成员必须同时满足：

1. User、Membership、Role 均启用，RoleAssignment 仍存在；
2. Membership 属于订单组织，角色数据范围覆盖该订单；
3. 该角色不是 `administrator`，并且角色显式关联
   `business.order.se.lock` 权限；
4. `is_bootstrap_admin=false`。

普通申请创建钉钉审批时，申请人和以上全部成员都必须有非空 `dingtalk_userid`。任何一方
未绑定，或根本没有有效业务角色成员，都把本地请求置为 `CONFIGURATION_FAILED` 并返回
具体成员/配置提示；禁止悄悄缩小候选集、把 bootstrap admin 塞入候选、改发机器人消息或
自动解锁。钉钉审批节点必须把候选人作为同一或签节点，任意一人同意即可，不允许申请人
自行选择或删减审批人。

### 钉钉审批 Outbox 与回调 Inbox

复用 `BackgroundTask` 的租约框架，新增任务类型 `DINGTALK_APPROVAL_CREATE`，并增加独立
的一对一载荷表 `DingTalkApprovalDispatch`：

- `background_task_id`、`unlock_request_id`；
- `process_code_snapshot`、申请人 DingTalk UserID、候选 UserID 列表快照；
- `request_payload_hash`、`dispatch_status`；
- 可空外部实例 ID、响应摘要和错误分类。

不得复用 `NotificationDelivery`，不得调用 `SendText` 冒充审批。Worker 只消费审批创建任务，
使用独立 `DingTalkApprovalGateway` 发起钉钉原生 OA 审批。

新增 `DingTalkApprovalInboxEvent`：

- 钉钉事件唯一标识；若官方事件不提供稳定 ID，则使用验签后的规范化事件散列；
- `corp_id`、`event_type`、`process_instance_id`、接收时间；
- 加密原文散列和最小必要解析摘要，不保存访问令牌或完整敏感报文；
- `status`：`RECEIVED | PROCESSED | IGNORED | FAILED`，以及处理结果/错误码。

事件先验签、解密并校验企业 ID/事件类型，再幂等落 Inbox 并快速按钉钉协议应答。后台处理
时通过官方审批查询接口读取权威实例结果，不直接信任回调正文中的显示文本。

`Security.DingTalk` 增加审批模板 `process_code`、事件回调 token 和 AES key 等必要配置；
配置只从环境注入，示例配置不得含真实秘密。实现时以钉钉当前官方“发起 OA 审批实例”和
审批事件订阅契约为准，不使用已废弃的自有 OA 待办案例代替原生审批。

外部创建调用的失败分类必须明确：

- 能证明请求未送达时才允许后台任务按现有机制重试；
- 已被钉钉明确拒绝时置 `DISPATCH_FAILED`；
- 超时或断线导致“可能已创建但未拿到实例 ID”时置 `DISPATCH_UNKNOWN`，禁止盲目重试
  生成重复审批；
- 不添加自动解锁、自动改发消息或自动选择其他审批人的兜底。

### MBL/HBL 不可变版本

新增 `SeaMasterBillVersion` 和 `SeaHouseBillVersion`。每个版本完整复制对应身份和内容：

- 组织、单证身份 ID、`version_no`；
- `source_entity_version`，指向创建快照时工作实体的乐观版本；
- 主/分单原始号码、规范化号码、签发主体及当时状态；
- `seaBillContentFields()` 的全部字段；
- MBL 另存运输执行 ID 和权威航程快照；HBL 另存订单 ID、MBL ID、签发来源和备注；
- `reason`、`source`：`ORDER_LOCK | AMENDMENT | SWITCH | VOID`；
- `created_by`、`created_at`；
- `content_hash`，用于核对同一工作版本内容，不用于跳过权限或版本检查。

`(document_id, version_no)` 和 `(document_id, source_entity_version, content_hash)` 唯一。身份表
增加可空 `current_version_id`，表示最近一次已经形成的有效不可变版本；身份表上的字段仍是
未锁定期间的当前工作内容。所有历史锁定记录、改单事件、Switch 事件及未来下游事实只引用
版本 ID，不通过身份 ID 回读“最新字段”。

共享 MBL 在某个成员已锁定后不能再修改，因此其他成员随后锁定时可以复用内容和来源版本
完全相同的 MBL 版本；HBL 版本只属于自己的操作票。再次锁定时：

- 工作实体版本与最近不可变版本相同且内容散列一致，可复用该版本；
- 工作实体已变更则追加新版本并推进身份表 `current_version_id`；
- 不允许仅凭散列跨越不同身份或跳过 `source_entity_version` 校验。

### 单改、作废与 Switch 事件

改单预览是只读计算，不保存服务器草稿。执行请求必须携带订单、目标单证、当前工作实体和
当前不可变版本的预期版本、原因及幂等键。事务内重算差异；无实际差异返回参数错误，不创建
空版本。成功后更新工作内容、追加不可变版本、切换 `current_version_id` 并写审计。

现有 `DRAFT/CONFIRMED/RELEASED` 只作为单证生命周期事实保留，不承担订单编辑锁职责，也不
新增无法由系统证明的外部结果。为作废和替代增加终态 `VOIDED`、`REPLACED`：

- 作废追加 `VOID` 版本和不可变作废事件，保留身份、号码、版本和原因；
- 被 Switch 替代的旧 HBL 进入 `REPLACED`，不能再作为当前有效 HBL 修改；
- 不允许物理删除已形成版本、已作废或已被替代的身份。

新增 `SeaDocumentVoidEvent`，通过互斥可空外键明确指向一张 MBL 或 HBL，并保存作废前后
版本、状态、原因、影响摘要、操作人和时间。迁移用 CHECK 保证恰有一种单证类型。

新增 `SeaHouseBillSwitchEvent`：

- `organization_id`、`order_id`、`master_bill_id`；
- `old_house_bill_id/version_id`、`new_house_bill_id/version_id`；
- `chain_id`、`sequence`；
- `reason`、原件交回/作废信息、外部与财务影响摘要；
- `idempotency_key`、`request_fingerprint`、`created_by`、`created_at`。

同一 HBL 只能被替代一次；一条替代链同一时点只能有一个非 `VOIDED/REPLACED` 的末端 HBL。
新 HBL 是同一订单、同一当前 MBL 下的真实身份，不创建新订单或隐藏 HBL。Switch 在一个
事务内创建新身份及首个版本、终止旧身份、创建替代事件和审计。

发现已确认账单行、发票、核销、提成或其他阶段 6 才能调整的下游事实时，改单、作废和
Switch 明确阻断并返回事实类型与编号；绝不自动复制、红冲、重算或隐藏影响。

## 权限与请求分流

### 新增权限

在 SE 订单操作中新增：

- `business.order.se.lock`，名称“锁定/直接解锁订单”，依赖
  `business.order.se.read` 和 `business.order.se.update`；
- `business.order.se.amend`，名称“改单”，依赖订单读取与编辑；
- `business.order.se.void`，名称“作废提单”，依赖订单读取与编辑；
- `business.order.se.switch`，名称“Switch B/L”，依赖订单读取与编辑。

普通解锁申请复用 `business.order.se.update`，不新增申请权限。锁定接口虽然由中间件检查
`lock`，biz/data 仍必须验证调用人通过非 `administrator` 业务角色实际取得该权限；这可阻止
bootstrap admin 或只拥有管理员角色的真实员工被当作日常锁单人。

### RequestOrderUnlock 分流顺序

在同一事务锁住 Order 后按固定顺序判断：

1. 订单未锁定：返回稳定冲突，不创建请求；
2. 幂等键命中：同指纹返回原结果，异指纹 409；
3. `is_bootstrap_admin=true`：`ADMIN_EMERGENCY`，同步直接解锁；
4. 调用人是有效锁定/解锁业务角色成员：`ROLE_DIRECT`，同步直接解锁；
5. 调用人只有订单编辑权限：解析所有有效业务角色成员和 DingTalk ID，创建
   `DINGTALK_APPROVAL` 请求、候选快照与 outbox；
6. 其他用户由权限层拒绝。

管理员分支必须放在业务角色分支之前，且业务角色查询始终排除 bootstrap admin。紧急解锁
原因可空；无论是否填写，都记录 `route=ADMIN_EMERGENCY`、操作人和时间。真实员工管理员
若未加入业务角色，不走紧急分支，而是按普通订单编辑人发起钉钉审批。

同一锁定代次已有活动审批时，普通编辑人重提只返回该活动请求；业务角色成员或 bootstrap
admin 仍可使用直接解锁。直接解锁事务先把旧请求置 `STALE`、写
`superseded_by_request_id`，再创建并完成新的直接解锁请求。之后到达的旧钉钉回调只能幂等
记录，不能改变订单。

## 命令契约

新增 `order_lock.proto` 与 `OrderLockService`：

1. `GetOrderLockState`
   - order read；返回当前锁摘要、当前代次、服务端允许动作、最近活动请求和具体阻断。
2. `LockOrder`
   - SE lock；请求含 `order_id`、`expected_order_version`、`idempotency_key`。
3. `RequestOrderUnlock`
   - SE update；请求含 `order_id`、`expected_order_version`、`idempotency_key`、可选原因；
     响应明确 `ROLE_DIRECT`、`ADMIN_EMERGENCY` 或 `DINGTALK_APPROVAL` 及真实状态。
4. `ListOrderUnlockRequests`、`GetOrderUnlockRequest`
   - order read；公共分页上限 200，按 `created_at DESC, id DESC` 稳定排序。

扩展或新增单证版本契约：

- `ListSeaMasterBillVersions`、`ListSeaHouseBillVersions`、`GetSeaDocumentVersion`；
- `Preview/ExecuteSeaDocumentAmendment`；
- `Preview/ExecuteSeaDocumentVoid`；
- `Preview/ExecuteSeaHouseBillSwitch`；
- `ListSeaDocumentEvents`。

全部写请求使用预期 Order 版本、目标实体版本、幂等键和服务端锁后重算。Preview 不是提交
令牌；Execute 不信任前端差异或影响摘要。错误使用 `order/v1/error_reason.proto` 的稳定 reason
并携带最小必要 metadata。

钉钉回调不挂在需要登录会话的生成业务服务下。新增固定路径，例如
`POST /api/integrations/dingtalk/order-unlock/events`，由专用 HTTP handler 在 SPA fallback
之前注册，仅按钉钉签名/加密协议鉴权，不接受浏览器用户身份，不复用普通 API 权限中间件。

## 锁定与解锁算法

### 锁定订单

`LockOrder` 在 `Transactor.WithinTransaction` 中：

1. `FOR UPDATE` 锁 Order，验证组织、SE、`ACTIVE`、`OPEN`、未锁定和预期版本；
2. 验证调用人是有效非管理员业务角色成员；
3. 锁当前 Active Link 和 MBL，再锁该订单 HBL，均按 UUID 升序；
4. 重验单证关系、工作版本和阶段 6 财务门禁；
5. 为 MBL 和全部有效 HBL 创建或复用不可变版本；
6. `lock_generation + 1`，设置 `locked_at/locked_by`，Order version + 1；
7. 创建 `OrderLockRecord`、HBL 快照关联和审计；
8. 提交后用普通上下文重读锁摘要。

任何版本、关系或审计写入失败全部回滚，不能出现“订单已锁但没有单证快照”。

### 同步直接解锁

角色成员直解和 bootstrap admin 紧急直解共用一个原子原语：

1. `FOR UPDATE` 锁 Order，验证仍是请求指向的代次与预期版本；
2. 锁当代 `OrderLockRecord`，确认尚未关闭；
3. 创建 `OrderUnlockRequest`；
4. 清空 `locked_at/locked_by`，Order version + 1；
5. 一次性填写 LockRecord 解锁字段，将请求置 `APPROVED` 并记录真实 route；
6. 写审计并提交。

原因可选，不添加“原因为空自动填系统文案”的兜底。审计 `details` 必须包含 route、代次、
请求人/实际解锁人；管理员分支包含 `admin_emergency=true`。

### 钉钉批准生效

回调 Inbox Worker 查询到审批终态后，在事务中：

1. 用外部实例 ID 定位唯一请求并锁 Order；
2. 锁请求、当代 LockRecord 和候选快照；
3. 验证请求仍 `PENDING_APPROVAL`、Order 仍锁定、代次和预期版本完全匹配；
4. 将钉钉批准人映射到唯一启用 User，并复验其当前仍是有效业务角色成员且存在于本次候选；
5. 权威结果为同意时先持久化 `APPROVED_PENDING_APPLY`，再由幂等生效事务清空锁、递增
   Order version、关闭 LockRecord、把请求置 `APPROVED` 并写审计；本地事务临时失败时页面
   能如实显示“已同意，待系统生效”并安全重试生效，而不是再次发起钉钉审批；
6. 拒绝时只把请求置 `REJECTED` 并写审计，订单保持锁定；
7. 代次/版本不一致或已被直接解锁取代时置 `STALE`，记录原因但绝不改订单；
8. 重复事件命中已处理 Inbox 或终态请求时幂等返回，不重复递增版本。

管理员不可能作为钉钉审批人通过上述校验。审批结果不明确、查询失败、验签失败或无法映射
用户时均不会解锁。

## 统一订单锁门禁

不能只在前端禁用或只给 `UpdateOrder` 增加判断。所有改变订单业务结果的入口，事务内首次
读取目标 Order 时都必须直接 `FOR UPDATE` 并调用同一个 `ensureOrderBusinessEditable`：

- 订单资料、状态流转、终止、结案；
- 标签、里程碑、附件、人员；
- 箱计划、实际箱、货物、装运单证、放货 POD、异常事件；
- MBL/HBL、DIRECT/HOUSE、箱货分配；
- 费用创建、修改、删除、取消/确认等业务写入；
- 拆票、整体改配、身份更正、改单、作废和 Switch。

普通单票锁定返回 `ORDER_BUSINESS_LOCKED`，metadata 至少包含 `order_id`、`order_no`、
`lock_generation`、`locked_at`、`locked_by_name`。页面展示“订单已锁定，如需修改请先申请
解锁”，有 update 权限时提供申请入口。

锁不替代既有领域门禁：订单解锁后，已经确认、进入账单、开票、核销或提成的费用仍按财务
错误阻断；不能把 `ensureOrderBusinessEditable` 当成“通过后所有业务都可写”。

## 共享 MBL 锁定门禁

修改 MBL 共享字段、身份更正、MBL 单改或作废之前：

1. 先按 UUID 升序锁住该 MBL 的全部 `ACTIVE` 成员 Order；
2. 收集所有 `locked_at IS NOT NULL` 的订单；
3. 任一存在即整体拒绝，之后不更新 MBL、运输执行或任何成员投影；
4. 返回 `SEA_MASTER_BILL_MEMBER_ORDER_LOCKED`，metadata 包含锁定数量和稳定排序的订单号；
5. 前端提示“需全部解锁后才能修改共享 MBL”，并为当前用户可编辑的订单提供逐票入口。

未锁定成员仍可修改自己的非共享订单资料、HBL、箱货和费用。解锁一张成员票不会自动解锁
其他票，也不会在仍有锁定成员时放开共享 MBL。

## 固定锁序与并发

所有相关写命令使用统一类别顺序：

```text
涉及的 Order（UUID 升序）
→ SeaMasterBill（UUID 升序）
→ Active/Historical Link（UUID 升序）
→ SeaTransportExecution（UUID 升序）
→ SeaHouseBill（UUID 升序）
→ Cargo/Container/Allocation/Fee 等既有子实体（各类 UUID 升序）
→ DocumentVersion / LockRecord / UnlockRequest / Candidate / SwitchEvent
→ AuditLog
```

首次需要写的读取直接 `FOR UPDATE`，禁止先 `FOR SHARE` 再升级。锁定动作和任一普通写都先
竞争同一 Order 行，因此只会出现两种完整结果：普通写先完成后锁定快照包含新值，或锁定先
完成后普通写等待并收到锁定错误；不会出现快照与业务字段交叉提交。

共享 MBL 动作必须先锁所有成员 Order 再锁 MBL。阶段 4 的拆票/改配、本阶段版本命令和既有
共享 MBL 更新均须对齐该顺序，不使用数据库死锁重试兜底顺序错误。

## 页面设计

### 订单顶部与表单门禁

- 订单详情 `PageHeaderShell` 显示“未锁定”或“已锁定 · 操作人 · 时间”；锁定时业务表单
  保持可阅读但禁用写入口，不默认折叠提单区块。
- 有效业务角色成员显示“锁定订单”或“直接解锁”；bootstrap admin 在锁定订单上显示
  “紧急解锁”；普通编辑人显示“申请解锁”。没有 update 权限只显示锁定原因。
- 点击紧急解锁时可填写原因但不强制；确认文案明确这是系统管理员兜底操作并会被审计。
- 被后端锁定阻断的任意写请求统一打开锁定摘要，不能只显示 toast“修改失败”。

### 解锁申请状态

详情页抽屉展示本订单最近请求和状态时间线：

- `PENDING_DISPATCH`：待创建钉钉审批；
- `PENDING_APPROVAL`：钉钉审批中；
- `APPROVED_PENDING_APPLY`：钉钉已同意，等待本地幂等解锁生效；
- `CONFIGURATION_FAILED/DISPATCH_FAILED/DISPATCH_UNKNOWN`：展示具体配置或发送问题；
- `REJECTED`：已拒绝，订单仍锁定；
- `STALE`：审批已过期，未影响当前锁定周期；
- `APPROVED`：已解锁及实际处理人/时间。

前端不自行推断“钉钉已经同意”，只展示服务端状态。活动请求存在时复用其详情，不重复创建。

### 版本、改单、作废与 Switch

- “提单信息”区块继续默认展开；主单/HBL 标签显示当前工作版本、最近封存版本和版本数。
- 历史列表、逐字段差异、作废和 Switch 替代链放在抽屉，不在普通录单页面平铺状态矩阵。
- 单改/作废/Switch 先调用 Preview，展示具体字段差异和财务/外部影响，再由用户明确确认
  Execute；改单、作废原因按 PRD 必填，解锁原因仍可选。
- 订单锁定时这些动作只显示“请先申请解锁”；共享 MBL 有其他锁定成员时列出具体订单号。
- DIRECT 不显示 HBL 版本或 Switch；HOUSE 支持一票多 HBL，动作只影响当前选择的 HBL。

## 错误契约

新增稳定 reason，至少包括：

- `ORDER_BUSINESS_LOCKED`
- `ORDER_ALREADY_LOCKED`
- `ORDER_NOT_LOCKED`
- `ORDER_LOCK_ROLE_REQUIRED`
- `ORDER_UNLOCK_REQUEST_ACTIVE`
- `ORDER_UNLOCK_APPROVER_NOT_CONFIGURED`
- `ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED`
- `ORDER_UNLOCK_DINGTALK_DISPATCH_FAILED`
- `ORDER_UNLOCK_DINGTALK_DISPATCH_UNKNOWN`
- `ORDER_UNLOCK_APPROVAL_STALE`
- `ORDER_UNLOCK_APPROVER_INVALID`
- `SEA_MASTER_BILL_MEMBER_ORDER_LOCKED`
- `SEA_DOCUMENT_VERSION_CONFLICT`
- `SEA_DOCUMENT_AMENDMENT_EMPTY`
- `SEA_DOCUMENT_CHANGE_BLOCKED`
- `SEA_DOCUMENT_VOIDED`
- `SEA_HOUSE_BILL_SWITCH_CONFLICT`
- `SEA_HOUSE_BILL_SWITCH_DOWNSTREAM_BLOCKED`

并发版本冲突统一 409；参数格式使用 400；锁定或财务事实导致的业务冲突使用 409；权限不足
使用 403。错误 metadata 不放完整钉钉报文、token、签名、单证全文或敏感联系人内容。

## 迁移与配置

- Ent Schema 是真相源，生成正式 PostgreSQL 迁移并审阅枚举 CHECK、FK、部分唯一索引和
  当前版本反向外键。
- 开发数据库按用户既有授权可清空后重建；迁移不按用户名识别旧 admin，也不把任意
  `administrator` 用户回填为 bootstrap admin。
- `bootstrap-admin` 在空库创建唯一初始化账号时写 `is_bootstrap_admin=true`；重复 bootstrap
  仍由现有“数据库必须无用户”门禁拒绝。
- 配置样例新增钉钉审批模板/回调键名，但真实 process code、token、AES key 只走环境变量。
- 不在迁移或启动时自动创建“锁定/解锁角色”；管理员由权限页面创建普通业务角色并分配
  `business.order.se.lock`，避免把某个角色名或 UUID 写死。

## 验证策略

### 单元与层级测试

- biz：锁定/解锁三分流、bootstrap 优先、非 bootstrap administrator 不误判、角色直接解锁、
  原因可空、幂等同键同/异指纹、终态状态机和具体错误 metadata。
- data：角色资格的 User/Membership/Role/Assignment/Permission 全条件；禁用和移除角色；
  DingTalk ID 缺失；候选快照；锁定代次；文档版本复用/追加；Switch 单当前末端。
- service/API：权限、UUID/版本/分页边界、DTO 状态、静态路由顺序、管理员紧急解锁响应。
- web：角色/管理员/普通编辑人三种按钮、可选原因、锁定只读、失败状态、共享 MBL 订单号、
  默认展开提单区块和 Preview/Execute 差异确认。

### PostgreSQL 与并发测试

使用隔离 Schema 的真实 PostgreSQL 测试，禁止 SKIP，至少覆盖：

- 普通写与锁定并发：只出现“写先完成并进入快照”或“锁先完成且写被阻断”；
- 两人并发锁定、两次并发解锁、角色直解与回调竞争，只有一个版本递增和一条关闭事实；
- 共享 MBL 多订单锁定，任一成员锁定均阻断共享更新且无部分提交；
- 钉钉重复回调、乱序拒绝/同意、旧代次批准、审批人资格撤销、候选外账号均不误解锁；
- 审计写入失败使锁定、解锁、改单、作废和 Switch 全部回滚；
- MBL/HBL 版本追加、复用、历史读取、作废、Switch 替代链和幂等重放；
- 解锁后仍有确认账单/核销/提成事实时，相关财务修改继续被原有门禁阻断。

钉钉测试使用本地假 HTTP 服务验证请求、错误分类、查询结果与回调，不调用真实钉钉环境。

## 回滚边界

阶段 5 的产品改动按实施步骤形成可验证提交，但 Schema、迁移、Proto、生成物、后端、页面和
测试必须保持同一最终版本。回滚时不得只删页面而遗留可写接口，也不得只删新表而保留身份表
的当前版本外键。已有锁定/版本/审批数据进入环境后，回滚必须先停写并制定显式数据方案；
不能用迁移静默删除业务历史。
