# 海运出口共享主单与分单契约

## Scenario：海运出口操作票、共享 MBL 与真实 HBL

### 1. Scope / Trigger

- 新增或修改海运出口（`SE`）订单、共享主单、运输执行、HBL、候选关联、拆票、
  改配或改单时适用。
- `Order` 是操作票；`SeaTransportExecution` 是共享航程事实；`SeaMasterBill` 是
  共享 MBL；`SeaMasterBillOrderLink` 是操作票当前 MBL 关系；`SeaHouseBill` 是
  真实 HBL。
- `OrderShippingDocument` 不再承载 SE 的 HBL。禁止从第一张 HBL 推断 MBL、创建
  虚拟 HBL，或把 MBL 内容复制到每张 HBL。
- 一个活动操作票最多关联一个当前 MBL；一张 MBL 可以关联多张操作票；一个
  操作票可以有多张真实 HBL，但这些 HBL 必须属于该操作票当前活动 MBL。

### 2. Signatures

- 候选查询：`GET /api/v1/orders/sea-master-bill-candidate`。
- 订单写入：
  - `CreateOrderRequest.sea_master_bill: SeaMasterBillInput`。
  - `CreateOrderRequest.sea_document: SeaOrderDocumentInput`。
  - `UpdateOrderRequest.sea_master_bill: SeaMasterBillInput`。
  - `UpdateOrderRequest.sea_document: SeaOrderDocumentInput`；普通整单更新不得提交
    HBL 集合变更。
- 单证命令：
  - `GET /api/v1/orders/{order_id}/sea-documents`。
  - `POST /api/v1/orders/{order_id}/sea-documents/mark-direct`。
  - `POST /api/v1/orders/{order_id}/sea-documents/cancel-direct`。
  - `POST /api/v1/orders/{order_id}/sea-documents/house-bills`。
  - `PUT /api/v1/orders/{order_id}/sea-documents/house-bills/{id}`。
  - `DELETE /api/v1/orders/{order_id}/sea-documents/house-bills/{id}`。
  - `PUT /api/v1/orders/{order_id}/sea-documents/master-bill-content`。
- 并发字段：
  - 结构变更和 HBL 命令携带 `expected_link_version`。
  - HBL 更新/删除同时携带 `expected_version`。
  - MBL 内容修改携带 `expected_mbl_version`。
- 数据库唯一键：
  - `sea_master_bills(organization_id, issuer_partner_id, normalized_master_no)`。
  - `sea_master_bill_order_links(order_id) WHERE status = 'ACTIVE'`。
  - 本公司 HBL：`(organization_id, issuer_organization_id, normalized_house_no)`，
    条件为 `issuer_source = 'SELF_ORGANIZATION'`。
  - 外部主体 HBL：`(organization_id, issuer_partner_id, normalized_house_no)`，条件为
    `issuer_source IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER')`。

### 3. Contracts

- SE 首次保存必须提交 MBL `master_no` 和 `issuer_partner_id`；HBL 可为空。新建时
  无 HBL 且未明确直单，结构为 `UNDETERMINED`。
- MBL 号码只允许 ASCII 字母和数字，规范化只把 ASCII 小写转大写；不得
  `TrimSpace`、删除标点或猜测签发主体。
- HBL 原号必须无损保存。`normalized_house_no` 只执行 Unicode NFC、去首尾
  Unicode 空白、ASCII 小写转大写；不得删除内部空白/标点、改变年份或前导零。
- 单证结构只允许：
  - `UNDETERMINED`：无 HBL，尚未明确直单。
  - `DIRECT`：明确无 HBL；禁止直接新增 HBL。
  - `HOUSE`：至少一张真实 HBL。
- `DIRECT -> HOUSE` 禁止一步完成。必须先执行 `cancel-direct` 回到
  `UNDETERMINED`，再新增第一张 HBL；删除最后一张 HBL 必须显式提交
  `return_to_undetermined=true`，不得静默变成 DIRECT。
- HBL 签发主体无默认值：
  - `SELF_ORGANIZATION`：沿订单所属组织向上解析最近的 `company` 或
    `headquarters`，保存真实 Organization ID。系统不存在“一家公司下选择多个
    对外签发品牌”的场景，不创建自有 Partner、品牌配置、选择器或 seed。
  - `CUSTOMER_PARTNER`：固定保存订单当前客户 Partner ID，请求不得另传
    `issuer_partner_id`。
  - `OTHER_PARTNER`：必须显式传入同组织且启用的真实 Partner ID。
- 已有 `CUSTOMER_PARTNER` HBL 时，普通订单保存不得把客户改成其他主体；用户先
  调整 HBL 签发主体，系统不自动改写历史语义。
- MBL 与每张 HBL 分别维护完整提单内容，内容字段可空。阶段 2 不根据订单件重尺
  做守恒校验，也不自动镜像订单字段。
- 命中已有 MBL 时只返回候选；写请求必须携带候选 ID 和版本明确确认，服务端在
  事务内重新锁定并校验。共享航程字段为承运人、起运港、卸货港、中转港、船名、
  航次、ETD、ETA；最终目的地仍归操作票。
- 单证写命令固定加锁顺序为 `Order -> SeaMasterBill -> Active Link -> HBL/后续
  验证对象`。用无锁查询定位 ID 后，取得写锁必须重新校验组织、订单、活动状态
  和 MBL 关系；禁止先锁 Link 再升级 MBL。
- 创建、结构切换、MBL 内容修改和 HBL 命令必须在同一事务写业务审计；审计失败
  必须回滚业务写入。事务提交后才用普通上下文重读响应。
- 所有仓储读取都通过 `Data.client(ctx)` 并带组织谓词；事务中禁止回落到
  `data.db`。
- PostgreSQL 集成测试只接受 `RONCIN_INTEGRATION_DATABASE_SOURCE`，每个测试在
  独立 Schema 执行真实迁移；禁止静默使用开发库或公共 Schema。

### 4. Validation & Error Matrix

| 条件 | 行为 |
|------|------|
| SE 首次保存缺 MBL、号码或签发方 | 400 `SEA_MASTER_BILL_INVALID_ARGUMENT` |
| MBL 含空格、标点、非 ASCII 字母或数字 | 400 `SEA_MASTER_BILL_INVALID_ARGUMENT` |
| 同组织、同签发主体、同规范化 MBL 已存在但未确认 | 409 `SEA_MASTER_BILL_CONFIRMATION_REQUIRED`，事务回滚 |
| 候选身份、版本或共享航程变化 | 409 对应 MBL 冲突错误，事务回滚 |
| 单证聚合找不到活动 Link | 400 `SEA_DOCUMENT_NO_ACTIVE_LINK`，不得构造虚假默认响应 |
| DIRECT 直接新增 HBL | 409 `SEA_DOCUMENT_DIRECT_ADD_HBL_BLOCKED` |
| 删除最后一张 HBL 未明确回到未确定 | 400 `SEA_DOCUMENT_DELETE_LAST_HBL_CONFIRMATION_REQUIRED` |
| Link、MBL 或 HBL 预期版本不一致 | 409 `SEA_DOCUMENT_STRUCTURE_CONFLICT`、`SEA_MASTER_BILL_CONFLICT` 或 `SEA_HOUSE_BILL_CONFLICT` |
| HBL 号为空或超过 128 个字符 | 400 `SEA_HOUSE_BILL_INVALID_ARGUMENT` |
| 同一真实签发主体的规范化 HBL 号重复 | 409 `SEA_HOUSE_BILL_EXISTS`，由数据库唯一索引兜底 |
| SELF/CUSTOMER 请求额外 Partner ID，或 OTHER 未选择 Partner | 400 `SEA_HOUSE_BILL_INVALID_ARGUMENT` |
| SELF 无法解析到 company/headquarters | 400 `SEA_HOUSE_BILL_INVALID_ARGUMENT`，不得创建占位主体 |
| 已有 CUSTOMER HBL 时修改订单客户 | 409 `ORDER_CUSTOMER_CHANGE_WITH_HOUSE_BILL_BLOCKED` |
| 件数、毛重、体积为负，浮点值为 NaN/Inf，或文本超限 | 400 `SEA_BILL_CONTENT_INVALID_ARGUMENT` |
| 审计写入失败 | 接口失败，业务写入和版本增加全部回滚 |

### 5. Good / Base / Bad Cases

- Good：新建 SE 订单填写 MBL，不填写 HBL，保存为 `UNDETERMINED`；用户明确点击
  “标记为直单”后才进入 `DIRECT`。
- Good：用户取消 DIRECT，再新增 `  hbl/001  `；原号按输入保存，唯一检索键为
  `HBL/001`，结构在同一事务变为 `HOUSE`。
- Good：HBL 选择“本公司”，订单属于部门，系统保存其最近公司/总部的 Organization
  ID，页面显示解析后的名称，不显示品牌选择器。
- Base：MBL/HBL 内容全部留空仍可保存；阶段 2 不要求用户实际使用全部单证字段。
- Bad：DIRECT 页面直接显示“添加 HBL”并让后端自动转 HOUSE；必须先执行独立取消
  动作，留下单独审计。
- Bad：创建 payload 静默过滤不完整 HBL；必须保留用户输入并由表单/服务端明确
  拒绝。
- Bad：查询故障被伪装成“未命中”或“无活动 Link”；必须原样返回错误，禁止创建
  新 MBL 或合成默认结构。

### 6. Tests Required

- Biz：MBL 必填及字符规则；HBL 原号无损和规范化边界；SELF/CUSTOMER/OTHER
  互斥；DIRECT 三态转换；内容长度、NaN/Inf、负数。
- Service/HTTP：请求 UUID、枚举和必填对象转换；可空 UUID 不输出全零 UUID；静态
  路由不能被 `/orders/{id}` 吞掉；错误 reason 可供前端稳定识别。
- Data/PostgreSQL：新建原子回滚、同主体同号并发唯一、不同主体同号可并存、一票
  第二条 ACTIVE 被部分唯一索引拒绝。
- Data/PostgreSQL：真实 `writeAudit` 失败后结构/版本/业务行回滚；并发单证命令无
  死锁；批量摘要必须按 `(order_id, active_master_bill_id)` 过滤历史 HBL。
- Data/PostgreSQL：从完整订单创建入口验证初始 HOUSE、HBL 数量和 `order.create`
  审计详情。
- Frontend：候选请求失败阻止保存；DIRECT 无添加入口；取消后可添加；最后 HBL
  删除确认；原号不 trim；不静默过滤；加载失败清空旧订单状态；区块默认展开。
- Migration：独立 Schema 从阶段 1 真实迁移到阶段 2 并核对表、列、约束、索引和
  revision；存在 SE 数据时迁移原子拒绝且不留下部分 DDL。
- Generation：连续运行 API、Ent/Wire、Proto 常量和 Web Client 生成命令，
  tracked/untracked 内容指纹保持一致。

### 7. Wrong vs Correct

#### Wrong

```go
// 同一事务先锁 Link 再锁 MBL，会与单证命令形成交叉死锁。
link := tx.SeaMasterBillOrderLink.Query().ForUpdate().OnlyX(ctx)
mbl := tx.SeaMasterBill.Query().Where(seamasterbill.ID(link.MasterBillID)).ForUpdate().OnlyX(ctx)
```

```typescript
// 丢弃用户已填写但不完整的 HBL，界面看似保存成功却发生数据丢失。
const houseBills = values.seaHouseBills.filter((item) => item.houseNo?.trim());
```

#### Correct

```go
// Order 已锁；先无锁定位 ID，再按全局顺序加锁并重验活动关系。
candidate := client.SeaMasterBillOrderLink.Query().Where(
    seamasterbillorderlink.OrderIDEQ(orderID),
    seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
).OnlyX(ctx)
mbl := client.SeaMasterBill.Query().Where(seamasterbill.IDEQ(candidate.MasterBillID)).ForUpdate().OnlyX(ctx)
link := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlink.IDEQ(candidate.ID)).ForUpdate().OnlyX(ctx)
if link.Status != seamasterbillorderlink.StatusACTIVE || link.MasterBillID != mbl.ID {
    return biz.ErrSeaDocumentStructureConflict
}
```

```typescript
// 不静默改写或丢弃；所有表单行进入 payload，再由明确校验给出错误。
const houseBills = (values.seaHouseBills ?? []).map(toSeaHouseBillInput);
```
