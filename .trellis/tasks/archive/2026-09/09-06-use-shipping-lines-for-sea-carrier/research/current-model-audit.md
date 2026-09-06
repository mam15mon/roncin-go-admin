# 当前船公司模型与数据审计

## 代码事实

- SE 新建和详情页的 `searchCarriers` 当前调用 `searchPartnersByRole(PARTNER_ROLES.CARRIER)`。
- Partner 列表按组织、启用状态和 `partner_roles.role_type = carrier` 过滤。
- ShippingLine 已有组织级列表接口，支持 `page/page_size/keyword/enabled`；查询覆盖 SCAC、中文名、
  英文名和 `search_keywords`，后者由统一 Hook 维护中文全拼与首字母。
- `orders.carrier_id` 与 `sea_transport_executions.carrier_id` 当前没有 ShippingLine/Partner 外键；
  `sea_master_bills.issuer_partner_id` 和版本 issuer 则带 Partner 外键。
- 订单创建/更新、共享 MBL 候选、拆票和改配已强制 Order carrier、TE carrier、MBL issuer 三方一致，
  因而只改下拉会把 ShippingLine UUID 写入 Partner FK 并失败。
- 费用新增应付当前使用 `bookingAgentId || carrierId` 默认结算单位；若直接改下拉而不改费用，会把
  ShippingLine UUID 误当 Partner ID。
- Partner carrier 角色除 SE 订单链路、Partner 角色 UI/转换和相关测试外没有独立业务消费者。

## 只读数据库审计

- 日期：2026-09-06。
- 数据源：当前本地开发 PostgreSQL，通过 `.env.local` 注入；未记录连接串或凭据。
- 执行方式：显式只读事务，查询结束回滚。

| 检查项 | 数量 |
| --- | ---: |
| ShippingLine 总数 | 358 |
| 启用 ShippingLine | 358 |
| Partner carrier role | 0 |
| SE Order | 0 |
| SeaMasterBill | 0 |
| SeaMasterBillVersion | 0 |

## 结论

下拉为空的直接原因是读错聚合；当前数据允许直接切换到 ShippingLine 模型。由于不存在可迁移的 SE 或
carrier role 数据，本任务不应引入名称匹配、SCAC 猜测、双写或自动角色转换。
