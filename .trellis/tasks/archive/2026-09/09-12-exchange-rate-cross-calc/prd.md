# 汇率交叉套算 PRD

## 1. Background & Scope

公司计划在加拿大设分公司（本位币 CAD）。当前汇率体系下总部财务只维护
`外币 → CNY` 单边基准行，而 `ResolveRate` 的目标币是**调用组织自己的本位币**
（`ResolveContext` 起点组织优先）。由此产生死角：

| 场景 | 需要的汇率 | 现状 |
|---|---|---|
| CNY 本币组织的 USD 费用 | USD→CNY | ✅ 直连行命中 |
| CAD 本币组织的 CAD 费用 | 恒等 1 | ✅ |
| **CAD 本币组织的 USD 费用** | USD→CAD | ❌ 无直连行，`ErrExchangeRateMissing`，费用无法录入 |
| **CAD 本币组织的 CNY 费用** | CNY→CAD | ❌ 同上 |
| CAD 本币组织的提成换算 | CAD→CNY | ✅ 直连行（09-12 已修的 `ResolveBaseRate` 特例，目标恒为 CNY） |

非 CNY 本位币组织今天只有"本币自身"和"目标恰为 CNY"两种解析可用，**任意第三币全部失败**。
加拿大分公司要在系统里正常录费用，交叉套算是前置条件。

## 2. Detailed Requirements

1. **套算规则**：直连行缺失时，经**总部本位币（基准币）**交叉推导——
   `from → to = (from → pivot) ÷ (to → pivot)`，pivot 为汇率表实际维护的基准币
   （树根总部本位币，通常 CNY）。两腿任一缺失即 `ErrExchangeRateMissing`（fail-closed，
   不做任何日期回退或静默 1）。
2. **直连行优先**：财务显式维护的直连行永远优先于推导值；两者同时存在时不套算、不告警、
   不覆盖。
3. **来源可审计**：套算得出的汇率在费用/账单/流水/发票快照上标记来源 `DERIVED`
   （推导），与 `SYSTEM`（直连命中）、`MANUAL`（手工覆盖）区分；解析结果需携带来源。
4. **覆盖面**：`ResolveRate`（单据折本位币）与 `ResolveBaseRate`（提成 CNY 折算）统一
   获得套算能力，全部消费方（费用/账单/流水/发票/提成/对冲快照）自动受益，无逐调用点改造。
5. **精度**：套算结果 `RoundBank(8)`；除零守卫（to→pivot 腿为零视为缺失）。
6. **无录入面变化**：汇率维护界面仍只录入 `X → 基准币` 单边行；交叉汇率是纯推导，
   不新增维护入口、不落汇率表行。

## 3. Acceptance Criteria

- [ ] CAD 本位币组织：USD 费用按 `(USD→CNY) ÷ (CAD→CNY)` 折算本位币，金额与手工推导一致；
      CNY 费用按 `1 ÷ (CAD→CNY)` 折算；
- [ ] 直连行存在时直连优先：手工在总部加一条 `USD→CAD` 行后，解析改用直连值且来源为 `SYSTEM`；
- [ ] 任一腿缺失（如财务未维护 CAD→CNY）→ `ErrExchangeRateMissing`，不做回退；
- [ ] 套算结果快照来源为 `DERIVED`，直连为 `SYSTEM`，手工覆盖仍为 `MANUAL`；
- [ ] CNY 本位币组织全部行为不变（回归：现有汇率/费用/账单/流水/发票/提成测试全绿）；
- [ ] **质量门禁**：`go -C server test ./...`（含集成）通过。

## 4. Out of Scope

- 汇率表直接维护 `X→Y` 交叉行的录入界面（财务仍只录单边基准行）；
- 历史快照重算/回填（存量单据快照不追溯套算）；
- 中间币选择策略（固定经总部基准币单跳，不做多跳寻优）。

## 5. 前置条件

- 等 09-12-netting-commission-and-followups 合并归档后实施（两者都触碰
  `biz/exchange_rate.go`，避免并行写冲突）。
