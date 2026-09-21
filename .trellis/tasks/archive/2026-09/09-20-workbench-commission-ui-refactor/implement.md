# 执行计划：工作台提成模块 UI 结构重构

前置：`prd.md`（需求与验收）、`design.md`（组件与布局规格）已评审通过。
所有命令在仓库根目录执行；定向测试路径相对 `web/`。

## 步骤清单（按序执行，每步含验证门禁）

### S1 共享样式常量（display.ts）

- [ ] 新增 `amountFont` / `heroAmountStyle` / `statAmountStyle`（见 design.md §5），只加不改。
- 验证：`pnpm --dir web exec vitest run src/pages/workbench/index.test.tsx`（存量应全绿）。

### S2 FlowStat 组件

- [ ] 新建 `FlowStat.tsx`：图标底色块 + 金额 + 笔数 + hint Tooltip + hover 浮起；
      颜色三系（gold/blue/green）按 design.md §2.2 规格实现。
- 验证：tsc 无错（组件先落地，接线在 S3）。

### S3 CommissionOverviewCard

- [ ] 新建 `CommissionOverviewCard.tsx`：SectionCard 外壳、hero 双指标（去掉
      Segmented 与 paidRange state）、三桶流程条（FlowStat + 箭头）、冲减单行摘要
      （有记录才渲染）、nextEffectiveDate Alert、Empty 空态。
- [ ] `index.tsx` 切换渲染新卡；删除 `CommissionSummaryCard.tsx`。
- [ ] 迁移 `index.test.tsx` 中提成卡相关断言（门禁三态、三桶/冲减数值、
      hero 双指标、明细/回款抽屉、组织切换竞态）。
- 验证：
  - `pnpm --dir web exec vitest run src/pages/workbench/index.test.tsx`
  - `pnpm --dir web exec biome check src/pages/workbench/CommissionOverviewCard.tsx src/pages/workbench/FlowStat.tsx web/../web/src/pages/workbench/index.tsx 2>/dev/null || pnpm --dir web exec biome check src/pages/workbench`

### S4 MyApplicationCard

- [ ] 新建 `MyApplicationCard.tsx`：SectionCard 外壳、`deriveStage` 纯函数 +
      Steps 三阶段、逐月分组副行、动作区（候选明细 / 去申请 primary）、
      驳回 warning 提示、预计可计提脚注；提交/刷新逻辑自 MyApplicationPanel 原样迁移。
- [ ] `index.tsx` 切换渲染新卡（applicationSummary 传入）；删除 `MyApplicationPanel.tsx`。
- [ ] `MyApplicationPanel.test.tsx` → `MyApplicationCard.test.tsx`：全部行为断言等价迁移
      + `deriveStage` 三分支纯函数用例 + 阶段渲染断言。
- 验证：
  - `pnpm --dir web exec vitest run src/pages/workbench/MyApplicationCard.test.tsx`
  - `pnpm --dir web exec biome check src/pages/workbench/MyApplicationCard.tsx`

### S5 全量回归与视觉自查

- [ ] `pnpm --dir web exec vitest run src/pages/workbench`（模块全绿）。
- [ ] `pnpm --dir web tsc`。
- [ ] `pnpm --dir web exec biome check src/pages/workbench`。
- [ ] 启动 `pnpm run dev:web`，以开发账号逐项核对 PRD 验收标准 R1-R4
      （门禁三态可用不同测试账号；无环境时以测试断言 + 组件规格核对并记录说明）。
- [ ] `pnpm --dir web test`（全量，提交前一次）。

### S6 收尾

- [ ] 更新 `.trellis/spec/web/frontend/capability-navigation.md`（若新增可复用能力；
      FlowStat 属页面内组件，预计无更新，确认即可）。
- [ ] `git diff --check`；按 Conventional Commits 提交（`feat: 工作台提成模块拆分双卡重构`）。

## 评审门禁

- S3 完成后：对照 design.md §2 自查一遍布局与文案（首个可见成果点）。
- S5 完成后：对照 prd.md 验收清单逐项勾选，未达成项记录原因。

## 回滚点

- S1-S2（纯新增）与 S3/S4（替换渲染）各自独立可 revert；全部改动单提交，
  整体 `git revert` 即可回滚，无数据影响。
