# 自动锁单实施与验证计划

## 进入条件与范围

PRD 的逐账单结清方案已获业务确认；本文件及 design.md 的第一阶段规划总览获确认后才激活本子任务。实现由 trellis-implement 执行，trellis-check 独立复核，主会话维护任务、规范和提交。实现阶段禁止 TDD，完成实现后补针对性回归。

## 顺序

- [x] 修改 `server/internal/data/order_auto_lock.go`：按 BillID 去重加载相关账单，分别聚合核销/对冲并逐账单精确结清；保留其它门禁与既有锁序。
- [x] 更新注释及旧合并账单测试预期，补 PRD AC1–AC6：100+200 收100、100+100 收100、全收300、多币种/多账单、纯对冲/混合、反转、草稿账单、共享 MBL 和幂等。
- [x] 使用 `getIntegrationData` 创建隔离 Schema 并重放正式 SQL 迁移；只使用专用 `RONCIN_INTEGRATION_DATABASE_SOURCE`，不输出凭据、不重置日常数据库。
- [x] 定向验证后检查差异，复核反转与自动锁的并发顺序；记录实际 PASS，SKIP 不算通过。
- [x] 同步 `.trellis/spec/server/backend/order-lock-and-document-version.md` 中受影响契约；不把未实现的更正模型写入现行规范。
- [x] 由主会话按可验证的一组提交代码、回归及规范；第一阶段与结案子任务集成后执行完整门禁。

## 验证命令

已注入专用数据库环境时执行：

```bash
go -C server test -p 32 ./internal/data -run 'TestAutoOrderLock_' -count=1 -v
go -C server test -p 32 ./internal/biz -run 'Test.*Auto.*Lock' -count=1
git diff --check
```

新增测试使用 TestAutoOrderLock_ 前缀，核查输出确实包含新增场景。若修改其它财务触发入口，追加对应核销/对冲事务测试。第一阶段最终执行 `pnpm run check:fast`；该命令不能替代真实数据库定向验收。本任务无构建配置变更，不固定运行 build。

## 检查与交付

- [x] check 代理确认非共享订单及共享组都无误锁，不同币种不轧差，未改变 CLOSED/TERMINATED 的既有自动锁定资格。
- [x] acceptance.md 记录测试命令、环境隔离方式、结果与限制，不写连接串。
- [x] 合并账单全收旧策略已一次性更新，无兼容分支；未修改无关工作。
