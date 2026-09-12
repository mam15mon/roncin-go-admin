# 汇率交叉套算 Implementation Plan

> 前置条件：09-12-netting-commission-and-followups 已合并归档（两者都改
> `biz/exchange_rate.go` / `data/exchange_rate.go`，必须串行）。
> 自 main 拉取 `feat/exchange-rate-cross-calc` 分支实施。

## Phase 1: 后端套算与来源传播（单原子单元，server/）

1. `data/exchange_rate.go`：`ExchangeRateContext` 加 `PivotCurrency`（根节点处捕获）；
   新增 `resolveWithCross`（直连优先 → 两腿套算 → fail-closed/冲突沿用），返回携带来源；
   `ResolveRate`/`ResolveBaseRate` 的 repo 实现改走该入口。
2. `biz/exchange_rate.go`：两个用例方法返回 `ResolvedRate{Rate, Source}`（或双返回值，
   按仓库惯例）；调用方签名同步。
3. 快照来源传播：order_fee / finance_bill / finance_cashflow / finance_invoice 创建与
   币种变更路径，`exchange_rate_source` 由固定 SYSTEM 改为透传解析来源（MANUAL 分支不动）。
4. 测试（design 第 4 节全量）：biz 套算金额/直连优先/腿缺失/来源三态；data 集成
   真实行套算往返与 CNY 组织回归；精度与除零。
5. 验证：`go -C server build/vet/test ./...` + data 集成全量（隔离 schema）。

无 proto/Schema/迁移/前端变更（DERIVED 枚举值已存在；维护界面不感知套算）。

## Phase 2: 门禁与收尾

1. `pnpm run check:server`；
2. spec：`exchange-rate-single-rate.md` 增补「交叉套算」节（直连优先、单跳 pivot、
   fail-closed、DERIVED 来源）；index 描述同步；
3. 提交：`feat(exchange-rate): 直连缺失时经基准币交叉套算并传播 DERIVED 来源`。
