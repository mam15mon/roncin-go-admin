# 执行计划

## 启动与范围确认

- [ ] 用户确认本版方案后 task.py start；实施前检查 git status 与更近 AGENTS.md。
- [ ] 核对原提交、迁移发布/应用范围和审计字段展示入口；保护其它未提交修改。

## 修改及实现后测试

1. [ ] 收紧旧费用清理迁移的冲减守卫，不自动修改金融调整。
2. [ ] 实现后补真实申请/费用/调整夹具：DRAFT、CONFIRMED、PAID、CANCELLED 有/无历史时间、无调整；验证失败原子性与成功历史保全。
3. [ ] 普通单条/批量删除在同事务补审计名称与单位快照，批量单位查询去重；补针对性内容和回滚断言。
4. [ ] 定位并修正自动锁定测试组织/本币夹具，验证核销触发、未建账阻断与删除后重评相邻路径；禁止 skip 或降低断言。
5. [ ] 只读核查已应用开发库的迁移影响；不执行数据修复、清库或重放删除。发现异常先交付清单。

## 验证

- [ ] 配置专用 `RONCIN_INTEGRATION_DATABASE_SOURCE`，仅用隔离 schema 与正式 SQL 迁移；禁止输出连接串或在开发库运行测试。SKIP 不算通过。
- [ ] 定向运行：

```bash
go -C server test -count=1 -p 1 ./internal/data -run 'TestOrderFeeStatusHardDeleteMigration|TestOrderFeeDeleteByBillOccupancyPostgres|TestOrderFeeBulkMaintenancePostgres|TestAutoOrderLock_SettlementTriggerPostgres|TestFeeSupplementGeneratedFeePlainDeleteForbiddenPostgres' -v
```

- [ ] 新增测试不在上述正则时补入；记录每个目标子用例是否实际执行，避免某一前置失败遮蔽后续断言。
- [ ] 相关 biz/data 测试、`git diff --check` 后分组提交：迁移+测试、审计+测试、自动锁定夹具+验收更正；每组可验证，中文 Conventional Commit。
- [ ] 最终 `pnpm run check:fast`；真实数据库定向结果单独记录，不能用无集成配置的门禁替代。
- [ ] 形成 research/verification.md，对照 A1—A6；其余基线失败仅登记，不盲目扩大范围。
- [ ] 更新必要现行规范，按 Trellis finish-work 收尾；证据不足不标全部完成。
