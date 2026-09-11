# 汇率极简化彻底重构与核心业务/安全漏洞闭环 Technical Design

## 1. Architecture & Design Principles

遵循系统当前交付原则（"优先完成最小业务闭环、不预建通用平台与一次性抽象"），本次技术设计采取**彻底精简架构、根治概念错位**的路线：
1. **单一基准汇率真相源**：全组织树汇率由总部唯一定义与维护，所有业务阶段（费用、账单、流水、发票、提成换算）统一拿单据业务日期查询唯一的折本币中间价；
2. **阻断门禁严格闭环**：消除"死代码"门禁，单证变更与订单变更严格落实业务锁、结案/终止状态机与下游财务事实不可逆拦截；
3. **借贷平衡自洽**：消除伪核销汇率，核销单头与对冲单头严格满足复式记账借贷平衡与汇兑损益沉淀；
4. **绩效与业务解耦**：提成冲平后自动释放费用锁，保证异常调账通路畅通。

**前置条件**：09-10 任务（`feat/finance-launch-readiness`，PR #1）已合入 `main`，`ensureOrderBusinessEditable` 统一门禁已在主线可用，本任务 A-3/A-4 直接消费；开工前 09-10 归档。

---

## 2. Core Modules & Data Models

### 2.1 汇率实体模型重构 (Exchange Rate Overhaul)

#### 数据库模型重构 (`server/internal/data/ent/schema/exchange_rate_setting.go`)
- **移除冗余字段**：`rate_type`、`receivable_rate`、`payable_rate`；
- **新增单一汇率**：`rate numeric(18,8)`；
- **唯一约束**：`(organization_id, from_currency, to_currency, effective_from)`；
- **下线实体**：物理删除 `ExchangeRateCustomSetting` 与 `ExchangeRateTimeStandard` 两张 Ent 模式及数据表。

```text
Table: exchange_rate_settings
  - id: uuid (PK)
  - organization_id: uuid (FK -> headquarters)
  - from_currency: varchar(3)
  - to_currency: varchar(3)
  - effective_from: timestamp with time zone
  - effective_to: timestamp with time zone (nullable)
  - rate: numeric(18,8)
  - is_active: boolean
  - created_at / updated_at
```

#### 存量迁移规则（`server/migrations/` 手写 SQL，时间戳命名，与 Ent Schema 同步）
1. `rate_type = 'BASE_CURRENCY'` 的行存活，`rate = receivable_rate`；其余类型行删除；
2. 塌缩后触发新唯一键 `(organization_id, from_currency, to_currency, effective_from)` 冲突时，迁移直接失败并报错，人工裁定后重跑，**禁止静默合并或任选一行**；
3. `DROP TABLE exchange_rate_custom_settings`、`exchange_rate_time_standards`（用户已明确授权）；
4. 开发库迁移校验和按 `migrate:dev` 规则允许重录。

#### 领域层内核重构 (`server/internal/biz/exchange_rate.go`)
- 废弃 `ResolveContext` 递归与 `TimeStandards` 瀑布流；
- 核心方法：
  ```go
  func (uc *ExchangeRateUsecase) ResolveRate(ctx context.Context, orgID uuid.UUID, currency string, targetDate string) (decimal.Decimal, error)
  ```
- 行为：
  1. 解析组织树根（总部）与基准币种；若 `currency == BaseCurrency`，直接返回 `1.0`；
  2. 校验 `targetDate` 格式（RFC3339 或 YYYY-MM-DD）；
  3. 查询总部组织下有效区间覆盖 `targetDate` 的唯一启用地汇率；未命中返回 `ErrExchangeRateMissing`。
- **写防御（D 组修复）**：
  - data 层复用 `headquartersOrganizationID` 助手（同 `fee_catalog.go` 模式）在 `Create`、`Update`、`Disable` 前校验调用组织即总部，否则拒绝；
  - **同时删除** biz 层现存的"写入重定向总部"逻辑（`normalized.OrganizationID = rateContext.OwnerOrganizationID`）：新政策是非总部一律拒绝，重定向与其互斥，保留即保留越权通道。

---

### 2.2 全业务单据汇率消费统一化

| 业务领域 | 消费时点（业务日期） | 查询方式 | 改造点 |
|---|---|---|---|
| **订单费用 (order_fee)** | `fee.ExpenseDate` | `ResolveRate(ctx, org, fee.Currency, fee.ExpenseDate)` | 移除 `BaseCurrencyRateType` 参数；`order_fee.go:349` 账单上下文预解析同步迁移 |
| **账单 (finance_bill)** | `bill.BillDate` | `ResolveRate(ctx, org, bill.Currency, bill.BillDate)` | 移除 `BillRateType` 参数 |
| **资金流水 (finance_cashflow)** | `cashflow.TransactionDate` | `ResolveRate(ctx, org, cashflow.Currency, cashflow.TransactionDate)` | 移除 `SettlementRateType` 参数；手工覆盖（`ExchangeRateOverride`）机制不动 |
| **发票 (finance_invoice)** | `invoice.InvoiceDate` | `ResolveRate(ctx, org, invoice.Currency, invoice.InvoiceDate)` | 移除 `InvoiceRateType` 参数 |
| **核销 (finance_verification)** | **不查汇率** | 取各分摊行 `cashflow_base_amount` | **移除核销时的汇率查询（B1 修复）**：单头 `base_amount = ∑ cashflow_base_amount`；删除单头汇率快照四字段（`exchange_rate`/`exchange_rate_source`/`exchange_rate_date`/`exchange_rate_setting_id`）与行级 `write_off_base_amount`（Schema + 迁移 + Proto 同步） |
| **提成换算人民币 (finance_commission)** | 提成生成日 `generation.ExchangeRateDate` | `ResolveRate(ctx, org, baseCurrency, ...)` | `finance_commission.go:559、678` 两处 `WRITE_OFF` 解析迁移；基准币种为 CNY 时自然返回 1 |
| **提成分母 (finance_commission)** | 账单日 | 消费账单行已快照的 `BaseCurrencyAmount` | **分母聚合改用账单行本币（B3 修复）**，拉齐收入与成本汇率口径 |

#### 汇率导入收敛（`exchange_rate_import.go`）
- 删除按类型命名的中文列映射（"核销汇率"等五类），模板收敛为单一"折本币汇率"列；
- 导入校验、批次实体中的类型字段一并清理，前端导入界面同步。

---

### 2.3 审计缺陷定向闭环设计

#### A 组：单证变更路径穿透业务锁 (P0)
1. **激活下游阻断（A-1）**：
   在 `server/internal/data/sea_document_change.go` 的 `collectDocumentImpacts` 中：
   ```go
   // 遍历费用、账单行、发票、核销、提成、提成调整及箱货分配时：
   impacts = append(impacts, biz.SeaDocumentImpact{
       FactType: factType, ReferenceID: refID, Message: msg,
       BlocksExecution: true, // 核心修复：显式置为阻断
   })
   ```
   补齐对 HBL 箱货分配（Container Allocation）存在的查询与阻断标记；同时清理 `includeAllocations` 参数从未使用的死参数。
2. **模式变更增加财务门禁（A-2）**：
   在 `ExecuteModeChange` 中收集 impacts 后，调用 `if hasBlockingImpact(impacts) { return impactError(biz.ErrSeaDocumentChangeBlocked, ...) }`。
3. **业务锁与终止状态拦截（A-3 / A-4）**：
   - 在 `executeMasterAmendment`、`executeHouseAmendment`、`executeMasterVoid`、`ExecuteModeChange` 中，对事务内已加载的关联订单执行 `ensureOrderBusinessEditable(ctx, tx, order)`（含共享 MBL 成员订单）；
   - 在 `sea_order_change.go` 的 `ExecuteTransportExecutionUpdate` 与 `ExecuteReassignment` 事务内，遍历已按主键序 `ForUpdate` 锁定的 `orders` 逐一校验 `ensureOrderBusinessEditable`，任一订单被业务锁定或处于 TERMINATING/CLOSED 即事务失败回滚。

#### B 组：对冲单平衡与汇差沉淀 (B2)
- 在 `server/internal/biz/finance_netting.go` 中，构建计划时分别累加两端本位币：
  ```go
  // base_currency_amount 语义固化为应收侧抵销本位币（现有字段，保留）
  plan.BaseCurrencyAmount = sum(allocation.BaseCurrencyAmount where Direction == RECEIVABLE)
  plan.PayableBaseAmount  = sum(allocation.BaseCurrencyAmount where Direction == PAYABLE)
  plan.ExchangeGainLoss   = plan.PayableBaseAmount.Sub(plan.BaseCurrencyAmount)
  ```
- Ent Schema 新增 `payable_base_amount`、`exchange_gain_loss`（numeric(28,8)），随迁移与 Proto 契约一起交付，对冲单详情页展示两端本位币与汇差；
- 同币种同汇率对冲 `ExchangeGainLoss = 0` 属正常结果；批量建账流程（`CreateBatch` 路径）同步两端累加。

#### C1：提成净额归零释放费用锁
- **净额公式（订单级）**：
  `net(order) = ∑ signed(commission_line where commission.status ∈ {CONFIRMED, PAID}) + ∑ signed(adjustment where status ∈ {CONFIRMED, PAID})`，
  冲减方向记负号；`net ≤ 0` 视为已全额冲减，释放费用编辑锁。
- **两处落点必须同一 SQL 语义**：
  1. `server/internal/biz/order_fee.go` 写入拦截（`ErrOrderFeeFinanceLocked`）；
  2. `server/internal/data/settlement.go:35-41` `financeLockedOrderPredicate`（台账列表 `finance_locked` 投影同源）。
  由 EXISTS 谓词改为相关子查询净额聚合；两处不得出现"列表已解锁、写入仍被拒"的漂移。
- PAID 提成仍为终态不改状态，保留已支付历史；净额释放只影响锁判定。

#### E 组：行业主数据总部共享 (E1)
- 在 `server/internal/data/industry_reference.go` 中，港口、机场、航司、船公司等全部实体查询：
  ```go
  headquartersID, err := r.headquartersOrganizationID(ctx, organizationID) // 复用既有助手
  query.Where(or(OrganizationIDEQ(organizationID), OrganizationIDEQ(headquartersID)))
  ```
- 候选列表按业务代码去重，本组织行优先（同代码同时存在时保留本组织行）；`is_active` 过滤保留；
  保证分公司即插即用总部标准主数据，同时支持分公司新增私有数据。

#### D 组：见 2.1 写防御（requireHeadquarters + 删除重定向）

---

## 3. Frontend & API Redesign

1. **Protobuf 契约精简 (`exchange_rate.proto`)**：
   - 彻底删除 `RateType` enum，删除 `receivable_rate`, `payable_rate`；
   - 消息体仅保留 `rate: string`；
   - 移除 `UpdateTimeStandards` 与 `UpdateCustomSetting` RPC 及消息；
   - 核销契约删除单头汇率快照字段与行级 `write_off_base_amount`；对冲契约新增 `payable_base_amount`、`exchange_gain_loss`；
   - 重新生成 Go 代码与 OpenAPI 规范，`generate:permission-keys` 重跑（权限码无增删，仅描述文案）。
2. **前端汇率管理页 (`web/src/pages/finance/exchange-rates`)**：
   - 移除 RateType 的 Tab 与表单项；
   - 表格列展示：币种、折本币汇率、生效起始日、失效日、状态、操作；
   - 新增/编辑弹窗仅输入单一汇率数值；导入模板与界面同步收敛；
   - 删除时间标准配置与继承策略相关路由与组件。
3. **核销与对冲详情页**：
   - 核销详情移除汇率快照展示，单头本币展示口径改为流水本位币合计；
   - 对冲详情新增应付侧本位币与对冲汇差展示。
