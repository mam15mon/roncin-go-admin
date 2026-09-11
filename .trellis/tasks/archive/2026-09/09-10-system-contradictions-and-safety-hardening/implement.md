# 系统核心业务矛盾与安全加固实施计划

> 本文件是执行顺序，不代表已经授权实施。必须在用户确认 PRD/design 后再进入代码修改。

## 0. 实施纪律

- 开始每阶段前检查 `git status`，只修改该阶段列出的文件，保留用户未提交改动。
- 不使用 TDD；先实现，再按风险补定向测试。
- 每阶段形成一组可独立验证的修改并提交；不得把五个领域压成一个巨型提交。
- 事务统一使用 `Data.WithTx`，仓储读取使用事务 client；多行加锁按 UUID 排序。
- 生成文件只由生成命令产生。本计划按当前设计不需要 Proto、Ent Schema 或权限 Manifest 变更。

## 阶段一：订单内容门禁与生命周期

目标提交：`fix(order): 加固生命周期写入门禁与退关关系`

### 1.1 错误与内容门禁

- [ ] 在 `server/internal/biz/order.go` 增加 TERMINATING、TERMINATED、CLOSED 的稳定领域冲突错误，中文含义准确。
- [ ] 将 `ensureOrderBusinessEditable` 重命名/重构为明确的内容写门禁，按 PRD 决策校验生命周期和业务锁。
- [ ] 用 `rg` 枚举现有调用点，记录“内容写入口/生命周期命令”分类；内容入口继续复用统一门禁。
- [ ] 删除 `order_release_pod.go` 的重复门禁并复用统一实现。

### 1.2 生命周期流转

- [ ] `TransitionStatus` 显式保留 ACTIVE + OPEN + unlocked 约束。
- [ ] `TransitionTermination` 移除内容门禁，按来源状态执行专属锁/状态校验，确保完成、取消和恢复路径不被误封。
- [ ] 目标 TERMINATED 时结束活动 SE Link，写结束事实并推进 Link 版本；异常活动 Link 数量 fail-closed。
- [ ] `TransitionClosure` 移除内容门禁，保留事务内版本/来源状态/readiness 重验，允许锁定放单订单结案。
- [ ] 核对 `orderAllowedActions` 与后端状态图一致；只有发现实际不一致时才修改返回动作。

### 1.3 汇总查询

- [ ] `ListConsolidationSummaries` 成员查询增加 Order ACTIVE 条件。
- [ ] 将 `WithShippingDocuments` 替换为当前 `SeaHouseBill` 读取，稳定去重 house numbers。
- [ ] 确保成员数和件重尺使用相同过滤后的订单集合。

### 1.4 测试与规范

- [ ] 补充内容门禁表驱动测试，覆盖代表性子资源和四类状态。
- [ ] 补充 PostgreSQL 集成测试：锁定后结案、终止完成/取消/恢复、Link 原子结束、不自动恢复、汇总过滤与 HBL。
- [ ] 更新 `.trellis/spec/server/backend/order-lock-and-document-version.md` 中管理员资格与内容生命周期门禁契约。
- [ ] 运行订单相关定向测试、`go test` 目标包、`git diff --check`。

阶段门禁：ORD-01、ORD-02、ORD-03、ORD-04 全部通过后提交。

## 阶段二：财务台账与核销候选

目标提交：`fix(finance): 统一结清口径并下推核销候选过滤`

### 2.1 台账有效结清金额

- [ ] 修改 `settlement.go` 的金融进度 SQL 谓词，加入有效 confirmed netting allocation。
- [ ] 为列表预加载有效核销与有效对冲关系，分别聚合后相加。
- [ ] 将 Biz 进度解析函数中的 `verifiedAmount` 语义改为 `settledAmount`，不改变外部枚举。
- [ ] 提取列表/详情共用投影，补齐详情的 bill no、financial progress、finance locked。

### 2.2 候选过滤下推

- [ ] 账单候选设置 `OnlyUnsettled: true`。
- [ ] 在 `FinanceCashflowFilter` 增加内部 `OnlyUnverified`，data Count/List 共用余额谓词。
- [ ] 候选资金流水设置 `OnlyUnverified: true`，保留 Go 层防御性余额检查。
- [ ] 确认组织、方向、往来单位、币种条件与余额谓词同时生效。

### 2.3 测试

- [ ] 单元/SQL 测试覆盖核销、对冲、混合、撤销后的列表过滤与状态。
- [ ] PostgreSQL 集成测试断言列表/详情投影一致。
- [ ] 账单和流水分别构造 `> 200` 已结清记录 + 排序靠后的未结清记录，断言候选可见。
- [ ] 运行财务相关定向测试、目标包测试、`git diff --check`。

阶段门禁：FIN-01、FIN-02、FIN-03 全部通过后提交。

## 阶段三：发票终态与账单并发

目标提交：`fix(finance): 收紧发票终态并同步账单版本`

### 3.1 后端状态机

- [ ] `Cancel` 只允许 DRAFT/ISSUED；显式拒绝 CANCELLED/RED_FLUSHED。
- [ ] `RedFlush` 继续只允许 ISSUED。
- [ ] 提取两个命令共用的活动账单释放辅助逻辑，避免状态判断仍留在辅助函数内。

### 3.2 事务与版本

- [ ] 按 Invoice → active links → sorted Bills 固定顺序 `FOR UPDATE`。
- [ ] 验证关联账单数量和组织；释放 link 后为每张 Bill `version + 1`，校验受影响行数。
- [ ] 保证 Invoice、links、Bills、audit 任一步失败整体回滚。
- [ ] 与 FinanceBill.Cancel 并发验证锁序；若发现反向锁等待，在本阶段统一修正，禁止重试掩盖。

### 3.3 前端与测试

- [ ] 区分草稿“取消”和已开票“作废”的按钮、确认标题、说明、原因 placeholder、成功提示。
- [ ] 已开票作废弹窗增加线下税务条件危险提示；保留独立红冲入口。
- [ ] PostgreSQL 集成测试覆盖状态矩阵、账单版本、link 释放、审计回滚、旧版本冲突和并发双命令。
- [ ] 运行发票定向后端测试、前端单文件测试、修改文件 Biome、`git diff --check`；类型受影响时运行 `pnpm --dir web tsc`。

阶段门禁：INV-01、INV-02 全部通过后提交。

## 阶段四：权限范围与前端访问

目标提交：`fix(auth): 对齐订单锁资格与跨组织访问`

### 4.1 订单锁资格

- [ ] 抽取可供单用户判断和候选查询复用的 lock grant 规则，支持 ORGANIZATION/ORGANIZATION_TREE/ALL。
- [ ] 移除 `role.CodeNEQ("administrator")`；普通 administrator 仍必须真实持有目标 lock 权限。
- [ ] `GetOrderLockState`、`LockOrder`、直接解锁、候选快照、钉钉回调复核使用同一口径。
- [ ] bootstrap admin 显式具备锁单和应急解锁；候选查询与回调不把 bootstrap 当普通审批人。
- [ ] 测试组织内角色、上级树角色、ALL administrator、bootstrap、无权限和权限撤销。

### 4.2 对冲网关

- [ ] 在 `scopedFinancePermissionWrites` 补入 netting read/create/confirm/reverse 的正确读写标志。
- [ ] 路由/鉴权测试覆盖附加组织只读与可写范围，不新增权限码。

### 4.3 前端 access 与用户页

- [ ] `UsersPanel` 按 access 条件请求 roles/organizations，普通组织管理员不发全组织请求。
- [ ] `UserFormModal` 普通流程只使用当前组织 roles；只有全局授权流程调用 `ListOrganizationRoles`。
- [ ] 保持 `admin.proto` 的 `ListOrganizationRoles = DATA_SCOPE_ALL`，确认生成鉴权表无变更。
- [ ] 前端测试覆盖组织管理员与全局管理员两类网络调用；运行修改文件 Biome、相关 Vitest、`pnpm --dir web tsc`、后端 auth 定向测试和 `git diff --check`。

阶段门禁：ACC-01、ACC-03、ACC-04 全部通过后提交。

## 阶段五：跨层回归与任务验收

目标：验证，不在此阶段顺便重构。

- [ ] 对照 PRD 的 12 个需求 ID 逐条记录证据（测试名或人工验证结果）。
- [ ] 运行 `go -C server test ./...`。
- [ ] 运行 `go -C server vet ./...`。
- [ ] 运行 `pnpm run check:web`。
- [ ] 运行根目录 `git diff --check`。
- [ ] 因本任务跨订单、财务、权限和前端公共 access，最终执行一次 `pnpm run check`；若它已完整包含前述门禁，避免重复运行同一检查。
- [ ] 本任务不涉及构建配置或生产入口，默认不运行 `pnpm run build`；仅在实施实际触及这些范围或用户要求发布验收时运行。
- [ ] 核查无手改生成物、无秘密、无临时数据、无未记录失败。
- [ ] 完整门禁通过后再归档任务；推送/合并仍需用户明确指示。

## 6. 预期文件范围

服务端核心：

- `server/internal/biz/order.go`
- `server/internal/biz/order_transition.go`
- `server/internal/biz/finance_cashflow.go`
- `server/internal/biz/settlement.go`（以实际符号位置为准）
- `server/internal/data/order_lock.go`
- `server/internal/data/order_release_pod.go`
- `server/internal/data/order_write.go`
- `server/internal/data/order_query.go`
- `server/internal/data/settlement.go`
- `server/internal/data/finance_verification.go`
- `server/internal/data/finance_cashflow.go`
- `server/internal/data/finance_invoice.go`
- `server/internal/data/dingtalk_approval_inbox.go`
- `server/internal/server/auth.go`
- 对应 `_test.go` / `_integration_test.go`

前端核心：

- `web/src/access.ts` 及 access 测试
- `web/src/pages/admin/users.tsx`
- `web/src/pages/admin/components/users/UserFormModal.tsx`
- `web/src/pages/finance/invoices/index.tsx`
- 对应 Vitest 文件

规范：

- `.trellis/spec/server/backend/order-lock-and-document-version.md`

明确不应修改：

- `server/api/**/*pb.go`
- `server/openapi.yaml`
- `web/src/services/roncin/**`
- `web/types/**`

除非先修改契约源并执行正式生成流程。

## 7. 最终验收记录（2026-09-11）

- trellis-check 全量复核：**通过**——12 个需求 ID（ORD-01~04、FIN-01~03、INV-01~02、
  ACC-01/03/04）全部有代码与测试证据；阶段间交互四项裁定成立
  （31 处门禁调用点逐一核对无意外语义变化；结清双口径在真实流转下等价；
  发票释放锁序无反向等待；administrator 例外全仓无残留）。
- 阶段五门禁：`go test ./...`（含 RONCIN_INTEGRATION_DATABASE_SOURCE 集成全集，
  干净环境）18 包 ok / 0 FAIL；`go vet` 通过；`buf lint` 通过；`govulncheck` 0 可达漏洞；
  `check:web` 在提交时点状态通过（admin/invoices 定向 vitest 38+11 用例复核通过）。
- 过程中修复的存量缺陷：海运航程冲突校验时区不一致（22d7debb，触发场景测试 a687ba2c），
  此前被集成测试静默 SKIP 掩盖——后续全量门禁应带集成环境变量执行。
- 遗留记录（P3，不阻断）：①未结清/未核销两套 SQL 口径（联不联父单据状态）靠
  「状态翻转必同事务翻转 allocation」不变量保持等价，建议后续统一；
  ②展示侧 ETD/ETA 日期格式化未统一 UTC/业务时区，存在展示日偏移的理论风险；
  ③门禁命令 `set -a && source .env.local` 会污染 TestProductionConfigUsesSafeDefaults
  等配置安全测试，跑门禁时只应注入必要变量。
