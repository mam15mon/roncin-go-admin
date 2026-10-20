# 当前实现证据

2026-10-20 已完成代码检查及开发数据库只读查询。实际仅有 CONT、BL、CBM、TON、SET、DOC、DAY 七个启用单位，缺少 KG、RT、HOUR 和具体箱型。

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

## 已收敛技术选择

- 独立工作分支 codex/billing-unit-quantity-assist 从 main 创建；旧工作分支保留，避免混入其他任务提交。
- 新接口 GET /api/v1/orders/{order_id}/fee-quantity-preview，billingUnitId 与 source（PLANNED/ACTUAL）显式传入。
- 箱型采用单位代码与 container_spec 代码精确匹配，无名称模糊匹配或 HC/HQ 自动等价。
- 货量来源为本订单 cargo_items，通过 PostgreSQL numeric 聚合并读为 decimal 字符串；不叠加箱货或 MBL 汇总。
- 自然箱与具体箱型支持计划/实际两个明确来源；关联共享箱时实际箱数不可确定则解释原因，保留手工录入。
- 验证使用独立一次性 PostgreSQL 数据库，安装 pg_trgm；不修改或清空开发数据库。

行业依据：https://terms.maersk.com/LCL 记载 1 CBM 对应 1000 KG 的 W/M 口径，但不是所有费用都采用此规则；本任务只提供显式确认建议，不引入通用运价引擎。
