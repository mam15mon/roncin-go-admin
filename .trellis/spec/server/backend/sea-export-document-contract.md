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
  操作票最多拥有一张真实 HBL，且该 HBL 必须属于其当前活动 MBL。
- 一票一分是 2026-09 业务确认的口径（所有分单独立建订单、独立建账单）：
  「一主多分」由多张操作票共享同一 `SeaMasterBill` 承载，契约不提供一订单
  多 HBL 形态；旧 `repeated house_bills`、分单 POST/DELETE 接口与
  `house_bill_count` / `house_nos` 摘要均已 reserved 移除。

### 2. Signatures

- 候选查询：`GET /api/v1/orders/sea-master-bill-candidate`。
- 订单写入：
  - `CreateOrderRequest.sea_master_bill: SeaMasterBillInput`。
  - `CreateOrderRequest.sea_document: SeaOrderDocumentInput`。
  - `UpdateOrderRequest.sea_master_bill: SeaMasterBillInput`。
  - `UpdateOrderRequest.sea_document: SeaOrderDocumentInput`；普通整单更新不得
    提交分单变更（分单只经单证命令维护）。
- 单证命令：
  - `GET /api/v1/orders/{order_id}/sea-documents`（聚合读取返回单数 `house_bill`）。
  - `PUT /api/v1/orders/{order_id}/sea-documents/house-bills/{id}`（更新唯一
    活动 HBL）。
  - `PUT /api/v1/orders/{order_id}/sea-documents/master-bill-content`。
  - `POST /api/v1/orders/{order_id}/sea-documents/mode-change/preview` 与
    `POST .../mode-change`：DIRECT/HOUSE 互转。DIRECT→HOUSE 随切换创建唯一
    HBL；HOUSE→DIRECT 将唯一活动 HBL 置为 VOIDED，不硬删除。
  - `POST .../amendments/preview` 与 `POST .../amendments`（改单）；
    `POST .../voids/preview` 与 `POST .../voids`（作废）。
  - 版本与事件：`GET .../versions/{version_id}`、`GET .../master-bill/versions`、
    `GET .../house-bills/{house_bill_id}/versions`、`GET .../events`。
  - 旧 `mark-direct` / `cancel-direct` 与分单 POST/DELETE 接口已随单值收敛移除。
- 放货记录命令：
  - `GET /api/v1/orders/{order_id}/release-pods`。
  - `POST /api/v1/orders/{order_id}/release-pods`。
  - `PUT /api/v1/orders/{order_id}/release-pods/{id}`。
  - `POST /api/v1/orders/{order_id}/release-pods/{id}/transition`。
  - `DELETE /api/v1/orders/{order_id}/release-pods/{id}`。
- 放货记录单证字段：旧模型使用 `shipping_document_id`；SE 使用
  `sea_document_type + sea_document_id`，其中类型只允许 `MASTER_BILL | HOUSE_BILL`。
- HBL 不提供硬删除；退出分单只能经 HOUSE→DIRECT 模式切换作废，历史版本与
  已关联 ReleasePod 记录全部保留。
- 并发字段：
  - HBL 更新与模式切换携带 `expected_link_version`。
  - HBL 更新携带 `expected_version`；模式切换额外携带 `expected_order_version`、
    `expected_house_bill_version` 与 `expected_current_version_id`。
  - MBL 内容修改携带 `expected_mbl_version`。
- 数据库唯一键：
  - `sea_master_bills(organization_id, shipping_line_id, normalized_master_no)`。
  - `sea_master_bill_order_links(order_id) WHERE status = 'ACTIVE'`。
  - 本公司 HBL：`(organization_id, issuer_organization_id, normalized_house_no)`，
    条件为 `issuer_source = 'SELF_ORGANIZATION'`。
  - 外部主体 HBL：`(organization_id, issuer_partner_id, normalized_house_no)`，条件为
    `issuer_source IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER')`。
- `order_release_pods` 持有三个可空真实外键：`shipping_document_id`、
  `sea_master_bill_id`、`sea_house_bill_id`；CHECK
  `num_nonnulls(...) <= 1`，Sea MBL/HBL 外键删除策略均为 `NO ACTION`。

### 3. Contracts

- SE 首次保存必须选择 `Order.shipping_line_id`（页面名称“船公司”）并提交 MBL
  `master_no`；HBL 可为空。新建时无 HBL 且未明确直单，结构为 `UNDETERMINED`。
- 海运船公司是 `ShippingLine` 行业参考数据，不是 Partner 商业往来方。所有 SE 创建、
  普通整单更新、拆票 NEW 目标和整票改配 NEW 目标必须满足
  `Order.shipping_line_id = SeaTransportExecution.shipping_line_id = SeaMasterBill.shipping_line_id`；
  前端和服务端只传递、保存一份船公司身份，不再从 Partner carrier 或 MBL issuer
  派生船公司。
- 新建或更换船公司时，必须选择同组织且启用的 `ShippingLine`；历史引用后来被停用时，
  未更换船公司的其他字段修改仍可保存。拆票和改配的 CANDIDATE 在 Preview 与 Execute
  都必须复验同组织且启用，Execute 在共享事务中用 `FOR SHARE` 稳定该引用。
- 普通 SE 整单更新仍必须提交 MBL 输入；独立的 MBL 内容更新可以不重复提交 MBL
  身份，但必须保持当前 shipping line 不变，并验证 Order、运输执行和 MBL 的三方身份一致。
- MBL 的唯一索引、不可变版本、锁定快照和响应都保存 `shipping_line_id`；不得同时保留
  carrier/issuer 两套身份。HBL 的 `issuer_partner_id` 是独立业务事实，继续引用 Partner，
  不适用船公司三方一致规则。
- Partner 角色只允许 `customer | supplier | foreign_agent`，不得恢复 `carrier`；Proto
  中已删除的 carrier 枚举编号和名称必须保留为 `reserved`，禁止复用。
- `ShippingLine` 只参与航线、船名航次、箱号和提单前缀等海运操作事实，绝不能写入
  `settlement_party_id`。应付费用的默认结算方只能使用订舱代理 Partner；没有订舱代理时
  留空，不得回退到 ShippingLine。
- MBL 号码只允许 ASCII 字母和数字，规范化只把 ASCII 小写转大写；不得
  `TrimSpace`、删除标点或根据号码猜测船公司。
- HBL 原号必须无损保存。`normalized_house_no` 只执行 Unicode NFC、去首尾
  Unicode 空白、ASCII 小写转大写；不得删除内部空白/标点、改变年份或前导零。
- 单证结构只允许：
  - `UNDETERMINED`：无 HBL，尚未明确直单。
  - `DIRECT`：明确无 HBL。
  - `HOUSE`：恰好一张真实 HBL。
- 结构互转只经模式切换命令完成（预览 + 执行，需外部确认与幂等键）：
  DIRECT→HOUSE 随切换提交唯一 HBL；HOUSE→DIRECT 将唯一活动 HBL 作废。
  不存在新增第二张 HBL、硬删除或回到 `UNDETERMINED` 的路径。
- HBL 签发主体无默认值：
  - `SELF_ORGANIZATION`：沿订单所属组织向上解析最近的 `company` 或
    `headquarters`，保存真实 Organization ID。系统不存在“一家公司下选择多个
    对外签发品牌”的场景，不创建自有 Partner、品牌配置、选择器或 seed。
  - `CUSTOMER_PARTNER`：固定保存订单当前客户 Partner ID，请求不得另传
    `issuer_partner_id`。
  - `OTHER_PARTNER`：必须显式传入同组织且启用的真实 Partner ID。
- 已有 `CUSTOMER_PARTNER` HBL 时，普通订单保存不得把客户改成其他主体；用户先
  调整 HBL 签发主体，系统不自动改写历史语义。
- MBL 与 HBL 分别维护完整提单内容，内容字段可空。阶段 2 不根据订单件重尺
  做守恒校验，也不自动镜像订单字段。
- 命中已有 MBL 时只返回候选；写请求必须携带候选 ID 和版本明确确认，服务端在
  事务内重新锁定并校验。共享航程字段为承运人、起运港、卸货港、中转港、船名、
  航次、ETD、ETA；最终目的地仍归操作票。
- 单证写命令固定加锁顺序为 `Order -> SeaMasterBill -> Active Link -> HBL/后续
  验证对象`。用无锁查询定位 ID 后，取得写锁必须重新校验组织、订单、活动状态
  和 MBL 关系；禁止先锁 Link 再升级 MBL。
- SE ReleasePod 最多关联当前活动 Link 指向的共享 MBL，或同组织、同订单且属于该
  MBL 的有效 HBL；非 SE 只允许旧 `OrderShippingDocument`。新增/更新必须先锁
  Order，再在同一事务内验证并写入；禁止事务外先查后写，也禁止根据 UUID 形态
  猜测单证类型。
- 海运单证页只有具备 `release_pod.read` 权限时才请求和展示关联记录；按真实
  MBL/HBL ID 分组展示放货编号、回单编号和状态，并明确区分空态与加载失败。
- HOUSE→DIRECT 模式切换在既有锁序内把唯一活动 HBL 置为 VOIDED 并递增版本；
  不删除 HBL 行与其关联 ReleasePod，历史关系继续可查。切换与 MBL/HBL 命令
  一样在同一事务写操作日志，日志失败全部回滚。
- MBL/HBL 作废不是硬删除，不得清空或级联删除已关联 ReleasePod；历史关系继续
  保留。
- 创建、结构切换、MBL 内容修改和 HBL 命令必须在同一事务写操作日志；日志失败
  必须回滚业务写入。事务提交后才用普通上下文重读响应。
- 所有仓储读取都通过 `Data.client(ctx)` 并带组织谓词；事务中禁止回落到
  `data.db`。
- PostgreSQL 集成测试只接受 `RONCIN_INTEGRATION_DATABASE_SOURCE`，每个测试在
  独立 Schema 执行真实迁移；禁止静默使用开发库或公共 Schema。

### 4. Validation & Error Matrix

| 条件 | 行为 |
|------|------|
| SE 创建或普通整单更新缺船公司、MBL 输入或 MBL 号 | 400 对应订单/MBL InvalidArgument，零写入 |
| MBL 内容独立更新改变 shipping line，或现有 Order/TE/MBL 身份不一致 | 400 `SEA_MASTER_BILL_INVALID_ARGUMENT`，零写入 |
| 新选/更换的 ShippingLine 不属于当前组织或已停用 | 400 对应订单/MBL InvalidArgument，零写入 |
| 拆票/改配 CANDIDATE 的 ShippingLine 被停用 | Preview 明确拒绝；Execute 锁内复验并回滚 |
| MBL 含空格、标点、非 ASCII 字母或数字 | 400 `SEA_MASTER_BILL_INVALID_ARGUMENT` |
| 同组织、同船公司、同规范化 MBL 已存在但未确认 | 409 `SEA_MASTER_BILL_CONFIRMATION_REQUIRED`，事务回滚 |
| 候选身份、版本或共享航程变化 | 409 对应 MBL 冲突错误，事务回滚 |
| 单证聚合找不到活动 Link | 400 `SEA_DOCUMENT_NO_ACTIVE_LINK`，不得构造虚假默认响应 |
| HOUSE→DIRECT 缺外部确认、幂等键，或 HBL 预期版本/当前版本 ID 不一致 | 400/409 对应单证冲突错误，事务回滚 |
| Link、MBL 或 HBL 预期版本不一致 | 409 `SEA_DOCUMENT_STRUCTURE_CONFLICT`、`SEA_MASTER_BILL_CONFLICT` 或 `SEA_HOUSE_BILL_CONFLICT` |
| HBL 号为空或超过 128 个字符 | 400 `SEA_HOUSE_BILL_INVALID_ARGUMENT` |
| 同一真实签发主体的规范化 HBL 号重复 | 409 `SEA_HOUSE_BILL_EXISTS`，由数据库唯一索引兜底 |
| SELF/CUSTOMER 请求额外 Partner ID，或 OTHER 未选择 Partner | 400 `SEA_HOUSE_BILL_INVALID_ARGUMENT` |
| SELF 无法解析到 company/headquarters | 400 `SEA_HOUSE_BILL_INVALID_ARGUMENT`，不得创建占位主体 |
| 已有 CUSTOMER HBL 时修改订单客户 | 409 `ORDER_CUSTOMER_CHANGE_WITH_HOUSE_BILL_BLOCKED` |
| 件数、毛重、体积为负，浮点值为 NaN/Inf，或文本超限 | 400 `SEA_BILL_CONTENT_INVALID_ARGUMENT` |
| 操作日志写入失败 | 接口失败，HBL、ReleasePod、Link 版本和其他业务写入全部回滚 |

### 5. Good / Base / Bad Cases

- Good：新建 SE 订单选择 ShippingLine 并填写 MBL 号，不填写 HBL；系统将同一个
  `shipping_line_id` 写入 Order、运输执行和 MBL，保存为 `UNDETERMINED`。用户明确点击“标记为直单”
  后才进入 `DIRECT`。
- Good：历史订单引用的 ShippingLine 后来被停用，用户不更换船公司而修改其他字段；
  系统保留历史引用，不把停用误判为身份变更。
- Good：DIRECT 订单经模式切换提交 `  hbl/001  `；原号按输入保存，唯一检索键为
  `HBL/001`，结构在同一事务变为 `HOUSE`。
- Good：HBL 选择“本公司”，订单属于部门，系统保存其最近公司/总部的 Organization
  ID，页面显示解析后的名称，不显示品牌选择器。
- Good：HOUSE 订单经模式切换退回 DIRECT，页面预览列出影响（唯一 HBL 作废）；
  用户一次确认后，结构、HBL 状态和操作日志原子完成，放单记录保留可查。
- Base：MBL/HBL 内容全部留空仍可保存；阶段 2 不要求用户实际使用全部单证字段。
- Base：ReleasePod 不关联任何单证；三个外键均为空，列表正常展示。
- Bad：在普通整单更新或编辑表单里夹带分单增删；分单只经单证命令维护，
  一切结构变化都经模式切换并留下操作日志。
- Bad：创建 payload 静默过滤不完整 HBL；必须保留用户输入并由表单/服务端明确
  拒绝。
- Bad：查询故障被伪装成“未命中”或“无活动 Link”；必须原样返回错误，禁止创建
  新 MBL 或合成默认结构。
- Bad：把 ShippingLine 当成 Partner carrier，或在没有订舱代理时把船公司 UUID 写入
  应付费用的 `settlement_party_id`。

### 6. Tests Required

- Biz：SE ShippingLine 与 MBL 必填、三方身份一致及字符规则；HBL 原号无损和
  规范化边界；SELF/CUSTOMER/OTHER 互斥；DIRECT 三态转换；内容长度、NaN/Inf、负数。
- Service/HTTP：请求 UUID、枚举和必填对象转换；可空 UUID 不输出全零 UUID；静态
  路由不能被 `/orders/{id}` 吞掉；错误 reason 可供前端稳定识别。
- Data/PostgreSQL：创建/单成员更正后 Order、TE、MBL 的 ShippingLine 三方一致；共享 MBL 的 shipping line
  修改被原子阻断；同船公司同号并发唯一、不同船公司同号可并存；一票第二条 ACTIVE
  被部分唯一索引拒绝。
- Data/PostgreSQL：新选/更换 ShippingLine 校验组织和启用状态；未更换的停用历史引用
  可保留；拆票/改配 CANDIDATE 在 Preview 与 Execute 均拒绝停用引用。
- Data/PostgreSQL：真实 `writeAudit` 失败后结构/版本/业务行回滚；并发单证命令无
  死锁；批量摘要必须按 `(order_id, active_master_bill_id)` 过滤历史 HBL。
- Data/PostgreSQL：ReleasePod 的 MBL/HBL 当前归属、跨组织/跨订单拒绝、三引用
  CHECK、Sea 外键 `NO ACTION`；模式切换 DIRECT↔HOUSE 的预览/执行、唯一 HBL
  创建与作废，以及操作日志失败后结构/版本/业务行全部回滚。
- Data/PostgreSQL：从完整订单创建入口验证初始单证结构与 `order.create`
  操作日志详情。
- Frontend：SE 船公司和 MBL 号必填且无 MBL 签发方输入；创建、详情、候选、拆票和
  改配都只传 ShippingLine；共享 MBL 同时禁用船公司与 MBL 号；候选请求失败阻止
  保存；结构互转只经模式切换预览与确认；原号不 trim；不静默过滤；加载失败清空
  旧订单状态；区块默认展开。
- Frontend：船公司选择器调用 ShippingLine 服务端关键字查询；详情即使历史船公司已停用，
  也能回显当前名称；应付结算方只默认订舱代理，不回退到船公司。
- Frontend：无 `release_pod.read` 权限时零请求；有权限时 MBL/HBL 分组、空态和
  错误态。
- Migration：独立 Schema 从阶段 1 真实迁移到阶段 2，并核对四个 ShippingLine 外键、
  `NO ACTION`、TE 非空、MBL 唯一索引、Partner role CHECK 和 revision；保留已有
  ShippingLine。存在旧 SE/TE/MBL/MBL Version 数据或非法 Partner role 时，迁移必须
  在任何 DDL 前原子拒绝且不留下部分变更，不猜测旧引用的映射关系。
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

```typescript
// ShippingLine 不是结算 Partner，禁止把船公司作为应付结算方兜底。
const settlementPartyId = values.bookingAgentId || values.shippingLineId;
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

```typescript
// 应付结算方只默认订舱代理；没有商业往来方时保持未选择。
const settlementPartyId = values.bookingAgentId;
```

```go
// 类型和 ID 显式成对；锁住订单后再验证当前活动 MBL/HBL 归属。
input := &biz.OrderReleasePod{
	SeaDocumentType: biz.SeaDocumentTypeHouseBill,
	SeaDocumentID:   &houseBillID,
}
return data.WithinTransaction(ctx, func(txCtx context.Context) error {
	client, err := data.client(txCtx)
	// Order -> MBL -> Active Link -> HBL，随后才写 ReleasePod。
	return err
})
```

## Scenario：拆票与改配目标引用闭环

### 1. Scope / Trigger

- 拆票请求用客户端键关联“结果票”和“目标 MBL”，或整体改配在候选 MBL 与本次新建
  MBL 之间选择目标时适用。
- 该键只用于同一次命令内建图；不能被 data 层解释成可缺省的提示字段。

### 2. Signatures

- 拆票：`targets[].client_target_key` 定义目标，
  `results[].client_target_key` 必须引用其中恰好一个目标。
- 拆票目标类型：`CURRENT | CANDIDATE | NEW`。
- 整体改配目标类型：`CANDIDATE | NEW`，不接受 `CURRENT`。
- 候选目标必须携带 MBL ID/version 与运输执行 ID/version；预期版本必须与候选字段一致。

### 3. Contracts

- `targets[].client_target_key` 去首尾空白后非空，并在请求内唯一；重复键不得按数组顺序
  覆盖。
- 每个结果键必须命中已定义目标；缺失目标不得按 `CURRENT` 处理。
- `CURRENT` 不得夹带候选或新建字段；`CANDIDATE` 必须提供完整候选身份与版本，并在
  锁内验证候选 MBL 与候选运输执行引用同一个 ShippingLine；`NEW` 不得夹带候选字段，
  必须提供同组织且启用的 ShippingLine，并满足新 MBL 的号码、港口和日期规则。NEW
  目标的 Order、运输执行和 MBL 均由同一个 `shipping_line_id` 维护。
- Preview 与 Execute 必须复用同一个 biz 输入校验；Execute 仍在事务锁内重验候选身份、
  版本、唯一性和共享航程。
- Execute 事务前只允许校验不依赖数据库当前状态的结构规则、幂等键和必填预期版本；
  禁止调用有状态 `PreviewSplit` 作为执行门禁。事务内必须先锁 Order 并比较
  `expected_order_version`，再校验 Link、Allocation、HBL、箱货、费用和候选 MBL/TE。
  这保证相同版本的并发请求稳定为一成功一 409，而不会被胜方提交后的新状态抢先
  解释为 400。
- data 层按已验证的显式类型 `switch`，`default` 返回领域错误，禁止依据某个可空 ID
  猜测目标类型。

### 4. Validation & Error Matrix

| 条件 | 行为 |
|------|------|
| 目标键为空或重复 | 400 `SEA_ORDER_SPLIT_INVALID_ARGUMENT` |
| 结果引用未知目标键 | 400 `SEA_ORDER_SPLIT_INVALID_ARGUMENT`，不得回退当前 MBL |
| 拆票目标类型未知 | 400 `SEA_ORDER_SPLIT_INVALID_ARGUMENT` |
| 改配目标为 `CURRENT`、空值或未知类型 | 400 `SEA_ORDER_REASSIGNMENT_INVALID_ARGUMENT` |
| CURRENT/NEW 夹带其他类型字段 | 对应 400 InvalidArgument |
| 候选身份字段缺失或版本与 expected map 不一致 | 对应 400/409，Preview 与 Execute 一致 |
| NEW 缺船公司，或船公司、港口、日期、MBL 号非法 | Preview 即明确拒绝，Execute 同样拒绝 |
| CANDIDATE 的 MBL 与运输执行 ShippingLine 不一致 | 409 对应候选冲突错误，事务回滚 |
| CANDIDATE 的 ShippingLine 不属于当前组织或已停用 | Preview 拒绝；Execute 在事务内 `FOR SHARE` 复验并回滚 |
| 两个不同幂等键以同一 Order 版本并发执行合法拆票 | 一个成功，另一个 409 `SEA_ORDER_SPLIT_VERSION_CONFLICT` |
| Execute 的 Order/Link/Allocation/HBL/箱货/费用版本过期 | 409 `SEA_ORDER_SPLIT_VERSION_CONFLICT`，不得先返回可变业务状态 400 |

### 5. Good / Base / Bad Cases

- Good：两张结果票都引用同一个已定义 `NEW_A`，船公司默认沿用来源但允许用户修改；
  服务端只创建一个目标 MBL，以该 ShippingLine 同时维护目标 Order、运输执行和 MBL，
  再分别建立显式关系。
- Base：结果引用 `CURRENT` 目标，顶层仍显式声明该键和类型，不靠空字符串表达沿用。
- Bad：结果传入 `MISSING`，data 查询不到后自动沿用当前 MBL。
- Bad：整体改配传 `UNKNOWN`，因候选 ID 为空而被猜成 NEW。
- Bad：Execute 事务前调用完整 Preview；并发失败方先看到胜方提交后的 HBL/费用状态，
  返回“数量不足”等 400，导致用户无法通过刷新恢复。

### 6. Tests Required

- Biz：空键、重复键、未知引用、未知类型、CURRENT/NEW 夹带字段、CANDIDATE 缺字段与
  expected version 不一致；Preview/Execute 断言相同错误。
- Service：畸形 DTO 转换后仍被领域校验拒绝，错误 reason 不变化。
- Data/PostgreSQL：未知键不写入；共享 NEW 键只建一个 MBL 且目标 Order、TE、MBL
  的 ShippingLine 三方身份一致；候选版本锁后变化、候选 MBL/TE 身份不一致或候选
  ShippingLine 停用时拒绝；Preview/Execute 对缺少/非法船公司、港口、日期和重复
  MBL 一致。
- Data/PostgreSQL：完整拆票父测试至少 `-count=3`；每轮断言同版本并发一成功一
  `SEA_ORDER_SPLIT_VERSION_CONFLICT`，且无双成功、孤儿行或重复事件。

### 7. Wrong vs Correct

#### Wrong

```go
target := targetMap[result.ClientTargetKey]
if target == nil || target.TargetType == SplitTargetTypeCurrent {
	return useCurrentMasterBill()
}
```

#### Correct

```go
target, ok := targetMap[result.ClientTargetKey]
if !ok {
	return ErrSeaOrderSplitInvalidArgument
}
switch target.TargetType {
case SplitTargetTypeCurrent, SplitTargetTypeCandidate, SplitTargetTypeNew:
	// 按显式类型处理；各分支的互斥字段已由共享 biz 校验确认。
default:
	return ErrSeaOrderSplitInvalidArgument
}
```

```go
// Execute 不调用有状态 Preview；仓储事务先取得版本裁决权，再重验业务状态。
if err := validateSplitTargetsAndResults(input.Targets, input.Results, input.ExpectedVersions); err != nil {
	return nil, err
}
return transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
	return repo.ExecuteSplit(txCtx, organizationID, actorID, input, operationLog)
})
```
