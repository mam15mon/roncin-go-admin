# 设计：客户收款约定收口

## 1. 数据模型

### 1.1 Ent Schema：`server/internal/data/ent/schema/partner_settlement_rule.go`

收口后的字段：

```go
field.UUID("partner_role_id", uuid.Nil),
field.Int64("credit_limit_minor").Optional().Nillable().Min(0),
field.String("credit_currency").Optional().Nillable().MinLen(3).MaxLen(3),
// 收款期限：NULL 未约定；0 现结；N 为建账后 N 天。
field.Int("payment_terms_days").Optional().Nillable().Min(0).Max(3650).Comment("收款期限天数"),
```

索引：

```go
index.Fields("partner_role_id").Unique().StorageKey("partner_settlement_rule_role_key"),
```

删除：`statement_mode`、`settlement_method`、`settlement_day`、`settlement_cycle_days`、
`settlement_base`、`settlement_currency`、`is_active`，以及索引 `partner_settlement_rule_key`、
`partnersettlementrule_partner_role_id_is_active`。

表名和实体名保持 `partner_settlement_rules` / `PartnerSettlementRule` 不变。
改名会牵动权限码、路由、审计 action 和大量文件，收益不大。界面用语统一改成「收款约定」。

修改后执行 `go -C server generate ./...` 重新生成 Ent 代码。

### 1.2 正式迁移：`server/migrations/<链尾时间戳>_partner_payment_terms_single_rule.sql`

编号必须晚于当前链尾（目前是 `20261117100000_retire_order_shipment_mode.sql`）。
按 `server/migrations/README.md` 的要求，不使用 `IF EXISTS`。顺序如下：

```sql
-- 0. 额度币种守卫：启用的客户规则中，正额度必须带公司生效本币（沿 parent_id 向上找第一个非空 base_currency）。
--    有异常就整体失败，不做任何修改。口径与 data/finance_bill_credit.go 的 partnerCreditSummaries 一致。
DO $$
DECLARE bad_count bigint;
BEGIN
  WITH RECURSIVE chain AS (
    SELECT o.id AS org_id, o.parent_id, o.base_currency, 0 AS depth FROM "organizations" o
    UNION ALL
    SELECT c.org_id, p.parent_id, p.base_currency, c.depth + 1
    FROM chain c JOIN "organizations" p ON p.id = c.parent_id
    WHERE c.base_currency IS NULL
  ), org_base AS (
    SELECT DISTINCT ON (org_id) org_id, base_currency
    FROM chain ORDER BY org_id, (base_currency IS NULL), depth
  )
  SELECT count(*) INTO bad_count
  FROM "partner_settlement_rules" r
  JOIN "partner_roles" pr ON pr.id = r.partner_role_id
  JOIN "partners" p ON p.id = pr.partner_id
  LEFT JOIN org_base ob ON ob.org_id = p.organization_id
  WHERE pr.role_type = 'customer' AND r.is_active AND r.credit_limit_minor > 0
    AND (ob.base_currency IS NULL OR r.credit_currency IS NULL OR r.credit_currency <> ob.base_currency);
  IF bad_count > 0 THEN
    RAISE EXCEPTION '收款约定迁移中止：% 条启用的客户结算规则的信用额度缺少币种或不是公司本币，请先在客户档案中修正额度后再迁移', bad_count;
  END IF;
END $$;

-- 1. 非客户角色不再有收款约定
DELETE FROM "partner_settlement_rules" r
USING "partner_roles" pr
WHERE r."partner_role_id" = pr."id" AND pr."role_type" <> 'customer';

-- 2. 停用规则不再生效
DELETE FROM "partner_settlement_rules" WHERE "is_active" = false;

-- 3. 同一客户角色有多条启用规则（此时已只剩启用的客户规则）：保留 updated_at DESC, id DESC 排第一的行。
--    - payment_terms_days = 组内非空值的 MAX；全空则为 NULL（MAX 天然如此）。
--    - 额度：只在组内存在 credit_limit_minor > 0 时改写，取最大正额度那一行的 (credit_limit_minor, credit_currency)
--      （同额时按 updated_at DESC, id DESC 取第一行）；组内没有正额度时，保留行的额度与币种保持原值。
--      额度 0 不参与取最大值，这与 partnerCreditSummaries 只认 > 0 的口径一致。
--    写法：用 CTE + 窗口函数算出每组 keep_id、merged_terms、best_credit_row，
--    先 UPDATE 保留行（只更新 count(*) > 1 的组），再 DELETE 同组其他行。

-- 4. 删除旧索引与废弃列
DROP INDEX "partner_settlement_rule_key";
DROP INDEX "partnersettlementrule_partner_role_id_is_active";
ALTER TABLE "partner_settlement_rules"
  DROP COLUMN "statement_mode", DROP COLUMN "settlement_method", DROP COLUMN "settlement_day",
  DROP COLUMN "settlement_cycle_days", DROP COLUMN "settlement_base",
  DROP COLUMN "settlement_currency", DROP COLUMN "is_active";

-- 5. 每个角色最多一条
CREATE UNIQUE INDEX "partner_settlement_rule_role_key" ON "partner_settlement_rules" ("partner_role_id");
```

说明：
- 第 0 步守卫的对象和现在建账时的报错对象完全一致：启用的客户规则，并且额度大于 0。
  停用规则和非客户规则即使币种异常，也不会被现有业务读取，所以不拦截，直接在第 1、2 步删除。
  守卫保证第 3 步取「最大额度」时比较的都是同一种本币，不会把异币额度合并进去，
  也不会把本该报错的记录在合并时悄悄删掉。
- 守卫的 SQL 与 `research/preflight-inventory.sql` 的 E 段口径相同。两边要一起维护：只读盘点先把异常列给用户看，守卫在迁移时兜底。
- 第 3 步的「取最大值」只是为了让迁移后客户的实际约定和迁移前建账时的实际取值一致，
  不是对旧行为的长期兼容。迁移之后，系统里不会再有多条规则的概念。
- 额度为空且账期为空的行可以保留，它表示「未约定」，不需要专门清理。
- 没有其他表外键引用 `partner_settlement_rules`（已检索确认），可以直接删除行。
- 这些列上没有 CHECK 约束（已检索 `20260820000000_baseline.sql`、`20260822023000_*`、
  `20260913090000_*`），不需要同步删除约束。

## 2. 契约（Proto）：`server/api/partner/v1/partner.proto`

```proto
message PartnerSettlementRule {
  string id = 1;
  string partner_role_id = 2;
  reserved 3, 4, 5, 6, 7, 8, 9;
  reserved "statement_mode", "settlement_method", "settlement_day", "settlement_cycle_days",
           "settlement_base", "settlement_currency", "is_active";
  string created_at = 10;
  string updated_at = 11;
  optional int64 credit_limit_minor = 12;
  optional string credit_currency = 13;
  optional int32 payment_terms_days = 14;
}

message PartnerSettlementRuleInput {
  reserved 1, 2, 3, 4, 5, 6, 7;
  reserved "statement_mode", "settlement_method", "settlement_day", "settlement_cycle_days",
           "settlement_base", "settlement_currency", "is_active";
  optional int64 credit_limit_minor = 8;
  optional string credit_currency = 9;
  optional int32 payment_terms_days = 10;
}
```

- 删除 `PartnerStatementMode`、`PartnerSettlementMethod`、`PartnerSettlementBase` 三个枚举。
  删之前先 `grep -rn` 确认 proto 里没有其他引用。
- 字段号用 `reserved` 保留。项目里已有同样的写法（`CreateBillBatchGroupInput` 的 `reserved "bill_date"`）。
  这不是兼容层，只是防止以后误用旧编号。
- RPC 不变：`ListPartnerSettlementRules` 仍返回 `repeated`，但现在最多一条；
  `Create`、`Update` 保持不变，权限码不变。这样可以少改权限、路由和前端调用。
- 生成：`make -C server api` → `pnpm run generate:web-client`。
  `web/src/enums.generated.ts` 会随之删掉三个枚举；如果它有独立的生成命令，以 `web/CLAUDE.md` 和 `package.json` 为准。

## 3. 服务端分层

### Biz：`server/internal/biz/partner_settlement_rule.go`
- `PartnerSettlementRule` 结构体只保留 `ID`、`PartnerRoleID`、`CreditLimitMinor`、`CreditCurrency`、`PaymentTermsDays`。
- 删除 `PartnerStatementMode`、`PartnerSettlementMethod`、`PartnerSettlementBase` 及其 `Valid()`。
- `normalizePartnerSettlementRule` 只校验：账期 0–3650、额度与币种必须成对、币种长度为 3。
- **只允许客户角色：** `PartnerSettlementRuleUsecase.Create/Update` 在 `roleType != PartnerRoleCustomer` 时返回
  `ErrPartnerSettlementRuleInvalidArgument`。主档聚合入口（`biz/partner.go` 约 637 行，`role.SettlementRule != nil`
  的分支）对非客户角色同样拒绝，不能静默丢弃。

### Data
- `data/partner_settlement_rule.go`：`create/updatePartnerSettlementRule`、`partnerSettlementRuleToBiz` 删除对应字段；
  删除 `validateSettlementCurrencies`（它只校验结算币种）；Create 的唯一约束映射改为新索引名
  `partner_settlement_rule_role_key` → `ErrPartnerSettlementRuleExists`。
- `data/partner.go` 的 `savePartnerRoleSettlementRules`：去掉 `validateSettlementCurrencies` 调用和
  「`len(existingRules) > 1` 拒绝」的分支（唯一索引已保证不会出现多条）；
  约束名映射同上。
- `data/finance_bill_credit.go` 的 `partnerCreditSummaries`：去掉 `IsActiveEQ(true)` 条件和「取最大值」的循环，
  每个客户直接取唯一一条的账期和额度。同步更新函数注释。

### Service
- `service/partner_settlement_rule.go`、`service/partner_convert.go`：删除枚举转换和废弃字段映射。

### 权限与 Manifest
- 权限码不变。如果 `internal/access/manifest.go` 中权限的 `Name`、`Description` 要改成「收款约定」用语，
  改完必须执行 `pnpm run generate:permission-keys`。权限键名不变时，生成物应当没有差异。

### 开发种子：`server/cmd/sync-dev/main.go`（约 660 行）
- 只为客户角色写入约定：`SetPaymentTermsDays(30)`，额度 50 万，币种为公司本币。
- 删除废弃字段的 setter。不再为供应商写规则。
- 顺便把 `_, _ =` 吞错改为返回错误，便于发现种子失败（只改这一处，不扩大范围）。

## 4. 前端

### 4.1 共用表单片段
新建 `web/src/pages/partners/components/PaymentTermsFields.tsx`，主档页和侧边抽屉都用它（同属 `pages/partners`，不需要提到 features）：
- 字段 `paymentTermsMode: 'none' | 'cash' | 'days'` + `paymentTermsDays`（mode 为 `days` 时必填，1–3650）；
- 字段 `creditLimit`（本币，沿用 `creditLimit.ts` 的 `creditLimitFields` 和 `hasInvalidCreditCurrency`）。
- 提供两个纯函数：
  - `paymentTermsToForm(rule)`：`NULL → none`，`0 → cash`，`N → days + N`；
  - `paymentTermsFromForm(values)`：`none → 不带 paymentTermsDays`，`cash → 0`，`days → N`。
  - 「未约定」时，提交载荷省略 `paymentTermsDays`（proto `optional` 未设置），服务端写入 NULL。
    已核实：`data/partner_settlement_rule.go` 的 `updatePartnerSettlementRule` 在 `PaymentTermsDays == nil`
    时会显式 `ClearPaymentTermsDays()`，额度为空时也会同时清空额度和币种。保留这个写法，不要改成只在有值时才 Set。

### 4.2 主档详情页
- `components/SettlementSection.tsx`：改为渲染 `PaymentTermsFields`，标题「收款约定」。
  删除对账方式、结算方式、结算币种、结算日期、「账期：基准 + 天数」这几个控件。
- `partner-detail.tsx`：
  - 只在 `roleType === 客户` 时渲染该分节，并组装 `settlementRule`；其他角色的 `roleInputs` 一律不带 `settlementRule`。
  - 删除新建时的默认值（`statementMode`、`settlementMethod`、`settlementBase`、`settlementDay`、`settlementCurrency`、`creditDays`）。
  - 回填（约 446–460 行）改用 `paymentTermsToForm`；提交（约 613–640 行）改用 `paymentTermsFromForm`，删除 `isTermFree` 和 `isActive` 逻辑。
  - 现有的权限与加载状态契约保持不变（规则读、建、改权限分开；查询未完成或失败不视为空规则），
    见 `.trellis/spec/server/backend/partner-aggregate-consistency.md`。

### 4.3 侧边抽屉
- `components/secondary/SettlementRulesPanel.tsx`：从「表格 + 新增弹窗」改为单条约定的查看与编辑表单（共用 `PaymentTermsFields`）。
  没有约定时显示「未约定」，并按 create 权限提供「设置收款约定」；已有时按 update 权限提供编辑。
  提交仍通过 `ModalForm.onFinish` 或页面内表单完成，遵循 AGENTS.md 的 ProComponents 范式。
- `partner-secondary.tsx`：「结算规则」页签改名为「收款约定」，只在往来单位有客户角色时显示。

## 5. 规范更新

- `.trellis/spec/server/backend/partner-credit-currency.md`：删除「结算币种与额度币种是两个字段」等关于结算币种的描述。
- `.trellis/spec/server/backend/partner-casual-contract.md`：「正式应收客户按启用结算规则带出默认账期」
  改为「按该客户唯一一条收款约定带出」。
- `.trellis/spec/server/backend/partner-aggregate-consistency.md`：「零/一/多规则与停用规则」改为「零/一条约定」；
  补充「非客户角色提交规则 → 参数错误」。
- `.trellis/spec/domain/glossary.md`：新增或改写「收款约定」词条
  （实体 `PartnerSettlementRule`，仅客户角色，每角色一条，NULL / 0 / N 三种期限 + 本币信用额度）。
- `.trellis/spec/domain/architecture-map.md` 如果提到结算方式、月结等，同步改写。

## 6. 测试设计

### 后端
- Biz：`normalizePartnerSettlementRule` 的边界（账期 -1 / 0 / 3650 / 3651，额度与币种不成对）；
  非客户角色 Create/Update 被拒绝。
- Data（真实 PostgreSQL）：
  - 同一角色第二次 Create → `PARTNER_SETTLEMENT_RULE_EXISTS`；
  - 主档聚合写入：客户角色零条用 create、一条用 update；非客户角色带规则时整次回滚；
  - 把账期从 30 改为「未约定」后，数据库里为 NULL；
  - `partnerCreditSummaries`：账期和额度直接取唯一一条。
- **迁移集成测试**（参考 `invoice_number_rule_migration_integration_test.go`、`migration_fixture_test.go`）：
  在隔离的 PostgreSQL 上先执行到本迁移之前的版本，再构造数据、执行本迁移。用例：
  - **合并与删除：** 供应商规则、停用客户规则、同一客户两条启用规则（15 天 / 10 万 CNY、45 天 / 30 万 CNY）
    → 每个客户角色一条，账期 45、额度 30 万 CNY；非客户规则和停用规则已删除；唯一索引 `partner_settlement_rule_role_key` 存在，废弃列已删除。
  - **额度 0 不参与：** 同组额度 0 与 20 万 → 20 万；同组只有 0 或空 → 保留行原值。
  - **账期全空：** 同组账期都为空 → 结果为空。
  - **守卫失败（三种）：** 启用客户规则的正额度缺币种；币种与公司本币不同；公司及上级都没有本币
    → 迁移返回错误，`partner_settlement_rules` 的行数与各列取值都和迁移前一致（事务整体回滚）。
  - **守卫不误伤：** 只有停用客户规则或供应商规则币种异常 → 迁移成功，这些行被删除。
  - **子公司继承本币：** 规则所属公司 `base_currency` 为空、上级公司为 CNY → 视为 CNY，守卫通过。
    （实施时以 `organizations` 的 CHECK 约束为准，核对哪些公司类型允许 `base_currency` 为空，构造合法数据。）
  - 空库完整迁移链可以执行。
  涉及 Schema 初始化的测试不能用 `t.Parallel()`；参考 `invoice_number_rule_migration_integration_test.go` 和 `migration_fixture_test.go`
  的「迁移到指定版本 → 构造数据 → 继续迁移」写法。
- `TestMigrationManifestCoversEntSchema` 通过。
- 现有结算规则、主档、建账预览、信用额度的测试同步删除废弃字段。

### 前端
- `PaymentTermsFields` 的两个纯函数：`NULL / 0 / N` 往返转换；「未约定」时载荷不带 `paymentTermsDays`。
- `partner-detail` 系列测试（`partner-detail.test.tsx`、`partner-detail-save.test.tsx`、
  `partner-detail-settlement-access.test.tsx`、`partner-detail-enabled.test.tsx`）：删除废弃字段的断言；
  新增「供应商角色不显示收款约定、提交载荷不带 settlementRule」。
- `SettlementRulesPanel.test.tsx`：改为单条约定的查看、新建、编辑，以及权限场景。
- 响应映射按 AGENTS.md 要求覆盖真实 wire 形态：`paymentTermsDays` 未出现（未约定）、显式 0（现结）、非零。

## 7. 回滚

后端和前端改动绑定同一个契约，按 AGENTS.md 在同一组提交里一起交付（可以先后两次提交，但不能只上线一半）。
迁移会删除数据，回滚只能靠恢复数据库备份；开发库执行迁移前，要先向用户说明影响并取得确认。
