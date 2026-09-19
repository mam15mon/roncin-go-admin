# 拆分 finance_commission.go 数据层大文件

## Goal

把 2768 行的 `server/internal/data/finance_commission.go` 按提成子域拆为
同包文件，纯移动、零行为变更。

## 实际拆分映射（9 文件）

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `finance_commission.go`（枢纽） | 43 | struct、构造器、接口断言 |
| `finance_commission_selectors.go` | 210 | 员工/候选/对冲候选选择器查询 |
| `finance_commission_rules.go` | 650 | 规则 CRUD、员工分配/复制、区间占用校验、规则映射 |
| `finance_commission_ledger.go` | 183 | 台账 List/Count/Export/GetByKey/Get |
| `finance_commission_mapping.go` | 197 | commission/line → biz 映射与快照枚举展开 |
| `finance_commission_calculation.go` | 573 | 计提引擎：来源装载、规则解析、金额计算 |
| `finance_commission_write.go` | 319 | Create/Transition、审批整批确认、锁序 helper |
| `finance_commission_adjustment.go` | 365 | 提成调整查询/创建/状态流转 |
| `finance_commission_order_summary.go` | 532 | 订单提成汇总聚合（含 bucket/aggregate 类型） |

## Acceptance Criteria（已验证）

- [x] build / vet / gofmt 通过。
- [x] 非 import 正文 2642 行与原文件多重集一致（零丢失零改动）。
- [x] `go test ./internal/data -run 'Commission' -count=1`（含真实库集成）
      全绿，61.2s。
