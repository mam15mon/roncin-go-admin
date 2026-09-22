# 执行计划：订单人员为提成归属唯一真相

前置：`task.py start` 之后才允许改产品代码。按序执行，每步含验证。

## 顺序清单

### 后端

1. **biz 三岗校验 + 新错误**
   - `server/internal/biz/order_usecase.go`：`normalizeOrder` `creating` 分支追加 `validateOrderCommissionPersonnel`。
   - 新增标签/顺序表与 `NewOrderCommissionPersonnelMissing`（就近放 order 错误定义处）。
   - 单测：缺岗文案（顺序+去重）、齐全通过、CREATOR 不计入。
   - 验证：`go -C server test ./internal/biz/ -run 'OrderCommissionPersonnel|NormalizeOrder'`

2. **删除旧门禁**
   - `server/internal/biz/partner.go:33-62` 三件套；`organization_interceptor_test.go` 中对应用例。
   - 验证：`rg -n "PARTNER_COMMISSION_ASSIGNMENT_MISSING|NewPartnerCommissionAssignmentMissing" server/` 仅剩（应为零）。

3. **快照取数源切换**
   - `server/internal/data/order_write.go` `snapshotOrderCommissionAttributions` 按 design 2.2 重写。
   - `order_create_transaction_test.go` 重写断言。
   - 验证：`go -C server test ./internal/data/ -run 'Snapshot|OrderCreate'`

4. **草稿换客户**
   - `order_write_draft.go:298-303` 改为更新 `customer_id`；新增/改造用例。
   - 验证：`go -C server test ./internal/data/ -run 'Draft'`

5. **后端回归**：`go -C server test -p 32 ./... && go -C server vet ./...`

### 前端

6. **三岗必填**
   - `SeaPersonnelSection.tsx`：`rules` prop + `isDetail` 区分；必填文案与委托单位同级。
   - 单测：新建/详情两模式规则断言 + 空值提交阻断。
   - 验证：`pnpm --dir web exec vitest run src/pages/orders/templates/components/sea/SeaPersonnelSection.test.tsx`（文件名按实际）

7. **档案与快建降级**
   - `partners/components/BasicInfoSection.tsx`、`orders/components/PartnerQuickAddSelect.tsx` 及相关注释、测试断言。
   - 验证：`pnpm --dir web exec vitest run src/pages/partners src/pages/orders/components/PartnerQuickAddSelect.test.tsx src/pages/orders/components/PartnerQuickAddSelect.failure.test.tsx`

8. **前端回归**：`pnpm --dir web tsc && pnpm --dir web biome:lint`（改动文件）

### 种子与文档

9. **sync:dev 种子补齐客户三岗**
   - `cmd/sync-dev/main.go`：`operatorRep`/`serviceRep` 字段 + 幂等补挂。
   - 验证：本地执行 `pnpm run sync:dev` 两次，查 `partner_assignments` 三岗齐全且无重复。

10. **规格文档修订**
    - `.trellis/spec/server/backend/operating-company-commission-attribution.md`：快照源、错误矩阵、换客户行为、档案职责。
    - `.trellis/spec/domain/` 相关描述（glossary/architecture-map 中提成归属条目）。

### 收尾

11. **全栈门禁**：`pnpm run check:fast`
12. **手工验收**：按 design §5 路径走通 AC1–AC7。
13. 提交（Conventional Commits，后端/前端/种子/文档同一逻辑组可拆多条）。

## 风险文件与回滚点

- 高风险：`order_write.go`、`order_write_draft.go`（事务内逻辑）——步骤 3/4 独立可回退。
- 中风险：`normalizeOrder`（创建/更新共用）——步骤 1 必须以 `creating` 严格限定，回归覆盖 `UpdateDraft` 用例。
- 前端 `SeaPersonnelSection` 为创建/详情共用组件——`isDetail` 分支缺失会卡死详情保存，步骤 6 测试双模式覆盖。

## 启动前检查

- [ ] 规划摘要已获用户批准
- [ ] `implement.jsonl` / `check.jsonl` 已含真实条目
