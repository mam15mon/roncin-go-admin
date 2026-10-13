# 自动锁单实现与定向验证记录

日期：2026-10-13（北京时间）。本记录由实现代理提供，提交和任务状态由主会话管理。

## 实现范围

- `server/internal/data/order_auto_lock.go`：使用活动订单行筛选应收账单集合，账单查询天然按 BillID 去重；草稿账单阻断。有效核销与对冲分别查询、按 BillID 相加，每张账单总额与其结清金额精确相等才通过，不跨账单或币种轧差、不把超额钳成零。
- `server/internal/data/order_auto_lock_integration_test.go`：原有合并账单全收测试改为允许锁单，保留部分结清不锁；夹具增加不同订单行金额、不同币种及对冲清理支持。
- `server/internal/data/order_auto_lock_settlement_integration_test.go`：补普通订单与共享 MBL 的同额/异额合账、全收与重复触发、多账单/多币种、超额异常、同票多行去重、纯对冲、多笔核销与对冲混合、父状态/分摊失效、草稿、已终止及结案回归。

## 验证

使用主会话准备的一次性专用 PostgreSQL 数据库，`getIntegrationData` 为每组测试创建随机 Schema，并重放正式 SQL 迁移。未读取或修改日常开发数据库；不记录连接凭据。

```bash
python3 /tmp/roncin-fee-phase1-ickh9kr0/run.py go -C server test -p 32 ./internal/data -run 'TestAutoOrderLock_' -count=1 -v
go -C server test -p 32 ./internal/biz -run 'Test.*Auto.*Lock' -count=1
git diff --check -- server/internal/data/order_auto_lock.go server/internal/data/order_auto_lock_integration_test.go server/internal/data/order_auto_lock_settlement_integration_test.go
```

最终 data 结果：PASS，8 个顶层测试、30 个子场景，0 FAIL、0 SKIP，17.815 秒。完整日志位于仓库外 `/tmp/roncin-auto-lock-tests.log`。biz 定向测试 PASS，差异检查 PASS。

首次回归新增夹具缺少反转/终止/结案 CHECK 要求的历史字段；已补齐操作者、时间、原因及终止类型，并完成最终验证。未删除或放宽正式数据库约束。

## 锁序复核及限制

- 自动检查仍在独立 `WithTx` 内锁订单后重读资格，共享 MBL 仍使用原有排序成员锁与整组原子写入。
- 账单 Confirm、Cancel、ConfirmBatch 均先调用 `lockExistingBillOrders`，再锁账单；反核销和反对冲先通过 `lockOrdersForFinanceReversal` 锁关联订单，再反转来源，维持现有串行顺序。
- 不新增锁实体、版本字段、接口、迁移或生成物；不自动解除已有业务锁，不扫描重写历史锁定记录。
- 定向验证包含自动锁定后反核销仍保留原锁事实，未新增财务反转竞争压力测试。第一阶段集成验收和完整门禁由主会话完成。
