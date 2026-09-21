# 工作台提成模块 UI 结构重构

## Goal

把工作台「我的提成」从单张巨石卡重构为「提成总览卡 + 月度申请卡」双卡结构，
建立大厂工作台式的信息层级（主数字 hero、状态流程可视化、统一卡片外壳），
提升员工日常最高频页面的视觉与交互品质。纯前端改造，不改后端接口与数据口径。

## 背景与问题（现状诊断）

当前 `CommissionSummaryCard`（web/src/pages/workbench/CommissionSummaryCard.tsx）把
三桶提成、本年/本月已发切换、冲减三桶、月度申请面板、预计可计提五层内容纵向堆进
一张 ProCard，存在五个具体问题：

1. 单卡高度超过 700px，层间只靠 1px 分隔线，无卡片矩阵节奏；
2. 桶组件为「3px 左色边 + 小字」，无图标底色块等图形语言；
3. 缺少 hero 主数字层级（本年已发缩在 18px 与普通读数混排）；
4. 月度申请的「累计中 → 可申请 → 财务审批」状态流不可视，核心动作「去申请」埋在第三列底部；
5. 金额未用等宽数字（tabular-nums），列宽跳动。

## Requirements

### R1 提成总览卡（CommissionOverviewCard，替换原 CommissionSummaryCard 的台账部分）

- 卡片外壳改用全站标准 `SectionCard`（components/ui，品牌蓝竖标 + 纯白 + 细边框 + 微阴影），
  右上角 extra 放「提成明细」「在途回款」入口按钮。
- hero 区：`本年已发` 为主数字（约 28-32px、加粗、tabular-nums），
  `本月已发` 为副指标（约 16px）同排展示，**取代原 Segmented 本年/本月切换**（两值同屏，减少一次交互）。
- 三桶流程条：`待财务确认 → 已确认待发 → 已发放` 三张 stat 小卡横向排列、箭头衔接，
  每卡含图标底色块（浅色圆底 + 主题色图标）、金额（约 20px、tabular-nums）、笔数；
  浅底色分别取 antd 色板 gold / blue / green 系；悬停有阴影浮起反馈。
- 冲减区：降级为总览卡底部单行摘要（待处理 / 已确认 / 已扣回 三段内联小统计），
  有任一冲减记录才渲染；三桶大展示不再占用整层。
- `nextEffectiveDate` 未来生效方案提示 Alert、`Empty` 空态、本位币 Tag 原语义保留。

### R2 月度申请卡（MyApplicationCard，从 MyApplicationPanel 提升为独立卡）

- 同样使用 `SectionCard` 外壳，右上角 extra 放「候选明细」「申请历史」按钮。
- 状态流可视化：以 Steps（或等价的轻量分段指示）呈现
  `本月累计中 → 可申请（截至上一自然月末）→ 财务审批`，当前所处阶段按数据推导：
  有 PENDING_REVIEW 申请 → 停在财务审批；有可申请分组 → 停在可申请；否则停在累计中。
  各阶段下方带金额/笔数。
- 「去申请」为主按钮，紧邻可申请阶段金额，禁用条件与权限门禁（canOperateBusiness）
  与现状一致；提交确认弹窗内容、成功后等待 Overview 刷新再解除 loading 的行为不变。
- 「预计可计提」（estimated）并入本卡底部作为前瞻脚注，保留「预计，非应发承诺」警示语义
  与 hasMore 部分统计声明。
- 最近申请被驳回时，在卡内以 warning 提示引导到申请历史重提（原语义）。

### R3 通用视觉与代码约束

- 金额文字统一 tabular-nums（`fontVariantNumeric: 'tabular-nums'`），样式常量收敛到模块内
  共享文件（display.ts 或新 token 文件），不在多处内联重复。
- 遵守全站纯白高密度企业级规范：不引入深色渐变、不硬编码 maxWidth、不加新依赖
  （不引入图表库；箭头/图标用 @ant-design/icons 现有图标）。
- 工作台其余卡片（FinanceSummaryCard、RecentOrdersCard、TodosCard、QuickEntriesCard、
  AccountBoundaryCard）本次**不动**，外壳迁移记录为后续任务（见 Notes）。

### R4 行为与数据不变式（重构红线）

- 数据源不变：仍消费 `GetWorkbenchOverview` 同一份响应，不新增/修改接口调用。
- 门禁不变：`hasCommissionEligibility !== true` 时从首次渲染起不出现任何提成 DOM；
  `canOperateBusiness` 为假时不出现申请办理入口（总部仅查看）。
- 文案口径不变：不出现付款承诺类表述；「本月累计中」不渲染任何申请入口；
  金额均为组织本位币口径。
- 下钻抽屉（MyCommissionDrawer / MyReceivablesDrawer / 候选 / 历史）的打开方式、
  分页与过滤行为、驳回重提流程全部保留。

## Acceptance Criteria

- [ ] 工作台提成区渲染为两张 SectionCard 卡片（总览 + 月度申请），原巨石卡不再存在；
- [ ] hero 区同时展示本年/本月已发，主数字层级明显且等宽数字对齐；
- [ ] 三桶为箭头衔接的 stat 小卡（图标底色块 + 金额 + 笔数），hover 有反馈；
- [ ] 月度申请卡呈现三阶段状态流，当前阶段随数据正确推导，「去申请」为主按钮；
- [ ] 冲减为单行摘要且仅在有记录时渲染；预计可计提保留警示语义；
- [ ] 全部既有行为断言在新测试下等价保留：门禁三态、提交流程（含确认弹窗内容、
      失败不刷新）、候选/历史抽屉、驳回重提、总部仅查看、组织切换竞态；
- [ ] `pnpm --dir web exec vitest run src/pages/workbench` 与
      `pnpm --dir web tsc`、改动文件 Biome 检查全部通过；
- [ ] `pnpm --dir web test`（全量）在提交前通过，无新增快照/控制台错误。

## 非目标（明确不做）

- 不做趋势图表、环比/同比（后端无月度序列数据，属方向 C，另行立项）；
- 不改后端 proto/服务/权限；
- 不迁移工作台其余五张卡片到 SectionCard（后续任务）；
- 不改 `/finance/commissions` 财务管理侧页面。

## Notes

- 后续任务候选：工作台其余卡片外壳统一迁移 SectionCard；
  Overview 增加月度序列以支持趋势图（方向 C）。
