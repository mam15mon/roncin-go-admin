# 实施计划：订单费用按账单占用关系删除

## 开始条件

- [x] 用户澄清已归档（follow-up-cancelled-fees.md），本任务承接其范围。
- [x] Remove/建账/取消账单代码路径、FK 引用清单、迁移与错误目录工具链已核实。
- [x] 共享文件并行改动已识别，沿用补丁级提交纪律。

## 变更组一：存储契约 + 服务端删除

- [ ] Ent Schema：`finance_bill_line.order_fee_id` 可空 + SetNull；`order_fee_enterprise_tags.order_fee` Cascade；`go -C server generate`。
- [ ] 迁移 `20260922130000_order_fee_delete_by_bill_occupancy.sql`（DROP NOT NULL + 置换两外键），本地 `pnpm run migrate:dev` 应用。
- [ ] data 层：`orderFeeRepo.Remove` 改物理删除（占用复核 → 删除 → 审计 `order.fee.delete`）。
- [ ] biz：`ErrOrderFeeBillOccupied` + `Remove` 注释与触发语义核对；重生成错误目录。
- [ ] 集成测试 `order_fee_delete_integration_test.go`（隔离 Schema 全链迁移）：删除/占用/取消后再删/再建账拒绝/快照可读/级联/版本/审计。
- [ ] `go -C server test ./internal/data -run OrderFeeDelete`（配集成库）、`go vet`。

## 变更组二：前端录入页

- [ ] `OrderFeeTableTabs`：两表过滤 CANCELLED（数据/最近结果/父级集合/笔数金额同源）；「作废」→「删除」。
- [ ] `fees.tsx`：`handleDeleteFee` 确认流（不采原因）、成功/失败提示、刷新与关联失效。
- [ ] 更新/新增前端定向测试：过滤、删除按钮、确认流、占用提示；修正 billTracking「已作废」断言。
- [ ] vitest 定向 + biome + tsc + architecture。

## 提交与收尾

- [ ] 组一、组二分别补丁级提交（避开并行会话未提交改动）。
- [ ] `error-catalog --check`、`check:server`、`check:web` 风险匹配执行。
- [ ] 更新 spec（错误目录已含新码；评估 finance-bill-currency/commission-lock 相关表述）、日志、归档。

## 验证命令

```bash
RONCIN_INTEGRATION_DATABASE_SOURCE=... go -C server test ./internal/data -run 'OrderFeeDelete|FinanceBill' -p 1
go -C server vet ./...
node scripts/generate-error-catalog.mjs && node scripts/generate-error-catalog.mjs --check
pnpm --dir web exec vitest run src/pages/orders/components/fees/OrderFeeTableTabs.billTracking.test.tsx src/pages/orders/components/fees/OrderFeeTableTabs.feeColumns.test.tsx src/pages/orders/fees.test.tsx <新增测试>
pnpm --dir web tsc && pnpm -w run check:architecture
```

## 回退点

- 组一：还原 Schema/迁移/删除实现回到软作废（迁移仅放宽约束，可随链保留）。
- 组二：还原前端文案与过滤。

## 风险

- 并行会话仍在编辑共享文件：编辑前等待文件静止，锚点避开其区域。
- 迁移冷启动等价性由集成测试隔离 Schema 验证；不修改历史迁移。
