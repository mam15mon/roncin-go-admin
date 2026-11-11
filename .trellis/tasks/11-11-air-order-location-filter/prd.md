# 空运地点筛选与权威投影一致

## 目标与需求

对应父任务 R6：空运主单地点变更后订单列表公开接口以新地点筛选，旧地点不再命中；起运与目的地点同样处理，海运及非空运路径保持原行为。

## 问题证据

server/internal/data/order_query.go:148；server/internal/data/air_export.go:178、406；server/internal/service/order_query.go:186。

## 验收

- [x] A→B 后按 B 命中按 A 不命中，起运与目的分别及组合。
- [x] 同主单多票。
- [x] 总数/分页一致。
- [x] 终止或缺关联按现有读投影语义。
- [x] 海运回归。
- [x] 公司隔离。

## 交付边界

独立交付，可与前端任务并行。
不重置数据库、不自动纠正已有业务金额；保留无关工作。按父任务授权完成独立 review、合并主分支和 push。

## 验收证据

本子任务已完成实现、触发与相邻路径回归及独立 review。具体命令和边界见本目录 `research/`，全栈验收与 Git 交付见父任务 `11-11-system-audit-fixes`。
