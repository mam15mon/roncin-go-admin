# 海运出口运输与主分单总体实施计划

## 阶段门禁

- [x] 用户确认按六阶段实施。
- [x] 父任务 PRD 与总体设计已收敛。
- [x] 六个子任务已创建且写明顺序依赖。
- [ ] 只启动下一交付物所属的子任务，不启动父任务。
- [ ] 每个子任务启动前补齐并复核自己的 `design.md`、`implement.md`、`implement.jsonl` 和 `check.jsonl`。
- [ ] 每阶段实施后由独立 `trellis-check` 检查并单独提交。

## 执行顺序

1. `09-02-sea-export-mbl-foundation`
2. `09-02-sea-export-document-content`
3. `09-02-sea-export-cargo-allocation`
4. `09-02-sea-export-split-reassignment`
5. `09-02-sea-export-document-versioning`
6. `09-02-sea-export-finance-allocation`

后一阶段不得在前一阶段检查和提交完成前启动。父子目录不表达依赖，依赖以各子任务 PRD 和实施计划为准。

## 每阶段统一流程

1. 核对依赖提交、工作区和数据库是否仍无历史数据。
2. 完成该阶段设计复核并激活子任务。
3. 由实施代理只实现当前阶段、运行生成和第一轮测试。
4. 主会话检查真实 Git 差异、生成来源、事务边界和跨层数据流。
5. 独立 `trellis-check` 复核；发现 P0/P1 必须修正并重查。
6. 重跑与风险匹配的 Go、前端、生成幂等和 PostgreSQL 集成测试。
7. 主会话提交、完成并归档子任务，再进入下一阶段。

## 最终整合验收

- [ ] DIRECT：必填 MBL、无 HBL、无虚拟分单。
- [ ] HOUSE：一票多 HBL、多票共享 MBL、每张分单独立内容。
- [ ] 一箱多 HBL、货物分配守恒。
- [ ] 部分拆票、整票改配、Switch B/L 三条流程互不混用。
- [ ] 已确认版本和外部/财务引用可重现。
- [ ] 共享费用分摊守恒，账单、发票、核销和提成链路回归通过。
- [ ] `pnpm run check`、`pnpm run build` 以及专用 PostgreSQL 测试明确 PASS，不把 SKIP 记为通过。

## 回滚点

- 每阶段一个或一组边界清晰的提交，按 6 → 1 逆序回滚。
- Schema、迁移、契约和生成物作为同一阶段原子边界。
- 若迁移前发现历史数据，与“无历史数据”前提冲突，停止执行并返回规划，不增加隐藏兼容分支。
