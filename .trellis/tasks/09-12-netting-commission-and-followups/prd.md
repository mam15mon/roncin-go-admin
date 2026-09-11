# 对冲提成通道与遗留修补 PRD

## 1. Background & Scope

上一任务（09-11，已归档）将 C2「对冲结清触发提成」拆出为独立任务。本任务完成它，并顺带
收敛遗留修补与潜伏缺陷：

1. **C2（P1，功能）**：提成创建硬绑核销——`CreateCommission` 强制 `verification_id`，
   计算源只认 `Status = ACTIVE && Direction = RECEIVABLE` 的核销分摊，已实现收入只从核销
   推导。纯对冲结清的应收账单走完对冲确认后账单结清，但无任何通道生成提成。
2. **PreviewSplit 锁定预览（P2，修补）**：拆票 Execute 已有统一内容门禁（09-10 起），
   但 `PreviewSplit` 不显示订单锁定/终态，预览通过、提交才被拒——与已修复的单证/船期/
   改配预览不一致（同族最后一处）。
3. **admin 集成测试遗留（P2，修补）**：`TestAdminEmployeeLifecyclePostgres` 仍用直连
   public schema + `AutoMigrate` 的旧夹具模式，本地库存在多个根组织（开发数据）时
   `Only()` 命中多行报 `ent: organization not singular`，在 HEAD 上即失败。
4. **GetChangeActions 业务锁感知（P2，修补）**：`GET /api/v1/sea/orders/{order_id}/change-actions`
   仅校验终止与关单，遗漏订单业务锁 `BusinessLockStatus == LOCKED`。订单业务锁定时仍返回
   `CanSplit = true, CanReassign = true`，导致前端按钮高亮可点、点开填写后在预览/提交时才拦截。
5. **非 CNY 本币提成折算修复（P2，缺陷修复）**：非 CNY 本币组织（如 USD）提成生成时，
   代码向总部请求 `CNY → USD` 汇率并进行倒数派生。但在单一汇率架构下总部仅维护外币折本币
   （`USD → CNY`），导致非 CNY 组织查汇率必报 `ErrExchangeRateMissing`；且 `1 / rate` 倒数
   公式方向颠倒（100 USD 乘汇率 3 算成 33.33 CNY 而非 300 CNY）。需拉齐为：查询组织本币
   折总部本币（`baseCurrency → CNY`）的基准汇率直接相乘。

## 2. Detailed Requirements

### C2：对冲提成通道

1. **来源模型**：`finance_commissions.verification_id` 由必填改为可空，新增可空
   `netting_id`（FK）与 `netting_no`（快照）；**恰好一个来源**（服务层校验二选一），
   不引入多态来源抽象。既有核销路径行为不变。
2. **计算口径**：对冲来源与核销同口径——CONFIRMED 对冲单的 RECEIVABLE 分摊按
   `分摊金额 / 账单总额` 比例摊入账单行本位币形成"已实现收入"，成本分摊、提成规则、
   CNY 换算（生成日单汇率）复用既有管线；分子分母均为账单行本位币（与 B3 拉齐后一致）。
3. **冲减联动**：对冲单 `REVERSE` 时对来源为该对冲单的提成执行与核销反冲同款处理——
   未支付则取消，有已支付敞口则生成 CONFIRMED 冲减调整（Clawback）；费用财务锁判定
   （净额口径）自动覆盖新来源，无需改锁。
4. **API**：`PreviewCommission` / `CreateCommission` 接受 `verification_id` 或 `netting_id`
   二选一；新增 `ListCommissionNettingCandidates`（CONFIRMED 对冲单候选，netting 形态
   响应），不把既有核销候选接口改成大杂烩。
5. **前端**：提成页新增「对冲提成」入口：CONFIRMED 对冲单列表 → 选规则/员工 → 预览 →
   创建，交互复用既有核销提成流程组件；提成列表/详情展示来源（核销单号或对冲单号）。

### PreviewSplit 与 GetChangeActions 门禁

- `PreviewSplit` 对拆票来源订单执行统一内容门禁（业务锁/终止/结案），未通过时写入
  `ValidationErrors`（既有 `SeaOrderSplitValidationError` 形态），预览端可见原因；
  Execute 的 409 语义与既有门禁保持不变。
- `GetChangeActions` 在既有终止与关单检查之外，同步感知订单业务锁（`ensureOrderBusinessContentEditable`
  或 `BusinessLockStatus == LOCKED`），锁定时将 `CanSplit` 与 `CanReassign` 置为 `false`
  并在原因列表追加锁定原因，消除前端按钮可点却在后续拦截的体验裂缝。

### 非 CNY 本币提成折算修复

- **汇率解析路由**：提成预览与创建生成上下文时，若组织的 `baseCurrency != "CNY"`，改为以该
  `baseCurrency` 解析总部基准折本币汇率（`USD → CNY`，即传入 `currency = baseCurrency`、目标
  指向总部 CNY 本币），获取正向折算比率。
- **金额换算公式**：`ResolveCommissionCNYRate` 去除倒数运算（`1 / resolvedRate`），直接保存
  解析到的 `USD → CNY` 汇率快照（来源 DERIVED），`ApplyCommissionAmount` 保持
  `commissionAmount × exchangeRate`，确保 100 USD 在汇率 7.2 时正确换算为 720 CNY。

### admin 集成测试迁移

- `TestAdminEmployeeLifecyclePostgres` 夹具迁到 `getIntegrationData`（隔离 schema +
  完整版本化迁移链）模式，消除对本地 public schema 开发数据的依赖；不改被测行为。

## 3. Acceptance Criteria

- [ ] **C2 通道**：
  - [ ] 纯对冲结清的应收账单可走 候选 → 预览 → 创建 生成提成；计算金额与同口径手工
        推导一致（分摊比例 × 账单行本位币）；
  - [ ] 同一 (netting, employee, rule) 幂等去重；verification 与 netting 来源互不冲突；
  - [ ] 对冲 REVERSE 后：未支付提成被取消、已支付生成 CONFIRMED 冲减；费用锁随净额释放；
  - [ ] 既有核销来源回归全绿（行为不变）。
- [ ] **拆票与操作门禁**：
  - [ ] 锁定/终止/结案订单的拆票预览返回 ValidationErrors 含订单号与原因；可编辑订单预览不受影响；Execute 错误语义不变；
  - [ ] 业务锁定订单的 `change-actions` 返回 `can_split = false`、`can_reassign = false` 且原因列表含锁定文案。
- [ ] **非 CNY 本币提成**：USD 等非 CNY 本币组织可正常命中总部基准汇率（如 USD->CNY=7.2）
      生成提成快照；CNY 提成金额为 `USD本币提成 × 7.2`（不再出现倒数取 1/7.2 缩水）；
      单测覆盖正向折算与多币种用例。
- [ ] **admin 测试**：`TestAdminEmployeeLifecyclePostgres` 在隔离 schema 下通过，不依赖
      public schema 数据。
- [ ] **质量门禁**：`go -C server test ./...`（含集成）、`pnpm run check:web` 全绿；
      迁移在开发库执行成功。

## 4. Out of Scope

- 提成规则引擎、多来源混合计算（一个提成单跨核销+对冲）——不建；
- 对冲单确认时自动生成提成（仍由人工从候选发起，与核销一致）。
