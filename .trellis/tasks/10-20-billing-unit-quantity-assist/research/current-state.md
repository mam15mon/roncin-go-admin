# 当前实现证据

2026-10-20 只读检查，未查询运行数据库。

| 事实 | 证据 |
| --- | --- |
| 全局单位与整数规则 | server/internal/data/ent/schema/billing_unit.go:16 |
| 早期箱型、RT、KG 种子依赖 HQ | server/migrations/20260824080000_billing_unit_container_flag.sql:13 |
| 后续种子为 CONT/BL/CBM/TON/SET/DOC/DAY | server/migrations/20260923120000_fee_catalog_seed.sql:8 |
| 开发数据只补六个基础单位 | server/cmd/sync-dev/main.go:396 |
| 已有规范要求保留管理员规则 | .trellis/spec/server/backend/billing-unit-quantity-rule.md |
| 新增费用默认数量 1 | web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx:668 |
| 费项选择只带默认单位 | web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx:994 |
| 弹窗数量手填 | web/src/pages/orders/components/fees/FeeFormModal.tsx:261 |
| 后端数量乘单价 | server/internal/biz/order_fee.go:836 |
| 计划箱量独立于实际箱号 | server/internal/biz/order_types.go:333 |
| 计划箱量已有接口字段 | server/api/order/v1/order.proto:360 |
| 实际箱引用 container_spec_id | server/internal/data/ent/schema/order_container.go:12 |
| 货物有毛重/体积，计费重后置 | server/internal/data/ent/schema/order_cargo_item.go:12 |
| 委托/实际测量投影含 float64 | server/internal/biz/order_types.go:379 |

既有任务 .trellis/tasks/archive/2026-09/09-23-fee-catalog-seed/prd.md 已记载新库种子完整性问题，本次只增量补齐，不重建费用科目目录。

行业依据：https://terms.maersk.com/LCL 记载 1 CBM 对应 1000 KG 的 W/M 口径，但不是所有费用都采用此规则；本任务只提供显式确认建议，不引入通用运价引擎。
