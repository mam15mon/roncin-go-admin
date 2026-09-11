# 总部共享主数据读取模式

## 1. Scope / Trigger

适用于"总部统一定义、分公司只读共享、可选本组织扩展"的主数据实体（费用科目、计费单位、
应税服务、港口/机场/航司/船公司等行业主数据、汇率）。新增此类实体或修改其组织可见性时
必须遵循。

## 2. Signatures

```go
// internal/data 统一模式（fee_catalog.go / industry_reference.go / exchange_rate.go）
headquartersOrganizationID(ctx, organizationID) (uuid.UUID, error) // 沿 parent 链找树根
requireHeadquarters(ctx, organizationID)                           // 写防御：非总部拒绝
```

```text
读共享: WHERE organization_id = :org OR (organization_id = :hq
         AND NOT EXISTS(本组织同业务代码行))    -- 本组织行优先去重
写共享: 仅总部组织可写（费用科目/汇率）或按实体策略（行业主数据同步仅本组织）
```

## 3. Contracts

- 树根必须是 `KindHeadquarters`，否则 fail-closed 报错（禁止静默退化为本组织查询）。
- **去重下推到 SQL**（`NOT EXISTS` 同码本组织行让位），保证分页 `total` 与去重后行集一致；
  禁止取回内存去重。
- 调用组织即总部时退化为仅本组织单条件，不得让总部自视重复。
- keyword 检索（业务代码/中英文名/别名/无声调全拼/拼音首字母）、`is_active` 过滤行为与共享
  前完全一致；注意 shadow 语义：本组织同码行不匹配关键字时会遮蔽匹配关键字的总部行——这是
  「本组织行优先」的既定取舍。
- 对冲参照：组织创建时 provisioning 只复制号码规则/订单选项/国家；行业主数据靠读取共享而非
  复制，新分公司即插即用。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 组织树根不是总部 | 查询报错 fail-closed |
| 本组织与总部存在同码行 | 仅返回本组织行 |
| 非总部写总部专属主数据（费用科目/汇率） | 拒绝 |

## 5. Good / Base / Bad Cases

- Good：新分公司开 SE 订单，船公司选择器直接命中总部启用的 ShippingLine。
- Base：总部自身查询只见自己一行（无自遮蔽子查询必要）。
- Bad：查询仍按 `organization_id = :org` 严格过滤——新分公司候选全空（2026-09 前的行业主数据缺陷）。
- Bad：取回双行后在 Go 内去重——分页计数与行集不一致。

## 6. Tests Required

- Data/PostgreSQL（`TestIndustryReferenceHeadquartersSharingPostgres`）：共享可见+同码本组织
  优先、启用/keyword 过滤不变、总部自视不重复不见分公司私有行、四实体覆盖。
- 写防御：非总部写汇率/费用科目被拒（总部共享实体扩展时同步覆盖）。

## 7. Wrong vs Correct

### Wrong

```go
query.Where(port.OrganizationIDEQ(organizationID)) // 新分公司查询为空
```

### Correct

```sql
-- 去重下推：同业务代码本组织行存在时，总部行让位（Ent 以 entsql 子查询表达）
WHERE (organization_id = :org)
   OR (organization_id = :hq AND NOT EXISTS (
        SELECT 1 FROM ports p2
        WHERE p2.organization_id = :org AND p2.code = ports.code))
```
