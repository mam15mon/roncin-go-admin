# 海运出口共享主单与分单契约

## Scenario：海运出口操作票关联共享 MBL

### 1. Scope / Trigger

- 新增或修改海运出口（`SE`）订单、共享主单、运输执行、HBL、候选关联、拆票或改单时适用。
- `Order` 是操作票；`SeaMasterBill` 是共享 MBL；`SeaTransportExecution` 是共享航程事实；`OrderShippingDocument` 只表示真实 HBL。
- 禁止从第一张 HBL 推断 MBL，也禁止把 MBL 号码复制到每张 HBL。一个活动操作票最多关联一个当前 MBL，一张 MBL 可以关联多张操作票。

### 2. Signatures

- 候选查询：`GET /api/v1/orders/sea-master-bill-candidate`。
- 订单写入：`CreateOrderRequest.sea_master_bill`、`UpdateOrderRequest.sea_master_bill`。
- 当前主单响应：`Order.sea_master_bill`。
- HBL 输入：`OrderShippingDocumentInput { id?, house_no, release_type?, note? }`，不含 MBL 字段。
- 数据库唯一键：
  - `sea_master_bills(organization_id, issuer_partner_id, normalized_master_no)`。
  - `sea_master_bill_order_links(order_id) WHERE status = 'ACTIVE'`。

### 3. Contracts

- SE 首次保存必须提交 `master_no` 和 `issuer_partner_id`；其他业务类型不受此门禁影响。
- `master_no` 只允许 ASCII 字母和数字。唯一规范化动作是 ASCII 小写转大写；不得 `TrimSpace`、删除标点或猜测签发主体。
- 命中已有 MBL 时只返回候选。写请求必须携带 `candidate_id` 和 `expected_candidate_version` 明确确认，服务端在事务内重新锁定并校验。
- 共享航程字段为承运人、起运港、卸货港、中转港、船名、航次、ETD、ETA；最终目的地仍归操作票，不参与共享匹配。
- 航程只在双方均有值时比较。多成员 MBL 不允许普通订单编辑改变共享航程；单成员只在 MBL 和订单均为草稿、版本匹配且无下游事实时允许修改。
- HBL 可为空；空占位行不持久化。填写 HBL 后号码为自由文本，仅去除首尾空白，一票可以有多张真实 HBL。
- 所有仓储读取都通过 `Data.client(ctx)`，并带组织谓词；事务中禁止回落到 `data.db`。

### 4. Validation & Error Matrix

| 条件 | 行为 |
|------|------|
| SE 首次保存缺 MBL 或签发主体 | 400 `SEA_MASTER_BILL_INVALID_ARGUMENT` |
| MBL 含空格、标点、非 ASCII 字母或数字 | 400，保留原输入供页面显示错误 |
| 同组织、同签发主体、同规范化 MBL 已存在但未确认 | 409 `SEA_MASTER_BILL_CONFIRMATION_REQUIRED`，事务回滚 |
| 候选 ID、组织、号码、主体或版本变化 | 409 状态/版本冲突，事务回滚 |
| 共享航程任一双方有值字段冲突 | 409 `SEA_MASTER_BILL_VOYAGE_CONFLICT`，禁止确认或保存 |
| 多成员 MBL 普通改号、改主体或改共享航程 | 409，并返回 `master_bill_id` 与受影响成员数 |
| 单成员修改缺当前 MBL 版本、存在下游事实或状态非草稿 | 409，不先删除 HBL、箱或其他下游数据 |
| HBL 空白且无其他内容 | 忽略占位行，不创建数据库记录 |

### 5. Good / Base / Bad Cases

- Good：用户输入 `cosco123`，页面转换为 `COSCO123`；候选未命中，在一个事务内创建订单、运输执行、MBL 和活动关联。
- Base：用户未填写 HBL，订单仍可保存；后续可以添加多张自由格式 HBL。
- Good：候选命中且航程一致，用户勾选确认后携带候选 ID/版本并入共享 MBL；已有 MBL 和运输执行不被修改。
- Bad：页面把候选查询失败当作“未命中”并创建新 MBL；必须阻止保存并提示重试。
- Bad：更新请求先删除已确认 HBL，再检查“无下游事实”；门禁必须先于订单、HBL、箱量等任何写入。

### 6. Tests Required

- Biz：MBL 空值、非法字符、小写转大写且不 trim；双方有值的逐字段航程冲突；最终目的地不参与。
- Service/HTTP：可空 UUID 不输出全零 UUID；静态候选路由不能被 `/orders/{id}` 吞掉。
- Data/PostgreSQL：新建原子回滚、同主体同号并发唯一、不同主体同号可并存、一票第二条 ACTIVE 被部分唯一索引拒绝。
- Data/PostgreSQL：候选版本重验、多成员航程冲突、单成员版本门禁、已确认 HBL/费用/账单等下游事实阻断。
- Frontend：候选请求失败阻止保存，冲突时不能确认；中转港及船名航次拆分与后端一致；空 HBL 不进入 payload。
- Migration：在一次性数据库分别验证冷启动、重复执行以及旧主分单表非空时任何 DDL 都未发生；禁止使用开发库充当集成测试库。
- Generation：连续两轮运行 API、Ent/Wire 和 Web Client 生成命令，tracked/untracked 内容哈希保持一致。

### 7. Wrong vs Correct

#### Wrong

```go
// 事务上下文被忽略；第一张 HBL 还被当成 MBL 真相。
client := r.data.db
masterNo := order.ShippingDocuments[0].MasterNo
```

```typescript
// 查询故障被伪装成候选未命中，可能误建重复 MBL。
try {
  await matchCandidate();
} catch {
  setCandidate(null);
}
```

#### Correct

```go
client, err := r.data.client(ctx)
if err != nil {
    return nil, err
}
masterBill := order.SeaMasterBill
```

```typescript
try {
  await matchCandidate();
} catch (error) {
  setCandidateQueryError(toMessage(error));
  // 表单校验读取该状态并阻止提交。
}
```
