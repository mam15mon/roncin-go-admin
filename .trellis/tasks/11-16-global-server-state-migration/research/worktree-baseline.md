# 规划阶段工作区基线

记录时间：2026-11-16T18:07:46+08:00。

以下是其他进行中工作的快照，包含已暂存删除和未跟踪新类型文件。文件正在变化，实施及提交前必须重新核对，不能把此快照当作可覆盖来源。

```text
 M AGENTS.md
 M server/api/order/v1/order_fee.pb.go
 M server/api/order/v1/order_fee.proto
 M server/internal/biz/order_fee_save.go
 M server/internal/data/order_fee_save_integration_test.go
 M server/internal/service/order_fee.go
 M server/openapi.yaml
 M web/config/openapi.generated.json
D  web/src/pages/orders/components/fees/FeeFormModal.tsx
 M web/src/pages/orders/components/fees/FeeQuantityForms.test.tsx
 M web/src/pages/orders/components/fees/FeeSupplementModal.tsx
 M web/src/pages/orders/components/fees/OrderFeeHeader.test.tsx
 M web/src/pages/orders/components/fees/OrderFeeHeader.tsx
 M web/src/pages/orders/components/fees/OrderFeeTableTabs.preview.test.tsx
 M web/src/pages/orders/components/fees/OrderFeeTableTabs.rateOverride.test.tsx
 M web/src/pages/orders/components/fees/OrderFeeTableTabs.test.tsx
 M web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx
 M web/src/pages/orders/components/fees/feeSave.test.ts
 M web/src/pages/orders/components/fees/feeSave.ts
 M web/src/pages/orders/components/fees/feeSummary.ts
 M web/src/pages/orders/components/fees/orderFeeOptionalColumns.tsx
 M web/src/pages/orders/fees.test.tsx
 M web/src/pages/orders/fees.tsx
 M web/src/pages/orders/use-fee-exchange-preview.test.tsx
 M web/src/pages/orders/use-fee-exchange-preview.ts
 M web/src/services/roncin/typings.d.ts
?? .trellis/tasks/11-16-global-server-state-migration/
?? web/src/pages/orders/components/fees/feeFormValues.ts
```

## 已核实重叠

- OrderFeeTableTabs 增加手工汇率改回系统汇率和入口调整，迁移必须保留。
- fees.tsx 删除旧费用弹窗并调整快捷新增往来单位；不恢复 FeeFormModal。
- FeeFormValues 已迁独立 feeFormValues.ts，补录及汇率 Hook 导入正在同步。
- 服务端 Proto、实现和生成物属于其他会话；本任务不暂存、不提交这些文件。
- 现有 index 中的删除不可跟随普通 git commit 意外带入。主会话负责按路径或差异提交。
