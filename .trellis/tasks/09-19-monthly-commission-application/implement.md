# 实施计划：员工月度提成申请与财务整批审批

## 1. 实施顺序

### 阶段 A：申请模型、契约与迁移

- [ ] 在 `server/internal/data/ent/schema/` 新增申请头和申请明细 Schema，补充状态 CHECK、组织/员工/提成 `NO ACTION` 外键、月度唯一索引、明细唯一约束及查询索引。
- [ ] 修改 Proto 源文件，定义员工本人申请预览/提交、申请列表/详情、财务批准/驳回和明细结构；接口不接受可改写的员工 ID/组织 ID作为授权来源。
- [ ] 生成 Go/API/OpenAPI/前端客户端，不手改生成物；新增权限时同步 Manifest 和权限键生成。
- [ ] 新增正式 SQL 迁移，验证空库迁移、现有提成历史不变、无虚假历史申请回填。

### 阶段 B：完整候选与员工提交

- [ ] 在 Biz 抽取“截止日以前、当前员工、当前组织、未取消、未被有效申请占用”的完整申请候选解析，复用既有提成来源、方案员工分配、费用和回款校验。
- [ ] 增加申请提交/重提领域用例和 Data 事务：自然月门禁、月度唯一键、来源固定锁序、`DRAFT` 提成复用/受控生成、明细快照和幂等冲突。
- [ ] 明确区分工作台有界预计金额与申请完整候选，禁止提交路径使用 20 条估算上限。
- [ ] 补重复点击、并发提交、跨组织/跨员工参数、空候选、当前月和迟到历史来源测试。

### 阶段 C：财务列表与整批审批

- [ ] 实现按目标组织权限读取申请头、明细和导出；员工只能读取本人申请。
- [ ] 实现整单批准/驳回事务：申请头、明细、订单和提成固定锁序；批准全部成功才把明细提成转为 `CONFIRMED`，驳回只改变申请状态并保存原因。
- [ ] 实现驳回原申请重提，禁止新建同员工同月份替代申请；保留每次提交/决策审计和版本。
- [ ] 补并发批准、重复批准、批准时来源指纹变化、草稿费用阻断、整批回滚和权限变更测试。

### 阶段 D：工作台与财务前端

- [ ] 在工作台增加“待申请 / 本月累计中 / 审批中 / 已批准”按自然月摘要和明细入口；未结束月份隐藏申请提交能力，服务端仍强制拒绝。
- [ ] 员工申请页默认展示截至上月末的全部合格未申请明细、覆盖月份和总额；提交后锁定原申请，不展示迟到数据追加按钮。
- [ ] 财务页面以申请批次为行，支持整单批准/驳回和驳回原因，明细下钻显示来源、方案快照、人员身份和金额；不提供部分批准控件。
- [ ] 补组织切换、本人隐私、空月份、跨月累计、驳回重提和迟到历史来源的前端测试。

### 阶段 E：整体校验与收口

- [ ] 运行受影响 Go 测试、生成检查、前端定向测试、Biome/tsc 和 `git diff --check`。
- [ ] 在隔离 PostgreSQL 库重放正式迁移，验证月度唯一键、申请头/明细外键、并发提交与整批回滚。
- [ ] 用代表性多月、多来源数据检查完整候选查询不是工作台有界估算，并记录查询计划与事务耗时。
- [ ] 更新任务实际验证结果，运行风险匹配的最终门禁；不引入银行或工资相关代码。

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

- [ ] PRD 中自然月、跨月累计、整批批准/驳回、迟到提成顺延和驳回原单重提均已确认。
- [ ] `design.md`、`implement.md`、`implement.jsonl`、`check.jsonl` 已完成并通过上下文校验。
- [ ] 用户已批准最新规划摘要后，才执行 `task.py start`；本阶段不启动任务。
