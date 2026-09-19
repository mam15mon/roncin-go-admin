# 拆分 sea_order_change.go 数据层大文件

## Goal

把 5185 行的 `server/internal/data/sea_order_change.go` 按变更类型拆为多个
同包文件，纯移动、零行为变更，让「按变更类型找实现」可以靠文件名直达。

## 拆分映射（函数块 → 目标文件）

| 目标文件 | 内容（原文件行段） | 预计行数 |
| --- | --- | --- |
| `sea_order_change.go`（保留为枢纽） | package/imports、`seaOrderChangeRepo` struct、`NewSeaOrderChangeRepo`、`GetChangeActions`、接口断言 `var _ biz.SeaOrderChangeRepo`（56–234、5185） | ~250 |
| `sea_order_change_split.go` | `GetSplitContext`、`PreviewSplit`、`GetSplitEventByIdempotencyKey`、`GetSplitEvent`（235–1204、4951–5074） | ~1100 |
| `sea_order_change_split_execute.go` | `ExecuteSplit` 单函数整体移动（1205–3388，内部约 2200 行不再细分，避免行为风险） | ~2190 |
| `sea_order_change_transport_update.go` | `transportExecutionDifferences`、`PreviewTransportExecutionUpdate`、`ExecuteTransportExecutionUpdate`（3389–3620） | ~230 |
| `sea_order_change_reassignment.go` | `PreviewReassignment`、`ExecuteReassignment`、`GetReassignmentEventByIdempotencyKey`、`GetReassignmentEvent`（3621–4419、5075–5166） | ~900 |
| `sea_order_change_events.go` | `ListChangeEvents`、`GetChangeEvent`、`decodeSplitResultSnapshotSummary`（4436–4728） | ~330 |
| `sea_order_change_shared.go` | 跨变更类型共用 helper：`externalConfirmationFromReassignment`、`validateNewMasterBillInput`、`validateTransportExecutionTargetInput`、`createNewMasterBillInTx`、`createNewTransportExecutionInTx`、`seaMasterBillShippingLineConsistent`、`enabledShippingLineExists`、`mblToSummary`、`makeDiff`、`sortAndDeduplicateUUIDs`（4420–4435、4729–4950、5167–5184） | ~450 |

`sortAndDeduplicateUUIDs` 与 `makeDiff` 被 `order_write.go`、
`sea_document_change.go` 与测试文件同包引用，放 shared 文件不影响引用方。

## 约束

- 纯移动：不改任何函数体、签名、导出名；Go 同包拆分无 import 面变化
  （新文件各自携带所需 import 子集，用 goimports 收敛）。
- 不新增公共 API、不改测试文件。

## Acceptance Criteria

- [ ] 拆分后 `go build ./...`、`go vet ./internal/data` 通过。
- [ ] `go test ./internal/data -run 'SeaOrderChange'` 与
      `sea_order_change_test.go` 全绿（集成测试无库时 SKIP 属正常）。
- [ ] 拆分零丢失：按原行序拼接各新文件正文（扣除重复 import 块）与原文件
      逐行一致；`git diff --stat` 显示总行数净增仅为新文件头。
- [ ] 单文件仅 `sea_order_change_split_execute.go`（单函数）超 1500 行，
      其余全部 <1500 行。
