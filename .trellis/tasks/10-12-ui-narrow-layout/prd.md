# 窄屏导航入口与订单布局修复

## 目标与来源

修复UI-UX审查 A10 / A11，让业务操作、反馈及状态一致。来源为 `.trellis/tasks/archive/2026-10/10-12-ui-ux-interaction-audit/audit-report.md`，所有原始浏览器证据仍在原审查归档。

## 已确认事实

- A10/P2：390px顶部菜单display none，抽屉仅工作台/订单管理/费用管理。`web/src/global.less:1393`、`web/config/routes.ts:38`、`:74`、`:102`、`web/src/router/adaptRoutes.tsx:128`；客户/主数据仍有 `web/src/pages/workbench/WorkbenchSideCards.tsx:240` 的快捷入口，不是全不可达。
- A11/P2：390px订单新建页scrollWidth537，工作台390；根因未锁定。排查入口 `web/src/components/ui/order-template/OrderFormTemplate.tsx:282`、`web/src/components/ui/form-row/FormRow.tsx:20`、`web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx:354`。

## 需求

R1：窄屏提供与桌面同权限来源的企业资源/设置常规入口，至少供应商、国外代理、企业资源配置及公司工作区设置项可发现；不依赖历史页签或先建单再返回列表。
R2：375/390px及断点附近订单新建/详情表单不发生整页横向溢出；多列字段合理降列，弹层和宽表横滚局限在自身容器，标签、输入、主要操作可见。
R3：保持桌面流式全宽、高密度与统一列宽行为，无新增maxWidth居中容器。

## 验收

- AC1/R1：375/390/767/768/769px公司及系统工作区有权限菜单可点击到达；无权限项不出现，当前路由高亮和drawer关闭符合现有行为。
- AC2/R2：375/390px稳定页面scrollWidth<=innerWidth+1；新建/详情主字段及保存入口可达，无overflow-x:hidden裁切掩盖；宽表仍可容器内横滚。
- AC3/R3：1440px对照布局和菜单行为正常，列冻结/拖宽不受影响。

## 非目标

不开发单独手机应用，不统一放大所有按钮，不换主题字体；财务弹窗尚未复现的窄屏候选不借机纳入。

## 共同约束

保持纯白高密度、PageContainer 流式全宽、12px 内容边距、既有权限和用户/组织隔离。复用现有能力，遵循 ProTable request/actionRef、ModalForm.onFinish；不加兼容回退，不手改生成物，不引入全站 KeepAlive 或自动保存平台。禁止 TDD，实现后按风险补回归验证；不修改真实业务数据进行验收。
最终规划已获用户批准，按总任务约定顺序实施；当前进度以 task.json 和 implement.md 为准。
