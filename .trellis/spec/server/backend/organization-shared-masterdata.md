# 主数据存储三型与维护权（A/B/C 型）

## 1. Scope / Trigger

适用于全部主数据/字典类实体（币种、国家、区划、箱型、费用大类、异常类型、计费单位、
航司、船司、港口、机场、费用科目、汇率、税务名称、企业资源等）。**新增此类实体或修改
其组织可见性/唯一索引时必须遵循**；维护权由存储型推导，禁止运行时按组织树动态判定。
（2026-09 主数据治理重构确立，取代旧的「总部行 + resolveHeadquartersOrganizationID
运行时共享」模型。）

## 2. Signatures

```go
// 存储三型（internal/data/ent/schema）
// A 型：无 organization_id 列，业务码 UNIQUE(code) 全局唯一
// B 型：organization_id 可空；NULL=总部基线行，非 NULL=本组织行
//       UNIQUE(code) WHERE organization_id IS NULL          -- 基线码全局唯一
//       UNIQUE(organization_id, code) WHERE organization_id IS NOT NULL
// C 型：organization_id 必填，(organization_id, code) 唯一

// internal/data/baseline_query.go —— B 型统一读取谓词工厂（仅此一处，勿在业务仓储自拼）
BaselineOrLocal(codeCol, code, orgID)              // 点查：本组织行优先，LIMIT 1
BaselineShadowedByLocal(table, codeCol, orgID)     // 列表：NOT EXISTS 同码基线整行遮蔽，
                                                   // 去重下推保证分页计数一致

// internal/biz/organization.go —— 写路径拦截器（权限码 + 组织身份双校验，缺一即拒）
RequireGlobalMasterDataWrite(ctx, principal)  // A 型写入（当前组织根节点 kind==headquarters）
RequireBaselineWrite(ctx, principal)          // B 型 NULL 基线行写入
```

## 3. Contracts

- **归型判定**：错误数据爆炸半径跨组织（单证/EDI/跨组织资金流/合并口径）→ A 型或 B 型
  基线；仅本组织内 → C 型。存在总部基线覆盖不到的本地真实对象（生僻港口等）→ B 型本地
  行；不存在 → A 型，不开本地口子。
- **读取**：A 型/C 型是普通查询；B 型列表必须走 `BaselineShadowedByLocal`（同码基线行
  整行不可见），点查走 `BaselineOrLocal`。**禁止**在分页列表使用 ORDER BY + LIMIT 1。
- **写入**：A 型与 B 型 NULL 行仅总部（拦截器双校验）；B 型 org 行仅归属组织（不可迁移
  归属）；C 型归归属组织。前端按钮经 `access.isHeadquartersOrganization` × 权限码组合
  门控，与服务端同源，页面不写第二套规则。
- **主数据读路径禁止** `resolveHeadquartersOrganizationID`（该函数仅存于写路径组织身份
  判定，收敛在 `data/organization_currency.go`）。
- 汇率特例（B 型）：`effective_from` 即当周周一，周窗口由服务端派生；`ar_rate`/`ap_rate`
  按费用收支方向取值；解析顺序 当周 org 行→回溯最近历史周（INHERITED_LAST_WEEK）→NULL
  基线直连/套算→MANUAL，不阻断单据；跨组织资金流按原币记账，不做系统折算。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 非总部上下文写 A 型表 / B 型 NULL 行 | 403（拦截器，权限码或身份缺一即拒） |
| 非归属组织读写 B 型 org 行 / C 型行 | 404（不泄漏行存在性），归属校验先于引用校验 |
| A 型同码第二行 / B 型两条 NULL 同码 | 数据库唯一索引物理拒绝 → `ent.IsConstraintError` 映射业务错误（409） |
| B 型列表同码本地行存在 | 基线行整行不可见（NOT EXISTS 遮蔽） |

## 5. Good / Base / Bad Cases

- Good：新分公司开单，港口选择器直接命中 NULL 基线行；录生僻码头走本地新增，仅本组织可见。
- Bad：给 A 型表查询传 organization_id 过滤，或读路径调用组织树解析——重构已废除，见
  `baseline_query.go`。
- Bad：B 型列表用 `ORDER BY organization_id = :org DESC LIMIT 1` —— 分页计数与行集不一致。
- Bad：分支上下文持权限码写 NULL 基线行被放行——拦截器必须双校验组织身份。

## 6. Tests Required

- 真实 PostgreSQL 集成（带 `RONCIN_INTEGRATION_DATABASE_SOURCE`，未配置会全部静默
  skip 造成假绿）：B 型两形态（遮蔽/穿透/分页计数一致）、A/B 型唯一冲突映射业务错误、
  拦截器双校验（分支持码 403、总部通过）、组织隔离（跨组织不可见不可改）。
- 迁移在一次性隔离库从零执行全链验证。

## 7. Wrong vs Correct

### Wrong

```go
// 读路径递归找总部 + 手拼共享谓词（旧模型，已废除）
hq, _ := resolveHeadquartersOrganizationID(ctx, organizationID)
query.Where(port.Or(port.OrganizationIDEQ(organizationID), port.OrganizationIDEQ(hq)))
```

### Correct

```go
// B 型列表统一走谓词工厂（internal/data/baseline_query.go）
query.Where(baselineShadowedByLocal("ports", port.FieldUnLocode, orgID))
```

## 8. Common Mistakes（迁移陷阱，两次实测踩坑）

- **改枚举值集必须同步重建 CHECK 约束**：PostgreSQL 列约束不会随 Ent schema 自动更新，
  旧 `*_check` 约束只认退役值集，新值写入直接 23514（阶段一 `master_data_items_kind_check`、
  阶段二四张财务表 `exchange_rate_source_check` 两次踩坑）。迁移中 `DROP CONSTRAINT IF
  EXISTS + ADD CONSTRAINT` 与数据重映射同批执行。
- **部分唯一索引生效期间先去重后归一化**：先 UPDATE 再去重会在 UPDATE 语句内撞索引中止；
  顺序必须是「按目标键去重 → 归一化」。
