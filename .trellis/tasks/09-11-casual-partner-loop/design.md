# 散客闭环：单次合作往来单位技术设计方案 (design.md)

> 范围说明：应收账期带出、信用额度双模管控、建账工作台预警已移交 `09-11-partner-terms-credit` 任务；本设计仅覆盖散客标识、出款账户硬约束与列表 / 选择器区分。

## 1. 架构目标与分层职责

本任务实现散客标识与散客供应商出款账户硬约束的全链路闭环。
核心原则：**风控充分预警、商业决策不强行锁死、强制只放在钱流出处**。

```
[前端界面]
  ├─ 快捷建档 (PartnerQuickAddSelect / QuickAddPartnerModal): 勾选「单次合作」(默认 true)
  ├─ 伙伴选择器 (Order/Fee/Bill): 契约携带 is_casual，散客动态渲染 [散客] Tag (label 保持纯净)
  ├─ 建账工作台 (BillGroupCard): 散客客户账期默认回填 0 并联动到期日；改大时黄色预警 (不阻断)
  ├─ 现金流付款 (cashflows/index.tsx): 散客供应商 Payable 流水强制要求填写对方账户
  └─ 伙伴主档/列表: 支持 is_casual 维护与筛选
       │
[传输层 DTO / Proto]
  ├─ partner.proto: Partner, CreatePartner, UpdatePartner, ListPartners (增加 is_casual)
  ├─ settlement.proto: FinanceSettlementPartyOption (增加 is_casual)
  │                   BillBatchPreviewGroup (增加 is_casual, default_payment_terms_days)
  └─ finance error_reason.proto:
       └─ FINANCE_CASHFLOW_CASUAL_SUPPLIER_ACCOUNT_REQUIRED (散客出款无账户)
       │
[领域层 biz]
  ├─ biz/partner.go: 实体与用例接入 IsCasual；快捷创建与导入口径对齐
  ├─ biz/finance_cashflow.go: Create 校验若为散客且 Payable 则强制 CounterpartyAccount 非空
  └─ biz/finance_bill.go: PreviewBatch 对散客客户账单组硬编码默认账期 0 (不读结算规则)
       │
[数据持久层 data / Ent]
  ├─ ent/schema/partner.go: 新增 is_casual 字段 (bool, default: false)
  └─ data/finance_cashflow.go: ResolveParty 返回 is_casual 状态供 biz 判定
```

---

## 2. 详细设计与改动清单

### 2.1 数据库 Schema 与迁移

1. **`Partner` 实体 (`server/internal/data/ent/schema/partner.go`)**:
   ```go
   field.Bool("is_casual").Default(false).Comment("是否单次合作（散客）")
   ```
2. **Atlas 迁移生成**:
   - 运行 `go -C server generate` 生成 Ent 代码；
   - 生成对应的 SQL 迁移脚本（`server/migrations/*_add_partner_casual.sql`）；
   - 存量历史数据 `is_casual` 默认为 `false`，平滑兼容。

### 2.2 Protobuf 契约与 OpenAPI 更新

1. **`server/api/partner/v1/partner.proto`**:
   - `message Partner`: 新增 `bool is_casual = 15;`
   - `message CreatePartnerRequest`: 新增 `optional bool is_casual = 10;`
   - `message UpdatePartnerRequest`: 新增 `optional bool is_casual = 11;`
   - `message ListPartnersRequest`: 新增 `optional bool is_casual = 6;`
   - 字段编号以实际 proto 当前空闲位为准，实施时核对。
2. **`server/api/finance/v1/settlement.proto`**:
   - `message FinanceSettlementPartyOption`: 新增 `bool is_casual = 4;`
   - `message BillBatchPreviewGroup`: 新增 `bool is_casual` 与 `optional int32 default_payment_terms_days`（字段编号以当前空闲位为准；散客场景 `default_payment_terms_days` 恒为 `0`）
3. **`server/api/finance/v1/error_reason.proto`**:
   - 新增错误码 `FINANCE_CASHFLOW_CASUAL_SUPPLIER_ACCOUNT_REQUIRED`，中文提示：`"向散客供应商出款时，对方收款账户为必填项"`。
4. **生成指令**:
   - `make -C server api`
   - `pnpm run generate:web-client`

### 2.3 后端业务逻辑实现

#### 1. 伙伴散客标识 (`biz/partner.go`)
- `Partner` 实体增加 `IsCasual bool`；
- `CreatePartnerInput` 与 `UpdatePartnerInput` 增加 `IsCasual *bool`；
- `ImportPartners` 批量导入逻辑中，显式设定 `is_casual = false`；
- 列表查询谓词支持按 `is_casual` 过滤。

#### 2. 散客供应商出款账户必填 (`biz/finance_cashflow.go` & `data/finance_cashflow.go`)
- `FinanceCashflowRepo` 的 `ResolveParty` 接口扩展返回 `isCasual`；
- `FinanceCashflowUsecase.Create`:
  若 `in.Direction == OrderFeePayable` 且 `isCasual == true`，强制要求 `CounterpartyAccount` 必须有非空文本，缺失则返回 `ErrFinanceCashflowCasualSupplierAccountRequired`。

#### 3. 散客客户账期底线 (`biz/finance_bill.go`)
- `PreviewBatch` 构建应收账单组时，若对方伙伴 `is_casual = true`：返回 `is_casual = true` 与 `default_payment_terms_days = 0`（症状级硬编码，**不读结算规则**；正式客户不返回默认值，维持手工录入现状）；
- 建账入账接口不设任何账期阻断拦截。

### 2.4 前端界面与交互实现

#### 1. 下拉选择器 (`options.ts`, `PartnerQuickAddSelect.tsx` 等)
- `SelectOption` 扩展 `isCasual?: boolean`；
- 散客候选项右侧渲染 `<Tag color="orange">散客</Tag>`；
- 选中回显 label 保持纯净文本。

#### 2. 快捷建档弹窗 (`PartnerQuickAddSelect.tsx`, `QuickAddPartnerModal.tsx`)
- 增加复选框 `[x] 单次合作往来单位（散客）`（默认勾选 `true`，可取消）；
- 提交请求透传 `isCasual`；
- `QuickAddPartnerModal` 客商类型由 `mode="multiple"` 改为单选，新增 `defaultRole` prop：由调用方（`order-fee-panel.tsx`、`fees.tsx`）按当前费用表单方向推导（应收→客户、应付→供应商；方向未选默认客户），弹窗内可切换为另一角色；与订单侧「上下文决定角色」的快捷建档契约对齐。

#### 3. 现金流表单 (`cashflows/index.tsx`)
- 往来结算单位下拉选项带出 `isCasual`；
- 散客付款方向动态标星必填，前置表单拦截空账户。

#### 4. 建账工作台 (`BillGroupCard.tsx`)
- 预览组携带 `is_casual` 时，账期天数默认回填 `default_payment_terms_days`（散客恒 0）并联动到期日 = 账单日；
- 散客账期修改为大于 0 天时，即时高亮黄色预警（"建议现结，请注意资金回款风险"），不禁用提交按钮；
- 正式客户组无默认回填，行为与现状一致。

#### 5. 伙伴主档与列表
- 列表支持合作类型（散客/正式）筛选，行内展示散客 Tag；
- 详情页支持修改散客 Switch。

---

## 3. 验证与门禁计划

1. **后端单元测试**:
   - `finance_cashflow_test.go`: 验证散客供应商出款无账户拒绝、正式供应商与收款方向不受影响；
   - `finance_bill_test.go`: 验证 `PreviewBatch` 对散客账单组返回默认账期 0、正式客户组无默认值；
   - `partner_test.go`: 字段存取、导入默认正式、列表过滤。
2. **前后端接口与类型检查**:
   - 运行 `pnpm --dir web tsc` 确保前端类型无报错；
   - 运行 Biome 检查相关前端文件。
3. **前端交互定向测试**:
   - `PartnerQuickAddSelect.test.tsx`: 测试快捷新增勾选与散客 Tag 渲染；
   - `BillGroupCard` 相关测试：散客默认 0 回填、改大预警、正式客户行为不变；
   - 现金流表单散客付款必填校验。

## 4. 启动前置与风险

- **前置**：在途伙伴 / 财务 UI 改动已合入 main（PR #1），前置已满足。
- 本任务触及 `PreviewBatch`（默认账期注入），但只读伙伴的 `is_casual`，**不依赖核销 / 对冲分摊口径，与 09-10 加固任务无依赖**；余额聚合类逻辑全部留在后续任务。
