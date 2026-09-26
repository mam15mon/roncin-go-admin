# 费用状态消费方盘点与审阅修正

日期：2026-09-26。范围：仓库代码与任务规划。本文用于定位，不以估算的命中数量代替逐项验收；实施前重新检索。最终决策以 prd.md/design.md 为准。

## 已核实的关键事实

| 证据 | 事实与处置 |
| --- | --- |
| `server/internal/data/order_fee.go:685`、`:857` | Remove/BulkRemove 已硬删除；移除多余状态依赖，保留事务事实复核与审计；补录来源拒绝普通删除 |
| `server/internal/data/order_fee_supplement_cancel.go:265` | 专用撤销仍写 CANCELLED，需真正改为 DELETE |
| 同文件撤销尾部 | 当前重查费用并返回 Fee；删除后不能沿用，需同步领域结果与响应 |
| `server/internal/service/order_fee_supplement.go:302` | 当前响应包含 result.Fee；改为刷新后的申请 data |
| `server/internal/data/ent/schema/finance_bill_line.go:14`、`:45`、`:53` | 历史快照保留，来源费用外键 SET NULL；已有 order_fee_id/active 索引和活动费用唯一索引 |
| `server/internal/data/ent/schema/order_fee_supplement_request.go` | 申请拥有不可变费用快照；费用通过 supplement_request_id 反向关联，并非独立链接表 |
| `server/internal/data/finance_bill_write.go:78`、`:87` | 插入账单行先于费用版本更新，后者不能机械改用 NOT EXISTS |
| `server/api/order/v1/order_fee.proto:647` | 撤销响应旧 fee 字段须退役，不能只改文案 |

旧方案的「保留 OrderFee.status 派生枚举」「普通删除从软删改硬删」「未建账从未被历史账单引用」「补录链接表清理」均已纠正。`cancellation_reason` 才是实际字段名。结案阻断未建账，拆票/改配等阻断已建账，不能把门禁方向混写。

## 后端逐项执行清单

| 文件/区域 | 处置与验证重点 |
| --- | --- |
| `data/order_fee.go` List/Get/GetByIdempotencyKey/orderFeeToBiz | 批量加载关联事实；不存在未加载默认 false；返回数据完整 |
| `data/order_fee.go` Update/ensureOrderFeeBulkUpdatable | 状态分支改关联事实；保留账单→账单行→费用锁序、字段限制、停用单位/币种既有策略、DRAFT 设置与单行换币 |
| `data/order_fee.go` Remove/BulkRemove | 原有硬删除/版本/占用/历史快照/审计保留；拒绝补录来源 |
| `biz/order_fee.go` | 删除状态类型与字段、创建默认值；改信用控制、费用目录快照、税率/名称覆盖、汇率继承与编辑准入；删除成功触发自动锁定，审计去状态键 |
| `data/finance_bill_write.go`、`data/finance_bill_batch.go` | 单张/批量建账、取消去状态写入；锁内复核事实，保留版本递增、行数检查；关注插入与检查顺序 |
| `data/finance_bill.go` | ListCreationCandidates 统一事实口径 |
| `biz/finance_bill.go`、`biz/finance_bill_batch.go` | 建账预览及过期校验、候选校验改关联事实，不能把预览反馈全部延迟到写入时 |
| `data/order_fee_supplement_approval.go` | 创建去状态；边际基线不再过滤 CANCELLED；保持同事务生成 |
| `data/order_fee_supplement_cancel.go` | 能力投影、倒序约束、已确认/已支付调整阻断、DRAFT 取消、硬删除、审计、重复撤销及返回值全部同步 |
| `biz/order_fee_supplement_usecase.go`、`biz/order_fee_supplement.go` | 删除旧状态分支与撤销结果 Fee 依赖；保持授权、事务和联动 |
| `data/order_auto_lock.go` | 未建账存在性阻断与至少一笔有效应收结清事实组合保持不变 |
| `data/order_write_lifecycle.go` | 结案 readiness：未建账费用阻断 |
| `data/sea_order_change.go` | 已建账阻断拆票/改配 |
| `data/sea_order_change_split.go`、`data/sea_order_change_split_execute.go` | 候选及执行锁内逐费复核、版本校验、整行迁移 |
| `data/sea_document_change_shared.go`、`data/order_write_sea_master_bill.go` | 下游已建账阻断事实改 EXISTS |
| `data/settlement.go` | 提取统一谓词；去 status 过滤和 CANCELLED 排除；保持 financial_progress/bill 投影与汇总一致 |
| `biz/settlement.go` | 删除费用状态筛选与枚举参数校验 |
| `data/workbench.go` | 待办未建账计数、提成归属有效费用来源，EXPLAIN 验证 |
| `data/finance_commission_calculation.go` | 新 fee_snapshot 及来源指纹去费用状态；检查重复计提防护，不重写历史金额 |
| `service/order_fee.go`、`service/settlement_fee_ledger.go`、`service/settlement_bill.go` | 去状态转换/筛选透传，OrderFee 输出 has_active_bill |
| `service/settlement_commission.go`、`service/order_fee_supplement.go` | 删除提成费用状态及 fee_status，撤销返回申请信息 |

路径前缀均为 `server/internal/`。检索补充其它 orderfee.Status、OrderFeeStatus、fee.status 消费，不能仅依赖本表。

## 契约、Schema 与数据

- `server/api/order/v1/order_fee.proto`：OrderFeeStatus、OrderFee.status、申请 fee_status、撤销响应 fee。
- `server/api/finance/v1/settlement.proto`：ListFeeLedgerRequest.status、FeeLedgerItem.status、CommissionFeeDetail.status。
- `server/internal/data/ent/schema/order_fee.go` 与 User 反向边：status、cancelled_at、cancelled_by、cancellation_reason、CHECK、索引、关联。
- 正式迁移在删列前处理 CANCELLED；保留旧有序迁移和升级测试输入，不执行全仓盲删。历史数据处理与开发库授权见 design.md §5。
- `server/cmd/sync-dev/main.go` 原费用创建、构造已建账与补录种子三处同步；不再只设 BILLED 模拟账单事实。
- `.trellis/spec/domain/glossary.md`、财务现行规范更新；旧任务归档不改写。

## 前端逐项执行清单

路径前缀 `web/src/`：

- `constants/statusMeta.ts`、`pages/orders/components/fees/feeConstants.ts`：删除费用状态枚举/映射与 feeStatusCode。
- `pages/orders/components/fees/OrderFeeTableTabs.tsx`：状态列、本地新行默认值、编辑/删除/批量准入与 CANCELLED 排除全部更新。
- `pages/orders/components/fees/{feeBaseColumns,orderFeeColumns,orderFeeOptionalColumns}.tsx`：删除状态列，财务进度保留。
- `pages/orders/{order-fee-panel,order-fee-panel-columns,fees}.tsx`：关联判定、编辑弹窗与 feeBillTracking 口径一致。
- `pages/orders/components/fees/FeeSupplementSection.tsx`：去 feeStatus，显示批准后生成费用已删除，撤销后刷新相关数据。
- `pages/orders/components/detail/OrderLockControl.tsx`：触发文案检查，触发事件类型不等于费用状态，不机械删除 FEE_BILLED/FEE_CANCEL。
- `features/finance/bill-creation/{BillCandidateSelectionStep,billWorkbenchFeeColumns}.tsx`：去状态列与双重状态判定，保护现有未提交修改。
- `pages/finance/fees/{index,components/feeLedgerColumns,components/FeeLedgerSearchFilter}.tsx`：列/筛选/建账按钮判定。
- `components/ui/finance-ledger-template/FinanceSummaryBoard.tsx`：移除 CANCELLED 与数值 4 排除，不用另一种过滤代替。
- 提成费用明细消费方、`services/roncin/` 与 `enums.generated.ts`：契约生成后按类型检查定位全部调用方。

## 验证入口

后端重点现有测试：order_fee_delete、order_fee_bulk、order_fee_supplement_transaction、finance_bill_transaction、finance_bill_batch、settlement_fee_ledger、order_auto_lock、workbench、sea_order_change、sea_document_change、finance_commission 系列集成测试；order_fee_status_migration 与 order_fee_schema_metadata 按正式迁移链更新，不能直接删掉历史升级覆盖。biz/service 相关状态测试改行为断言。

前端重点：statusMeta、OrderFeeTableTabs 各分拆文件、FeeSupplementSection、orders/fees、use-order-fee-panel-rate、OrderLockControl 及实际受影响费用台账/建账测试。复杂组件测试按仓库耗时/数量约束拆分。

`scripts/acceptance-finance-bill-batch.mjs` 存在旧 ORDER_FEE_STATUS_CONFIRMED 断言，随契约清理；payable 脚本财务进度断言保留并检查相邻流程。

数据库集成测试依赖 `RONCIN_INTEGRATION_DATABASE_SOURCE`，缺配置会 SKIP。必须报告真实执行结果，不能仅以 go test 退出码代替集成验收。
