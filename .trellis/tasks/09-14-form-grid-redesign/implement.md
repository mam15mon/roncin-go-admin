# 实施清单 (implement.md)

## Phase 1: 根布局 vertical 与钳制清理
- [ ] OrderFormTemplate.tsx:249 `layout="horizontal"` → `vertical`；
- [ ] OrderFormTemplate.less：删除 label 宽度钳制（flex: 0 0 96px / max-width: 96px / label height 32px）等横向专属规则；
- [ ] global.less：`.ant-form-item:has(textarea.ant-input)` 反制块精简——240px 固定高与 resize:none 必须保留；label 上置与控件列 flex 反制在根 vertical 后如仍需要则保留，否则删除；
- [ ] Playwright 回归：登录 → /orders/sea-export/new，截图确认整页无塌陷/截断/重叠，业务信息与配舱信息卡（尚未迁移 row 模板）观感可接受。

## Phase 2: FormRow 组件与货物与提单信息卡迁移
- [ ] 新建 `web/src/components/ui/form-row/`（FormRow + index 导出）：props `cols={2|3|4|5}`，CSS Grid `grid-template-columns: repeat(cols, 1fr)`，gutter 16/12，响应式 <992px 单列；
- [ ] 迁移货物与提单信息卡：委托品名/特殊要求（row-2）、对照三行（row-4×3，按钮移至「委托 vs 实际件重尺对照」标题行右侧）、提单特别条款（row-2 左格）、发货人/收货人（row-2）、通知人/外国代理（row-2）；
- [ ] 删除迁移后不再使用的 Row/Col 与对照区灰盒样式残留。

## Phase 3: 验证
- [ ] `pnpm --dir web exec tsc`；biome 定向检查全部改动文件；
- [ ] Playwright：登录 → 新建海运出口 → 量测（脚本思路见 /tmp/measure.mjs、/tmp/probe6.mjs，playwright 在 web/node_modules/@playwright/test，chromium 已装）：
  - 委托品名 textarea x == 提单特别条款 x == 对照区 col1 label x；
  - row-2 两列、row-4 四列各自 x 一致；
  - 件数拼接控件总宽 210px；
  - 无两控件包围盒相交；
  - 截图整卡 + 对照区局部；
- [ ] `gofmt` 不涉及；`git diff --check` 干净。

## 风险与回滚
- 根 vertical 会让业务信息/配舱信息卡也变标签上置（属设计预期）；若观感问题回滚 = layout 改回 horizontal。
- 全部改动仅前端 sea 模板相关文件与 global.less；出现意外回滚 = revert 本任务提交。
