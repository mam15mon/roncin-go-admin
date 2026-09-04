# 海运提单改单、作废、Switch 与不可变历史

## 1. Scope / Trigger

- 修改 MBL/HBL 不可变版本查询、改单、作废、Switch B/L、事件历史或相关前端确认流程时适用。
- 这些命令会同时改变 API、订单/单证版本、数据库不可变事实、权限与页面交互，必须按跨层契约整体实现和验证。
- 本契约只允许阻断既有下游事实；不得自动复制费用、红冲账单、撤销核销、重算提成或创建第二套审批流。

## 2. Signatures（API / Biz / DB）

HTTP/API：

```text
GET  /api/v1/orders/{order_id}/sea-documents/master-bill/versions
GET  /api/v1/orders/{order_id}/sea-documents/house-bills/{house_bill_id}/versions
GET  /api/v1/orders/{order_id}/sea-documents/versions/{version_id}
GET  /api/v1/orders/{order_id}/sea-documents/events
POST /api/v1/orders/{order_id}/sea-documents/amendments/preview
POST /api/v1/orders/{order_id}/sea-documents/amendments
POST /api/v1/orders/{order_id}/sea-documents/voids/preview
POST /api/v1/orders/{order_id}/sea-documents/voids
POST /api/v1/orders/{order_id}/sea-documents/house-bill-switches/preview
POST /api/v1/orders/{order_id}/sea-documents/house-bill-switches
```

核心 Biz 命令：

```go
type SeaDocumentAmendmentCommand struct {
    OrderID, DocumentID, ExpectedCurrentVersionID uuid.UUID
    DocumentType                                  SeaDocumentType
    ExpectedOrderVersion, ExpectedDocumentVersion uint64
    Reason, IdempotencyKey                        string
    Input                                         *SeaDocumentAmendmentInput
}

type SeaDocumentVoidCommand struct {
    OrderID, DocumentID, ExpectedCurrentVersionID uuid.UUID
    DocumentType                                  SeaDocumentType
    ExpectedOrderVersion, ExpectedDocumentVersion uint64
    Reason, IdempotencyKey                        string
}

type SeaHouseBillSwitchCommand struct {
    OrderID, OldHouseBillID, ExpectedCurrentVersionID uuid.UUID
    ExpectedOrderVersion, ExpectedHouseBillVersion    uint64
    Reason, IdempotencyKey                             string
    SurrenderInfo                                      *string
    NewHouseBill                                       *SeaHouseBillInput
}
```

关键数据库约束：

- `sea_master_bill_versions`、`sea_house_bill_versions` 只追加；改单版本按
  `(organization_id, idempotency_key)` 唯一，身份表的 `current_version_id` 指向新版本。
- `sea_document_void_events` 同时引用作废前、作废后版本；`order_id`、幂等键、请求指纹必填，
  MBL/HBL 外键组合互斥且历史外键使用 `ON DELETE NO ACTION`。
- `sea_house_bill_switch_events` 的 `(organization_id, idempotency_key)`、
  `old_house_bill_id`、`(chain_id, sequence)` 分别唯一。

## 3. Contracts（请求 / 响应 / 状态）

- Preview 和 Execute 都必须携带大于零的 Order/工作单证预期版本以及当前不可变版本 ID；
  Execute 额外要求去空白后非空、最长 128 字符的幂等键。
- 原因必填、去首尾空白、最长 500 字符且不得含控制字符；Switch 的 `surrender_info` 可空，
  但非空时遵循同样长度和控制字符约束。
- Preview 不保存服务器草稿。响应必须给出基线不可变版本、最终逐字段差异、下游影响和
  `executable`；前端只有在本次 Preview 成功且完整展示差异/影响后才能启用 Execute，任一输入
  变化都必须清空 Preview。
- Execute 必须在统一事务中重新计算差异和门禁；不得信任客户端传回的 Preview。成功时写工作
  实体、追加不可变版本/事件、切换当前指针、递增相关版本并写审计，任一步失败整体回滚；提交后
  使用普通上下文重读响应。
- MBL 写入先定位全部活动成员 Order，按 UUID 升序加锁，再锁 MBL 和全部活动 Link；取得 MBL
  锁后必须重查成员集合。集合改变返回结构冲突，不允许用首次定位的旧集合执行财务门禁。
- HBL 写入固定锁序为 `Order -> MBL -> Active Link -> HBL`，并重验组织、订单、当前 MBL、
  工作版本和当前不可变版本。
- 下游门禁直接查询已确认费用、活动账单行、发票、核销、提成、提成调整；HBL 还查询箱货分配。
  不能假设发票或核销始终可由当前活动账单行间接覆盖。错误 metadata 至少含
  `blocked_count`、首个 `fact_type`、`reference_id`、`reference_no`。
- 单改追加 `AMENDMENT` 版本；作废追加 `VOID` 版本和作废事件并把身份置为 `VOIDED`；Switch
  在同订单、同当前 MBL 下创建真实新 HBL 及首个 `SWITCH` 版本，把旧 HBL 置为 `REPLACED`，
  后续 Switch 沿用 `chain_id` 并递增 `sequence`。
- `VOIDED`、`REPLACED` 是服务端终态。普通 HBL 更新、删除、箱货分配/汇总入口和页面普通编辑
  都必须失败关闭，不能只隐藏改单按钮。
- 历史 DTO 只能读取不可变版本或事件中保存的号码、状态和内容；不得回读当前可变身份字段。
  共享 MBL 的版本和作废历史对全部活动成员订单可见。
- Preview 路径使用显式 `/preview` 子路径；不得使用 `:preview`，当前 OpenAPI 客户端生成器会把
  冒号动作误解释为运行时路径变量。
- 版本/事件列表复用公共分页校验：`page >= 1`、`1 <= page_size <= 200`，并按时间/版本和 ID
  提供稳定排序。

## 4. Validation & Error Matrix

| 条件 | 稳定错误 / 行为 |
| --- | --- |
| UUID、枚举、预期版本、原因、输入对象或 Execute 幂等键非法 | 400 `SEA_DOCUMENT_INVALID_ARGUMENT` |
| 单改相对当前不可变版本没有最终差异 | 400 `SEA_DOCUMENT_AMENDMENT_EMPTY` |
| 工作版本或当前不可变版本已变化 | 409 单证版本/结构冲突，事务不落库 |
| 订单或共享 MBL 任一成员已锁定 | 409 订单业务锁或 `SEA_MASTER_BILL_MEMBER_ORDER_LOCKED` |
| MBL 锁前后活动成员集合变化 | 409 `SEA_DOCUMENT_STRUCTURE_CONFLICT` |
| 已有费用、账单、发票、核销、提成、提成调整或相关箱货分配 | 409 `SEA_DOCUMENT_CHANGE_BLOCKED`；Switch 使用 `SEA_HOUSE_BILL_SWITCH_DOWNSTREAM_BLOCKED` |
| 同幂等键、同请求指纹 | 返回原版本/事件/新 HBL，不追加事实 |
| 同幂等键、不同请求指纹 | 409 版本或 Switch 冲突 |
| 对 `VOIDED` 单证再次修改 | 409 `SEA_DOCUMENT_VOIDED` |
| 对 `REPLACED` HBL 再次修改或 Switch | 409 `SEA_HOUSE_BILL_SWITCH_CONFLICT` |
| Switch 新 HBL 号码或签发主体不合法/重复 | 400 HBL 参数错误或 409 `SEA_HOUSE_BILL_EXISTS` |
| 审计写入失败 | 命令失败，工作实体、当前指针、版本和事件全部回滚 |

## 5. Good / Base / Bad Cases

- Good：用户修改 HBL 后先 Preview；页面展示签发主体解析后的最终差异，用户确认 Execute，服务端
  锁内重算后追加版本并切换指针，旧版本内容仍可完整读取。
- Good：两个相同幂等键、相同 payload 的并发请求只有一个写入版本，另一个在唯一冲突事务结束后
  从普通上下文读取并返回同一事实。
- Good：第二次 Switch 以第一张新 HBL 为旧单，沿用替代链且 `sequence + 1`；链上只存在一个
  非 `REPLACED`/`VOIDED` 当前末端。
- Base：没有下游事实且订单可编辑时，具备动作权限的用户无需额外审批即可执行改单、作废或 Switch。
- Bad：Preview 通过后直接提交客户端缓存差异，不在锁内重查 Order/Link/HBL/财务事实。
- Bad：事件历史通过关联当前 HBL 读取号码；后续普通更正会让历史事件展示发生变化。
- Bad：只在新动作接口阻断终态，仍允许旧的 Update/Remove 或箱货汇总修改 `VOIDED/REPLACED` HBL。

## 6. Tests Required

- Biz/Service：必填 UUID/枚举/预期版本、原因/幂等键长度、MBL/HBL 输入互斥、分页 1/200/201、
  DTO 的零 UUID 与可空字段映射。
- Data/PostgreSQL：MBL/HBL 单改、作废、两次 Switch、旧版本不变、当前指针、事件前后版本、审计
  失败回滚和稳定历史读取。
- 并发：不同请求同时改单或 Switch 只有一个成功；相同幂等键并发返回同一事实；共享 MBL 在锁前后
  增加成员时返回结构冲突且不使用旧成员集合提交。
- 门禁：构造非活动账单行仍保留的发票/核销事实，证明直接门禁可命中；同时覆盖费用、活动账单、
  提成/调整和 HBL 箱货分配。
- 终态：普通 HBL Update/Remove、箱货分配和汇总入口均拒绝 `VOIDED/REPLACED`，数据库无部分更新。
- Frontend：Execute 初始禁用；Preview 成功并展示差异后启用；输入变化重新禁用；锁态、加载失败及
  终态都失败关闭；动作成功先刷新 Order version，再刷新单证历史。
- Migration/Generation：从版本化基线迁移到改单迁移并重复执行；检查 CHECK/FK/唯一索引；连续运行
  Proto/OpenAPI、Ent、Wire、Web Client 和权限/枚举生成后差异指纹不变。

## 7. Wrong vs Correct

### Wrong

```go
// 锁 MBL 前取得成员集合，之后一直用旧集合检查财务事实并提交。
memberIDs := locateMemberIDs(ctx, tx.Client(), mblID)
lockOrders(ctx, tx, memberIDs)
lockMasterBill(ctx, tx, mblID)
return publish(ctx, tx, memberIDs)
```

```typescript
// 只要曾经 Preview 成功，表单变化后仍可 Execute。
const executeDisabled = !preview;
```

### Correct

```go
// Order 按 UUID 固定顺序锁定；取得 MBL 锁后再锁并重验全部活动 Link。
memberIDs := locateMemberIDs(ctx, tx.Client(), mblID)
orders := lockOrdersInOrder(ctx, tx, memberIDs)
mbl := lockMasterBill(ctx, tx, mblID)
link, err := lockAndValidateMasterMemberLinks(
    ctx, tx, orgID, orderID, mbl.ID, expectedLinkID, memberIDs,
)
if err != nil {
    return err
}
return publishWithLockedFacts(ctx, tx, orders, mbl, link)
```

```typescript
// Preview 与当前输入绑定；字段变化立即清空，且服务端明确允许执行后才开放按钮。
const executeDisabled =
  !preview || preview.executable !== true || !previewMatchesCurrentInput;
```
