# R4 提成真实库夹具修复

## 根因与修复边界

本组只调整集成测试夹具和失效断言，不修改产品权限、真实清偿投影、经济重算或提成计算规则，不放宽数据库约束。

- `finance_commission_transaction_integration_test.go`：通用提成夹具的经营组织从 `system` 改为 `company`，与当前公司持有经营数据的契约一致。
- `commission_adjustment_transition_integration_test.go`：人工调整取消场景先通过现有 `approveInitialFees` 建立真实费用与批准快照，再创建来源提成；避免空来源被当前经济证据校验拒绝。
- `finance_commission_application_integration_test.go`：原夹具对同一张 1000 应收账单反复创建多张全额 ACTIVE 核销，第二来源超出实际可分配收入，当前经济投影正确排除该来源。现在每个来源有独立订单、应收 1000/应付 400、两方向账单及批准快照；员工归属同步覆盖所有来源订单。第一来源保留现有字段引用，让陈旧来源和重提刷新测试仍真实修改被测账单/费用。每来源提成仍为 60，两来源仍为 120，迟到来源不会改写首个订单快照。
- `finance_commission_netting_integration_test.go`：原合并应付账单标头 800，而明细只有 100+200，账单清偿与费用行不守恒。增加无员工归属、只有 500 应付费用的第三张订单补齐账单；原两张计提订单利润仍为 200+300，提成仍为 20+30。支付反转断言改为明确来源对冲 ID、逐订单实际追回 20/30、非空独立幂等键，不再要求已经退役的 `nt:<id>` 字符串前缀。父单保持 PAID、总冲减 50、净额归零、财务锁释放与费用批准冻结的断言保留。

## 验证

- 独立 PostgreSQL 实际执行：`go -C server test -p 32 -count=1 -json -timeout 25m -run Commission ./internal/data`，退出码 0；全部 196 个测试事件 PASS，0 FAIL，0 SKIP，无 panic 或超时。日志 `/tmp/roncin-audit-postgres-fixtures.jsonl`。
- 此筛选同时包含费用更正提成、月度申请审批、创建共享事务、来源重复计提、核销/对冲支付反转、逐订单冲减、财务锁、月度锁顺序、方案分配和工作台查询；完整 data/migration 包留主会话执行。
- 四个修改过的 Go 文件执行 `gofmt`，`git diff --check` 通过；完整差异 `/tmp/roncin-audit-commission-fixtures.diff`。

## 交接

使用主会话创建的独立一次性 PostgreSQL 库，连接与凭据仅留 `/tmp` 私有状态文件，不写入文档或提交。未触碰开发库，未提交；主会话负责整包验证、资源清理与最终提交。
