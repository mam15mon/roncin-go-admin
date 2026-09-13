# 海运表单栅格统一重构 PRD

## Goal

海运订单模板表单统一到单一列网格系统：标签全部上置（vertical）、四种行模板（row-2/3/4/5）、删除特异性对抗的 !important 钳制。本期范围 P1+P2；P3（配舱/业务信息卡迁移）与 P4（旧栅格规则全删）为后续阶段。

## 已定设计决策（2026-09-14 用户确认）

1. 表单根 layout 改 **vertical**，所有标签上置（长标签不再截断，控件左缘=字段左缘）；
2. 四种行模板（CSS Grid）：row-2=12/12、row-3=8/8/8、row-4=6/6/6/6、row-5=5/5/5/4×5——**跨行共享列边界**（row-2 中线 = row-4 第二列边界）；
3. 控件宽度=列宽（width 100%）；拼接控件 Space.Compact 铺满列宽、内部比例固定（主输入 flex1 + 下拉 104px）；
4. 操作按钮（带入委托件重尺）放区块标题行右侧，不占字段网格；
5. Card body padding 0 已落地（70f56f67）。

## Requirements

- **R1（P1）**：OrderFormTemplate.tsx:249 ProForm 根 `layout="horizontal"` → `vertical`；删除 OrderFormTemplate.less 的 label 96px 钳制等横向专属规则；精简 global.less 的 `:has(textarea)` 反制规则（**textarea 240px 固定高与 resize:none 保留**——用户明确要求）。
- **R2（P2）**：新建 FormRow 行模板组件（@/components/ui，CSS Grid 实现 row-2/3/4/5 + 响应式降级）；迁移货物与提单信息卡全部字段行：委托品名/特殊要求、对照三行（委托/实际/条款）、提单特别条款、发货人/收货人、通知人/外国代理。
- **R3**：迁移后所有字段/联动逻辑不变（name、校验、onChange、默认值）。

## Acceptance Criteria

- [ ] 海运新建单页面：标签零截断、无 96px 空带、无元素重叠；
- [ ] 货物与提单信息卡所有字段控件左缘落在统一列边界（0/25%/50%/75%）——Playwright 量测：委托品名 textarea x == 提单特别条款 x == 对照区 col1 label x；row-2 各字段 x 一致；
- [ ] 件重尺拼接控件（件数+单位）总宽保持 210px；
- [ ] tsc/biome 绿；海运模板测试维持 skip；截图验收。

## Out of Scope

- 配舱信息/业务信息卡的 FormRow 迁移（P3 后续任务）；
- OrderFormTemplate.less 旧栅格规则的整体删除（P4）；
- 其他非海运模板表单。
