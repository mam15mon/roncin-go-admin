# 对冲提成通道与遗留修补 Implementation Plan

> 自 `main` 拉取 `feat/netting-commission-and-followups` 分支实施。

## 1. Phase Overview

| Phase | Description | Key Deliverables | Risk |
|---|---|---|---|
| **Phase 1** | C2 后端原子单元（Schema/迁移/Proto/biz/data/service + 两个修补 + 测试） | 来源模型、计算管线推广、冲减联动、新候选 RPC、PreviewSplit 门禁、admin 测试迁移 | High（契约+Schema） |
| **Phase 2** | C2 前端 | 对冲提成 Tab、来源展示 | Medium |
| **Phase 3** | 全量门禁 + spec 沉淀 | check:server/check:web、规范更新 | Low |

Phase 1 不再细拆：verification 字段改 Optional 后 service/biz/data 会立刻编译不过，
契约、Schema 与调用方必须一次到位（与 09-11 Phase 1+2 原子化同理）。

## 2. Detailed Phase Tasks

### Phase 1: C2 后端 + 两个修补（单原子单元，只改 server/）

1. **Ent Schema**（finance_commission.go）：verification_id/no 改 Optional，新增
   netting_id/netting_no + edge；索引按 design 2.1 拆部分索引/唯一约束；
   `go -C server generate`。
2. **迁移 SQL**：DROP NOT NULL → 加列加 FK → 重建部分索引；存量核销行不动。
3. **Proto**（settlement.proto）：Preview/Create 的 verification_id 改 optional + 新增
   netting_id；新增 ListCommissionNettingCandidates RPC（权限
   system.finance.commission.manage）；CommissionCalculation/FinanceCommission 消息
   verification 字段 optional + 新增 netting 字段；`make -C server api`。
4. **biz**：Create/Preview 输入二选一校验；计算仓储入口推广为按来源加载（核销 ACTIVE
   RECEIVABLE / 对冲 CONFIRMED RECEIVABLE 分摊 → 同一 orderRealized 聚合）；
   fingerprint 加来源段；netting Reverse 接入 PlanCommissionReversal 公共助手。
5. **data**：读写映射补字段；ListNettingCandidates 候选查询；创建事务幂等与锁序沿用
   既有模式（多行主键序）。
6. **service**：DTO 二选一校验与映射；新 RPC handler。
7. **拆票与操作门禁**：
   - `PreviewSplit`：GetSplitContext 后对来源订单执行内容门禁，未通过追加 ValidationErrors（design 2.5），不改 Execute 语义；
   - `GetChangeActions`：同步检查订单业务锁，锁定时 CanSplit/CanReassign 置 false 并追加锁定原因。
8. **非 CNY 本币提成换算修复**：
   - `Preview` / `Create` 时若 `BaseCurrency != CNY`，改查 `BaseCurrency → CNY` 正向汇率；
   - `ResolveCommissionCNYRate` 移除倒数，直接使用正向汇率，单测更新。
9. **admin 测试迁移**：TestAdminEmployeeLifecyclePostgres 夹具迁 getIntegrationData。
10. **测试**：
   - biz：二选一校验、对冲来源计算金额（分摊比例 × 账单行本位币，正/负/多单）、
     指纹来源段、幂等；非 CNY 本币提成正向汇率快照与金额换算；
   - data/集成：纯对冲结清 → 候选 → 创建 → 金额断言；对冲 REVERSE → 未支付取消 /
     已支付冲减 → 费用锁随净额释放；同来源幂等去重；核销来源回归；
     PreviewSplit 锁定/终态预览 ValidationErrors + Execute 409 不变；
     GetChangeActions 业务锁返回 false；
     admin 生命周期集成在隔离 schema 通过；
   - service：新 RPC 映射、二选一 400。
11. **验证**：build/vet/test 全量 + data 集成全量（隔离 schema）+ migrate:dev。

### Phase 2: C2 前端

1. 提成页「核销提成 / 对冲提成」Tab；对冲候选列表 → 员工/规则候选 → 预览 → 创建
   （请求体 nettingId）；交互与组件复用核销流程。
2. 提成列表/详情来源展示（核销单号或对冲单号）。
3. 验证：tsc 全绿、biome、定向 vitest（改写断言旧必填字段的用例）。

### Phase 3: 门禁与收尾

1. `pnpm run check:server` + `pnpm run check:web`；
2. spec：finance-commission-lock.md 补来源模型与对冲冲减；order-lock spec 的预览拦截
   家族补 PreviewSplit 与 GetChangeActions；index 登记；
3. 提交分组：
   - `feat(commission): 对冲结清触发提成与非CNY本币折算修复`
   - `fix(order-change): 拆票预览与操作查询提前拦截订单业务锁`
   - `test(admin): 员工生命周期集成测试迁隔离 schema`
   - `feat(web): 对冲提成入口与来源展示`
