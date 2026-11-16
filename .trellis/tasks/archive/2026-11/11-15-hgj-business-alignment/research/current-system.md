# 当前系统订单能力盘点

只读源码调研，未运行测试；文件存在不等于当前全量通过。

| 结论 | 源码证据 |
| --- | --- |
| 仅有海运出口/空运出口注册 | `web/src/pages/orders/order-kinds/types.ts:17`、`registry.ts:9`、`web/config/routes.ts:132` |
| 创建门禁拒绝其他类型及进口 | `server/internal/biz/order_usecase.go:321`、`order.go:72` |
| SI/AI/LAND/RAIL 只在枚举中预留 | `server/api/order/v1/order.proto:94` |
| AE 共享主单、分单、成员/历史已实现 | `server/api/order/v1/air_export.proto:8`、`server/internal/biz/air_export.go:27`、`:127` |
| AE HAWB 目前为自由文本 | `web/src/pages/orders/templates/air-template.tsx:156`、`air_export.proto:69` |
| 现有制作页仅 HBL | `web/config/routes.ts:196`、`web/src/pages/orders/hbl.tsx` |
| AE 普通保存禁止换主单 | `server/internal/biz/air_export.go:332` |
| AE 已有独立重量契约和费用来源 | `.trellis/spec/server/backend/air-export-contract.md`、`server/internal/biz/air_weight.go` |
| SE 已有拆票、改配和共享航程 | `server/api/order/v1/sea_order_change.proto:28`、`:46`、`:64` |
| SE 已有改单/作废/模式切换及结构化单证 | `server/api/order/v1/sea_document.proto:44`、`:62`、`:80`、`:184` |

## 历史决定（新批准覆盖项见下节）

来源：`.trellis/tasks/archive/2026-10/10-20-air-export-business-loop/prd.md:69` 起。

- MAWB/HAWB 建票必填，一操作票一个当前主单、一张当前分单；无 HAWB 直单此前不提供。
- HAWB 同主单唯一、跨主单可重复；共享资料显式关联，不按号码隐式拼单。
- 体积除数 6000、分泡 1、最终 HALF_UP 至 0.001 kg，不默认最低重或半公斤进位。
- 签单、实际起降与财务结案独立；缺资料硬拦，单纯超时记原因。
- 双航司未确认；复杂改配/拆拼、成本分摊、跨公司拼单、多航段、库存/安检/自动配载此前延期。本次需要重新提出具体建议，不能自动当作已获批。

## 可复用验证入口

后端：`air_export_test.go`、`air_weight_test.go`、`air_export_integration_test.go`、`air_weight_integration_test.go`、`air_order_location_integration_test.go`、`air_workbench_integration_test.go`。
前端：`detail-air-house-save*.test.tsx`、`detail-change-notice-air.test.tsx`、`air-master.permissions.test.tsx`、`templates/air-create-layout.test.tsx`、`AirWeightPanel.test.tsx`。

## 本次需求已覆盖的历史决定

用户已明确 AE/AI 同时提供 MAWB 直单与主分单，取代此前“无 HAWB 直单不提供”的范围限制。现有代码尚未更改；这是新需求，不应删除原有历史记录。主、分空运单结构化及制作/预览/PDF打印也已批准。复杂拆拼、改配及多航段仍等待明确范围选择。

> 状态同步：后续用户已确认直单一主一票不混HOUSE、高级空运与批量导入后续、完整筛选CSV本期；最新具体方案以父prd.md/design.md最终复核稿为准，本研究中的候选/历史未决项不再单独决定实施范围。
