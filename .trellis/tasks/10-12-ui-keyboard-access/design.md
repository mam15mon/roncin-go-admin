# 公共键盘交互与焦点修复：设计草案

对目标下拉使用Antd trigger受控能力及原生button激活，必要ArrowDown处理在触发层，避免全局HeaderDropdown默认变更影响右键/其它消费者。明确open/aria-expanded与返回焦点。
SectionCard只有collapsible时将点击div改原生button，样式保持标题尺寸与品牌竖线；useId或既有sectionKey形成内容关联，统一受控状态。
页签/关闭局部focus-visible描边，采用已有品牌色/token；不得删除outline而无等价反馈。触达测试使用真实可访问名称，考虑图标对按钮名称的贡献。
键盘任务在列表任务完成后处理TagsView相关文件；global.less与窄屏任务顺序串行。

## 影响文件

- `web/src/components/layout/HeaderMenus.tsx`
- `web/src/components/OrganizationSwitcher.tsx`
- `web/src/components/HeaderDropdown/index.tsx`
- `web/src/components/ui/page-shell/SectionCard.tsx`
- `web/src/components/layout/TagsView.tsx`
- `web/src/global.less`
