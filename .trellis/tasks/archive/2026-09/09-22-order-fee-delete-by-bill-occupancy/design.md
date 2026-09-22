# 技术设计：订单费用按账单占用关系删除

## 1. 背景与目标

用户澄清（`../archive/2026-09/09-22-order-fee-columns-tracking/follow-up-cancelled-fees.md`）将费用「作废」升级为真删除：

- 无未取消账单占用即可删除（草稿账单也算占用）；
- 占用时明确拒绝并提示先取消对应账单；取消账单后恢复可删，不因曾建账永久禁删；
- 其他费用是否建账不影响本行删除资格；
- 删除不连带删除历史账单与金额快照；删除操作日志独立保留；
- 既有 CANCELLED 行在录入页不显示；批量清理历史数据未获授权，不做。

## 2. 存储契约（FK 调整 + 迁移）

引用 order_fees 的外键仅两处（全量扫描 schema 已核实）：

| 表 | 约束 | 处理 |
|---|---|---|
| finance_bill_lines.order_fee_id | `finance_bill_lines_order_fees_finance_bill_lines`（NOT NULL, NO ACTION） | 改为可空 + `ON DELETE SET NULL`；账单行自带费用代码/名称/数量/单价/税额/币种快照，历史展示不依赖费用行 |
| order_fee_enterprise_tags.order_fee_id | `order_fee_enterprise_tags_order_fees_enterprise_tag_links`（NO ACTION） | 改为 `ON DELETE CASCADE`；标签关联离开费用无意义 |

- Ent Schema 同步：`FinanceBillLine.order_fee_id` 去 Required（Optional）、`order_fee` 边加 `entsql.OnDelete(entsql.SetNull)`；`OrderFeeEnterpriseTag.order_fee` 边加 `entsql.OnDelete(entsql.Cascade)`；执行 `go generate`。
- 新增正式迁移 `20260922130000_order_fee_delete_by_bill_occupancy.sql`：DROP NOT NULL、置换两条外键；无数据变更，不清理既有 CANCELLED 行。
- 补录来源 `order_fees.supplement_request_id` 是费用侧外键（指向申请），删除费用不触碰该约束；申请自身快照保留，不禁删。锁定/补录审批流程不在本期改写：补录费用只存在于锁定订单，`lockOrderForFeeMutation` 的业务可编辑校验自然覆盖。

## 3. 删除事务（data 层）

`orderFeeRepo.Remove` 由「置 CANCELLED」改为物理删除，沿用既有事务与锁模式：

1. `lockOrderForFeeMutation`：订单行 ForUpdate + 业务可编辑校验 + 财务锁净额谓词（与修改/作废现状一致，权限与锁规则不变）；
2. 费用行 ForUpdate + `expectedVersion` 比对（409 已被更新语义保留）；
3. **占用复核（事务内事实）**：存在 `active = true` 且所属账单状态 ≠ CANCELLED 的账单行 → 拒绝。不检查整张订单是否有账单，不信任前端状态；
4. 通过后 `DeleteOne` 物理删除：标签关联由 DB CASCADE 清除，历史（inactive）账单行 order_fee_id 由 SET NULL 置空、快照保留；
5. 审计事件 action 改为 `order.fee.delete`，Details 保留费用代码/方向/金额/币种/前状态/前版本 + 原因（选填）。

并发（删除 vs 建账）：建账事务先按 ID 序 ForUpdate 锁费用并校验 CONFIRMED 后才插行；删除事务锁订单后锁同一费用行，二者在费用行锁上串行——建账先提交则删除的占用复核命中新活动行而拒绝；删除先提交则建账的费用存在性校验（len(fees)≠len(feeIDs)）失败。无新锁顺序，不与「先订单后费用」的既有顺序冲突（建账不锁订单）。

账单取消后删除：取消事务把费用 BILLED→CONFIRMED、账单行 active=false，随后删除的占用复核不再命中 → 可删；再建新账单则再次命中活动行 → 拒绝。

## 4. biz / service / 契约

- 新错误 `ErrOrderFeeBillOccupied = errors.Conflict("ORDER_FEE_BILL_OCCUPIED", "费用已进入未取消的账单，请先取消对应账单后再删除")`（409；账单号由页面「账单号」列与订单财务详情呈现，错误文案不带动态编号）；同步重生成 `.trellis/spec/domain/error-catalog.md`。
- `OrderFeeUsecase.Remove` 签名不变：保留先读当前状态供 `triggerAutoLock`（删除草稿仍触发结清自动锁定检查，与原作废一致）；reason 保持选填仅进审计。
- proto / OpenAPI 不改（RemoveFee 端点复用，语义由服务端实现决定），无生成物变更。

## 5. 前端（订单费用录入页）

- `OrderFeeTableTabs`：两表 request 结果在按方向过滤时同时排除 `FEE_CANCELLED`，返回数据、最近请求结果、`setAll*` 父级集合、笔数与金额消费同一有效集合（不再只金额排除）；操作列「作废」改「删除」（danger，仍仅 DRAFT/CONFIRMED 展示；BILLED 定义上即被占用）。
- `fees.tsx`：`handleCancelFee` 改为 `handleDeleteFee`——`modal.confirm` 确认（不再采集原因），成功提示「费用已删除」并 `reloadFeeTables()`（连带失效关联账单查询）；占用拒绝时由 `getErrorMessage` 透出服务端文案。
- 关联账单列中原「已作废」单元格分支随过滤移除（作废行不再进入录入表）。

## 6. 测试与验证

- 服务端集成测试（`RONCIN_INTEGRATION_DATABASE_SOURCE` 隔离 Schema 跑全量迁移链，冷启动即验证新迁移）：未建账草稿/已确认删除、占用拒绝（草稿与确认账单）、其他费用建账不影响本行、取消账单后可删、取消后再建账再拒绝、历史账单行 order_fee_id 置空且快照可读、费用标签级联清理、版本冲突、审计写入。
- 前端：录入表过滤作废行（含迟到响应/切换订单）、删除按钮与确认流、占用错误提示透出；更新既有 billTracking 测试中「已作废」断言。
- 门禁：定向 `go test`（集成库本地隔离库执行）+ `go vet` + 前端定向 vitest/biome/tsc + `check:architecture` + `error-catalog --check`；最终 `check:server`/`check:web` 按风险执行。

## 7. 回退与并行会话约束

- 工作区存在并行会话未提交改动（行内汇率联动、reason 选填、confirmWithReason 等）。共享文件（`biz/order_fee.go`、`fees.tsx`、`OrderFeeTableTabs.tsx`）沿用「HEAD 基底 + 仅本任务补丁」暂存提交，不覆盖、不纳入他人改动。
- reason 选填是并行会话对作废原因的演化；删除语义下不再采集原因，其 `Reopen`（撤回）侧的 reason 选填不受影响。
- 回退：还原本任务服务端提交即回到软作废语义（新迁移仅放宽约束，向后兼容旧代码路径）；前端回退恢复作废文案。
