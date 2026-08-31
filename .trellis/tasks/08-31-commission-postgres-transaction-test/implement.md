# 实施计划：提成完整 PostgreSQL 事务集成测试

## 1. 构建真实夹具

- 在 `server/internal/data` 新增或扩展提成事务集成测试文件。
- 复用现有账单、核销集成测试辅助结构；只补提成计算所需实体和逆序清理。
- 先确认成功路径能通过真实 `CommissionUsecase.Create`，再增加失败场景。

## 2. 增加四类场景

- 成功：断言提交后重读、主单、明细、CNY 快照和审计。
- 汇率失败：真实时间标准缺口，断言无事务内产物。
- 保存失败：测试装饰器修改持久化输入后委托真实仓储，断言错误及无半成品。
- 审计失败：非法审计枚举委托真实仓储，断言主单和明细被回滚。

禁止为了测试修改生产接口或增加测试开关。实现若发现生产事务契约不满足 PRD，停止
并报告，不自行修改生产业务语义。

## 3. 整理归档 PRD

- 对照已有提交、独立检查和测试报告，把阶段 1/2/3 与静态路由 PRD 中已完成的
  Acceptance Criteria 改为 `[x]`。
- 不修改归档 `implement.md`，不新增无法从历史证据证明的结论。

## 4. 验证

```bash
go -C server test -v ./internal/data -run '^TestCommissionCreateSharedTransactionPostgres$' -count=1
go -C server test ./internal/data -count=1
go -C server test ./...
go -C server vet ./...
git diff --check
```

- 未设置环境变量时，第一条命令必须明确显示 SKIP。
- Codex 主会话使用名称固定到本任务的隔离临时数据库和临时角色显式注入
  `RONCIN_INTEGRATION_DATABASE_SOURCE`，重跑全部子用例，并在结束后验证库与角色
  已删除。
- 独立 `trellis-check` 重点检查装饰器是否真正委托真实仓储、清理顺序和错误是否
  可能被吞、失败断言是否包含主单/明细/审计。

## 5. 提交与回滚点

- Agy 禁止任何 Git 操作；Codex 最终提交建议：
  `test: 补齐提成 PostgreSQL 事务集成验证`。
- 归档 PRD 与测试放在同一提交，作为本轮历史证据整理。
- 若出现生产缺陷，测试与文档不先提交，返回规划拆出修复任务。
