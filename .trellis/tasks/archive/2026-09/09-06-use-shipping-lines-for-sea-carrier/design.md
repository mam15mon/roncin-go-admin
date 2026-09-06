# 技术设计：海运出口船公司改用航运公司主数据

## 1. 设计目标与边界

本次修正的是领域身份类型，而不只是下拉数据源。目标是让 ShippingLine 成为 SE 船公司的唯一真相源，
Partner 只承载真实往来与结算主体，并把这一边界贯穿数据库、领域模型、API、共享 MBL、拆票、改配、
版本快照、列表筛选和费用默认值。

HBL issuer 仍然是真实 Organization/Partner 签发主体，任何名为 `issuer_partner_id` 的 HBL 字段不属于
本次删除范围。

## 2. 目标数据模型

| 聚合/表 | 旧字段 | 新字段 | 约束 |
| --- | --- | --- | --- |
| `orders` | `carrier_id` | `shipping_line_id` | 可空；SE 业务层必填；FK ShippingLine `NO ACTION` |
| `sea_transport_executions` | `carrier_id` | `shipping_line_id` | 非空；FK ShippingLine `NO ACTION` |
| `sea_master_bills` | `issuer_partner_id` | `shipping_line_id` | 非空；FK ShippingLine `NO ACTION` |
| `sea_master_bill_versions` | `issuer_partner_id` + `carrier_id` | `shipping_line_id` | 非空不可变；FK ShippingLine `NO ACTION` |

共享 MBL 唯一索引变为：

```text
(organization_id, shipping_line_id, normalized_master_no)
```

Ent 为 Order、SeaTransportExecution、SeaMasterBill、SeaMasterBillVersion 增加指向 ShippingLine 的显式
edge。写入时仍必须查询 `organization_id`，不能只靠单列 FK 判断租户归属。

PartnerRole Ent 枚举只保留 `customer | supplier | foreign_agent`。新增数据库 CHECK 约束防止绕过应用写入
carrier 或其他未知值。

## 3. 契约变更

### 3.1 Order 与共享 MBL

- Order/Create/Update/List filter 中的 `carrier_id` 改为 `shipping_line_id`。
- `SeaTransportExecution` 返回 `shipping_line_id`、`shipping_line_name`。
- `SeaMasterBillSummary`、`SeaMasterBillCandidate`、`SeaMasterBillDetail` 返回 ShippingLine 身份和名称，
  删除 MBL issuer Partner 字段。
- `SeaMasterBillInput` 只保留主单号、候选确认信息和更正原因；船公司由订单 `shipping_line_id` 提供。
- `MatchSeaMasterBillCandidateRequest` 只接收 `shipping_line_id` 加 MBL 号和航程信息。
- 拆票/改配摘要与目标输入统一改用 `shipping_line_id` / `shipping_line_name`，删除 issuer/carrier 双字段。
- `SeaDocumentVersion` 新增/改用可空 `shipping_line_id` 表示 MBL 版本；原可空 `issuer_partner_id` 只用于
  HBL Partner issuer。

字段改名会改变 JSON 契约。服务端 Proto、生成 Go 代码、OpenAPI 和 Web Client 必须在同一组变更中生成；
不保留旧字段别名或双写逻辑。

### 3.2 PartnerRole

Partner Proto 将枚举值 4 和名称 `PARTNER_ROLE_TYPE_CARRIER` 设为 reserved，服务转换与 Biz 类型删除
carrier 分支，前端重新生成枚举后删除角色选项和结算规则标签。

## 4. 跨层数据流

### 4.1 选择器

```text
SE 表单输入关键字
→ ListShippingLines(page=1, page_size=50, enabled=true, keyword)
→ Data 按组织 + enabled + SCAC/中英文/search_keywords 查询
→ 返回“中文名 / 英文名（SCAC）”选项
→ 表单保存 shippingLineId
```

复用现有 ShippingLine 列表接口和拼音检索键，不另建全量缓存或前端循环分页。订单新建、详情、拆票、改配
和列表筛选共用一个 `searchShippingLineOptions` 转换函数；组织切换仍沿用当前请求代次隔离，避免旧组织响应
污染当前页面。

详情加载时使用 MBL/订单响应携带的 `shipping_line_name` 注入当前选项，保证当前 ShippingLine 后续停用时
仍显示可读名称；搜索结果仍只返回启用项。

### 4.2 创建与普通更新

```text
Create/Update.shipping_line_id
→ Service UUID 转换
→ Biz 要求 SE 非空
→ Data 在现有事务中校验同组织 ShippingLine
→ 新选或变更时要求 enabled=true
→ 写 Order、TE、MBL 同一 ID
→ 共享候选按 ShippingLine + normalized MBL no 匹配
```

数据写入继续使用现有 `Order -> MBL -> Link -> TransportExecution` 锁序、候选版本和订单乐观版本。
多成员 MBL 禁止从单票修改 ShippingLine。内容独立更新和无关字段更新不得改变当前 ShippingLine。

### 4.3 拆票与整票改配

- CURRENT 目标沿用当前 MBL 的 ShippingLine。
- NEW 目标必须提交一个 `shipping_line_id`，创建目标 Order、TE、MBL 时三方写同一个值。
- CANDIDATE 目标携带 ShippingLine、候选 MBL/TE ID 与版本；锁内重新验证候选 MBL、TE 和所选
  ShippingLine 一致。
- 摘要名称从 ShippingLine 查询，不再读取 Partner legal_name。

## 5. 读模型与费用边界

- 订单列表船公司筛选改用 `shipping_line_id`；资源名称解析从 ShippingLine 获取。
- MBL 摘要、拆票、改配、锁定版本等所有船公司名称由 ShippingLine 解析。
- 费用页订单概要显示 `shipping_line_name`，不得把 ID 直接渲染给用户。
- 应付费用默认结算单位只取 `booking_agent_id`；不存在时留空。ShippingLine 不参与
  `settlement_party_id`、账单、账户、合同或结算规则查询。

## 6. 停用与引用规则

- 新建订单或更换 ShippingLine：要求同组织且启用。
- 已有引用的 ShippingLine 被停用：FK 和历史 ID 保留，详情/版本仍能读取名称。
- 更新无关字段且 ShippingLine 未变化：允许保留停用引用，不进行静默替换。
- 尝试把订单改成停用 ShippingLine：明确拒绝。
- ShippingLine 当前没有删除接口；FK 使用 `NO ACTION`，为未来删除能力提供数据库保护。

## 7. 数据迁移

新增最新版本迁移，保持单事务执行，顺序如下：

1. 在任何 DDL 前检查 SE Order、SeaTransportExecution、SeaMasterBill、SeaMasterBillVersion 是否存在数据；
   任一非空立即抛错。
2. 检查 `partner_roles` 是否包含 carrier 或其他非目标角色；存在时立即抛错。
3. 删除旧 MBL Partner 外键和相关旧索引。
4. 重命名 Order/TE/MBL 列，MBL version 收敛为单个 `shipping_line_id`，删除重复 snapshot carrier 列。
5. 建立 ShippingLine 外键、新索引和 MBL 新唯一索引。
6. 建立 Partner role CHECK 约束。

当前只读审计为 358 条 ShippingLine、0 条 carrier role、0 张 SE Order/MBL/version，因此迁移无需映射。
迁移测试必须覆盖冷启动、当前 Schema 升级、前置数据阻断和失败后无部分 DDL。

## 8. 用户未提交导入改动的隔离

`package.json`、`server/internal/data/industry_reference_sync.go` 和 `server/cmd/sync-shipping-lines/` 是用户已有
未提交工作。本任务可以消费既有 ShippingLine 表和列表接口，但不得重写、回退或加入本任务提交。若生成或
格式化命令触及这些文件，提交前必须逐文件排除并核对差异归属。

## 9. 验证策略

- Biz：SE ShippingLine 必填；MBL 输入不再含 Partner issuer；拆票/改配目标校验。
- Data/PostgreSQL：组织/启用校验、三方一致、共享候选、唯一索引、锁序、多成员保护、版本快照。
- Service/HTTP：新字段 UUID 转换、HBL issuer 保留、Proto reserved 枚举行为。
- Frontend：四个船公司入口、搜索标签、候选 payload、详情回填、列表筛选、费用默认值。
- Migration：空库完整链、已有 Schema 增量、旧数据阻断原子性、358 条 ShippingLine 保留。
- Generation：API、Ent/Wire、OpenAPI/Web Client、枚举生成重复运行后无非预期差异。

## 10. 风险与回滚

- 最大风险是误删 HBL `issuer_partner_id`。所有搜索、替换和检查必须按 MBL/HBL 类型逐入口核对。
- 字段改名跨越大量生成物和读写入口，编译通过不足以证明正确；检查必须跟踪创建、候选、拆票、改配、
  锁定版本、列表和费用的完整数据流。
- 不通过兼容分支回滚。代码发布前若迁移前置检查失败，事务会保持旧 Schema；停止发布并另开数据治理任务。
- 当前开发库可通过备份或空库重建恢复；不得修改已经执行的历史迁移。
