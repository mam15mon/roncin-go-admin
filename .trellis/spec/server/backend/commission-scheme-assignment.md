# 提成方案与员工分配契约

## 1. Scope / Trigger

修改 `FinanceCommissionRule`（提成方案主表）、`FinanceCommissionRuleAssignment`
（员工有效期分配）、计提候选/预览/生成的规则解析、工作台提成资格门禁时必须遵循。方案是组织内可复用规则；员工资格由分配关系决定；
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

- **配置与办理权限分离**：方案维护、名单分配、复制使用 `system.finance.commission.configure`；计提、确认、付款、调整与业务申请审批使用 `system.finance.commission.manage`。方案配置和经营办理均须进入获授权的当前公司工作台；系统管理工作台不维护公司提成方案，原 manage 不作为 configure 的隐式兼容授权。

- **实际有效区间** = 方案区间 ∩ 分配区间，`YYYY-MM-DD` 字符串闭区间比较；
  起点必填，空终点 = 正无穷。同一 `组织 × 员工 × 身份` 的实际区间不得跨方案重叠；
  唯一索引仅作并发兜底，判定靠锁内区间校验。
- **更新校验**：创建启用方案的起始日不得早于业务日期；更新不能机械复用此限制而拒绝已生效方案名称/备注维护。已到期方案保留原终止日不变时仍允许非核心修改；纯名称/备注修改且全部资格字段不变时，仍锁现存 Membership，但不要求历史员工当前启用。创建/启用及任何资格字段变化继续检查成员、账号有效性，不得借纯元数据例外恢复停用员工资格；实际修改结束日才适用禁止追溯规则。所有启用方案更新都按每个未取消分配段的实际交集复核其他方案冲突，排除本方案分配，不能只在停用转启用时检查。预读成员集合与锁内集合变化返回冲突，禁止持方案锁后补锁新成员。
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
- **订单列表提成退役**：海运出口订单列表不展示提成，不附加提成摘要，也不调用专用聚合；提成办理和查看继续使用财务模块与工作台。不得因增加列表字段而恢复订单级提成旁路。
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
- 更新：已生效/已到期方案名称备注可改，核心参数与追溯修改拒绝；未来启用方案延长到后继方案区间拒绝且零写入。区间计算必须先分别求方案与分配的交集，再比较实际交集，不能用两个区间对各自相交代替。
- 解析与资格：`TestCommissionResolutionPostgres`（多员工共享、员工差异、多身份
  独立候选、固定薪无候选、加入/退出日期边界、离职历史资格、复制衔接后历史命中）。
- 订单列表退役验证：接口不含提成摘要字段，列表无提成列，财务与工作台相邻用例继续通过。
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


## 方案表单日期与对冲验收

- 新建/编辑方案的 `ProFormDateRangePicker` 提交契约必须与回调一致：回调调用
  Dayjs `.format('YYYY-MM-DD')` 时，外层表单显式设置 `dateFormatter={false}`。
  仅将 TypeScript 字段标为 Dayjs 不会改变 ProForm 默认字符串转换。
- 新建启用方案须在前端校验起始日必填且不早于业务日期，后端继续最终校验；
  已生效方案修改名称/备注按原有更新规则处理，不能套用新建禁止过去日期的限制。
- 完整有限区间创建成功须有明确提示、关闭与列表刷新；失败保留输入。回归测试保留
  真实 ProForm 日期转换与提交，仅 mock 请求；不能直接调用 onFinish 代替交互测试。
- 对冲提成验收应从真实 `CreateRule` 建立员工分配，串联员工候选、预览、生成草稿和确认。
  无方案时空候选合法；来源 REVERSED 或分摊失效不再是有效计提来源。
- 金额按订单所有当前有效费用及账单快照推导应收 R、应付 P，分摊成本为本次实现收入
  × P / R；不得只取本次对冲的应付分摊。R=300、P=200、实现收入200、毛利10%时，
  成本133.33333333、毛利66.66666667、提成6.66666667，界面显示6.67。
  亏损时提成基数归零，仍可返回零额候选、生成及确认，不能将零额当作引擎失效。
- 实际验收先核对来源当前状态、方案/分配区间与固化员工归属；不复活已反转来源、
  不回填历史归属或更改原费用来制造正额结果。开发验收数据须有明确标识，验收到
  CONFIRMED 不等于已发放。

- 客商编码 `Partner.code` 选填；`FinanceCommissionLine.customer_code` 快照允许空字符串，
  保留长度上限与不可变属性，不得强制 NotEmpty 或用伪造编码补齐。真实库回归必须
  包含无编码客商的预览、生成与确认，避免预览成功而生成草稿时 Ent 校验返回500。
