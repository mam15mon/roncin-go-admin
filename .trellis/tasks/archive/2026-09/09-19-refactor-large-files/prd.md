# 大文件职责拆分重构

## Goal

把手写源码中的超大文件按职责拆分为同包（Go）/ 就近目录（前端）文件，
**零行为变更**，目标是让人工与 AI 读取文件变简单：文件能独立装进上下文、
按文件名即可定位职责。不需要重构的文件明确不动。

## 判定标准（哪些拆、哪些不拆）

**拆**：手写源码 >1000 行，且包含 ≥2 个可独立命名的职责区块。

**不拆（明确排除）**：

- 一切生成物：`*.pb.go`、`server/internal/data/ent/`（除 schema）、
  `server/openapi.yaml`、`access_rules_gen.go`、`wire_gen.go`、
  `web/src/services/roncin/`、`web/config/openapi.generated.json`、
  `typings.d.ts`、`enums.generated.ts`、`permissions.generated.ts`。
- 测试文件（本计划不碰；超大集成测试如 `order_lock_integration_test.go`
  2141 行若需拆分另立任务）。
- `web/src/components/ui/` 模板基座：共享复用面大，拆动风险与收益不成比例
  （`fields-meta.ts` 1078 行为配置数据，单文件反而便于整体替换）。
- P2 观察名单（1000–1150 行、职责相对内聚，暂不动，P0/P1 完成后按痛点复查）：
  `biz/sea_order_change.go`(1144)、`service/sea_order_change.go`(1132)、
  `biz/auth.go`(1114，Principal 权限判定一体)、`data/enterprise_resource.go`(1054)、
  `audit-presentation.ts`(968) 以下全部文件。
- <1000 行的文件全部不动。

## 拆分纪律（所有子任务共同约束）

1. **纯移动、零行为变更**：不改函数签名、不改导出名、不加抽象层、不换实现、
   不做无关格式化。Go 同包拆分不产生任何 import 变化。
2. **每文件（或每配对组）一个独立提交**，`refactor:` 前缀，提交信息列明
   移动前后文件与行数。
3. **验证**：Go 拆分跑目标包定向测试 + `go -C server vet ./internal/data`
   （或对应包）；前端拆分跑对应定向 vitest + `biome check` + 类型链变化时
   `pnpm --dir web tsc`。每个子任务收口时跑 `git diff --check`。
4. 拆分后 `grep` 复核：旧文件内无残留引用、包内符号解析不变、
   测试引用路径无需修改（Go 同包；前端仅当文件路径变化才改 import）。
5. 遵守仓库快速交付原则：不为拆分引入新目录层级或新公共 API；
   文件命名沿用领域前缀（如 `sea_order_change_split.go`）。

## 子任务地图（批次顺序即执行顺序）

### P0 高价值（各为独立子任务，优先执行）

| 子任务 | 文件 | 行数 | 拆分轴 |
| --- | --- | --- | --- |
| refactor-sea-order-change-data | `server/internal/data/sea_order_change.go` | 5185 | 按变更类型拆 5 个同包文件：`_actions`（GetChangeActions/GetSplitContext）、`_split`（Preview/ExecuteSplit + 拆单校验 helper，ExecuteSplit 单函数约 2200 行保持原样整体移动）、`_transport_update`、`_reassignment`、`_events`（事件查询 + 共享映射 helper） |
| refactor-commission-ledger-data | `server/internal/data/finance_commission.go` | 2768 | 规则管理（rules CRUD/assign/copy）与台账查询（list/count/export/candidates）两个文件 + 共享 helper 文件 |
| refactor-order-split-page | `web/src/pages/orders/split.tsx` | 2709 | 页面骨架保留；抽 `hooks/useSeaOrderSplitFlow`（状态与提交）、子组件（目标编辑、费用汇总、结果展示）、纯函数移 `splitUtils.ts`；`split.test.tsx` 全绿为准绳 |
| refactor-sea-document-section | `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx` | 2587 | 按业务分节抽子组件，就近放 `sea/` 目录；既有测试为准绳 |

### P1 中价值（按领域配对分组，P0 完成后逐个执行）

| 子任务 | 文件（行数） | 说明 |
| --- | --- | --- |
| refactor-order-write-lock | data/order_write.go (1590) + data/order_lock.go (1589) | 同域配对；注意二者是 AGENTS.md 引用的并发范本，拆分后范本指向需同步更新 |
| refactor-sea-document-change | data/sea_document_change.go (1483) | 按单证变更职责拆 |
| refactor-finance-bill | biz/finance_bill.go (1613) + data/finance_bill.go (1378) | 上下层配对拆分 |
| refactor-fee-supplement | data/order_fee_supplement.go (1327) | 按补录职责拆 |
| refactor-commission-biz-app | biz/finance_commission.go (1361) + data/finance_commission_application.go (1669) | 提成域配对；application 为上月新写，结构较新，拆分保守 |
| refactor-admin-enterprise-web | enterprise-resources/index.tsx (1626) + OrgChartCanvas.tsx (1292) + RoleFormModal.tsx (1291) | 页面拆分节卡片组件；Canvas 拆绘制函数与交互逻辑 |
| refactor-drawer-workbench-web | SeaSharedContainerDrawer (1091) + CommissionRulesDrawer (1029) + BillCreationWorkbench (1026) + partner-detail.tsx (1017) | 各自抽内部区块子组件与 hooks |

## 验收标准（父任务）

- [x] P0 四个子任务全部完成归档，四个大文件均 <1500 行，相关定向测试全绿。
- [x] P1 七个子任务全部完成归档，涉及文件均降到职责单一的可读规模。
- [x] 全程零行为变更：无契约/生成物/迁移变化，无公开 API 变化；
      Go 拆分以「非 import 正文与原文件逐行多重集一致」机器证明，
      前端拆分以定向测试全绿 + tsc 零错误佐证。`pnpm run check`
      最终门禁结果见下方收尾记录。
- [x] P2 观察名单复查（2026-09-19，结论：均不拆）：
  - `biz/sea_order_change.go`（1144）：拆票/改配领域命令与预览对象定义，
    与 data 层新文件一一对应，职责内聚。
  - `service/sea_order_change.go`（1132）：同名域的 DTO 转换薄层，
    拆开反而割裂转换上下文。
  - `biz/auth.go`（1114）：Principal 权限判定是一个完整概念，
    拆分会制造人为文件边界。
  - `data/enterprise_resource.go`（1054）：已低于新格局下的拆分阈值。
- [x] AGENTS.md / `.trellis/spec/` 中引用的文件路径已同步：
      `order_write.go` 的 UpdateDraft 并发范本指向改为
      `order_write_draft.go`（AGENTS.md 与 database-guidelines.md）。

## 收尾记录（2026-09-19）

- 11/11 子任务全部完成归档；后端 8 个 1300+ 行手写文件拆为 ~40 个职责文件，
  前端 7 个 1000+ 行组件拆出 ~40 个就近子组件/工具文件。
- 拆分后手写大文件格局：唯一 >1300 行的是
  `sea_order_change_split_execute.go`（2221，单函数 ExecuteSplit 整体
  移动的既记录例外，拆函数体超出纯移动边界）；其余 Go ≤1144、前端 ≤1278。
- 执行方式：Go 侧主会话机械脚本拆分（每文件零丢失机器证明 + 真实库
      PostgreSQL 定向集成测试）；前端 P0 主会话手工抽取，P1 两组由
  trellis-implement 子代理在独立 worktree 并行执行、主会话复核收尾。
- 最终门禁 `pnpm run check` PASS：`go test ./...` 全包 ok 零失败
  （旧开发机记录的 3 个存量失败在本机未复现）、前端全量 vitest
  852 通过/12 跳过（152 文件）、buf lint、go vet、漏洞审计
  （豁免 GO-2026-6452 一项）全部通过。

## Notes

- 父任务自身无直接代码工作，只承载总计划、跨子任务验收与最终集成复查；
  实现全部落在子任务。
- 子任务激活时各自补齐自己的 design/implement 细节（拆分文件清单、
  函数归属映射、验证命令）。
