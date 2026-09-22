# 技术设计：订单人员为提成归属唯一真相

## 1. 边界与分层

| 层 | 变更 |
|----|------|
| `server/internal/biz` | `order_usecase.go` 创建路径新增三岗校验 + 新错误构造器；删除 `partner.go` 旧错误构造器 |
| `server/internal/data` | `order_write.go` 快照改从 `order_personnels` 取数；`order_write_draft.go` 换客户改为更新 `customer_id` |
| `server/api` | 无 proto 变更（错误码为 Kratos reason 字符串，不进契约） |
| `web/src/pages/orders` | `SeaPersonnelSection.tsx` 新建模式三岗必填 |
| `web/src/pages/partners`、`orders/components` | 档案表单与快建三岗放宽为选填，注释修正 |
| `server/cmd/sync-dev` | 客户种子补齐操作/客服责任人 |
| `.trellis/spec` | 规格文档口径修订 |

## 2. 后端设计

### 2.1 biz 层三岗校验（R2）

`normalizeOrder(input, creating)` 中 `creating == true` 时（仅 `Create` 走到；`UpdateDraft` 传 false），在现有人员去重校验后追加：

```go
// 提成相关岗位（销售/操作/客服）创建时必须配齐，缺配直接拒绝。
if creating {
    if err := validateOrderCommissionPersonnel(output.PersonnelAssignments); err != nil {
        return nil, err
    }
}
```

`validateOrderCommissionPersonnel`：收集缺失岗位，按 销售→操作→客服 固定顺序输出；空缺失返回 nil。错误构造器仿照旧 `NewPartnerCommissionAssignmentMissing` 的去重+排序模式：

- `orderCommissionPersonnelRoleLabels` / `orderCommissionPersonnelRoleOrder`（biz/order_types.go 或就近）
- `NewOrderCommissionPersonnelMissing(missing []OrderPersonnelRole) error` → `errors.BadRequest("ORDER_COMMISSION_PERSONNEL_MISSING", "订单缺少"+labels+"人员，请在内部信息区补全后再开单")`

注意：`sea_order_change_split_execute.go` 直接调 `createOrderPersonnel` 复制源订单人员，不经过该校验——源订单创建时已保证三岗，复制天然齐全，无需加校验。

### 2.2 快照取数源切换（R3）

`order_write.go` `snapshotOrderCommissionAttributions` 重写：

- 签名不变（`organizationID, orderID, customerID, attributedAt`）——`customerID` 仍用于归属行冗余列。
- 查询：`tx.OrderPersonnel.Query().Where(OrderIDEQ(orderID), OrganizationIDEQ(organizationID), RoleIn(SALES, OPERATOR, CUSTOMER_SERVICE)).WithUser()`。
- 写入：`SetSourceAssignmentID(personnel.ID)`（订单人员行 ID）、`SetEmployeeID(personnel.UserID)`、`SetEmployeeName(user.DisplayName)`、`SetPersonnelRole(...)`、`SetCustomerID(customerID)`、`SetAttributedAt(attributedAt)`。
- 去重键沿用 `userID:role`；三岗齐全由 biz 门禁保证，此处不再做缺配校验（`order_personnels` 每岗唯一已由 `normalizeOrder` 去重保证，配合 `ordercommissionattribution_order_role` 唯一索引兜底）。
- 创建事务调用点（`order_write.go:133`）位于 `createOrderPersonnel`（:130）之后，同事务读未提交行无问题，顺序不动。

`order_commission_attributions.source_assignment_id` 为 `uuid NOT NULL` 无外键（`migrations/20260827212000`），语义切换不需要迁移；`personnel_role` CHECK 约束 `IN('SALES','OPERATOR','CUSTOMER_SERVICE')` 与订单人员角色枚举字符串一致。

### 2.3 草稿换客户（R4）

`order_write_draft.go:298-303` 替换为：

```go
if _, updateErr := tx.OrderCommissionAttribution.Update().
    Where(ordercommissionattributionent.OrderIDEQ(id)).
    SetCustomerID(input.CustomerID).Save(ctx); updateErr != nil {
    return updateErr
}
```

人员与归属内容不变，仅同步冗余 `customer_id`；零行匹配（存量草稿无归属行）为无操作。

### 2.4 旧门禁清理（R5）

- 删除 `biz/partner.go:33-62` 三件套与 `biz/organization_interceptor_test.go` 中 `TestPartnerCommissionAssignmentMissingListsOnlyMissingRoles`。
- 全仓 `rg PARTNER_COMMISSION_ASSIGNMENT_MISSING` 清零（前端两处为注释引用，随 R5 前端改动一并删除）。

## 3. 前端设计

### 3.1 三岗必填（R1）

`SeaPersonnelSection.tsx`：

- `buildSeaPersonnelSection(props)` 读取 `props.isDetail`（`templates/types.ts:33` 已有）。
- `PersonnelAssignmentFields` 新增 `rules?: Rule[]` prop；新建模式下三岗传 `[{ required: true, message: '请选择XX人员' }]`，其余字段与其余模式不传。
- 详情模式渲染完全不变。

必填触发时机与「请选择委托单位」一致（ProForm 提交校验），不引入额外状态。

### 3.2 档案必填降级（R5）

- `partners/components/BasicInfoSection.tsx:127-137`：删除 `requiresCommissionStaff` 分支，三岗 `rules` 恒为空（保留字段本身与带入语义）。
- `orders/components/PartnerQuickAddSelect.tsx:86-88`：客户快建仍展示三岗选择（作为默认值录入），移除"必须同步配置"的必填约束与注释。
- `SeaBasicInfoSection.tsx` 带入逻辑零改动。

## 4. 种子设计（R6）

`cmd/sync-dev/main.go` `partnerSeedDef` 增加可选 `operatorRep`/`serviceRep`（沿用 `salesRep` 的账号名解析方式）；5 个客户种子补配操作/客服（使用既有种子员工，如操作/客服岗员工）。幂等策略与现有 `salesRep` 补挂逻辑一致（缺则插，不重复）。

种子订单的归属行、提成演示数据均由种子直接写入，不受快照源切换影响。

## 5. 测试与验证

后端（`go -C server test`）：

- biz：三岗校验单测（缺岗列名顺序与去重、全齐通过、CREATOR 不计入）。
- data：`order_create_transaction_test.go` 重写为从订单人员取数断言；新增草稿换客户仅更新 `customer_id` 的用例；删除旧缺配拦截用例。
- 既有引用旧错误的用例（`organization_interceptor_test.go` 等）同步改写。

前端（`pnpm --dir web exec vitest run`）：

- `SeaPersonnelSection` 必填规则渲染（新建有、详情无）。
- 新建表单提交校验集成用例（空三岗阻断，错误文案）。
- 既有 `SeaBasicInfoSection.customer-personnel.test.tsx`、`PartnerQuickAddSelect` 系列用例按降级口径调整断言。

全栈门禁：`pnpm run check:fast`。

手工验收路径：`pnpm run sync:dev` → 用 CUST-HY-001 开单（带入仅销售，手选操作/客服）→ 提交成功 → 查 `order_commission_attributions` 三行；再验证空三岗提交被表单拦截、API 直发缺岗请求返回 400。

## 6. 风险与回滚

- 财务提成链路（`finance_commission_*`、`workbench`、拆单复制）只读归属行，不感知来源，风险低；AC3 手工验收覆盖。
- 详情页人员编辑不落库为既有缺口，本次不扩大（PRD Out of Scope），必填仅限新建模式避免踩到。
- 回滚点：后端切换（2.2/2.3）与前端必填（3.1）可独立回退；无 Schema 迁移，回滚无数据残留问题（`source_assignment_id` 新旧语义混存仅影响开发库，不做清洗）。
