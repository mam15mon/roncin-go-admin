# 提成方案与员工分配契约

## 1. Scope / Trigger

修改 `FinanceCommissionRule`（提成方案主表）、`FinanceCommissionRuleAssignment`
（员工有效期分配）、计提候选/预览/生成的规则解析、工作台提成资格门禁、订单列表
提成摘要投影时必须遵循。方案是组织内可复用规则；员工资格由分配关系决定；
任何「谁在什么日期按哪个方案计提」的判定都以本契约为唯一真相源。

## 2. Signatures

```text
finance_commission_rules        方案主表：组织、名称、SALES/OPERATOR/CUSTOMER_SERVICE 身份、
                                计提口径、比例、effective_from(必填)/effective_to(可空)、
                                enabled、version、legacy_readonly
finance_commission_rule_assignments  分配表：organization_id、rule_id、employee_id、
                                effective_from(必填)/effective_to(可空)、cancelled_at/by、
                                terminated_at/by（外键全部 NO ACTION）

锁序（所有方案/分配写入口共用，internal/data/finance_commission.go）：
  Data.WithTx → 员工 ID 排序去重 → 逐个 Membership 行 FOR UPDATE（校验成员有效）
  → 目标方案行锁 + expected_version 校验 → 锁内 ensureCommissionRuleIntervalFree
  → 写入并 version+1

解析（核销/对冲候选与工作台预计机会同口径）：
  resolveCommissionRuleForDate(来源日期) → 唯一命中「已启用方案 ∩ 该员工未取消分配」
  × 订单同身份 OrderCommissionAttribution；任一条件不满足不产生候选
```

## 3. Contracts

- **实际有效区间** = 方案区间 ∩ 分配区间，`YYYY-MM-DD` 字符串闭区间比较；
  起点必填，空终点 = 正无穷。同一 `组织 × 员工 × 身份` 的实际区间不得跨方案重叠；
  唯一索引仅作并发兜底，判定靠锁内区间校验。
- **生效后参数锁定**：方案到达生效日后，身份、口径、比例、起始日不可原地修改，
  不得用 `enabled = false` 追溯失效；调整走【复制为新方案】（新方案当天/未来生效，
  旧方案终止日 = 新方案生效日前一日）。`enabled = false` 仅用于未生效草稿/作废
  与 `legacy_readonly` 旧规则。
- **legacy_readonly**：迁移前无员工分配的旧组织级角色规则的持久标记。不可补挂
  员工、不可重新启用、只可复制为新方案；与「停用草稿」靠该列区分，禁止用
  「enabled=false 且无分配」推断。
- **名单变更**：方案生效前可直接调整；生效后新增/移除必须带 `change_effective_date`
  ≥ 系统业务日期（Asia/Shanghai 服务端口径，`biz.Today` 显式入参便于测试）；
  移除 = 提前终止（`terminated_at`），取消未生效段 = `cancelled_at`，
  禁止物理删除任何分配行。成员停用被「当前/未来未终止分配」阻断，须先以离开
  日期终止分配。
- **计算指纹只含稳定资格标识**：方案 id+身份+口径+比例+起始日、分配段 id+员工+起始日。
  禁止纳入 name/version/enabled/终止日等易变字段——名单变更与复制衔接不得使
  已有提成的确认失败或触发重算（历史快照不可变）。
- **工作台资格门禁**：`has_commission_eligibility`（optional bool，服务端恒赋值）
  = 存在本人当前/未来有效分配（方案启用且实际区间未结束）OR 本人
  DRAFT/CONFIRMED/PAID 提成单或调整单。订单协作与归属不开启门禁；已取消记录
  不单独开启；门禁为假时服务端不执行提成金额查询。
- **订单列表提成摘要隐私**：对页面每个订单组织单独判定调用者
  `system.finance.commission.read`——持有 → ORGANIZATION 全员汇总，
  否则 → EMPLOYEE 且 `employee_id = caller` **必须出现在 SQL WHERE 条件内**
  （先限定后聚合）；普通员工响应无他人姓名/数量/状态/金额及「整票是否有提成」
  等旁路字段；本人无记录返回一致空态。事实集合可并存（预计/待确认/待发/已发/
  待冲减），PAID 不宣称整票结清，冲减 DRAFT ≠ 已扣回，取消不入桶。
- **离职历史资格**：来源日期落在离职前有效分配期间的提成仍可生成/展示；
  查询资格以「来源日期 × 方案区间 × 分配区间」判定，不以当前账号/成员状态抹除。

## 4. Validation & Error Matrix

| 条件 | 行为 |
| --- | --- |
| 实际区间与本人同身份其他启用方案重叠 | 锁内拒绝（重叠领域错误），唯一索引兜底映射同语义 |
| 启用方案起始日早于业务日期 / 无有效员工 | 拒绝启用 |
| 生效方案修改身份/口径/比例/起始日，或 enabled=false | 拒绝；引导复制为新方案 |
| `change_effective_date` < 业务日期 | 拒绝（禁止回溯名单） |
| 停用成员存在未终止当前/未来分配 | 拒绝并返回需先处理的方案清单 |
| 来源日期命中 0 或 >1 个方案×分配 | 不产生候选 / 明确领域错误 |
| expected_version 不匹配 | Conflict「已被更新，请刷新后重试」 |
| 摘要同订单出现多币种 | 数据异常直接报错，禁止裸相加 |

## 5. Good / Base / Bad Cases

- Good：10 人同比例销售方案 → 一个方案多选分配；个别人比例不同 → 另建方案。
- Base：员工 3 月 1 日退出方案（分配 2 月 28 日终止），2 月核销 3 月 5 日生成 →
  来源日期 2 月仍在分配区间内，资格保留并命中原方案。
- Bad：把旧角色规则重新启用或按角色批量补挂员工——恢复组织级歧义，禁止。
- Bad：普通编辑直接缩短已生效方案终止日来「停用」——追溯失效历史来源，禁止。

## 6. Tests Required

- PostgreSQL 并发：`TestCommissionRuleConcurrentOverlapPostgres`（两事务把同一
  员工身份加入重叠方案恰好一个成功）、分配 Schema/锁序集成
  （`TestCommissionRuleAssignmentSchemaPostgres` 等）。
- 解析与资格：`TestCommissionResolutionPostgres`（多员工共享、员工差异、多身份
  独立候选、固定薪无候选、加入/退出日期边界、离职历史资格、复制衔接后历史命中）。
- 摘要隐私矩阵：`TestOrderCommissionSummaryPrivacyPostgres`（仅本人/组织级/
  跨组织权限不同/同事有记录本人无记录无泄露/多状态并存）。
- 工作台门禁矩阵（`workbench_integration_test.go`）：无分配无历史/未来分配/
  过期分配但有历史/仅取消记录/订单协作但固定薪。
- 前端：门禁 `!== true` 首帧无提成 DOM；生成枚举构造测试数据。

## 7. Wrong vs Correct

### Wrong

```go
// 先查后写的无锁冲突判断：首条分配无可锁行，并发必重复
if exists := queryOverlap(ctx, emp, role, from, to); !exists { insert(...) }
```

### Correct

```go
// 稳定父行锁串行化同一员工的全部方案写入，首条分配同样安全
for _, empID := range sortedUniqueEmpIDs { lockMembershipForUpdate(ctx, empID) }
lockRuleForUpdate(ctx, ruleID, expectedVersion)
ensureCommissionRuleIntervalFree(ctx, orgID, empID, role, from, to)
```

### Wrong（指纹）

```go
// 指纹含 name/version/effective_to：名单变更或复制衔接即确认失败，违背快照不可变
fingerprint := fmt.Sprintf("name=%s;ver=%d;to=%s", rule.Name, rule.Version, rule.EffectiveTo)
```

### Correct（指纹）

```go
// 只含影响来源日期资格的稳定标识
fingerprint := fmt.Sprintf("rule=%d;role=%s;basis=%s;rate=%s;from=%s;seg=%d;segfrom=%s", ...)
```
