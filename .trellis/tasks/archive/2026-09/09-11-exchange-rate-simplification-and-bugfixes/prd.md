# 汇率极简化彻底重构与核心业务/安全漏洞闭环 PRD

## 1. Background & Scope

在当前货代实际业务运营中，财务部门按月/按日维护统一的外币折合本位币基准汇率（中间价），不存在跨单据类型的多重独立汇率字典，亦不需要现汇买入/卖出双轨输入。原系统把"单据发生在不同时间点"抽象为 5 套独立的 `RateType`，并配套了收付双列汇率（`receivable_rate`/`payable_rate`，代码中实际生效）与继承瀑布流配置；相对本业务的实际输入方式，这些是冗余的输入面与配置面。

缺陷因果界定（避免误读）：

- **B1（核销汇率失真）**是多汇率体系的直接产物，随体系废除一并消除；
- **B3（提成分子分母汇率口径混用）**与 **D（分公司越权覆盖总部汇率）**是独立缺陷，与汇率类型无关，由 Part 2 定向修复；
- A、C1、E 组为业务审计独立发现，与本部分无耦合。

同时，近期对 `main` 分支的业务审计发现了以 **A 组（单证变更穿透业务锁、财务阻断门禁失效，P0）** 为代表的一组核心业务漏洞。

本任务采取**彻底推平重构（路径一）**策略：
1. **汇率体系彻底极简化**：废除多 `RateType` 与收付双轨，砍掉冗余的 `CustomSetting` 与 `TimeStandards` 表，全系统收敛为唯一的总部基准折本币汇率；
2. **审计缺陷闭环（C2 拆出另立任务）**：
   - **A 组 (P0)**：修复单证与订单变更穿透业务锁，激活 `BlocksExecution` 财务阻断门禁，补齐共享航程船期更新与改配的状态与锁校验；
   - **B 组 (P1/P2)**：核销单头取实际流水本币、两端差额进汇兑损益（B1）；对冲单补齐应付侧本位币与汇兑损益（B2）；提成分摊分母拉齐账单行汇率（B3）；
   - **C1 (P2)**：反核销冲减后订单有效提成净额 ≤ 0 时自动释放费用财务锁；
   - **D 组 (P1)**：汇率写操作仅限总部，删除"分公司写入重定向总部"通道；
   - **E 组 (P1)**：港口/航司/船公司等行业主数据实现"总部共享 + 本组织扩展"，杜绝新分公司候选项全空。
3. **C2（对冲结清触发提成）拆出为独立后续任务**：提成创建硬绑核销单（`VerificationID` 外键、核销分摊口径），打通对冲通道需要来源模型改造与新计算口径，属新功能开发，不阻塞本任务的 P0/P1 落地。

---

## 2. Detailed Requirements

### Part 1: 汇率体系极简重构（彻底推平）

1. **实体与 Schema 极简化**：
   - `exchange_rate_settings` 表删除 `rate_type`、`receivable_rate`、`payable_rate` 字段，收敛为单一 `rate numeric(18,8)`；
   - 唯一索引调整为 `(organization_id, from_currency, to_currency, effective_from)`；
   - 物理下线并删除 `exchange_rate_custom_settings` 与 `exchange_rate_time_standards` 两张实体表（已获用户明确授权）；
   - **存量数据迁移规则**：保留 `rate_type = 'BASE_CURRENCY'` 的行，`rate = receivable_rate`；其余类型行删除；塌缩后出现同唯一键冲突时迁移失败、报错人工处理，不做静默合并。
2. **API 契约收敛**：
   - Protobuf 契约废除 `RateType` 枚举与相关请求字段；
   - 废除 `UpdateTimeStandards` 与 `UpdateCustomSetting` RPC；
   - 汇率列表与增删改仅保留 `from_currency`, `to_currency`, `effective_from`, `effective_to`, `rate`, `is_active`；
   - 权限清单无权限码增删（时间标准/继承策略 RPC 复用 `update` 权限码），仅更新涉及"时间标准"字样的权限描述文案，并重跑 `generate:permission-keys`。
3. **汇率查询纯粹化**：
   - 领域层查询签名精简为 `ResolveRate(ctx context.Context, orgID uuid.UUID, currency string, rateDate string) (decimal.Decimal, error)`（保留 `orgID` 用于解析组织树根与基准币种）；
   - **严格限定汇率只有总部可写**：非总部组织调用 Create/Update/Disable 一律拒绝（D 组修复）；同时**删除** biz 层现存的"分公司写入重定向到总部行"逻辑——两者语义互斥，保留重定向即保留越权通道。
4. **全系统业务单据统一消费**：
   - 费用（按费用发生日）、账单（按账单日）、流水（按到账日）、发票（按开票日）统一调用单一 `ResolveRate`；
   - **提成人民币换算**（`finance_commission.go` 预览与创建两处）：原 `WRITE_OFF` 类型解析改用 `ResolveRate` 按提成生成日解析；
   - **核销（Verification）**：彻底移除汇率查询解析，单头 `base_amount` 改为 ∑ 行级 `cashflow_base_amount`；单头汇率快照四字段（`exchange_rate`/`exchange_rate_source`/`exchange_rate_date`/`exchange_rate_setting_id`）与行级 `write_off_base_amount` 随契约和 Schema 一并删除（无生产历史包袱）。
5. **汇率导入流程收敛**：
   - `exchange_rate_import.go` 移除按类型命名的中文列映射（"核销汇率"等），导入模板收敛为单一"折本币汇率"列；
   - 前端导入界面与校验提示同步收敛。
6. **前端管理页精简**：
   - 移除 RateType Tab 切换与下拉选择；
   - 输入表单收敛为单一"折本币汇率"输入框；
   - 移除时间标准设置与继承策略页面/抽屉；
   - 核销详情页移除汇率快照展示。

### Part 2: 核心业务与安全漏洞闭环

1. **A 组：单证变更路径穿透业务锁 (P0)**
   - **A-1**: `collectDocumentImpacts` 中，对费用、账单、发票、核销、提成、提成调整 6 类下游事实及 HBL 箱货分配，将 `BlocksExecution` 真正赋值为 `true`，使阻断门禁生效。
   - **A-2**: `ExecuteModeChange`（模式变更，尤其 HOUSE -> DIRECT）接入 `hasBlockingImpact`，有下游不可逆事实时直接返回 `SEA_DOCUMENT_CHANGE_BLOCKED`。
   - **A-3**: Master Amendment、House Amendment、Master Void、Mode Change 的事务内全面校验目标订单 `ensureOrderBusinessEditable`（ACTIVE + OPEN + 未业务锁定），并校验共享 MBL 成员订单锁状态。
   - **A-4**: `sea_order_change.go` 船期更新与改配流程，在锁定的订单集合上校验终止、结案与业务锁状态。
2. **B 组：结算口径平衡 (P1/P2)**
   - **B-1**: 核销单头 `BaseAmount` 严格定义为 ∑ cashflowBase，消除伪汇率带来的单头金额畸变。
   - **B-2**: 对冲单在既有 `base_currency_amount`（语义明确为**应收侧抵销本位币**，保留不动）之上，新增 `payable_base_amount` 与 `exchange_gain_loss`（应付侧本币 − 应收侧本币）两个字段，贯通 Ent Schema、迁移、Proto 契约与对冲单详情展示；同币种同汇率对冲汇差为 0 属正常结果。
   - **B-3**: `finance_commission.go` 提成聚合分母改用账单行 `BaseCurrencyAmount`，拉齐分子分母汇率基准。
3. **C1：提成净额归零释放费用锁 (P2)**
   - 财务锁谓词改为**订单级净额判定**：净额 = ∑(该订单 CONFIRMED/PAID 提成线符号金额) + ∑(该订单 CONFIRMED/PAID 调整单符号金额)；净额 ≤ 0 视为已全额冲减，自动释放费用编辑锁；
   - 谓词有**两处落点必须同步修改**：`order_fee.go` 写入拦截与 `settlement.go` 的 `financeLockedOrderPredicate`（台账列表 `finance_locked` 投影同源），避免"列表显示已解锁、写入仍被拒"。
4. **D 组：分公司越权防御 (P1)**
   - 汇率写入口在 data 层复用 `headquartersOrganizationID` 校验模式（同费用科目 `requireHeadquarters`），非总部直接拒绝；
   - 删除 biz 层写入重定向逻辑（见 Part 1 第 3 条）。
5. **E 组：新分公司主数据全空 (P1)**
   - 港口、机场、航司、船公司查询调整为"本组织 + 总部共享"并查（`organization_id IN (currentOrg, headquartersID)`），复用 data 层 `headquartersOrganizationID` 助手；
   - 候选列表按业务代码去重，**本组织行优先**，保留启用状态过滤，保证新分公司开单具备可用主数据且支持本组织私有扩展。

---

## 3. Acceptance Criteria

- [ ] **汇率模型精简**：
  - [ ] 数据库完成迁移：删除 `rate_type`、`receivable_rate`、`payable_rate`，增加单一 `rate` 字段；删除 2 张旧配置表；存量迁移执行"BASE_CURRENCY 行存活、rate=receivable_rate、唯一键冲突报错"规则；
  - [ ] Protobuf 与前端 OpenAPI 客户端重新生成，编译 0 报错；
  - [ ] 汇率管理页仅保留单汇率维护，无死字段；导入模板收敛为单一汇率列且可用；
  - [ ] 提成人民币换算按生成日经 `ResolveRate` 解析，本位币组织返回 1；
  - [ ] 权限描述文案更新并重新生成 `permissions.generated.ts`。
- [ ] **A 组业务锁与财务门禁**：
  - [ ] 当订单已有费用/账单/发票/提成时，Master/House 改单与作废 Preview 返回 `Executable: false`，Execute 报错 409 `SEA_DOCUMENT_CHANGE_BLOCKED`；
  - [ ] 模式变更在有下游事实时被拦截；
  - [ ] 订单处于业务锁定或 TERMINATING/CLOSED 时，单证变更、船期更新、改配均返回 409 拒绝。
- [ ] **B 组与 C1 结算闭环**：
  - [ ] 核销单头本币金额严格等于行级流水本位币合计，单头与行级不再携带核销日汇率快照字段；
  - [ ] 对冲单正确记录应收侧、应付侧本位币与对冲汇差，详情页可见；
  - [ ] 提成计算中收入分母与分子均以账单日为汇率基准，提成计算无波动失真；
  - [ ] 反核销后全额冲减提成的订单，费用财务锁自动释放且台账 `finance_locked` 投影同步为解锁，允许修改费用。
- [ ] **D 组与 E 组多组织安全**：
  - [ ] 非总部用户调用汇率增删改直接被拒绝；biz 层写入重定向逻辑已删除；
  - [ ] 新建分公司查询港口/机场/航司/船公司，可正常拉取总部共享候选项，重复代码去重且本组织行优先。
- [ ] **质量门禁**：
  - [ ] `go -C server test ./...` 全量通过；
  - [ ] `pnpm --dir web biome:lint` 与 `pnpm --dir web tsc` 0 报错。

## 4. 延后范围（不在本任务内）

- **C2 对冲结清触发提成**：提成来源模型（`VerificationID` 硬绑）改造、对冲分摊收入实现口径、候选入口与页面，另立独立任务实施；本任务不得为其预留半成品通道。
