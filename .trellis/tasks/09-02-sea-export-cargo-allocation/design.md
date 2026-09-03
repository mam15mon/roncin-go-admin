# 海运出口箱货分配设计

## 设计目标与阶段边界

阶段 3 在阶段 2 的真实多 HBL 上建立可审计的定量分配事实，让一条订单货物可以
拆到多张 HBL 和多只实际箱，也允许一只箱承载多张 HBL。系统在草稿阶段允许逐步
补齐，但从第一次保存起就拒绝超分；确认时强制件数、毛重和体积完整守恒。

本阶段不实现拆票、整体改配、甩柜、Switch B/L、不可变签发版本或费用重分摊。
订舱箱型计划仍只是计划，不会被自动扩成虚假箱号。

## 领域关系与聚合边界

```text
Order（操作票） 1 ── N OrderCargoItem（货物来源）
       │
       ├── N OrderContainer（已知箱号的实际箱，仅 FCL）
       │
       └── 1 Active SeaMasterBillOrderLink（当前 MBL 成员）
                        │
                        ├── allocation_status/version
                        ├── N SeaHouseBill（HOUSE 单证归属）
                        └── N SeaCargoAllocation
                              ├── 1 CargoItem
                              ├── 1 HouseBill
                              └── 0..1 Container
```

- 分配聚合属于“操作票在某次 MBL 成员关系下的箱货事实”，因此分配行必须关联
  `SeaMasterBillOrderLink`，不能只挂 Order。阶段 4 终止旧成员关系时，旧分配仍能
  随该关系追溯。
- `OrderCargoItem` 是件数、毛重和体积的唯一操作票级数量来源。
- `OrderContainer` 只表示已有真实箱号的实际箱；`OrderContainerRequest` 永远不
  参与分配。
- `SeaHouseBill` 是分配的单证归属。HBL 内容里的件重尺是显示快照，不是分配来源。

## 数据模型

### `SeaMasterBillOrderLink` 扩展

- `cargo_allocation_status`：`DRAFT|CONFIRMED`，默认 `DRAFT`。
- `cargo_allocation_version`：从 1 开始；保存草稿、确认、撤回，以及货物/HBL 集合/
  实际箱变化时递增。
- `cargo_allocation_confirmed_at`、`cargo_allocation_confirmed_by`：确认时写入，撤回时
  清空。

单证结构仍使用既有 `version`；箱货编辑使用独立 `cargo_allocation_version`，避免
普通提单内容保存无意义地制造分配冲突。HBL 增删会同时改变集合版本和分配版本。

### `SeaCargoAllocation`

每行保存：

- `organization_id`、`order_id`、`master_bill_order_link_id`
- `cargo_item_id`、`house_bill_id`、可空 `container_id`
- `package_count`：正整数
- `gross_weight_kg`：`numeric(18,3)`，正数
- `volume_cbm`：`numeric(18,6)`，正数
- 创建/更新时间

同一聚合内建立两组条件唯一键：

- 无箱：`link_id + cargo_item_id + house_bill_id`，条件为 `container_id IS NULL`；
- 有箱：`link_id + cargo_item_id + house_bill_id + container_id`，条件为
  `container_id IS NOT NULL`。

批量草稿保存采用全量替换语义和聚合版本，不为分配行再增加一套独立乐观版本。
已有行 ID 可回传以保持稳定；请求遗漏的旧行在同一事务删除。

### 货物与实际箱扩展

`OrderCargoItem` 增加 `organization_id` 和 `version`；`OrderContainer` 增加
`organization_id`、`version`、正整数 `package_count`。两类更新/删除均携带
`expected_version`。

移除 `OrderContainer.shipping_document_id` 及其旧 API、领域字段、Ent 边和海运
页面下拉。阶段 2 的真实多 HBL 与分配表是唯一关系，不建立单值 HBL 替代字段。

现有货物和实际箱重量/体积列改为 PostgreSQL `numeric(18,3)` / `numeric(18,6)`；
Proto 兼容保留 `double`，biz 使用 `decimal.NewFromFloat` 转为十进制并拒绝超过规定
小数位的输入，不做静默舍入。新分配契约直接使用十进制字符串，前后端不经过
JavaScript 浮点计算。

## 分配状态与守恒规则

### 允许状态

- 只有 `HOUSE` 可以保存、确认或撤回 HBL 分配。
- `UNDETERMINED` 和 `DIRECT` 请求分配时返回状态冲突；DIRECT 绝不创建虚拟 HBL。
- `DRAFT` 允许缺少部分货物、HBL 或实际箱分配，但每次保存都必须是合法正数，且
  任一已知基准均不得被超过。
- `CONFIRMED` 禁止普通货物、实际箱、HBL 和分配修改；必须携带聚合预期版本显式
  撤回到 `DRAFT`。

### 草稿保存校验

服务端按完整请求重新汇总，而不是相信前端计算：

1. 所有引用必须属于当前组织、当前 Order、当前活动 Link/MBL；
2. 每个 cargo item 的分配合计不得超过其件数、毛重、体积；
3. 有 container 的行，其分配合计不得超过该箱件数、毛重、体积；
4. 单张 HBL 的分配合计不得超过整个操作票对应维度总量，用于直接定位明显错误；
5. LCL/散杂行的 `container_id` 必须为空；FCL 草稿可暂时为空，等待真实箱号取得。

### 确认校验

在草稿校验基础上：

- 每个 cargo item 的三个维度均必须精确分完；
- 每张当前 HBL 至少有一条正数分配；
- FCL 的每行必须关联真实箱，每只当前实际箱必须至少被使用，且三个维度精确分完；
- LCL/散杂不要求、不生成操作票级实际箱，只确认货物到 HBL 的守恒；
- 确认只冻结分配事实，不写 HBL/MBL 显示内容。

所有相等与差值都使用 `decimal.Decimal` 精确计算；件数用整数。重量最多 3 位小数，
体积最多 6 位小数，禁止 epsilon 容差、自动舍入或静默纠正。

## 明确填入提单内容

提供两个显式命令，均只更新目标单证的件数、毛重、体积，不修改品名、唛头、包装
单位、发通收或条款：

- HOUSE：逐张“用分配汇总填入本张 HBL”，携带分配聚合和 HBL 预期版本；汇总
  目标 HBL 的当前分配并更新该 HBL，递增 HBL version，写审计。
- DIRECT：“用操作票货物汇总填入 MBL”，携带 MBL 预期版本；汇总当前货物行并
  更新共享 MBL，递增 MBL version，写审计。

上述命令是用户明确动作，不在保存或确认分配时自动执行。HOUSE 命令允许在草稿或
已确认分配上执行；若聚合版本已经变化则返回 409，防止用过期汇总覆盖显示值。

## API 契约

新增 `SeaCargoAllocationService`：

- `GetSeaCargoAllocation(order_id)`：返回结构、装载类型、聚合状态/版本、货物、
  实际箱、HBL、分配行、三视角服务端汇总和允许动作。
- `SaveSeaCargoAllocationDraft(order_id, expected_allocation_version, allocations)`：
  全量替换草稿；请求行使用十进制字符串。
- `ConfirmSeaCargoAllocation(order_id, expected_allocation_version)`。
- `WithdrawSeaCargoAllocation(order_id, expected_allocation_version)`。
- `ApplySeaHouseBillAllocationSummary(order_id, house_bill_id,
  expected_allocation_version, expected_house_bill_version)`。
- `ApplySeaOrderCargoSummaryToMasterBill(order_id, expected_mbl_version)`。

读取复用订单 `read` 权限；保存、确认、撤回和提单填入复用订单 `update` 权限与组织
数据范围，不新增平行权限真相。

现有货物/实际箱响应增加 `version`；更新和删除请求增加必填 `expected_version`。
实际箱新增 `package_count`，删除旧 `shipping_document_id`。生成 API、OpenAPI 和 Web
客户端，禁止手改生成物。

## 精确错误契约

领域错误至少包括：

- `SEA_CARGO_ALLOCATION_INVALID_ARGUMENT`
- `SEA_CARGO_ALLOCATION_INVALID_REFERENCE`
- `SEA_CARGO_ALLOCATION_EXCEEDED`
- `SEA_CARGO_ALLOCATION_INCOMPLETE`
- `SEA_CARGO_ALLOCATION_CONFLICT`
- `SEA_CARGO_ALLOCATION_STATUS_CONFLICT`

数量错误通过 Kratos error metadata 返回：

- `object_type`、`object_id`、`object_label`
- `dimension`：`package_count|gross_weight_kg|volume_cbm`
- `baseline_value`、`allocated_value`
- `remaining_value` 或 `excess_value`
- 可定位时补 `cargo_item_id`、`house_bill_id`、`container_id`

消息必须由同一 biz 格式化函数生成。例如单张 HBL 明显超过操作票总量时返回：
“HBL001 的件数已分配 100，超过操作票可分配总量 80，超出 20，请调整”。多个 HBL
共同造成货物行超分时，则准确指出货物名称和该货物行的基准/已分配/超出值。页面
用 metadata 定位并高亮相应行，不能只弹“分配不平衡”。

## 页面设计

订单列表和详情为海运 HOUSE 操作票提供“箱货分配”动作，打开宽抽屉；不把复杂
矩阵塞进新建订单长表单。抽屉默认完整展示，不默认折叠，分为：

1. 顶部状态条：操作票号、MBL、HOUSE、DRAFT/CONFIRMED、版本和保存/确认/撤回；
2. 分配编辑表：货物、HBL、实际箱、件数、毛重、体积；FCL 可选实际箱，LCL/散杂
   不显示箱选择；
3. 即时进度区：货物行与实际箱显示基准/已分配/剩余；HBL 显示本张已分配汇总、
   操作票尚未分配总量和与 HBL 当前显示值的差额。

前端以十进制字符串运算：

- 剩余大于 0 使用品牌蓝/进行中；等于 0 使用成功绿；小于 0 使用错误红；
- 本地输入一旦出现非法值或超分，立刻高亮单元格、显示具体文本并禁用保存；
- 未分完仍可保存草稿，但确认按钮不可用并展示逐项未完成清单；
- 服务端返回 metadata 时滚动并聚焦对应货物/HBL/箱行；
- 每张 HBL 的汇总卡显示“用分配汇总填入本张 HBL”，并在旁边展示填入前差额；
- 确认动作不包含“同时更新全部 HBL”的复选框或隐式逻辑。

DIRECT 不显示箱货分配入口，在提单信息 MBL 卡显示“用操作票货物汇总填入 MBL”。

## 事务、锁与并发

所有写命令使用共享事务，统一固定锁序：

```text
Order
→ SeaMasterBill
→ Active SeaMasterBillOrderLink
→ OrderCargoItem（ID 升序）
→ SeaHouseBill（ID 升序）
→ OrderContainer（ID 升序）
→ SeaCargoAllocation（ID 升序）
→ AuditLog
```

- 首次需要修改的行直接 `FOR UPDATE`，不得先 `FOR SHARE` 后升级。
- 锁住 Link 后重验 organization/order/master_bill/status，拒绝旧 MBL 关系。
- 分配保存比对 `cargo_allocation_version`；货物/实际箱写入同时比对各自 version。
- HBL/货物/实际箱集合变化时在同一事务递增分配聚合版本，防止打开的分配页面继续
  提交旧集合。
- 保存、确认、撤回、显式填入和审计同事务完成；错误原样外传并完整回滚。提交后
  使用普通上下文重读响应，不继续使用 `txCtx`。

现有货物、实际箱、HBL 和订单装载类型写入口都要接入同一门禁：CONFIRMED 时阻断；
更新后若会造成已保存草稿超分则阻断，而不是删除或改写分配行。

## 审计

至少记录：

- `order.sea_cargo_allocation.save_draft`
- `order.sea_cargo_allocation.confirm`
- `order.sea_cargo_allocation.withdraw`
- `sea_house_bill.apply_allocation_summary`
- `sea_master_bill.apply_order_cargo_summary`

审计记录组织、操作人、Order/Link/MBL、前后版本、状态、分配行数和受影响 ID；不
记录完整请求矩阵或单证长文本。货物/实际箱既有增删改审计补前后版本。

## 迁移与回滚

- 新建 `sea_cargo_allocations`，扩展 Link、CargoItem、Container，删除实际箱旧单值
  单证外键。
- 当前无历史数据；迁移开始时显式检查相关 SE 货物、实际箱和旧单证关系为空。
  发现非空立即失败，不回填、不双写、不生成虚假箱/HBL。
- 数据库 CHECK 保证枚举、正数和小数列范围；外键与条件唯一索引作为持久层防线。
- 回滚必须同时撤销 Schema、迁移、Proto、生成物、后端、页面和测试；不保留旧
  `shipping_document_id` 兼容读取。

## 明确不做

- 不新增 HBL 分配目标表，不把 HBL 显示值当作守恒基准。
- 不自动分摊剩余量、不自动生成行、不把 FCL 订舱箱计划伪造成实际箱。
- 不在确认时批量改写 HBL/MBL 内容。
- 不实施拆票、改配、单证签发版本或财务成本分摊。
