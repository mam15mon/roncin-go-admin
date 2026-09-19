# 实施计划：员工月度提成申请与财务整批审批

## 1. 实施顺序

### 阶段 A：申请模型、契约与迁移

- [x] 在 `server/internal/data/ent/schema/` 新增申请头和申请明细 Schema，补充状态 CHECK、组织/员工/提成 `NO ACTION` 外键、月度唯一索引、明细唯一约束及查询索引。
- [x] 修改 Proto 源文件，定义员工本人申请预览/提交、申请列表/详情、财务批准/驳回和明细结构；接口不接受可改写的员工 ID/组织 ID作为授权来源。
- [x] 生成 Go/API/OpenAPI/前端客户端，不手改生成物；新增权限时同步 Manifest 和权限键生成。
- [x] 新增正式 SQL 迁移，验证空库迁移、现有提成历史不变、无虚假历史申请回填。

### 阶段 B：完整候选与员工提交

- [x] 在 Biz 抽取“截止日以前、当前员工、当前组织、未取消、未被有效申请占用”的完整申请候选解析，复用既有提成来源、方案员工分配、费用和回款校验。
- [x] 增加申请提交/重提领域用例和 Data 事务：自然月门禁、月度唯一键、来源固定锁序、`DRAFT` 提成复用/受控生成、明细快照和幂等冲突。
- [x] 明确区分工作台有界预计金额与申请完整候选，禁止提交路径使用 20 条估算上限。
- [x] 补重复点击、并发提交、跨组织/跨员工参数、空候选、当前月和迟到历史来源测试。

### 阶段 C：财务列表与整批审批

- [x] 实现按目标组织权限读取申请头、明细和导出；员工只能读取本人申请。
- [x] 实现整单批准/驳回事务：申请头、明细、订单和提成固定锁序；批准全部成功才把明细提成转为 `CONFIRMED`，驳回只改变申请状态并保存原因。
- [x] 实现驳回原申请重提，禁止新建同员工同月份替代申请；保留每次提交/决策审计和版本。
- [x] 补并发批准、重复批准、批准时来源指纹变化、草稿费用阻断、整批回滚和权限变更测试。

### 阶段 D：工作台与财务前端

- [x] 在工作台增加“待申请 / 本月累计中 / 审批中 / 已批准”按自然月摘要和明细入口；未结束月份隐藏申请提交能力，服务端仍强制拒绝。
- [x] 员工申请页默认展示截至上月末的全部合格未申请明细、覆盖月份和总额；提交后锁定原申请，不展示迟到数据追加按钮。
- [x] 财务页面以申请批次为行，支持整单批准/驳回和驳回原因，明细下钻显示来源、方案快照、人员身份和金额；不提供部分批准控件。
- [x] 补组织切换、本人隐私、空月份、跨月累计、驳回重提和迟到历史来源的前端测试。

### 阶段 E：整体校验与收口

- [x] 运行受影响 Go 测试、生成检查、前端定向测试、Biome/tsc 和 `git diff --check`。
- [x] 在隔离 PostgreSQL 库重放正式迁移，验证月度唯一键、申请头/明细外键、并发提交与整批回滚。
- [x] 用代表性多月、多来源数据检查完整候选查询不是工作台有界估算，并记录查询计划与事务耗时。
- [x] 更新任务实际验证结果，运行风险匹配的最终门禁；不引入银行或工资相关代码。

## 2. 重点文件（实现前以搜索结果为准）

```text
server/api/finance/v1/settlement.proto 或新增 commission_application.proto
server/api/workbench/v1/workbench.proto
server/internal/biz/finance_commission_application.go              # 新增
server/internal/data/finance_commission_application.go              # 新增
server/internal/data/ent/schema/finance_commission_application.go   # 新增
server/internal/data/ent/schema/finance_commission_application_line.go # 新增
server/internal/service/settlement_commission_application.go        # 新增
server/internal/service/workbench.go
server/migrations/<timestamp>_commission_applications.sql
web/src/pages/workbench/**
web/src/pages/finance/commissions/**
web/src/services/roncin/**                                           # 仅生成器更新
```

## 3. 最小验证命令

```bash
go -C server test ./internal/biz -run 'Commission|Workbench'
go -C server test ./internal/data -run 'Commission|Workbench|Migration'
go -C server test ./internal/service -run 'Commission|Workbench'
pnpm --dir web exec vitest run <受影响测试文件>
pnpm --dir web exec biome check <受影响文件>
git diff --check
```

最终门禁按风险执行：

```bash
go -C server test ./...
go -C server vet ./...
pnpm --dir web tsc
pnpm --dir web biome:lint
pnpm run check
```

## 4. 开始实施前检查

- [x] PRD 中自然月、跨月累计、整批批准/驳回、迟到提成顺延和驳回原单重提均已确认。
- [x] `design.md`、`implement.md`、`implement.jsonl`、`check.jsonl` 已完成并通过上下文校验。
- [x] 用户已批准最新规划摘要后，才执行 `task.py start`；本阶段不启动任务。

## 5. 实际验证结果（阶段 E 收口，2026-09-19）

集成验证使用一次性 PostgreSQL 库 `roncin_phase_e_it`（凭据取自 `.env.local` 同源本地账号，
每个测试在库内创建隔离 Schema 并重放完整迁移链；采样探针文件用后删除，一次性库在收口后删除）。

### 5.1 门禁命令与结果

| 检查 | 命令 | 结果 / 耗时 |
| --- | --- | --- |
| Go 静态检查 | `go -C server vet ./...` | PASS，30.6s |
| Go 全量测试（含真实库集成） | `RONCIN_INTEGRATION_DATABASE_SOURCE=… go -C server test ./... -timeout 20m` | 本任务相关包全绿（service 2.1s 等）；`internal/data` 768.6s 内 3 个存量失败（见 5.5），逐项在任务前基线 `5c086efa` 完全一致复现 |
| 数据层 `Postgres$` 全量套件 | `go -C server test ./internal/data -run 'Postgres$'`（全量 verbose 日志核对） | 79 PASS / 0 FAIL |
| 迁移链冷启动重放 | `RONCIN_POSTGRES_MIGRATION_TEST=1 go -C server test ./internal/platform/migration -run 'TestPostgres'` | `TestPostgresColdStartMigration` PASS（4.5s，隔离 Schema 全链重放 + Ent 全表存在性核对，含申请两张新表）；`TestPostgresChecksumRepairMigration` PASS；4 个海运迁移用例失败为存量（见 5.5） |
| 申请 Schema 专项 | `TestCommissionApplicationSchemaPostgres` | PASS：状态/非负 CHECK、组织/员工/提成 `NO ACTION` 外键与决策人 `SET NULL`、月度唯一键（23505）、提成事实全局唯一索引、删除 `20260920110000_commission_applications.sql` 后重放不改既有数据且零回填 |
| 申请集成用例（11 个） | Submit/并发提交/门禁/重提/来源冲突/财务读取/批准/驳回重提批准/整批回滚/并发批准驳回 | 全部 PASS，合计 90.5s：并发 4 提交恰 1 成功 3 冲突、并发批准/驳回恰一成功、指纹漂移与草稿费用阻断整单回滚且申请头保持待审 |
| 前端类型检查 | `pnpm --dir web tsc` | PASS，12.8s |
| 前端 Lint | `pnpm --dir web biome:lint` | PASS（exit 0）；7 条存量 info 级风格建议 + `.agents/skills` 符号链接的环境性内部诊断，均非阻塞 |
| 前端定向套件 | `vitest run` workbench 2 文件 + finance/commissions 2 文件 | 33/33 PASS，24.9s（版本冲突提示、驳回原因必填、无 `commission.manage` 隐藏动作、明细下钻、组织切换迟到响应、月度申请摘要分组等） |
| 生成物漂移 | `check:permission-keys` / `check:proto-constants` / `lint:proto` / `pnpm run generate:web-client` 后 `git status` | 302 权限键、75 枚举 + 7 错误原因域、buf lint 全部无漂移；Web 客户端重新生成后零差异 |
| 全量最终门禁 | `pnpm run check` | PASS：web lint + 全量 vitest（152 文件 / 850 用例通过，12 跳过）+ buf lint + `go test ./...` + `go vet` + 漏洞审计（豁免 GO-2026-6452 一项） |
| 工作区检查 | `git diff --check` | PASS |

### 5.2 端到端 403 冒烟

服务层 `TestCommissionApplicationServicePermissionGate` 在既有 `ErrPermissionDenied` 断言基础上，
本期补强 kratos `errors.Code == 403`（`http.StatusForbidden`）冒烟断言：仅持
`system.finance.commission.read` 的主体调用 `Approve/RejectCommissionApplication` 必须得到
HTTP 403。路由层 `access_rules_gen.go` 对两操作强制 `system.finance.commission.manage`
（组织范围），双层口径一致。测试 PASS。

### 5.3 性能采样基线（design §7）

代表性数据量：单员工 12 个历史自然月 × 每月 250 个核销来源 ≈ 3000 条提成行
（一次性库、空闲负载；临时探针测试测量后删除）。无既定目标值，仅记录基线：

| 操作 | 耗时 |
| --- | --- |
| 完整候选解析（`ListMyCandidates`，全量解析 + 分页） | 29.9s |
| `Submit` 事务（3000 明细固化，含受控创建 DRAFT） | 1m45.9s |
| `Approve` 整批事务（3000 明细整批复核 + 转确认） | 40.2s |

结论：完整候选解析确认为无 20 条估算上限的全量解析（3000 行全量纳入申请）。三者随来源数
近似线性；按现实规模（员工每月数十来源、累计约数百行）外推 Submit 约在秒级到十秒级，可接受。
未引入缓存/快照；若未来真实数据超标，优先按 design §7 做来源批量查询与索引优化。

### 5.4 期间发现并处理的问题

- 性能探针初版因 `time.Format` 布局误用（`"2006-01-15"` 中的 `15` 为小时占位符）生成非法
  日期 `YYYY-MM-00`，导致月窗口解析失败、候选为 0——探针自身缺陷，非实现缺陷；修正后
  全量解析通过。该教训：生成日期字符串必须使用 `2006-01-02` 布局或 `AddDate`。

### 5.5 遗留风险与存量问题（非本任务引入，均有基线复现证据）

1. `internal/data` 全量套件下 3 个存量失败（基线 `5c086efa` 与 HEAD 签名完全一致）：
   - `TestAdminCreateOrganizationInitializesEnabledCurrencies`：测试传入空 `Action` 的
     `AuditEvent`，被 `audit_logs.action NotEmpty` 校验拒绝；单独运行同样失败，属测试用例
     自身缺陷。
   - `TestFeeSupplementSchemaFKMetadata` / `TestFeeSupplementSourceUniqueIndexMetadata`：
     单独运行 PASS、全量套件内确定性失败（补录申请相关表的 Ent 生成元数据外键/唯一索引在
     进程内被前置测试就地改写，属 AGENTS.md 已警示的全局表指针污染类问题）；污染源未深挖。
2. `internal/platform/migration` 4 个海运迁移用例全量运行失败（基线一致复现）：阶段性迁移
   用例对后续迁移演进后的索引/结构预期漂移，非本任务引入。
3. 上述存量问题不影响本任务任何验收路径；建议另开 `test(server):` 修复任务处理。
