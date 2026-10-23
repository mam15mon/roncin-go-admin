# 实施与验证计划

## 启动门槛

- [ ] 用户明确批准最新完整规划后，才执行 task.py start 并进入实现。
- [ ] 刷新 git 状态及任务上下文，保留其他任务；当前分支 codex/unified-order-review-lock，目标 main。
- [ ] 实施代理加载 implement.jsonl、检查代理加载 check.jsonl；遵循 trellis-before-dev、roncin-web-stack 及 Ant Design 规范。本阶段不派发实施。

## 实施顺序

1. [ ] 审计受影响引用：订单锁／整票审核／自动锁／提成财务锁、普通及批量费用写入、三类建账入口、补录／更正、解锁／生命周期、通知、种子、权限及测试。记录迁移前态，不输出秘密。
2. [ ] 修改 Proto、Ent 与权限 Manifest：统一人工审核事实、补审绑定审核代次、逐费能力、必要阻断信息；审阅正式迁移及无歧义升级范围。遇到旧记录无法转换，列出影响并请求具体数据处置，不重置数据库。
3. [ ] 用现有锁单和审核快照实现审核并锁定；移除整票待审冻结与自动结清锁触发；统一同公司同品类审核及补审资格，保留单证历史与现有解锁授权路径。
4. [ ] 统一普通费用编辑门禁和建账准入：草稿占用即禁改，未审核可建账，已审核正式费可结算；普通改费接入必要提成重算，避免放开财务锁后出现差额遗漏。
5. [ ] 收敛补录／更正入口；未建账源费才可提交／批准，待审仅阻止该原费建账，其他费用继续办理；新增／替换／调减／作废原子生效，解锁及结案感知待审申请。
6. [ ] 复用财务回退动作，完善实际依赖提示和合并单据影响范围；完整同步来源反转及提成差额。
7. [ ] 完成订单列表／详情／费用、台账、建账候选、工作台相关投影及通知文案；统一锁、费用能力和七状态，删除旧双锁入口和独立整票费用审核 UI。
8. [ ] 依次执行 go -C server generate、make -C server api、pnpm run generate:web-client、pnpm run generate:permission-keys；涉及 Wire 接线时执行项目现有生成目标，禁止手改生成物。同步开发种子及所有桩实现。
9. [ ] 功能实现后补充风险匹配测试并做独立检查；每组可验证修改提交中文 Conventional Commit，最终完整跨层验收通过再归档。

## 验证矩阵

| 场景 | 验证内容 | 验收 |
| --- | --- | --- |
| 权限／公司 | 原人及另一合格人、其他品类、跨公司、停用／撤权、自审 | AC3、AC10 |
| 普通费用 | 未锁／已锁、未建账／草稿／确认、批量操作、隐藏 API 入口 | AC1、AC2 |
| 补审 | ADD／REPLACE／调减／VOID、批准／退回／撤回、前后快照与正式明细 | AC3、AC4 |
| 建账 | 未审核先分供应商建账、锁后建账、同票待审只阻止源费 | AC2 |
| 回退 | 草稿／已开票、部分／全部核销、草稿／确认对冲、合并账单／合并发票 | AC4、AC5 |
| 生命周期 | 无自动锁、人工审核、待审阻止解锁／结案、结案后补审、终止边界 | AC6、AC7 |
| 竞争与回滚 | 审核 vs 编辑、建账 vs 编辑／申请、批准 vs 撤回／回退／发放、重复请求 | AC8 |
| 提成守恒 | 普通新增成本、部分来源回退、重建账重核销、连续调整、已付及手工奖惩 | AC9 |
| 迁移 | 全迁移链、审核依据约束、自动锁／旧待审异常明确阻断、无静默丢历史 | AC10 |
| 前端 | 状态刷新、失败保留表单、并发冲突、公司切换迟到响应、准确金额／本地时间 | AC1—AC7 |

## 验证命令与阶段

先跑直接受影响的最小集（实际文件新增／移除后同步命令）：

```bash
go -C server test -p 32 ./internal/biz ./internal/service -run 'OrderLock|OrderFee|FinanceBill|Commission'
go -C server test -p 32 -timeout 25m ./internal/data -run 'OrderLock|OrderFee|FeeCorrection|FinanceBill|FinanceInvoice|Verification|Netting|Commission' -count=1
pnpm --dir web exec vitest run src/pages/orders/fees.test.tsx src/pages/orders/use-order-lock-state.test.ts src/features/orders/fee-correction/FeeCorrectionSection.test.tsx src/pages/orders/components/fees/FeeCorrectionCreate.test.tsx
pnpm --dir web test:changed
pnpm --dir web tsc
git diff --check
```

修改文件单独 Biome 检查；测试重型文件超过 20 秒按功能拆分，不以 TDD 或重复实现式测试推进。新增普通费用变更提成用例必须验证实际金额及失败零写入。

真实 PostgreSQL 使用显式专用测试库与正式 SQL 迁移，不调用 Schema.Create，不指向开发库、不输出连接串。注入 RONCIN_INTEGRATION_DATABASE_SOURCE；迁移验收额外启用 RONCIN_POSTGRES_MIGRATION_TEST，相关子进程 DATABASE_SOURCE 指向同一隔离库。记录 PASS／FAIL／SKIP 和实际执行的并发用例，SKIP 不算验收。

最终因修改权限、Schema、跨层契约及财务原子性，运行 pnpm run check:fast，并补齐真实数据库定向验收。只有构建入口／依赖变化或明确发布验收才额外执行 pnpm run build。

## 收尾与交付

- [ ] 检查代理独立审查业务矩阵、权限、锁序、财务历史及生成物；问题修复后复跑受影响检查。
- [ ] 同步 domain/glossary.md、architecture-map.md、order-lock-and-document-version.md、order-fee-review.md、finance-commission-lock.md、order-fee-correction.md 及前后端能力导航，以实际实现更新，规划期不把提案写成现行规范。
- [ ] 删除已退役流程的运行代码和失效测试，保留必要账务／审计历史；确保没有第二套锁或兼容回退。
- [ ] 提交完整代码、生成物、正式迁移及验证记录；未运行项说明原因。验收通过后归档，推送／合并按用户交付指令执行。
