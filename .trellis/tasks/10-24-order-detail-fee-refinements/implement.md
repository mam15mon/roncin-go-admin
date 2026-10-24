# 执行计划

按序执行；每阶段独立提交。约定：先做 schema/契约，再做后端实现，最后前端；生成物与源文件同组提交。

## 阶段 1：Schema 与迁移

- [ ] 1.1 Ent schema：`fee_setting.go` 删 charge_category 字段/edge/索引；`fee_setting_template.go` 删 charge_category_id；`master_data_item.go` kind 删 `charge_category`、删 edge；`order_fee.go` 删 `expense_date`。
- [ ] 1.2 `go -C server generate` 重新生成 Ent 代码。
- [ ] 1.3 新增迁移 `server/migrations/20261024100000_drop_fee_charge_category.sql`（删大类数据行、重建 kind 枚举、删两表大类列与索引）与 `20261024110000_order_fee_drop_expense_date.sql`（删 order_fees.expense_date）。
- [ ] 1.4 `pnpm run migrate:dev` 应用迁移（开发库）。
- 验证：`go -C server build ./...`；迁移校验和生效。

## 阶段 2：契约与权限

- [ ] 2.1 proto：`order_fee.proto`（OrderFee/AddFee/UpdateFee/BulkUpdate/ResolveFeeExchangeRate 删发生日期与 expense_date 参数）；`finance/v1/fee_catalog.proto` 删大类字段；`finance/v1/settlement.proto` 台账过滤改 created_at。
- [ ] 2.2 `make -C server api` 重新生成绑定与 openapi。
- [ ] 2.3 `manifest.go` 新增 `system.finance.order_fee.exchange_rate.override`，调整 FinanceExchangeRateUpdate 描述；`pnpm run generate:permission-keys`。
- [ ] 2.4 `pnpm run generate:web-client`。
- 验证：生成差异审查，无手工改动物。

## 阶段 3：后端实现

- [ ] 3.1 人员带入防绕过：订单写入路径按客户档案覆盖 SALES/CUSTOMER_SERVICE 角色（`order_write.go` / `order_write_draft.go`）。
- [ ] 3.2 状态跳转：`biz/order_transition.go` 目标集合改为全部后续节点；`data/order_write_lifecycle.go` 一次事务补齐中间事件；`validateAirFlowData` 放宽为序比较、仅校验目标节点。
- [ ] 3.3 一键完结：`TransitionClosure` 未完成主流程时事务内补齐全部节点再结案。
- [ ] 3.4 费用科目：`fee_catalog` / `masterdata` 清理大类；种子与同步脚本清理。
- [ ] 3.5 费用日期：`biz/order_fee.go` 输入删 ExpenseDate、BulkUpdate 删维度、排序改 created_at；`data` 层查询/写入同步。
- [ ] 3.6 汇率按 ETD：`orderETDDay` + `ErrOrderEtdMissing`；`ResolveExchangeRate` 与费用创建/更新解析改 ETD。
- [ ] 3.7 汇率覆盖权限：新错误 + 权限校验 + 同币种集合校验，覆盖 Add/Update/补录/更正入口。
- [ ] 3.8 后端测试：更新受影响用例（order_test、order_fee_test、fee_catalog_test、transition 相关集成测试），补跳转/完结补齐/权限/ETD 用例。
- 验证：`go -C server test -p 32 ./...`（集成测试依赖本地 PG）。

## 阶段 4：前端实现

- [ ] 4.1 `SeaPersonnelSection.tsx` 业务/客服只读、去必填；`SeaBasicInfoSection` 带入逻辑保持；相关测试更新。
- [ ] 4.2 `OrderStatusSection` / `order-detail-transitions` 跳转文案与测试更新。
- [ ] 4.3 费用设置页删大类；费用表单/列表/台账/导出/补录/更正的日期口径改录入日期。
- [ ] 4.4 `use-fee-exchange-preview.ts` 触发条件与错误处理更新；`access.ts` 权限改绑。
- [ ] 4.5 前端定向测试更新与补充。
- 验证：`pnpm --dir web tsc`；受影响文件 `pnpm --dir web exec vitest run <files>`。

## 阶段 5：收尾

- [ ] 5.1 `pnpm run check:fast` 全量门禁。
- [ ] 5.2 分组提交（schema+迁移 / 契约+生成物 / 后端 / 前端）。
- [ ] 5.3 review（trellis-check）后合并、push、归档任务。

## 回滚点

- 每阶段独立提交；迁移回滚依赖 Git revert + 开发库重建（需用户授权）。
