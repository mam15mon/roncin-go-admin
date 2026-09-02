# 海运出口主分单内容设计

## 设计目标与边界

阶段 2 在阶段 1 的共享 MBL 与当前成员关系上，补齐操作票单证结构、共享 MBL
内容、真实 HBL 身份和每张 HBL 的独立内容，并把这些字段呈现在默认展开的订单
“提单信息”区块。

本阶段只维护可编辑的当前内容，不建立不可变单证版本，不实施箱货定量分配、
拆票/改配、外部 SI/VGM/舱单状态、Switch B/L 或费用分摊。阶段 3 负责分配和守恒，
阶段 5 负责确认后版本和改单。

## 领域关系与所有权

```text
Organization(company/headquarters) ──┐
                                     ├── SeaHouseBill.issuer（互斥真实主体）
Partner(customer/other) ─────────────┘

SeaMasterBill 1 ── N SeaMasterBillOrderLink N ── 1 Order
      │                         │
      │ shared content          └── document_structure
      │                              UNDETERMINED | DIRECT | HOUSE
      │
      └── 0..N SeaHouseBill N ── 1 Order
                 └── independent content
```

- 单证结构属于“当前操作票在当前 MBL 下的单证关系”，保存到活动
  `SeaMasterBillOrderLink`，不属于整张共享 MBL。同一 MBL 下可以同时存在采用
  DIRECT 和 HOUSE 的不同操作票。
- MBL 内容保存到唯一的 `SeaMasterBill`，所有成员操作票读取同一记录；禁止在
  `Order` 上保存副本。
- 海运 HBL 使用新的 `SeaHouseBill` 实体，不继续扩张同时服务空运的
  `OrderShippingDocument`。旧实体保留给非海运消费者；SE 写入必须拒绝继续使用
  `shipping_documents`，避免双真相。
- 每张 `SeaHouseBill` 同时关联一个 Order 和该 Order 当前的 MBL。新增和更新时
  后端在事务内重验活动成员，不能相信请求中的 MBL ID。

## 单证结构状态

### 状态语义

| 状态 | HBL 基数 | 页面含义 |
| --- | ---: | --- |
| `UNDETERMINED` | 0 | MBL 已录入，单证结构尚未确认或 HBL 尚未维护 |
| `DIRECT` | 0 | 用户已明确确认直单，不存在 HBL |
| `HOUSE` | 1..N | 存在一张或多张真实 HBL |

### 允许转换

```text
UNDETERMINED --明确标记直单--> DIRECT
UNDETERMINED --添加首张 HBL--> HOUSE
DIRECT --取消直单标记--> UNDETERMINED
HOUSE --明确删除最后一张 HBL--> UNDETERMINED
```

- 不提供 DIRECT → HOUSE 一步转换；DIRECT 下添加 HBL 返回领域错误。
- “取消直单标记”是独立命令，携带成员关系 `expected_version`，写业务审计。
- 删除 HOUSE 最后一张 HBL 时，请求必须显式声明返回 `UNDETERMINED`；没有明确
  意图则拒绝，后端不自动判为 DIRECT。
- 新建订单默认 `UNDETERMINED`。用户在未保存页面主动选择 DIRECT 后，可以随
  订单创建；添加首张 HBL 后页面明确显示 HOUSE。
- 任意状态下都不创建虚拟、隐藏、空号 HBL。

## 数据模型

### `SeaMasterBillOrderLink` 扩展

- `document_structure`：非空枚举 `UNDETERMINED|DIRECT|HOUSE`，默认
  `UNDETERMINED`。
- 复用现有 `version`。任何结构转换以及 HBL 集合增删都递增版本，使页面对整组
  HBL 的并发修改具备聚合级冲突检测。

数据库不能单靠当前 link 表表达 HBL 数量约束；`DIRECT/HOUSE` 与 HBL 基数在同一
事务内由 data 强校验，真实 HBL 的外键和唯一索引作为持久层防线。

### 通用 `SeaBillContent`

Proto 与 biz 定义可复用的内容值对象，MBL 与每张 HBL 分别保存以下字段：

- `shipper_text`、`consignee_text`、`notify_party_text`、`second_notify_party_text`
- `marks_text`、`goods_description_text`
- `package_count`、`package_unit`
- `gross_weight_kg`、`volume_cbm`
- `freight_terms`、`transport_terms`
- `bill_form`、`release_type`、`clauses`

所有内容字段在阶段 2 均为选填。文本只做首尾空白和长度校验，不从订单字段实时
镜像；件数、毛重、体积为非负可空值。阶段 2 不校验其与订单货物总量相等。

企业资源中的发货人、收货人、通知人、唛头和英文品名只作为显式预填来源；用户
选中后把当时文本复制进表单。保存的是提单独立文本，不保存实时联动关系，后续
资源修改不得改写已录提单内容。

### `SeaMasterBill` 扩展

- 直接增加一组当前可编辑 `SeaBillContent` 字段。
- 复用已有 `version`；任何身份或内容更新都必须锁定 MBL、比对预期版本，并将
  `version + 1`。
- 关联已有共享 MBL 时只读取已有内容；确认关联命令禁止夹带内容更新。
- 阶段 5 引入不可变版本时，以当前内容生成首个版本并改为当前版本指针；阶段 2
  不提前建立无消费者的版本表。

### `SeaHouseBill`

身份与关系字段：

- `organization_id`
- `order_id`
- `master_bill_id`
- `house_no`：原始展示值，按请求无损保存
- `normalized_house_no`：唯一性和检索键
- `issuer_source`：`SELF_ORGANIZATION|CUSTOMER_PARTNER|OTHER_PARTNER`
- `issuer_organization_id`：仅本公司来源非空
- `issuer_partner_id`：仅委托单位/其他主体来源非空
- `status`：沿用当前草稿/已确认/已放货状态，不在本阶段扩展状态机
- `version`：从 1 开始
- `note`
- 一组独立 `SeaBillContent`

主体检查约束：

```text
SELF_ORGANIZATION => issuer_organization_id IS NOT NULL
                     AND issuer_partner_id IS NULL
CUSTOMER_PARTNER | OTHER_PARTNER => issuer_organization_id IS NULL
                                    AND issuer_partner_id IS NOT NULL
```

“本公司”由服务端从订单所属组织向上查找最近的 `company/headquarters`，保存该
组织 ID；找不到时拒绝，不回退到部门或任意 Partner。“委托单位”必须等于订单
当前 customer_id；“其他主体”必须是当前租户内启用的明确 Partner。

如果已有 `CUSTOMER_PARTNER` HBL，订单更新把 customer_id 改为其他主体时必须
拒绝，并提示先调整相关 HBL；禁止 HBL 签发主体随客户字段静默变化。

## HBL 号码与唯一性

### 原始值

- `house_no` 不套用 MBL 的 ASCII 正则。
- 不改变大小写，不删除首尾或内部字符，不替换空格、连字符、斜线、年份、前导零。
- 只限制不能为空（规范化后非空）和最多 128 个 Unicode 字符。

### 规范化键

所有写入入口只调用同一个 biz 纯函数：

1. Unicode NFC 规范化；
2. 去除首尾 Unicode 空白；
3. 统一英文字母大小写用于比较；
4. 保留内部空白、全部标点、年份和前导零。

规范化结果只写 `normalized_house_no`，不得覆盖 `house_no`。

### 唯一索引

由于签发主体可能是 Organization 或 Partner，使用两个带条件的数据库唯一索引：

- 本公司：`organization_id + issuer_organization_id + normalized_house_no`，条件为
  `issuer_source='SELF_ORGANIZATION'`。
- 外部主体：`organization_id + issuer_partner_id + normalized_house_no`，条件为
  `issuer_source IN ('CUSTOMER_PARTNER','OTHER_PARTNER')`。

同一真实 Partner 无论通过“委托单位”还是“其他主体”选择，都属于同一唯一范围。
不同真实主体允许使用相同原号；不得增加忽略签发主体的订单级唯一索引。

## API 契约

### 订单契约

- 新增 `SeaDocumentStructure`、`SeaBillContent`、`SeaHouseBill`、
  `SeaHouseBillInput` 和 `SeaOrderDocumentInput`。
- `Order` 响应返回当前 `sea_document_structure`、活动成员版本和 HBL 摘要；列表只
  返回结构和数量摘要，不内嵌大段提单内容。
- `CreateOrderRequest` 可携带 `sea_document`：结构、MBL 初始内容和零到多张 HBL；
  创建 Order、MBL、成员与 HBL 在现有统一事务中完成。
- `UpdateOrderRequest` 的 `sea_document` 为显式变更意图；没有该字段时不得触碰
  MBL 内容、结构或 HBL。提交时携带成员、MBL 和 HBL 的预期版本。
- SE 请求继续携带旧 `shipping_documents` 时明确拒绝；非 SE 仍沿用旧契约。

### 单证读取与命令

新增海运单证聚合接口：

- `GetSeaOrderDocuments(order_id)`：返回结构、成员版本、共享 MBL 内容/版本、HBL
  完整内容及可用动作。
- `MarkSeaOrderDirect(order_id, expected_link_version)`。
- `CancelSeaOrderDirect(order_id, expected_link_version)`。
- HBL 的新增、更新、删除接口；更新携带 `expected_version` 和成员
  `expected_link_version`，删除最后一张时额外携带明确的
  `return_to_undetermined=true`。
- `UpdateSeaMasterBillContent(order_id, expected_mbl_version, content)`：通过当前
  成员定位共享 MBL，不接受前端任意 MBL ID。

接口复用订单 read/update 权限与组织数据范围，不新增第二套权限真相。独立命令与
订单整单保存必须调用同一 biz 规则和 data 写入原语。

## 事务、锁与并发

- 所有跨 Order、成员、MBL、HBL 的写入通过 `biz.Transactor.WithinTransaction`。
- 仓储客户端只经 `Data.client(ctx)`；不在事务中调用原生 SQL 事务仓储方法。
- 固定锁顺序：Order → SeaMasterBill → Active Link → SeaHouseBill（多行按 ID
  排序）→ 相关 Organization/Partner 校验读取。首次需要写的行直接 `FOR UPDATE`，
  不先 `FOR SHARE` 再升级。
- MBL 内容使用 MBL version；结构和 HBL 集合使用 link version；单张 HBL 更新同时
  比对 HBL version。任一不匹配返回 409“已被更新，请刷新后重试”。
- HBL 创建、更新、删除和结构转换与审计在同一事务提交；错误原样外传并完整回滚。
- 事务提交后用普通上下文重读完整响应，不在提交后继续使用 `txCtx`。

## 审计

至少记录：

- `order.sea_document.mark_direct`
- `order.sea_document.cancel_direct`
- `sea_master_bill.content.update`
- `sea_house_bill.add|update|remove`

审计保存组织、操作人、Order/MBL/HBL ID、变更字段名、前后版本和结构转换；不记录
完整发通收、货描等长文本内容。创建订单时把初始结构和 HBL 数量记入订单审计。

## 页面设计

### 区块位置

- “配舱信息”继续承载必填 MBL 号、主单签发方、候选关联和航程。
- 海运模板在配舱信息后、货物信息前新增独立“提单信息”区块。
- 区块默认展开；可以保持不可折叠，或设置 `defaultCollapsed=false`，绝不因没有
  制单服务、没有 HBL 或内容为空而默认隐藏。

### 结构与标签

- 顶部显示 `未确定 / DIRECT / HOUSE` 状态和明确动作。
- 未确定时展示“标记为直单”和“添加 HBL”；DIRECT 时只展示“取消直单标记”，
  不展示可直接添加 HBL 的入口；HOUSE 时展示“添加 HBL”。
- 内容区采用紧凑 Tabs：主单 MBL 始终存在；HOUSE 增加 `HBL-001` 等真实分单
  标签。共享主单显示“共 N 票”提示。
- 每张 HBL 标签内独立显示号码、签发主体和完整内容；签发主体初始无默认值。
- 新增第 2 张及后续 HBL 时提供“复制上一张内容”，只复制内容字段，不复制 HBL
  号码、ID、版本和签发主体；复制结果进入本张本地表单，之后不联动。
- 删除最后一张 HBL 弹出明确确认，说明结构将回到“未确定”；确认请求携带明确
  标记。任何删除都不自动变成 DIRECT。

### 签发主体控件

- `本公司 / 委托单位 / 其他主体` 三选一，初始均未选择。
- 本公司显示解析到的公司/总部名称，不要求用户维护自有 Partner。
- 委托单位显示当前客户名称；更换客户导致已有 HBL 不一致时展示后端阻断提示。
- 其他主体选择后出现服务端 Partner 关键字检索，不一次性加载全量。

## 迁移与回滚

- 创建 `sea_house_bills`，扩展 `sea_master_bill_order_links` 与
  `sea_master_bills`；不把海运新字段强塞进空运共用旧表。
- 迁移开始时检查是否存在 SE 订单或海运 HBL 历史数据；发现非空立即停止并报告，
  不回填、不双写、不生成占位主体或 HBL。
- 数据库 CHECK 约束保证结构枚举、签发主体互斥和数值非负；条件唯一索引保证
  HBL 身份。
- 回滚必须同时撤销新表、字段、约束、Proto/生成物、后端和页面；只允许在开发库
  尚未承载阶段 2 业务数据时执行数据库回滚。

## 明确不做

- 不校验提单件重尺与订单/货物一致；阶段 3 实施分配和守恒。
- 不创建确认内容快照、改单或 Switch 替代链；阶段 5 实施。
- 不把 HBL 内容复制到 MBL，也不从订单字段持续镜像。
- 不创建本公司 Partner、签发品牌配置或 seed。
- 不增加隐式状态推断、静默 HBL 删除或兼容双写。
