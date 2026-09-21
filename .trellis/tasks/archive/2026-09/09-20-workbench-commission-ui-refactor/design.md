# 技术设计：工作台提成模块 UI 结构重构

## 1. 改动范围与组件边界

```text
web/src/pages/workbench/
  index.tsx                     改：提成区渲染新双卡；抽屉提升到本层管理
  CommissionSummaryCard.tsx     删：拆分为下面两个组件
  CommissionOverviewCard.tsx    新：提成总览卡（hero + 三桶流程条 + 冲减摘要行）
  MyApplicationPanel.tsx        删：提升为独立卡
  MyApplicationCard.tsx         新：月度申请卡（状态流 + 动作区 + 预计可计提脚注）
  FlowStat.tsx                  新：三桶 stat 小卡（图标底色块 + 金额 + 笔数 + hover）
  display.ts                    改：新增金额排版共享常量（等宽数字等）
  MyCommissionDrawer.tsx        不动
  MyReceivablesDrawer.tsx       不动
  MyApplicationCandidatesDrawer.tsx  不动
  MyApplicationHistoryDrawer.tsx     不动
  index.test.tsx                改：断言新结构，行为断言等价保留
  MyApplicationPanel.test.tsx   改名→MyApplicationCard.test.tsx 并适配
```

不新增依赖；`SectionCard`、图标均来自现有 `components/ui` 与 `@ant-design/icons`。

## 2. CommissionOverviewCard 设计

### 2.1 数据映射（全部来自现有 Overview 响应，无新请求）

| UI 元素 | 字段 |
| --- | --- |
| hero 主数字「本年已发」 | `commissionSummary.paidAmountThisYear` |
| hero 副指标「本月已发」 | `commissionSummary.paidAmountThisMonth` |
| 三桶 | draft/confirmed/paid 三组 Count+Amount |
| 冲减摘要行 | decreaseDraft/Confirmed/Paid 三组 |
| 未来生效提示 | `data.nextEffectiveDate` |
| 本位币 | `data.baseCurrency || summary.baseCurrency` |
| 空态判定 | 三桶 + 冲减三组 + estimated.opportunityCount 全零 |

### 2.2 布局（自上而下）

```text
SectionCard title=我的提成 [+本位币Tag]  extra=[在途回款][提成明细(primary)]
├─ (可选) nextEffectiveDate Alert
├─ hero 行：本年已发 ¥1,284,000 (30px/700/tabular-nums)
│            本月已发 ¥98,200 (16px，纵向左侧对齐 baseline 错落)
├─ 三桶流程条：[FlowStat gold] → [FlowStat blue] → [FlowStat green]
│   箭头用 ArrowRightOutlined (#cbd5e1 18px)；窄屏 flexWrap，箭头在换行时隐藏
│   (用 CSS container query 或简单 flexWrap+箭头随卡折行，实现取简：箭头元素随
│    flex 换行自然掉到行尾即可，不做媒体查询)
├─ (有冲减记录时) 冲减摘要行：标签「冲减」+ 三段内联
│   「待处理 ¥x · n笔」「已确认 ¥x」「已扣回 ¥x」，13px，段间 divider 分隔
└─ 空态：Empty（保持现有文案）
```

- hero 数字 `workbenchAmount()` 空值处理沿用 `display.ts`。
- FlowStat 规格：
  - 容器：`flex:1; min-width:150px; padding:12px 14px; border-radius:8px;
    background: <浅色底>; border: 1px solid <同系浅边>`；hover：
    `boxShadow: 0 4px 12px rgba(15,23,42,.08); translateY(-1px); transition`。
  - 图标底色块：`28px 圆形背景 <浅色底>` 内 14px 主题色图标
    （gold: `#faad14`/`#fff7e6`；blue: `#1677ff`/`#e6f4ff`；green: `#52c41a`/`#f6ffed`）。
  - 图标语义：待财务确认 `HourglassOutlined`、已确认待发 `WalletOutlined`、
    已发放 `CheckCircleOutlined`。
  - 金额 20px strong + tabular-nums；笔数 12px secondary。
  - hint Tooltip 保留原三桶提示文案（迁移到 FlowStat 的 title/hint prop）。

## 3. MyApplicationCard 设计

### 3.1 阶段推导（纯函数，便于单测）

```ts
type ApplicationStage = { step: 0 | 1 | 2; done: boolean[] };
// step 0 本月累计中 / step 1 可申请 / step 2 财务审批
function deriveStage(summary): ApplicationStage {
  if ((summary.pendingReviewCount ?? 0) > 0) return { step: 2, ... };
  if ((summary.applyGroups?.length ?? 0) > 0) return { step: 1, ... };
  return { step: 0, ... };
}
```

（审批中优先于可申请：有在途申请时用户动作是等待/查看历史，不是再次提交；
与服务端「同月唯一申请」约束一致。）

### 3.2 布局

```text
SectionCard title=月度申请 [+本位币Tag]  extra=[申请历史]
├─ Steps(size=small, 3 项)：
│   ① 本月累计中 —— 累计 ¥82,000 · 12 笔（accumulatingAmount/Count）
│   ② 可申请（截至上一自然月末）—— 合计 ¥156,000 · 8 笔（applyGroups 汇总；
│      多月份时副行逐月列出「YYYY-MM n笔 ¥x」）
│   ③ 财务审批 —— 审批中 n 张 / 已批准 m 张（pendingReviewCount/approvedCount）
├─ 动作行：[候选明细] [去申请(primary, disabled=无可申请分组)]  ← 紧跟 Steps 下方右侧
├─ (最近申请被驳回) Alert warning：引导到申请历史重提（原文案语义）
└─ 脚注行：预计可计提 ¥39,000 · n 笔机会（预计，非应发承诺；hasMore 声明）/
           暂无法估算（原文案）
```

- Steps 用 antd `Steps` 现成组件（`progressDot` 或默认小尺寸），不自绘。
- 「去申请」点击后的 `modal.confirm` 内容、`workbenchServiceSubmitMyCommissionApplication`
  调用、成功 `message` + `await onOverviewRefresh()`、失败不刷新——逻辑原样迁移。
- 提交按钮的 loading/disabled/canOperateBusiness 门禁逻辑原样迁移；
  总部（无 canOperateBusiness）时不渲染按钮但仍显示摘要与历史入口（原行为）。

## 4. index.tsx 接线

- `showCommission` 门禁不变；门禁内渲染：
  `<CommissionOverviewCard data onOpenCommissions onOpenReceivables />`
  `<MyApplicationCard summary currency onOverviewRefresh />`
  （两张卡上下排列，全宽，与工作台其余卡片节奏一致）。
- 四个抽屉（明细/回款/候选/历史）状态位置：
  明细与回款留在 index.tsx（现状）；候选与历史留在 MyApplicationCard 内部（现状）。
- 删除 `CommissionSummaryCard.tsx`、`MyApplicationPanel.tsx` 及其旧引用。

## 5. 共享样式常量（display.ts 增量）

```ts
export const amountFont = {
  fontVariantNumeric: 'tabular-nums',
} as const;
export const heroAmountStyle: React.CSSProperties = { fontSize: 30, fontWeight: 700, ...amountFont, color: '#0f172a' };
export const statAmountStyle: React.CSSProperties = { fontSize: 20, fontWeight: 600, ...amountFont };
```

新增只做加法，不动 `workbenchAmount` 等现有导出。

## 6. 测试策略

### 6.1 必须等价保留的行为断言（改写选择器/文案定位，不改语义）

来自 `index.test.tsx`：
- 门禁 false / undefined：无提成 DOM；true 且无数据：空态 + 未来生效提示；
- 三桶数值与冲减三阶段数值仍可断言（新定位：FlowStat 文本 / 冲减摘要行文本）；
- 「本年/本月已发切换」用例改为「双指标同屏展示」断言；
- 明细/回款抽屉打开与分页；
- 组织切换竞态；
- 申请摘要展示（可申请分组 + 本月累计）。

来自 `MyApplicationPanel.test.tsx`（迁至 `MyApplicationCard.test.tsx`）：
- 总部仅查看、可申请分组与合计、累计中无入口；
- 提交确认弹窗内容（覆盖月份/笔数/总额/截止日说明）、成功后等待刷新、失败不刷新；
- 候选/历史抽屉、驳回重提携带 ID+版本、驳回引导提示。

### 6.2 新增断言

- 阶段推导纯函数 `deriveStage` 三分支单测（放 MyApplicationCard.test.tsx 顶部，轻量）；
- hero 主/副指标同时渲染；冲减摘要行无记录时不渲染。

### 6.3 约束

- 每个测试文件重型渲染用例数 ≤10（AGENTS.md LPT 约束）；两文件当前用例数已在限内，
  迁移时新增轻量纯函数用例不计入渲染负担。

## 7. 风险与回滚

| 风险 | 缓解 |
| --- | --- |
| 测试大量依赖旧 DOM 定位 | 6.1 清单先行迁移，跑定向 vitest 作为每步门禁 |
| Steps 在窄屏换行观感差 | size=small + 窄屏允许 description 换行；不做响应式断点特判 |
| SectionCard 与工作台其余 ProCard 混排观感差异 | 接受（本次范围仅提成区）；记录后续任务 |
| Segmented 移除影响既有用户习惯 | 双指标同屏信息量 ≥ 切换，PRD 已确认 |

回滚：单次提交整体 revert 即可，无数据/契约影响。
