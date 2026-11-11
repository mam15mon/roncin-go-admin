# 资金登记幂等重试

## 目标与需求

对应父任务 R4：同一登记弹窗内相同提交意图在结果未知或失败后重试复用幂等键；更改内容产生独立意图；成功后新登记使用新键。保留失败输入与现有公司、汇率、散客和权限校验。

## 问题证据

web/src/pages/finance/cashflows/index.tsx:543；server/internal/biz/finance_cashflow.go:141、184；server/internal/data/ent/schema/finance_cashflow.go:30。

## 验收

- [x] 第一次已成功但返回 undefined/抛超时后第二次键一致。
- [x] 普通失败保留输入。
- [x] 内容变更键不同，改回原载荷键恢复。
- [x] 成功后同内容新登记键不同。
- [x] 组织切换隔离。
- [x] 后端同键回放、不一致拒绝。

## 交付边界

可与 R1/R2/R6 并行，生成物和提交由主协调者串行。
不重置数据库、不自动纠正已有业务金额；保留无关工作。按父任务授权完成独立 review、合并主分支和 push。

## 验收证据

本子任务已完成实现、触发与相邻路径回归及独立 review。具体命令和边界见本目录 `research/`，全栈验收与 Git 交付见父任务 `11-11-system-audit-fixes`。
