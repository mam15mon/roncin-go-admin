# 汇率管理完全下沉分公司：实施计划

## 前置状态

当前 planning，本文件是待审阅执行计划。仅补齐文档，不执行 task.py start、不改产品代码、不运行数据库写入。
实施前用户审阅最新 PRD/design/implement；本计划不以文档齐全代替实施批准。

## 1. 实施上下文与范围确认

- [ ] 读取根目录及目标目录 AGENTS.md，确认 git status，保留用户未提交改动。
- [ ] 加载 implement.jsonl/check.jsonl 中真实规范与任务设计；读 trellis-before-dev、前端相关栈规范。
- [ ] 确认公司历史周继承、手工覆盖保持，公共基线仅留作旧事实、不参与新取值的方案已获审阅。
- [ ] 用 rg 复查 ResolveRate、PivotCurrency、allowBaseline、OrganizationIDIsNil、RequireBaselineWrite 及汇率相关 CLI/seed 引用，形成改动文件清单；不要改其他主数据公共基线能力。

## 2. 公司权限与维护闭环（R1、R2）

- [ ] Manifest 将汇率权限纳入公司经营范围，更新描述、工作台权限投影和授权测试；保持权限键，避免波及费用目录。
- [ ] ResolveContext 仅解析合法公司；部门定位公司；移除系统本币依赖。
- [ ] List/Create/Update/Disable 使用 OwnerOrganizationID；移除 allowBaseline，仓储查询以记录 ID + 公司 ID 限定。
- [ ] 模板下载、预检、确认、批次读取、抓取预览、同步均拒绝系统工作台，确认服务端 principal 权限检查完整。
- [ ] 导入/同步行只写同一公司，保留预检、幂等、锁与事务保护；核对审计归属和提醒收件人。
- [ ] 前端 access 和维护面板仅显示公司配置，移除公共参考行、系统基线按钮与说明，保留现有页面和 request/actionRef 范式。

## 3. 解析及业务调用收口（R2—R4）

- [ ] 删除公共基线直连/历史/套算路径及内部 PivotCurrency 参数，同步接口实现和测试桩。
- [ ] 保留同币种 1、公司当周及历史周、双轨方向取价与原有合法手工覆盖。
- [ ] 核对 order_fee、finance_bill、finance_bill_batch、finance_cashflow、finance_invoice 的组织来源及事务上下文。
- [ ] 核对提成、核销、对冲消费快照和跨组织原币记账路径，不新增实时折算。
- [ ] 统一缺失提示，不能再提示系统管理员维护公共汇率。
- [ ] 修改历史汇率导入工具及开发种子，禁止无公司目标写公共基线；不自动运行导入或清理数据。

## 4. 契约与针对性验证

先完成实现，再按风险补测试，禁止 TDD。每组验证成功后提交可独立审阅的变更；紧耦合接口与调用方不得拆成不能编译的提交。

| 场景 | 预期 | 验收映射 |
| --- | --- | --- |
| system 直接请求所有维护 API | 权限拒绝，菜单不可见 | 1 |
| 公司 A/B 同币种不同报价 | 查询与计算各自独立，跨公司 ID/token 不可用 | 2、3 |
| 部门属于 A | 经权限校验读取/维护 A，无部门汇率行 | 2、3 |
| 公司当周缺失、有历史公司行 | 继承公司历史，来源及提示保持 | 3 |
| 公司缺失、只有公共直连/交叉价 | 明确缺失，不产生公共结果 | 4、5 |
| 本币、应收/应付、非 CNY 公司 | 恒等与方向价格正确，外部报价同步仍可用 | 3、5 |
| 导入确认/重复同步/失败 | 公司隔离、原子回滚、既有幂等保持 | 2 |
| 切换公司/并发改价/已有快照 | 页面不串数据，共享事务不撕裂快照，存量不重算 | 3、6 |
| 旧公共记录仍存在 | 不删除，已有引用保留，新业务不使用 | 4、6 |

建议定向后端检查（新增测试命名需包含相关业务前缀）：

```bash
go -C server test -p 32 ./internal/access ./internal/biz ./internal/service ./internal/data -run 'ExchangeRate|FinanceBill|FinanceInvoice|FinanceCashflow|OrderFee|Workspace' -count=1
```

真实库测试必须配置专用 `RONCIN_INTEGRATION_DATABASE_SOURCE`，使用现有测试辅助及有序正式 SQL 迁移；禁止针对开发业务库执行清理，禁止 Schema.Create 并发建表。检查测试输出，SKIP 不算集成通过；无法运行时列明缺口。

前端先运行修改/新增测试文件（相对 web/ 的真实路径）；现有同步测试：

```bash
pnpm --dir web exec vitest run src/pages/finance/exchange-rates/components/ExchangeRateSyncModal.test.tsx
pnpm --dir web test:changed
pnpm --dir web tsc
```

对修改文件运行 Biome；补足 access、列表维护与组织切换场景，不能只跑同步弹窗就宣称前端验收完成。

有 proto 变更时：

```bash
make -C server api
pnpm run generate:web-client
```

Manifest 变更后：

```bash
pnpm run generate:permission-keys
pnpm run check:permission-keys
```

若方案仍不改 Schema，无需生成结构迁移；实际引入结构变更则补齐 Ent 源、生成物、正式迁移、空库迁移验证后再提交。

## 5. 最终检查与交付

- [ ] Trellis 检查审阅：重点权限投影、公司归属、公共基线残留入口、事务读取、CLI 和快照保护。
- [ ] 更新 `.trellis/spec/server/backend/exchange-rate-single-rate.md`、后端索引及受影响的组织主数据规范，移除旧公共兜底描述，保留已存来源展示语义。
- [ ] 完整任务涉及权限及跨层折算，执行 `pnpm run check:fast`；普通 UI 改动不额外构建，只有涉及构建配置或明确发布验收才运行 build。
- [ ] `git diff --check`，核对无秘密、无手改生成物、无无关修改。
- [ ] 中文 Conventional Commits 提交；记录定向/全量/真实库结果与未运行原因。
- [ ] 经检查通过再归档任务，记录进度和提交号。

## 风险停止点与回退

发现需要删除公共记录、批量复制公共价格、改变公司内历史继承/人工覆盖或新增锁协议时，先回到设计审阅，不能自行扩大需求。
若发现跨公司归属泄漏或快照被重算，必须修正后才能完成验收。数据库测试不可用不等于行为已验证。
回退以本任务提交组为单位，不 reset 用户改动、不删除数据库；回退会恢复旧公共行为，必须明确说明，不能自动启用。
