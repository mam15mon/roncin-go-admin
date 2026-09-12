# 应收账期与信用额度消费技术设计方案 (design.md)

> 依赖前置：`09-11-casual-partner-loop`（is_casual 字段与契约）已交付；09-10 加固任务（核销 / 对冲分摊口径）已收尾；在途财务 / 伙伴 UI 改动已落地。

## 0. 设计决策（已于 2026-09-13 定稿）

1. **默认账期回填位置**——定稿采用**方案 A：服务端兜底注入**：创建账单请求未显式提供账期时，服务端注入默认值（散客恒 0；正式客户取结算规则 `payment_terms_days`）。默认值注入不属于拦截，覆盖 API 直录路径；
2. **账单编辑场景**——已核实账单状态机：**草稿（Draft）是唯一可编辑账期 / 到期日的状态**（`FinanceBillUsecase.Update` 对非草稿一律拒绝）。定稿：**创建与草稿编辑两条路径都触发**散客预警与超额预警展示（复用同一套预警组装），非草稿状态不可编辑、无需处理。

## 1. 架构目标与分层职责

本任务实现伙伴主档账期带出、自定义设置信用额度管控策略（仅提醒 vs 直接干预拦截）的全链路闭环。
核心原则：**风控充分预警、商业决策不强行锁死、强制只放在钱流出处与管理层明确启用的干预规则上**。

```
[前端界面]
  ├─ 自定义设置 (CustomSettingsPanel): 新增「往来单位信用额度管控策略」卡片
  │    └─ 开关「往来户超信用额度后是否可以选择」(默认 true: 仅提醒; 关闭: 直接干预)
  ├─ 伙伴选择器 (Order/Fee/Bill): 契约携带 credit_exceeded (is_casual 由散客任务先行提供)
  │    └─ 直接干预模式下，超额客户置灰禁用 (disabled: true) 并标注 [已超信用额度]
  ├─ 建账工作台 (BillCreationWorkbench / BillGroupCard):
  │    ├─ 应收账期: 散客默认带出 0，正式客户按主档规则带出，均联动到期日 (可修改)
  │    ├─ 散客账期警示: 散客修改为 > 0 天时即时高亮黄色警告 (不阻断提交)
  │    └─ 信用超额预警: 未核销本币余额 + 本单 > 额度时高亮黄色警告 (不阻断提交)
  └─ 伙伴主档: 结算规则支持 payment_terms_days 维护
       │
[传输层 DTO / Proto]
  ├─ partner.proto: PartnerSettlementRule, PartnerSettlementRuleInput (增加 payment_terms_days)
  ├─ settlement.proto:
  │    ├─ BillBatchPreviewGroup (增加 is_casual, default_payment_terms_days, credit_limit_amount 等)
  │    ├─ FinanceSettlementPartyOption (增加 credit_exceeded)
  │    └─ CreditLimitControlPolicy, Get/UpdateCreditLimitControlPolicy RPC
  └─ error_reason.proto:
       └─ PARTNER_CREDIT_LIMIT_EXCEEDED (超信用额度直接干预拦截)
       │
[领域层 biz]
  ├─ biz/finance_custom_setting.go: CreditLimitControlPolicy 实体、用例与生效判定
  ├─ biz/partner_settlement_rule.go: 实体与校验接入 PaymentTermsDays (0~3650 天)
  ├─ biz/finance_bill.go: PreviewBatch 解析默认账期与未核销应收超额；入账不阻断
  └─ biz/order.go / biz/order_fee.go: 直接干预模式下，校验委托客户与结算单位是否超额
       │
[数据持久层 data / Ent]
  ├─ ent/schema/partner_settlement_rule.go: 新增 payment_terms_days 字段 (int, optional, nillable)
  ├─ ent/schema/finance_custom_setting.go: 新增 credit_limit_selection_allowed (bool, default: true)
  └─ data/finance_bill.go: 实现 GetPartnerUnsettledReceivableBaseAmount(orgID, partnerID)
```

---

## 2. 详细设计与改动清单

### 2.1 数据库 Schema 与迁移

1. **`PartnerSettlementRule` 实体 (`server/internal/data/ent/schema/partner_settlement_rule.go`)**:
   ```go
   field.Int("payment_terms_days").Optional().Nillable().Min(0).Max(3650).Comment("默认信用账期天数")
   ```
2. **`FinanceCustomSetting` 实体 (`server/internal/data/ent/schema/finance_custom_setting.go`)**:
   ```go
   field.Bool("credit_limit_selection_allowed").Default(true).Comment("超信用额度后是否仍允许选择往来单位（true: 仅提醒; false: 直接干预拦截）")
   ```
3. **Atlas 迁移生成**:
   - 运行 `go -C server generate` 生成 Ent 代码；
   - 生成对应 SQL 迁移脚本；存量数据 `payment_terms_days` 默认 `NULL`，`credit_limit_selection_allowed` 默认 `true`，平滑兼容。

### 2.2 Protobuf 契约与 OpenAPI 更新

1. **`server/api/partner/v1/partner.proto`**:
   - `message PartnerSettlementRule`: 新增 `optional int32 payment_terms_days = 14;`
   - `message PartnerSettlementRuleInput`: 新增 `optional int32 payment_terms_days = 10;`
   - 字段编号以实际 proto 当前空闲位为准，实施时核对。
2. **`server/api/finance/v1/settlement.proto`**:
   - `message BillBatchPreviewGroup`:
     ```protobuf
     bool is_casual = 20;
     optional int32 default_payment_terms_days = 21;
     optional string credit_limit_amount = 22;
     optional string credit_currency = 23;
     optional string current_unsettled_amount = 24;
     bool is_credit_exceeded = 25;
     ```
     （`is_casual` 与 `default_payment_terms_days` 已由散客任务交付，本任务仅新增信用相关四个字段）
   - `message FinanceSettlementPartyOption`:
     ```protobuf
     bool credit_exceeded = 5;
     ```
   - 新增信用额度管控策略消息与 RPC：
     ```protobuf
     message CreditLimitControlPolicy {
       string organization_id = 1;
       bool allow_selection_when_credit_exceeded = 2;
       uint64 version = 3;
       string updated_at = 4;
       string updated_by = 5;
     }
     message GetCreditLimitControlPolicyRequest {
       optional string organization_id = 1;
     }
     message GetCreditLimitControlPolicyResponse {
       bool success = 1;
       int32 code = 2;
       string message = 3;
       CreditLimitControlPolicy data = 4;
       string trace_id = 5;
       bool can_update = 6;
     }
     message UpdateCreditLimitControlPolicyRequest {
       optional string organization_id = 1;
       bool allow_selection_when_credit_exceeded = 2;
       uint64 expected_version = 3;
     }
     message UpdateCreditLimitControlPolicyResponse {
       bool success = 1;
       int32 code = 2;
       string message = 3;
       CreditLimitControlPolicy data = 4;
       string trace_id = 5;
     }
     ```
     在 `SettlementService` 中添加：
     ```protobuf
     rpc GetCreditLimitControlPolicy(GetCreditLimitControlPolicyRequest) returns (GetCreditLimitControlPolicyResponse);
     rpc UpdateCreditLimitControlPolicy(UpdateCreditLimitControlPolicyRequest) returns (UpdateCreditLimitControlPolicyResponse);
     ```
3. **`server/api/finance/v1/error_reason.proto`**:
   - 新增错误码 `PARTNER_CREDIT_LIMIT_EXCEEDED`，中文提示：`"该客户应收未核销总额已超出信用额度，系统已限制选择"`。
4. **生成指令**:
   - `make -C server api`
   - `pnpm run generate:web-client`

### 2.3 后端业务逻辑实现

#### 1. 结算规则默认账期 (`biz/partner_settlement_rule.go`)
- `PartnerSettlementRule` 增加 `PaymentTermsDays *int`；
- `normalizePartnerSettlementRule` 增加对 `PaymentTermsDays` 的取值范围校验 `[0, 3650]`。

#### 2. 自定义设置：信用额度管控策略 (`biz/finance_custom_setting.go`)
- 领域模型增加 `CreditLimitControlPolicy`:
  ```go
  type CreditLimitControlPolicy struct {
      OrganizationID uuid.UUID
      AllowSelectionWhenCreditExceeded bool // true: 仅提醒; false: 直接干预
      Version uint64
      UpdatedAt *time.Time
      UpdatedBy *uuid.UUID
  }
  ```
- 实现 `GetCreditLimitControlPolicy` 与 `UpdateCreditLimitControlPolicy`（并发乐观锁保护）；
- 提供便捷判定方法 `IsCreditLimitInterventionActive(ctx, orgID) (bool, error)`。

#### 3. 客户超额判定与聚合 (`biz/finance_bill.go` & `data/finance_bill.go`)
- 数据层实现：
  `GetPartnerUnsettledReceivableBaseAmount(ctx context.Context, orgID, partnerID uuid.UUID) (decimal.Decimal, error)`
  - **必须复用 09-10 修复后的未结清 / 核销分摊口径（`billUnsettledPredicate()` 等），不得另写一套余额算法**；
  - 过滤当前组织、当前往来户的所有已确认（`CONFIRMED`）应收账单；
  - 汇总其折本币未核销金额（`unverified_base_amount`，负值钳零）。
- 超额判定规则：
  - 读取客户角色的激活规则；
  - 若 `rule.CreditLimitMinor != nil && *rule.CreditLimitMinor > 0`:
    - `creditLimitBase := decimal.NewFromInt(*rule.CreditLimitMinor).Div(100)`
    - 若 `unsettledBaseAmount.GreaterThan(creditLimitBase)`，判定为超额（`credit_exceeded = true`）。
  - 未设额度或额度为 0 的伙伴，判定为不超额。

#### 4. 订单与费用写入门禁拦截 (`biz/order.go`, `biz/order_fee.go`)
- 当组织处于「直接干预模式」（`AllowSelectionWhenCreditExceeded == false`）时：
  - 创建或更新订单若更换/设定了委托客户（`CustomerID`），校验该客户是否超额；
  - 创建应收费用若指定了结算单位（`SettlementPartyID`），校验该客户是否超额；
  - 若超额，直接返回 `ErrPartnerCreditLimitExceeded`，阻断入库。
- **并发口径**：校验为尽力而为（查询时点余额），不做串行化加锁；并发窗口漏拦以事后预警 / 报表校正。

#### 5. 建账工作台账期带出与信用预警 (`biz/finance_bill.go`)
- `PreviewBatch` 构建叶子 `group` 时：
  - 散客默认带出 `0` 天账期（已由散客任务交付，保持不变）；
  - 正式客户带出主档规则的 `payment_terms_days`；
  - 计算未核销总额与信用额度比对，若超额标记 `is_credit_exceeded = true`；
- 建账入账接口（`CreateBillBatch` / `CreateBill`）**不设阻断拦截**，支持财务核对后正常入账。

### 2.4 前端界面与交互实现

#### 1. 自定义设置面板 (`CustomSettingsPanel.tsx`)
- 新增卡片：**往来单位信用额度管控策略**；
- 展示说明文案：
  - 标题：**往来户超信用额度后是否可以选择**
  - 描述：开启后（默认），当往来单位应收未核销金额（本币）超出信用额度时，系统仅在选择与建账时提供黄色预警提醒，仍允许选择与录单；关闭后，超额往来单位将被直接干预拦截，在下拉选择器中置灰禁用且无法选择该往来户。
- Switch 绑定 `allowSelectionWhenCreditExceeded`，权限受财务设置更新权限控制。

#### 2. 下拉选择器 (`options.ts`, `PartnerQuickAddSelect.tsx` 等)
- `SelectOption` 扩展 `creditExceeded?: boolean`；
- 当组织开启直接干预模式且 `creditExceeded` 时：
  - 候选项设置 `disabled: true`；
  - 候选项右侧渲染 `<Tag color="error">已超信用额度</Tag>`；
- 选中回显 label 保持纯净文本。

#### 3. 建账工作台 (`BillCreationWorkbench.tsx`, `BillGroupCard.tsx`)
- 自动回填默认账期天数并联动到期日；
- 散客账期修改为 > 0 天时，即时高亮黄色风险预警（"建议现结，请注意资金回款风险"）；
- 信用额度超限时，即时高亮黄色超额预警（展示未核销本币金额 vs 信用额度）；
- 预警均为软提示，不禁用提交按钮。

#### 4. 伙伴主档结算规则
- 结算规则面板支持维护「默认信用账期天数」（`paymentTermsDays`）与信用额度。

---

## 3. 验证与门禁计划

1. **后端单元测试**:
   - `finance_custom_setting_test.go`: 验证信用额度管控策略的默认值（开启）、切换与权限；
   - `finance_bill_test.go`: 验证本币未核销余额聚合与超额判定（复用 09-10 口径的一致性）；建账预览预警；散客默认账期带出；
   - `order_test.go` & `order_fee_test.go`: 验证直接干预模式下超额客户被阻断、仅提醒模式下允许通过；
   - `partner_settlement_rule_test.go`: `payment_terms_days` 范围校验。
2. **前后端接口与类型检查**:
   - 运行 `pnpm --dir web tsc` 确保前端类型无报错；
   - 运行 Biome 检查相关前端文件。
3. **前端交互定向测试**:
   - `CustomSettingsPanel.test.tsx`: 测试策略卡片渲染与 Switch 切换；
   - `PartnerQuickAddSelect.test.tsx`: 测试超额置灰禁用；
   - `BillCreationWorkbench.test.tsx`: 测试账期带出与超额预警。
