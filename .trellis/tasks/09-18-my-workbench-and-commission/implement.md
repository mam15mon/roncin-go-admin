# 实施计划：我的工作台、提成方案员工分配与订单提成摘要

## 1. 实施原则

- 严格按 Proto → 生成物 → Biz → Data → Service → Web 的契约顺序推进，不手改生成文件。
- 不兼容旧组织级角色规则的新生成行为，但保留被历史提成引用的规则与提成快照；不清空数据库。
- 每组修改完成定向验证并单独提交，使用 Conventional Commits 中文提交信息。
- 功能实现完成后补测试，不采用 TDD。
- 工作台和订单列表只新增读投影；审批、冲减、发放继续调用原写端点。

## 2. 有序实施清单

### 阶段 A：提成方案员工分配 Schema 与迁移

- [ ] 保留 `server/internal/data/ent/schema/finance_commission_rule.go` 作为方案主表；新增 `FinanceCommissionRuleAssignment` Schema、方案/员工/组织 `NO ACTION` 外键、有效期字段、审计字段、CHECK 与查询索引。
- [ ] 增加正式 SQL 迁移：创建员工有效期分配表，停用没有分配的旧角色规则，保留所有历史提成与规则引用；不清空数据、不自动给同角色员工分配方案。
- [ ] 在迁移与 Ent Schema 元数据测试中验证约束名、表达式、外键删除策略和索引。
- [ ] 运行 Ent 生成，审阅所有生成差异；不得手改 `server/internal/data/ent/` 生成物。
- [ ] 定向运行 Schema/迁移测试及 `git diff --check`。
- [ ] 提交：`feat(finance): 支持提成方案分配多名员工`。

### 阶段 B：方案名单领域逻辑与并发控制

- [ ] 扩展 `internal/biz/finance_commission.go` 的方案、员工分配、过滤器和输入；新增成员资格、实际区间重叠、回溯名单变更及旧无分配规则错误。
- [ ] 修改 `internal/data/finance_commission.go`：方案列表关联当前/未来员工；关键字覆盖方案与员工；批量写入对员工 ID 排序后按 Membership → Rule 固定锁序执行，并在锁内校验跨方案重叠。
- [ ] 创建/更新/启用方案及增删员工均校验当前组织有效成员；已生效名单只允许今天或未来生效，不物理删除历史分配；驱动约束与版本冲突映射到稳定领域错误。
- [ ] 在组织成员停用路径增加当前/未来方案分配检查；存在未结束分配时拒绝停用并返回需先处理的方案，不自动删除或截断历史分配。
- [ ] 已生效方案锁定身份、口径、比例和起始日；实现【复制为新方案】及旧方案终止日衔接，禁止通过停用或修改参数追溯影响历史来源。
- [ ] 实现方案 CRUD、员工批量分配和生效日定向测试；实现 PostgreSQL 并发集成测试验证同一员工身份加入重叠方案时只有一个成功。
- [ ] 提交：`feat(finance): 强制提成方案员工区间唯一`。

### 阶段 C：计提契约与规则自动解析

- [ ] 修改 `server/api/finance/v1/settlement.proto`：规则输入/输出增加员工；候选与预览/创建改为员工 + 人员身份，由服务端解析规则。
- [ ] 重新生成服务端 API、OpenAPI 与前端客户端。
- [ ] 重构核销及对冲候选：按来源订单归属发现员工身份组合，再按来源日期解析唯一有效方案和员工分配。
- [ ] 预览、创建和确认重算均校验方案、员工分配、身份、日期与归属；保留现有来源指纹、幂等和活跃唯一索引。
- [ ] 更新方案管理抽屉和提成创建弹窗：员工远程多选、适用人数/名单、生效日变更、候选合并展示、旧无分配规则提示。
- [ ] 补充 Biz/Data/Service 与前端定向测试，覆盖多员工共享方案、员工差异方案、多身份、固定薪无候选、名单加入/退出边界、已生效方案复制衔接、离职员工历史资格、核销/对冲同构和历史快照。
- [ ] 提交：`refactor(finance): 按方案员工分配自动解析提成`。

### 阶段 D：工作台后端读模型

- [ ] 新增 `server/api/workbench/v1/workbench.proto`，定义 Overview、本人提成、本人应收和本人近期订单分页契约；不接受员工或组织改写参数。
- [ ] 新增 `internal/biz/workbench.go`、`internal/data/workbench.go`、`internal/service/workbench.go` 及 Wire/HTTP/gRPC 注册。
- [ ] 实现 `has_commission_eligibility`：本人当前/未来有效方案分配或非取消历史提成/调整；订单归属不单独开启。
- [ ] 实现本人提成分桶、预计机会、应收未结、近期海运出口订单和可靠作业待办；不同币种不裸合计。
- [ ] 复用补录申请实时审批资格，聚合财务提成/冲减摘要；只返回计数、少量记录和现有页面入口。
- [ ] 补 Biz/Data/Service 测试及权限矩阵，验证固定薪、未来方案分配、历史方案分配、仅取消记录、财务权限和组织切换。
- [ ] 生成 OpenAPI 与前端客户端，确认访问规则生成结果。
- [ ] 提交：`feat(workbench): 提供按资格组合的工作台读模型`。

### 阶段 E：订单列表提成隐私投影

- [ ] 在 `server/api/order/v1/order.proto` 增加可选提成摘要及状态事实结构，重新生成契约与前端客户端。
- [ ] 为提成用例增加当前页订单批量摘要查询；按目标组织逐个解析 `system.finance.commission.read`，普通视图在 SQL 层固定本人。
- [ ] 在 `OrderService.ListOrders` 对已授权海运出口订单页批量附加摘要；不修改订单详情默认投影，不逐行查询。
- [ ] 补普通员工、组织级财务、混合组织权限、同事有记录本人无记录、多身份/多状态的隐私与语义测试。
- [ ] 提交：`feat(order): 按提成权限投影订单列表摘要`。

### 阶段 F：工作台前端与自适应布局

- [ ] 重构 `web/src/pages/Welcome.tsx`，使用 React Query/现有请求层加载工作台；请求未完成前不预渲染零金额提成卡片。
- [ ] `hasCommissionEligibility !== true` 时完全移除提成 DOM，并让近期订单或待办顶格；纯职能人员显示欢迎、组织、账号边界和已有授权入口。
- [ ] 门禁为真时实现提成状态卡、预计说明、历史调整、回款和下钻；未来方案分配展示生效日期。
- [ ] 财务/审批卡按独立权限与响应能力组合出现，不与个人提成模块互斥。
- [ ] 海运出口订单列表增加提成事实 Tag 与下钻；普通员工和财务视图只消费服务端已裁剪投影。
- [ ] 补页面测试：false/undefined 无提成 DOM、true 空态、未来生效、组合权限、组织切换迟到响应、订单列表本人/财务投影。
- [ ] 提交：`feat(web): 上线自适应工作台与提成摘要`。

### 阶段 G：整体校验与文档收口

- [ ] 对所有修改文件运行格式化和定向 lint；运行受影响 Go/前端单测。
- [ ] 运行 `go -C server test ./...`、`go -C server vet ./...`、`pnpm --dir web tsc`、`pnpm --dir web biome:lint`。
- [ ] 运行 `pnpm run check`，验证 Proto/OpenAPI/Ent/权限生成物无漂移；本任务不改生产入口，除非检查发现构建相关风险，不固定运行 `pnpm run build`。
- [ ] 在 PostgreSQL 集成环境验证正式迁移、方案员工分配并发、工作台聚合和订单摘要查询计划；记录无法运行的集成项。
- [ ] 运行 `git diff --check` 和秘密/临时产物检查，更新任务文档中的实际验证结果。
- [ ] 提交：`test(workbench): 补齐员工提成与隐私投影验证` 或按实际剩余内容使用准确前缀。

## 3. 重点受影响文件

```text
server/api/finance/v1/settlement.proto
server/api/order/v1/order.proto
server/api/workbench/v1/workbench.proto                         # 新增
server/internal/biz/finance_commission.go
server/internal/biz/workbench.go                                # 新增
server/internal/data/finance_commission.go
server/internal/data/workbench.go                               # 新增
server/internal/data/ent/schema/finance_commission_rule.go
server/internal/data/ent/schema/finance_commission_rule_assignment.go # 新增，准确命名实施时确认
server/internal/service/settlement_commission.go
server/internal/service/order_query.go
server/internal/service/workbench.go                             # 新增
server/internal/server/http.go
server/internal/server/grpc.go
server/internal/{biz,data,service}/provider*.go 或现有 Wire 集合文件
server/migrations/<timestamp>_commission_rule_assignments.sql    # 新增
web/src/pages/Welcome.tsx
web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx
web/src/pages/finance/commissions/components/CommissionCreateModal.tsx
web/src/pages/orders/**                                          # 海运出口列表实际组件
web/src/services/roncin/**                                       # 仅生成器更新
```

实际实现前通过搜索定位 Wire/provider 的准确文件名与海运出口列表组件，不根据上述通配说明盲目创建重复文件。

## 4. 最小验证命令

开发过程中按修改阶段选择最小集合：

```bash
go -C server test ./internal/biz -run 'Commission|Workbench'
go -C server test ./internal/data -run 'Commission|Workbench|Migration'
go -C server test ./internal/service -run 'Commission|Workbench|Order'
pnpm --dir web exec vitest run <受影响测试文件>
pnpm --dir web exec biome check <受影响文件>
git diff --check
```

契约和生成阶段：

```bash
make -C server api
go -C server generate ./...
pnpm run generate:web-client
```

最终门禁：

```bash
go -C server test ./...
go -C server vet ./...
pnpm --dir web tsc
pnpm --dir web biome:lint
pnpm run check
```

## 5. 开始实施前检查

- [ ] 用户已明确批准最新的 PRD、技术设计与实施摘要，而不只是批准某一条产品决定。
- [ ] `implement.jsonl` 与 `check.jsonl` 已包含真实规范条目。
- [ ] 已执行 `task.py start`，任务状态进入实施阶段。
- [ ] 已运行 `trellis-before-dev` 读取目标包规范。
- [ ] 工作区状态已检查，用户现有改动不会被覆盖。
- [ ] PostgreSQL 集成测试连接可用性已确认；不可用时预先记录替代验证与遗留风险。
