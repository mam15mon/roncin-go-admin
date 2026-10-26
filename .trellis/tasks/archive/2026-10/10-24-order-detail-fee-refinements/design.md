# 技术设计：订单详情与费用录入规则调整

对应 PRD 六个需求点。整体原则：契约先行（proto → 生成物 → 前后端）、正式 SQL 迁移、无历史兼容分支。

## 1. 内部人员业务/客服只读（客户带入）

**现状**：`SeaBasicInfoSection.tsx` 选客户时已按客户档案 `partner_assignments`（SALES / CUSTOMER_SERVICE 等角色）带入订单人员；`SeaPersonnelSection.tsx` 中业务/客服仍为可编辑下拉，且新建时三岗必填。海/空运模板共用该区块（`sea-template.tsx` / `air-template.tsx`）。

**改动**：

- 前端 `SeaPersonnelSection.tsx`：
  - 业务/客服两个字段 `disabled` 恒置（新建与详情一致）；去掉这两岗的 `commissionStaffRules` 必填（操作人员保留必填）。
  - 只读展示用现有 `SearchableSelect disabled`，选项仍来自 `personnelOptions` 以正确显示姓名。
- 服务端防绕过：订单创建/更新写 `PersonnelAssignments` 前，读取客户（订单 CustomerID）的 partner assignments，用 SALES / CUSTOMER_SERVICE 角色覆盖（或删除）请求中的对应角色；客户缺岗时这两个角色置空。落在 `internal/data` 订单写入路径（`order_write.go` / `order_write_draft.go` 的 personnel 保存处集中处理），`biz` 层不引入 partner 依赖。

**边界**：不改动 partner 维护页与带入的其它岗位（操作/商务/联系人/单证仍带入但可手改）。

## 2. 订单状态跳转补齐 + 一键完结

**现状**：`biz/order_transition.go` 的 `allowedSEFlowTransitions` 只允许相邻推进；结案要求 `FlowStatus == DOCUMENT_RELEASED` 或已退关；每个流转写一条 `OrderLifecycleEvent`，`order_status_records.go` 按 dimension+toStatus 取最新一条投影为时间轴。

**改动**：

- `biz/order_transition.go`：
  - `allowedSEFlowTransitions` 语义改为「所有严格大于当前状态的主流程状态」（DRAFT 场景保持：新订单首节点为已订舱，仍属"大于"）。`AllowedTargetFlowStatuses()` 不变地消费它，前端时间轴自动把所有后续节点渲染为可点。
  - `validSEFlowTransition(from, to)` 改为枚举序比较（`to > from` 且 to 为合法主流程状态）。
- `data/order_write_lifecycle.go` `TransitionStatus`：
  - 校验不变（版本、终止 ACTIVE、结案 OPEN、业务锁）；`validateAirFlowData` 仅对 target 执行，但其中的 `ValidAirFlowTransition(from, target)` 需放宽为序比较（`biz` 侧同步调整，中间节点不再校验空运数据）。
  - 写入时为 `from+1 … target` 的**每个**状态各写一条 FLOW 事件（`changed_at = now`、from/to 相邻衔接），`flow_status` 置 target，版本 +1。一次事务完成。
- 一键完结：`biz TransitionClosure` 与 `data TransitionClosure` 中，`flowFinished` 不再阻断；结案事务内先执行与 TransitionStatus 相同的「补齐到 DOCUMENT_RELEASED」事件写入（同一 `occurredAt`），再走既有结案更新。readyness 中 `HasActiveException / HasUnbilledOrderFees / HasPendingFeeSupplement` 保留。已退关订单（TERMINATED）完结行为不变（不补主流程）。
- 前端 `OrderStatusSection` 无需结构改动（`allowedTargetFlowStatuses` 已驱动）；流转确认弹窗文案补一句「跳转将自动完成中间节点」。

## 3. 费用大类取消

**改动**：

- Ent schema：`fee_setting.go` 删 `charge_category_id` 字段、`charge_category` edge、`(organization_id, charge_category_id, abnormal_case_id)` 索引；`fee_setting_template.go` 删 `charge_category_id`；`master_data_item.go` kind 枚举删 `charge_category`、删 `charge_category_fee_settings` edge。
- 迁移 SQL（新文件 `server/migrations/20261024xxxxxx_drop_fee_charge_category.sql`）：
  1. 删 `master_data_items` 中 `kind = 'charge_category'` 行；
  2. 重建 kind 枚举类型（PostgreSQL 不支持 REMOVE VALUE：新建枚举 → `ALTER COLUMN ... TYPE ... USING` → drop 旧类型）；
  3. `ALTER TABLE fee_settings DROP COLUMN charge_category_id`、`fee_setting_templates DROP COLUMN charge_category_id`；删除受影响索引。
- proto `server/api/finance/v1/fee_catalog.proto`：删 `charge_category_id` / `charge_category_name` 字段（reserved 占位），费用科目列表/下拉响应同步。
- `biz/fee_catalog.go`、`service/fee_catalog.go`、`data/fee_catalog.go`、`biz/masterdata.go`（删 `MasterDataKindChargeCategory`）及种子/同步脚本中费用大类清理。
- 前端 `pages/finance/fee-settings/**` 删大类选择与筛选；`services/roncin` 生成物随之更新；主数据页若展示费用大类分类则移除入口。

## 4. 发生日期 → 录入日期（不可变）

**改动**：

- Ent schema `order_fee.go`：删 `expense_date` 字段；排序索引 `(order_id, direction, created_at)` 已存在，无需新增。
- 迁移 SQL（`20261024xxxxxx_order_fee_expense_date_to_created_at.sql`）：`ALTER TABLE order_fees DROP COLUMN expense_date`。
- proto `order_fee.proto`：
  - `OrderFee` 消息删 `expense_date`（reserved 14）；
  - `AddFeeRequest` / `UpdateFeeRequest` 删 `expense_date`（reserved）；
  - `BulkUpdateOrderFeeRequest` 删 `expense_date` 目标（批量修改仅剩结算单位维度）；
  - `ResolveFeeExchangeRateRequest` 删 `expense_date`（见 §5）。
- `settlement.proto` 费用台账：`expense_date_from/to` 过滤改为 `created_at_from/to`（按创建时间范围）；展示消息中 `expense_date` 删除（前端展示 `created_at` 格式化日期）。
- `biz/order_fee.go`：
  - 创建/更新输入不再有 ExpenseDate；`normalizeFee` 校验删除；`BulkUpdate` 只支持 settlementPartyID；
  - `expenseDateDay` / `isValidExpenseDate` 退役（汇率日期改由 ETD 提供，见 §5）；
  - 排序 `ByExpenseDate` → `ByCreatedAt`（`data/finance_bill.go` 台账查询等）。
- 前端：`FeeFormModal.tsx` 删发生日期选择器；`feeBaseColumns.tsx` / 台账列 / 导出 / 补录与更正弹窗中「发生日期」列改「录入日期」绑定 `created_at`（`formatDate` minute 精度）；筛选组件改按录入日期。

## 5. 汇率按订单 ETD 解析

**现状**：`ExchangeRateUsecase.ResolveRate(ctx, org, direction, currency, targetDate)` 按周匹配（direction → ar/ap 双轨）；`OrderFeeUsecase.ResolveExchangeRate` 与费用创建 `applyResolvedRate` 均传 `expenseDateDay(fee.ExpenseDate)`。

**改动**：

- `biz/order_fee.go`：
  - `OrderFeeUsecase` 增加私有 `orderETDDay(ctx, orgID, orderID) (string, error)`：读订单 `ETD`（orders.etd，RFC3339），取日期部分；为空返回新业务错误 `ErrOrderEtdMissing`（中文提示「请先填写订单 ETD 后再解析汇率」，HTTP 400/409 语义用 InvalidArgument 家族）。
  - `ResolveExchangeRate` 签名去掉 `expenseDate`，内部用 ETD 日期调 `exchangeRate.ResolveRate`。
  - 费用创建/更新（含补录审批生成、更正生成路径）同样以 ETD 日期解析；`fee.ExchangeRateDate = ETD 日期`。
- proto：`ResolveFeeExchangeRateRequest` 删 `expense_date`（reserved）。
- 前端 `use-fee-exchange-preview.ts`：解析触发不再依赖 `expenseDate`，币种变化即触发；展示 `exchangeRateDate` 为「按 ETD 2026-10-19 周汇率」类文案（沿用现有卡片）。

## 6. 财务调整汇率独立权限 + 同币种条件

**现状**：手工覆盖入口由 `canOverrideFeeExchangeRate` 控制，目前是 `financeExchangeRateUpdate` 的别名；后端接受 `exchange_rate_override` 的校验点待查（费用 add/update service 层）。

**改动**：

- `internal/access/manifest.go`：新增
  `FinanceOrderFeeExchangeRateOverride = "system.finance.order_fee.exchange_rate.override"`，名称「调整订单费用汇率」，Group「费用管理 · 汇率」，Requires `FinanceExchangeRateRead`；`FinanceExchangeRateUpdate` 描述去掉「在业务单据中手工输入汇率」。
- `migrate` 权限同步（`cmd/migrate` 自动挂 administrator）+ `pnpm run generate:permission-keys`。
- 后端校验（`biz/order_fee.go`，Add/Update/补录/更正所有接受 `ExchangeRateOverride` 的入口）：
  1. principal 持新权限（DataScopeOrganization）；
  2. 订单应收费用币种 distinct 集合 == 应付费用 distinct 集合，且两者非空（在事务内按当前订单费用查询，包含本次新增前费用；新建首笔费用时以「已有费用 + 本笔」合并计算）。
  不满足返回新业务错误 `ErrFeeExchangeRateOverrideForbidden`（中文提示「应收应付币种不一致，不能手工调整汇率」/「无调整订单费用汇率权限」）。
- 前端 `access.ts`：`canOverrideFeeExchangeRate` 改绑 `permissions.financeOrderFeeExchangeRateOverride`；手工汇率 UI 入口不变。

## 生成与验证顺序

1. schema + `server/migrations` 迁移 SQL；
2. proto 修改 → `make -C server api`（同时生成 openapi）；
3. `pnpm run generate:web-client`；
4. manifest → `pnpm run generate:permission-keys`；
5. 后端 biz/data/service 实现与测试；
6. 前端实现与定向测试；
7. `pnpm run check:fast` 全量门禁。

## 风险与回滚

- 迁移删列不可逆，依赖 Git 回滚 + 迁移前开发库由 `sync:dev` 可重建。
- 状态跳转放宽不改变事件表结构，仅新增写入模式；一键完结保留全部业务前置检查。
- 权限收紧后原持「编辑汇率」角色不再能手工覆盖费用汇率，需管理员为新权限授角（migrate 只给 administrator 补挂）。
