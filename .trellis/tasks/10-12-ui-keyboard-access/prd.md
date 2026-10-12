# 公共键盘交互与焦点修复

## 目标与来源

修复UI-UX审查 A07 / A08 / A09，让业务操作、反馈及状态一致。用户已同意创建修复任务并进入规划；来源为 `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/audit-report.md`，所有原始浏览器证据仍在原审查归档。

## 已确认事实

- A07/P2：企业资源/设置中心Enter、Space、ArrowDown不打开，hover可打开；组织切换Enter/ArrowDown不打开。`web/src/components/layout/HeaderMenus.tsx:173`、`:195`、`web/src/components/OrganizationSwitcher.tsx:133`、`web/src/components/HeaderDropdown/index.tsx:44`。
- A08/P2：客户四个可折叠分节标题DIV/tabIndex=-1且无expanded语义。`web/src/components/ui/page-shell/SectionCard.tsx:38`、`:103`。
- A09/P2：页签focus-visible=true但outline none、shadow none；关闭按钮无outline由代码确认。`web/src/global.less:653`、`:693`、`web/src/components/layout/TagsView.tsx:378`、`:401`。

## 需求

R1：三个顶栏下拉通过原生按钮Enter/Space及约定的ArrowDown展开；Esc关闭并返回触发按钮焦点，保留hover，不另建菜单键盘系统。
R2：可折叠SectionCard具有原生按钮语义，Enter/Space切换，aria-expanded与真实显示同步并关联唯一内容ID；非折叠标题保持展示。
R3：业务页签及关闭按钮出现清晰focus-visible样式，定位不改变active/hover语义和布局尺寸。

## 验收

- AC1/R1：三个菜单纯键盘能打开、选择或Esc退出，焦点可知；鼠标hover和正常权限过滤仍有效。
- AC2/R2：受控/非受控SectionCard均键盘可切换且语义一致，非折叠标题不新增焦点；标题extra按钮不意外触发折叠。
- AC3/R3：Tab到页签/关闭可见焦点，Enter/Space仍有效；点击不产生持续焦点装饰，关闭仍走未保存guard。

## 非目标

不追加页签roving tabindex/Home/End体系，不重设计导航样式；联系人icon命名为待复核，不作为已确认无名缺陷。

## 共同约束

保持纯白高密度、PageContainer 流式全宽、12px 内容边距、既有权限和用户/组织隔离。复用现有能力，遵循 ProTable request/actionRef、ModalForm.onFinish；不加兼容回退，不手改生成物，不引入全站 KeepAlive 或自动保存平台。禁止 TDD，实现后按风险补回归验证；不修改真实业务数据进行验收。
本子任务处于 planning，产品代码未变。规划待总任务范围决策及最终审阅；不得仅凭任务创建或本方案草案进入实施。
