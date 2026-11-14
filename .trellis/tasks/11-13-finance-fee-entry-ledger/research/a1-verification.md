# A1 核对结论（阶段 A 步骤 1）

日期：2026-11-14。结论：**不推翻 design.md 的 1.1 与 1.3**，可继续实施；有两处补充（见 c 与 b 的注意事项）。

## a) `orders.trade_direction`

- Ent 字段 `field.Enum("trade_direction").Values("export","import")`（`server/internal/data/ent/schema/order.go:61`）：**非 Optional、非 Nillable，数据库 NOT NULL，无空值可能**。所有 business_type（SE/SI/AE/AI/LAND/RAIL）共用同一列，取值只有 export / import。
- 当前创建与更新入口 `normalizeOrder`（`biz/order_usecase.go:318-323`）校验 `TradeDirection.Valid()`，并且只放行 `SE/AE + export`，其余返回 `ErrOrderBusinessUnsupported`；因此 LAND/RAIL/SI/AI 目前根本无法落库，更无空值问题。进口、LAND、RAIL 上线时 `OrderRateDay` 已按方向分支，无需再改。
- 开发库只读 SQL：`orders` 共 10 行，全部 `SE / export`，`etd`、`eta` 均非空，无空值样例。
- 结论：设计 1.1「按 trade_direction 判定」成立；方向非法或为空时 `OrderRateDay` 显式返回 `ORDER_FEE_INVALID_ARGUMENT`，不默认出口。

## b) 发票、核销、对冲、建账是否先锁 Order

均在写关联前先按主键升序对相关 Order `FOR UPDATE`，再锁账单/流水/费用行（锁序为 Order → 账单 → 账单行 → 费用）：

| 路径 | 先锁 Order | 备注 |
| --- | --- | --- |
| 建账 `finance_bill_write.go:27-37` | `lockBillFeeOrders` (:33) | 随后费用行升序 `ForUpdate` (:37) |
| 草稿账单更新/确认/取消 `finance_bill_write.go:117-121, 214-218, 240-244`；批量 `finance_bill_batch.go:61-73`、费用 :127 | `lockExistingBillOrders` | 与既有 spec 一致 |
| 发票创建 `finance_invoice.go:259-265`、开具 :329-339 | `lockExistingBillOrders` | 其后锁发票档案、账单 |
| 核销创建 `finance_verification.go:188-198`、反转 :351-363 | `lockOrdersForFinanceReversal`（订单升序 ForUpdate，`order_finance_reversal_lock.go:11`） | 其后锁流水、账单 |
| 对冲确认 `finance_netting.go:401-419`（`LockNetting`/`LockNettingBills` 内 `lockFinanceEconomicOrders`，:152-159, 293-300）、反转 :508-527（`lockOrdersForFinanceReversal`） | 是 | 草稿创建（:356）与取消（:478）不锁 Order，但草稿对冲不属于受保护事实 |
| 发票取消/红冲 `finance_invoice.go:441-470` | **否**（Invoice → 活动开票关联 → 账单，`lockAndReleaseInvoiceActiveBills` :388-） | 只会释放保护，不制造新的保护事实 |

这些写路径都不对 `order_fees` 行本身加锁，保护事实落在账单、发票、核销、对冲表上，并由 Order 锁串行化。

联动事务内的并发校验结论：

1. 联动已按主键升序持有成员订单锁，所有「新增保护事实」的写路径（建账、开票、核销、对冲确认）都先要这把 Order 锁，因此**在联动持锁期间不会有新的保护事实提交**；联动在 Order 锁内重读 `feeHasProtectedFinanceFact`（读已提交，每条语句新快照）即得到最新事实，无需再给这些写路径补费用行锁，也无需锁账单。
2. 发票取消/红冲不先锁 Order，但它只移除保护：联动可能读到取消前的事实而保守跳过该费用，属 fail-safe（不会误改受保护费用）。联动不得为此去锁账单行，否则会引入「Order → 账单」与取消路径「发票 → 账单」之间的新锁序组合；联动只锁 Order 与费用行即可，与现有「订单 → 费用」锁序一致，不出现死锁环。
3. 风险边界：联动只能保证持有的订单集合内的事实最新。账单经济闭包可能涉及其他订单，但其他订单上的写路径都会一并锁闭包内全部订单（`validateFinanceEconomicOrderSet`），与联动订单集合相交时同样互斥。

## c) 页面显示的 ETD/ETA 来源

- **海运（SE）**：订单详情的 ETD/ETA 在存在活动 MBL 关联且运输执行（TE）有值时，来自 **TE 覆盖值**（`data/order_convert.go:153-171`，覆盖 `result.ETD/ETA`；MBL 摘要 `data/sea_master_bill.go:249-253` 同理）；否则读 `orders.etd/eta` 列。
- **空运（AE）**：订单列 `orders.etd/eta` 在建单时由主单复制（`data/air_order_sync.go:59-60`）；之后主单更新（`data/air_export.go:385-427`）只写主单自身列，**不写成员订单列**。
- 因此目前存在已知分叉：共享海运航程更新（`data/sea_order_change_transport_update.go:207-214`）与空运主单更新只改 TE/主单，订单列与页面显示（SE）或订单列自身（AE）可能不一致；费用汇率现读订单列。这正是设计 1.2「共享改期时同事务写回成员订单 `etd/eta`」要解决的问题，设计方向无须改动；单票内联改 MBL/TE（`order_write_sea_master_bill.go:443-451, 546-554`）与 `UpdateDraft`（`order_write_draft.go:267-268`）已写订单列。
- 改配路径 `sea_order_change_reassignment.go:794-799` 已同步订单列（用 TE 值 Set/Clear）。拆票、恢复的订单日期写入留待步骤 5 单独核对。
- 补充：`OrderRateDay` 读的是订单列，步骤 5 落地前，对共享改期场景订单列可能是旧值；本步骤不改。
