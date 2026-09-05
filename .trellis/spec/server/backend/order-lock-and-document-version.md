# 全业务订单锁、SE 单证快照与共享 MBL 写入

## 1. Scope / Trigger

- 适用于 SE、SI、AE、AI、LAND、RAIL 六类订单的锁定、角色/管理员直接解锁、解锁申请事实，以及所有会修改订单业务资料或订单费用的 data 层写入口。
- SE 锁定额外覆盖不可变 MBL/HBL 版本和共享 MBL 写入口；非 SE 锁定不得创建或引用海运单证快照。
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
- 锁定仅允许具备目标类型显式 `business.order.<type>.lock` 权限、有效非 `administrator` 业务角色且数据范围覆盖订单的非 bootstrap 用户；锁权限只依赖同类型 read/update，不得跨类型授权。
- 解锁固定分流：bootstrap admin → `ADMIN_EMERGENCY`；合格业务角色成员 → `ROLE_DIRECT`；普通订单编辑人 → `DINGTALK_APPROVAL`。
- 所有类型的锁定事务原子创建锁定周期记录、更新订单锁状态与审计；SE 同一事务额外创建/复用完整 MBL/HBL 不可变版本与 HBL 快照，任何一步失败必须全部回滚。
- 六种类型共用一个 `Security.DingTalk.approval_process_code`；OA 实例必须携带业务类型、订单号、申请人、原因和锁代次，不按类型复制模板配置。
- `Security.DingTalk.approval_process_code/event_token/event_aes_key` 只接受环境注入；仓库配置仅保留空值或占位符。
- 列表分页必须满足 `page >= 1`、`1 <= page_size <= 200`；缺省值由 service 统一补齐。

## 4. Validation & Error Matrix

| 条件 | 稳定错误 / 行为 |
| --- | --- |
| 目标订单已业务锁定，调用普通写入口 | `ORDER_BUSINESS_LOCKED`，metadata 含订单、代次、时间和锁定人 |
| 调用人没有目标订单类型的合格业务锁角色 | `ORDER_LOCK_ROLE_REQUIRED` |
| 订单已锁 / 未锁或预期版本不匹配 | `ORDER_ALREADY_LOCKED`、`ORDER_NOT_LOCKED` 或 `ORDER_STATUS_CONFLICT` |
| 同幂等键、同请求指纹 | 返回原结果，不追加事实 |
| 同幂等键、不同请求指纹 | `ORDER_IDEMPOTENCY_KEY_REUSED` |
| 共享 MBL 实际写入时任一活动成员订单已锁 | `SEA_MASTER_BILL_MEMBER_ORDER_LOCKED`，订单号按字典序输出 |
| MBL 锁定前后活动成员集合变化 | `SEA_DOCUMENT_STRUCTURE_CONFLICT` 并回滚；不得后置补锁 Order |
| 共享 MBL 字段完全未变化，仅更新未锁成员自身普通字段 | 允许更新，不因其他成员锁定而误拦截 |
| 无有效审批候选人或相关用户缺少 DingTalk UserID | 保存配置失败事实并返回对应稳定错误，不缩小候选集 |

## 5. Good / Base / Bad Cases

- Good：按 UUID 升序一次性锁住共享 MBL 的全部活动成员 Order，再锁 MBL、目标 Link、TransportExecution；锁后重验成员集合，随后执行共享写入。
- Base：普通订单字段更新不携带 MBL 身份、权威航程或 `MasterBillContent` 写入时，只锁目标 Order 并执行统一业务锁门禁。
- Bad：先锁目标 Link 或 MBL，再查询并 `FOR UPDATE` 其他成员 Order；这会与拆票、改配、锁单路径形成反向锁序。
- Bad：只检查 `SeaMasterBillInput`，遗漏 `SeaDocumentInput.MasterBillContent` 也会修改共享 MBL 的事实。

## 6. Tests Required

- Biz：锁定/三路解锁资格、bootstrap 优先、真实 administrator 边界、幂等同键同/异指纹、错误 metadata。
- Data/PostgreSQL：六类型锁定/解锁、跨类型权限隔离、每类订单资料与费用写阻断、非 SE 空快照、SE 版本快照、DIRECT 零 HBL、共享版本复用、审计失败回滚、普通写与锁定竞争、双锁/双直解竞争、活动请求唯一。
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
