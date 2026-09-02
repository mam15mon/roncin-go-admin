# 海运出口箱货分配

## Goal

实现操作票货物、集装箱与多张 HBL 的定量分配和总量守恒

## Requirements

- 依赖 `09-02-sea-export-document-content` 已完成、检查并提交。
- 建立货物行/集装箱到 HBL 的多对多定量分配，支持一箱多 HBL 和一张 HBL 多箱。
- 分配保存件数、重量、体积并验证订单、HBL、集装箱三个维度的总量守恒。
- DIRECT 不创建 HBL 分配；主单内容从操作票货物和运输事实明确生成，不制造虚拟分单。
- 本阶段不实施拆票、改配或财务费用重分摊。

## Acceptance Criteria

- [ ] 一箱可分配给至少两张 HBL，一张 HBL 可覆盖至少两个箱。
- [ ] 一个货物行可定量分配到多个 HBL/箱，超分、负数和维度不守恒被拒绝。
- [ ] 并发修改使用版本与固定锁顺序，失败完整回滚。
- [ ] DIRECT、HOUSE 和订单货物既有链路回归通过。

## Notes

- 后续依赖：`09-02-sea-export-split-reassignment`。
