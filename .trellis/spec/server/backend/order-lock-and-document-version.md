# 全业务订单锁、SE 单证快照与共享 MBL 写入

## 1. Scope / Trigger

- 适用于 SE、SI、AE、AI、LAND、RAIL 六类订单的锁定、角色/管理员直接解锁、解锁申请事实，以及所有会修改订单业务资料或订单费用的 data 层写入口。
- 订单“业务内容写入门禁”与“生命周期流转资格”分离：内容门禁只约束业务字段及子资源写入；终止、恢复、结案、反结案等生命周期命令在锁定订单行后执行专属状态校验，不得复用内容门禁。
- SE 锁定额外覆盖不可变 MBL/HBL 版本和共享 MBL 写入口；非 SE 锁定不得创建或引用海运单证快照。
- SE 订单最终流转到 `TERMINATED` 时，同一事务结束其活动 `SeaMasterBillOrderLink`；恢复订单不复活历史 Link。
- 新增订单写入口、修改 MBL 身份/权威航程/内容、改变 MBL 活动成员关系时，必须复核本文的事务门禁与固定锁序。
- 钉钉审批的外部派发、回调与本地生效另行实现；不得把当前 `PENDING_DISPATCH` 事实误当成审批已经发起。

## 2. Signatures（API / Biz / DB）

HTTP/API：

```text
GET  /api/v1/orders/{order_id}/lock-state
POST /api/v1/orders/{order_id}/lock
POST /api/v1/orders/{order_id}/unlock
GET  /api/v1/orders/{order_id}/unlock-requests
GET  /api/v1/orders/{order_id}/unlock-requests/{request_id}
```

核心仓储契约：

```go
LockOrder(ctx, caller, orderID, expectedOrderVersion, idempotencyKey, audit)
RequestOrderUnlock(ctx, caller, orderID, expectedOrderVersion, idempotencyKey, reason, audit)
ListOrderUnlockRequests(ctx, organizationID, orderID, page, pageSize)
GetOrderUnlockRequest(ctx, organizationID, orderID, requestID)
```

关键数据库约束：

- `orders.locked_at/locked_by/lock_generation/version` 表示当前锁状态；成功锁定和成功解锁都递增 `version`，仅锁定递增 `lock_generation`。
- `order_lock_records(order_id, generation)` 唯一；锁定事实只追加、不覆盖，并保存当代次权威 `business_type`。
- `order_unlock_requests(organization_id, idempotency_key)` 唯一；同订单同代次最多一个活动请求，并保存与锁定事实一致的 `business_type`。
- `order_lock_records` 的 MBL/MBL 版本引用仅 SE 必填，其他五类必须为空；历史快照外键即使字段可空也保持 `ON DELETE NO ACTION`。
- `sea_master_bill_versions`、`sea_house_bill_versions` 是不可变历史；身份表的 `current_version_id` 只能指向自身版本。

## 3. Contracts（请求 / 响应 / 配置）

- `LockOrderRequest` 必须包含合法 UUID `order_id`、大于零的 `expected_order_version` 和去除空白后非空的 `idempotency_key`。
- `RequestOrderUnlockRequest` 使用相同并发与幂等字段；`reason` 可选，管理员紧急解锁不得伪造默认原因。
- 锁定资格按「有效 lock grant」统一计算（`internal/data/order_lock.go` 的 `qualifiedBusinessLockGrantPredicate`，单用户判断与候选查询必须复用同一谓词）：用户 enabled、membership enabled、角色 enabled 且真实持有目标类型显式 `business.order.<type>.lock` 权限（不按角色代码排除 `administrator` 等任何角色），角色数据范围覆盖目标订单组织——`ORGANIZATION` 要求 membership 组织即目标组织，`ORGANIZATION_TREE` 要求 membership 组织是目标组织或其祖先（沿 parent 链计算，带环保护），`ALL` 覆盖任意组织；锁权限只依赖同类型 read/update，不得跨类型授权。
- bootstrap admin 显式具备锁单资格（`GetOrderLockState`/`LockOrder` 直接视为可锁），但不得进入普通审批候选池，回调也不得绕过候选快照。
- 解锁固定分流：bootstrap admin → `ADMIN_EMERGENCY`；持有效 lock grant 的用户（含真实持锁权限的 `administrator` 角色）→ `ROLE_DIRECT`；普通订单编辑人 → `DINGTALK_APPROVAL`。
- 审批候选人快照保存实际 `membership_id`、`role_id` 与 DingTalk UserID；同一用户有多个合格 grant 时按 RoleAssignment ID 升序取首个可追溯 grant；钉钉回调先校验审批人存在于快照，再以同一资格口径实时复核，权限或成员关系被撤销后旧候选回调稳定标记 `APPROVER_NOT_QUALIFIED`。
- 所有类型的锁定事务原子创建锁定周期记录、更新订单锁状态与审计；SE 同一事务额外创建/复用完整 MBL/HBL 不可变版本与 HBL 快照，任何一步失败必须全部回滚。
- 六种类型共用一个 `Security.DingTalk.approval_process_code`；OA 实例必须携带业务类型、订单号、申请人、原因和锁代次，不按类型复制模板配置。
- `Security.DingTalk.approval_process_code/event_token/event_aes_key` 只接受环境注入；仓库配置仅保留空值或占位符。
- 列表分页必须满足 `page >= 1`、`1 <= page_size <= 200`；缺省值由 service 统一补齐。
- 统一内容写门禁（`internal/data/order_lock.go` 的 `ensureOrderBusinessContentEditable`）：所有订单业务资料及子资源写入口在 Order `FOR UPDATE` 后校验，订单必须满足 `termination_status = ACTIVE`、`closure_status = OPEN` 且未业务锁定；校验顺序为先生命周期后业务锁，禁止另写第二套重复门禁（如原 `ensureReleasePodOrderEditable` 已删除并复用统一实现）。
- 生命周期命令的专属校验：`TransitionStatus` 显式要求 ACTIVE + OPEN + 未业务锁定（主流程推进属业务写入，锁定后不得推进）；`TransitionTermination` 只要求结案维度保持 OPEN，其中 `ACTIVE → TERMINATING` 仍要求订单未业务锁定，`TERMINATING → ACTIVE/TERMINATED` 与 `TERMINATED → ACTIVE` 不因业务锁被阻断；`TransitionClosure` 不校验业务锁，已锁定的 `DOCUMENT_RELEASED` 订单允许结案，反结案不受历史业务锁阻断，结案 readiness 在事务内重验。
- SE 订单流转到 `TERMINATED` 时在同一事务结束活动 Link：按 Link ID 排序 `FOR UPDATE`，超过一条活动 Link 返回 `SEA_DOCUMENT_STRUCTURE_CONFLICT`（fail-closed，不静默修复）；更新为 `ENDED`、写入 UTC `ended_at`、固定中文 `ended_reason = "订单退关"` 并递增 Link 版本，不修改 MBL/TransportExecution 版本；Order、Link、生命周期事件、审计任一步失败全部回滚。`TERMINATED → ACTIVE` 只恢复订单状态，历史 Link 保持不可变。

## 4. Validation & Error Matrix

| 条件 | 稳定错误 / 行为 |
| --- | --- |
| 订单处于终止流程（`termination_status = TERMINATING`），调用内容写入口 | `ORDER_TERMINATION_IN_PROGRESS`（409，“订单已进入终止流程，不允许修改业务数据”） |
| 订单已终止（`termination_status = TERMINATED`），调用内容写入口 | `ORDER_TERMINATED`（409，“订单已终止，不允许修改业务数据”） |
| 订单已结案（`closure_status != OPEN`），调用内容写入口 | `ORDER_CLOSED`（409，“订单已结案，不允许修改业务数据”） |
| 目标订单已业务锁定，调用普通写入口 | `ORDER_BUSINESS_LOCKED`，metadata 含订单、代次、时间和锁定人 |
| 已终止订单残留历史锁信息 | 优先返回终止类错误，不得误提示“先解锁即可编辑” |
| 锁定的 `DOCUMENT_RELEASED` 订单执行结案、`TERMINATING/TERMINATED` 执行完成/取消/恢复 | 生命周期命令放行，不得被内容门禁误杀 |
| 锁定订单发起退关（`ACTIVE → TERMINATING`） | `ORDER_BUSINESS_LOCKED`，保持“锁定资料不可发起退关” |
| 调用人（非 bootstrap）没有覆盖目标组织的有效 lock grant | `ORDER_LOCK_ROLE_REQUIRED` |
| 钉钉回调审批人不在候选快照中，或快照后实时资格复核失败 | 分别稳定标记 `APPROVER_NOT_IN_SNAPSHOT` / `APPROVER_NOT_QUALIFIED`，订单保持锁定 |
| 订单已锁 / 未锁或预期版本不匹配 | `ORDER_ALREADY_LOCKED`、`ORDER_NOT_LOCKED` 或 `ORDER_STATUS_CONFLICT` |
| 同幂等键、同请求指纹 | 返回原结果，不追加事实 |
| 同幂等键、不同请求指纹 | `ORDER_IDEMPOTENCY_KEY_REUSED` |
| 共享 MBL 实际写入时任一活动成员订单已锁 | `SEA_MASTER_BILL_MEMBER_ORDER_LOCKED`，订单号按字典序输出 |
| MBL 锁定前后活动成员集合变化 | `SEA_DOCUMENT_STRUCTURE_CONFLICT` 并回滚；不得后置补锁 Order |
| 共享 MBL 字段完全未变化，仅更新未锁成员自身普通字段 | 允许更新，不因其他成员锁定而误拦截 |
| 退关完成时同一订单存在多条活动 Link | `SEA_DOCUMENT_STRUCTURE_CONFLICT` 并整体回滚，不静默修复 |
| 无有效审批候选人或相关用户缺少 DingTalk UserID | 保存配置失败事实并返回对应稳定错误，不缩小候选集 |

## 5. Good / Base / Bad Cases

- Good：按 UUID 升序一次性锁住共享 MBL 的全部活动成员 Order，再锁 MBL、目标 Link、TransportExecution；锁后重验成员集合，随后执行共享写入。
- Good：SE 订单完成退关的事务先锁 Order，再按 Link ID 锁定并结束唯一活动 Link，随后写生命周期事件与审计，任一步失败整体回滚。
- Base：普通订单字段更新不携带 MBL 身份、权威航程或 `MasterBillContent` 写入时，只锁目标 Order 并执行统一内容写门禁（生命周期 + 业务锁）。
- Base：已锁定的 `DOCUMENT_RELEASED` 订单直接结案；`TERMINATED → ACTIVE` 恢复订单但不复活历史 Link，后续显式建立新 Link 由活动 Link 唯一索引兜底。
- Bad：在生命周期命令（终止/恢复/结案/反结案）里复用内容写门禁，导致合法流转被 `ORDER_BUSINESS_LOCKED`/`ORDER_TERMINATED` 误杀。
- Bad：先锁目标 Link 或 MBL，再查询并 `FOR UPDATE` 其他成员 Order；这会与拆票、改配、锁单路径形成反向锁序。
- Bad：只检查 `SeaMasterBillInput`，遗漏 `SeaDocumentInput.MasterBillContent` 也会修改共享 MBL 的事实。
- Bad：退关完成时发现多条活动 Link 后静默结束全部或仅结束其一；必须 fail-closed 返回结构冲突。

## 6. Tests Required

- Biz：锁定/三路解锁资格、bootstrap 优先、幂等同键同/异指纹、错误 metadata。
- Data/PostgreSQL 锁资格矩阵：组织内 ORG 角色、上级组织 TREE 角色（按真实上级 membership/role 快照）、ALL 范围 `administrator` 角色、无锁权限的 `administrator` 角色、bootstrap（可锁、应急解锁、不进候选池）、ORG 范围越组织与 TREE 范围兄弟组织拒绝、撤权后旧候选回调 `APPROVER_NOT_QUALIFIED`。
- Data 单元：内容门禁表驱动测试覆盖 TERMINATING/TERMINATED/CLOSED/已锁定四类阻断、生命周期原因优先于业务锁、正常 ACTIVE+OPEN+未锁定放行。
- Data/PostgreSQL：六类型锁定/解锁、跨类型权限隔离、每类订单资料与费用写阻断、非 SE 空快照、SE 版本快照、DIRECT 零 HBL、共享版本复用、审计失败回滚、普通写与锁定竞争、双锁/双直解竞争、活动请求唯一。
- Data/PostgreSQL：代表性子资源（草稿、POD、货物、里程碑等）在四类状态下写阻断且零部分写入；锁定 `DOCUMENT_RELEASED` 订单结案与反结案；退关完成原子结束活动 Link（ENDED、ended_at、ended_reason、version+1、事件与审计），取消退关保留 Link，恢复不复活历史 Link，旧版本稳定冲突。
- 自拼汇总：Link ACTIVE 且 Order ACTIVE 的 LCL 成员才计入件重尺/成员数；house numbers 来自当前活动 MBL 的有效 `SeaHouseBill`，旧 `OrderShippingDocument` 与 VOIDED HBL 不返回。
- 共享 MBL：至少分别覆盖身份/权威航程和 `MasterBillContent` 两个入口；断言成员锁定时 Order、MBL、TransportExecution 均未部分更新。
- 不误拦截：共享字段无变化时，未锁成员的普通订单字段可更新，且 MBL/TransportExecution 版本不变。
- 迁移：在临时 PostgreSQL schema 中按正式迁移链执行；从升级前基线构造既有 SE 锁定周期和解锁请求，验证 `business_type=SE` 回填、NOT NULL、SE/非 SE 单证引用 CHECK、历史快照外键 `NO ACTION`、revision/checksum 与重复执行。
- Service/Web：分页 1/200/201、允许动作 fail-closed、锁定详情、三种解锁入口和稳定状态文本。

## 7. Wrong vs Correct

### Wrong

```go
// 仅锁目标订单，随后直接修改共享 MBL；其他成员的锁可被绕过。
order := tx.Order.Query().Where(order.IDEQ(orderID)).ForUpdate().OnlyX(ctx)
mbl := tx.SeaMasterBill.Query().Where(seamasterbill.IDEQ(mblID)).ForUpdate().OnlyX(ctx)
```

### Correct

```go
// 定位阶段只读；成员 ID 去重并按 UUID 升序后一次性加锁。
members := tx.Order.Query().
    Where(order.OrganizationIDEQ(orgID), order.IDIn(sortedMemberIDs...)).
    Order(order.ByID()).
    ForUpdate().
    AllX(ctx)

// 随后按 MBL → 目标 Link → TransportExecution 加锁，并在 MBL/Link 锁后
// 重验活动成员集合；集合变化立即返回结构冲突并整体回滚。
```
