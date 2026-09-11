# 汇率极简化彻底重构与核心业务/安全漏洞闭环 Implementation Plan

> 前置条件：09-10 已合入 `main`（PR #1）并归档；本任务自 `main` 拉取 `feat/exchange-rate-simplification-and-bugfixes` 分支实施。
> C2（对冲结清触发提成）已拆出为独立后续任务，本计划不含任何 C2 步骤。

## 1. Phase Overview

| Phase | Description | Key Deliverables | Risk Level |
|---|---|---|---|
| **Phase 1** | 全部 Schema、Proto 契约、迁移与代码生成 | 汇率表精简、两张配置表下线、核销汇率快照字段删除、对冲两新字段、Proto 收敛、迁移 SQL、OpenAPI/权限生成物 | High (契约变更) |
| **Phase 2** | 汇率业务内核与全领域统一对接 | `ResolveRate` 极简化、D组总部写防御+删重定向、B1核销单头去汇率、B3提成拉齐账单日、提成CNY换算迁移、导入收敛 | High (跨层核心) |
| **Phase 3** | A 组 P0 与结算/主数据闭环 | A组财务门禁与业务锁对齐、B2对冲汇差计算与展示、C1净提成释放锁（双落点）、E组主数据总部共享 | Critical (业务正确性) |
| **Phase 4** | 前端精简与展示对齐 | 汇率管理页/导入收敛、核销详情去汇率、对冲详情新字段 | Medium (UI) |
| **Phase 5** | 全量回归测试与门禁验证 | Go 全量测试、前端 Biome/TSC、迁移校验和处理 | Low (门禁验收) |

---

## 2. Detailed Phase Tasks

### Phase 1: 全部 Schema、Proto 契约、迁移与代码生成

1. **汇率 Ent Schema 精简**（`server/internal/data/ent/schema/exchange_rate_setting.go`）：
   - 删除 `rate_type`、`receivable_rate`、`payable_rate`；新增 `rate`（numeric(18,8)）；
   - 唯一索引调整为 `(organization_id, from_currency, to_currency, effective_from)`；
   - 删除 `exchange_rate_custom_setting.go`、`exchange_rate_time_standard.go` 两个 Schema 文件；
   - 同步清理导入批次 Schema 中残留的类型字段（如有）。
2. **核销 Schema 字段删除**：
   - `finance_verification.go` 删除 `exchange_rate`、`exchange_rate_source`、`exchange_rate_date`、`exchange_rate_setting_id`；
   - `finance_verification_allocation.go` 删除 `write_off_base_amount`。
3. **对冲 Schema 新增字段**（`finance_netting.go`）：
   - `payable_base_amount`、`exchange_gain_loss`（numeric(28,8)，Immutable）。
4. **迁移 SQL**（`server/migrations/<timestamp>_exchange_rate_simplification.sql`，手写、时间戳命名）：
   - 汇率表：删 `BASE_CURRENCY` 以外的行 → `rate = receivable_rate` → 删三列 → 唯一键替换；塌缩后唯一键冲突时迁移失败报错，禁止静默合并；
   - `DROP TABLE exchange_rate_custom_settings`、`exchange_rate_time_standards`；
   - 核销表删除单头四列与行级 `write_off_base_amount` 列；对冲表新增两列。
5. **Proto 契约**（`server/api/finance/v1/exchange_rate.proto` 等）：
   - 删除 `RateType` 枚举、`UpdateTimeStandards`、`UpdateCustomSetting` RPC 及消息；列表请求移除 `rate_type` 筛选；
   - 核销消息删单头汇率快照字段与行级 `write_off_base_amount`；对冲消息新增 `payable_base_amount`、`exchange_gain_loss`；
   - `make -C server api` 重新生成。
6. **生成物同步**：
   - `go -C server generate`（Ent）；
   - `pnpm run generate:web-client`；
   - `access/manifest.go` 更新"时间标准"字样的权限描述后 `pnpm run generate:permission-keys`。

### Phase 2: 汇率业务内核与全领域统一对接

1. **汇率仓储与用例重构**：
   - `internal/biz/exchange_rate.go`：签名精简为 `ResolveRate(ctx, orgID, currency, targetDate)`；删除 `ExchangeRateTimeStandardSetting` 与 `ExchangeRateCustomSetting` 领域对象、用例与相关错误码；
   - `internal/data/exchange_rate.go`：适配单一 `rate` 存储；`Create`/`Update`/`Disable` 前经 `headquartersOrganizationID` 校验调用组织即总部，否则拒绝（**D 组**）；
   - **删除 biz 层写入重定向总部逻辑**（与总部写防御互斥，保留即保留越权通道）。
2. **各领域汇率调用方对齐**（日期语义见 design 2.2 表）：
   - `order_fee.go:453/471/349`、`finance_bill.go:690/1043`、`finance_cashflow.go:164`、`finance_invoice.go:332` 全部改用 `ResolveRate`；
   - `finance_commission.go:559/678` 提成 CNY 换算改用 `ResolveRate`（生成日）。
3. **B1：核销移除汇率查询**：
   - `finance_verification.go:210` 彻底移除 `Resolve` 调用；
   - 单头 `BaseAmount = ∑ cashflowBase`；`CalculateVerificationAllocationAmounts` 删除 `writeOffBase` 出参与核销汇率入参。
4. **B3：提成分母拉齐账单日**：
   - `data/finance_commission.go:674-689` 总应收/总应付聚合改用关联账单行 `BaseCurrencyAmount`。
5. **汇率导入收敛**：
   - `exchange_rate_import.go` 删除五类中文列映射，模板收敛为单一"折本币汇率"列，批次校验同步。

### Phase 3: 核心业务与安全漏洞闭环 (A 组 P0 + B2 + C1 + E 组)

1. **A 组 (P0)**：
   - **A-1**: `data/sea_document_change.go:390-452`：6 类下游事实及 HBL 箱货分配 `BlocksExecution = true`；补箱货分配查询；清理 `includeAllocations` 死参数；
   - **A-2**: `ExecuteModeChange` 接入 `hasBlockingImpact` 阻断；
   - **A-3**: Master/House Amendment、Master Void、Mode Change 事务内对关联订单（含共享 MBL 成员）执行 `ensureOrderBusinessEditable`；
   - **A-4**: `sea_order_change.go` 船期更新（3394 行锁定集合）与改配流程逐单执行 `ensureOrderBusinessEditable`。
2. **B2：对冲两端本币与汇差**：
   - `biz/finance_netting.go`：计划与批量建账两路径分别累加应收侧（既有 `BaseCurrencyAmount`，语义固化为应收侧）与应付侧，`ExchangeGainLoss = Payable − Receivable`；
   - data 层持久化两新字段，列表/详情读出。
3. **C1：净提成释放费用锁（双落点）**：
   - `biz/order_fee.go` 写入拦截与 `data/settlement.go:35-41` `financeLockedOrderPredicate` 同步改为订单级净额聚合判定（公式见 design 2.3）；台账 `finance_locked` 投影同源更新。
4. **E 组：行业主数据总部共享**：
   - `data/industry_reference.go` 港口/机场/航司/船公司等查询改 `organization_id IN (currentOrg, HQ)`，按业务代码去重、本组织行优先，保留启用过滤。

### Phase 4: 前端精简与展示对齐

1. **汇率管理页**（`web/src/pages/finance/exchange-rates`）：
   - 删除 RateType Tab 与下拉、收付双列；表单收敛为：币种、折本币汇率、生效起始日、失效日、状态；
   - 删除时间标准配置 Tab 与继承策略组件；导入模板与界面同步收敛。
2. **核销详情页**：移除汇率快照展示，单头本币口径改为流水本位币合计。
3. **对冲详情页**：新增应付侧本位币与对冲汇差展示。

### Phase 5: 全量回归测试与门禁验证

1. **服务端**：`go -C server test ./...`、`go -C server vet ./...`；
2. **前端**：`pnpm --dir web biome:lint`、`pnpm --dir web tsc`；
3. **迁移**：开发库执行 `pnpm run migrate:dev`（允许校验和重录），验证存量数据按规则塌缩、冲突场景正确报错；
4. **提交规范**（Conventional Commits，按 Phase 分组）：
   - `feat(exchange-rate): 推平汇率类型与收付双轨为单一折本币基准汇率`
   - `fix(finance): 核销单头改用流水本位币并删除汇率快照`
   - `fix(finance): 提成分母拉齐账单行汇率并对冲单沉淀两端本币与汇差`
   - `fix(finance): 净提成归零自动释放费用财务锁`
   - `fix(sea-document): 修复单证变更财务阻断死代码与业务锁门禁穿透`
   - `fix(masterdata): 行业主数据总部共享并去重`

---

## 3. 延后任务（另立）

- **对冲结清触发提成（C2）**：commission 来源模型（`VerificationID` 硬绑 → 通用来源）、对冲分摊收入实现口径、候选入口与页面；本任务归档后创建独立任务实施。
