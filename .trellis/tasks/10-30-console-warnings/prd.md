# 修复海运表单与路由控制台告警

## 目标
消除用户日志中由本项目代码引起的路由初始加载与表单默认值冲突告警，保留现有录单行为，并查明 Permissions-Policy 告警的可控范围。

## 已确认事实
- 用户附件包含 5 条不识别 Permissions-Policy 功能项告警、1 条路由 HydrateFallback 告警，以及付款方式和主分单各四个条款字段的默认值冲突告警。
- web/src/router/adaptRoutes.tsx:80 的页面懒加载及 :116 的布局懒加载没有配置初始加载占位。
- web/src/pages/orders/templates/components/sea/SeaCargoSection.tsx:34 重复声明 paymentTerm 默认值。
- web/src/pages/orders/templates/components/sea/SeaBillContentFormFields.tsx:1279、:1332、:1381、:1415 声明字段默认值；web/src/pages/orders/sea-document-form.ts:13 已统一提供主分单条款默认值。
- 全仓搜索未找到 Permissions-Policy 或对应广告功能项配置，尚不能确定响应头来自应用、代理还是外部资源。

## 范围与要求
- R1：为首次进入懒加载路由提供沿用现有视觉的加载占位，覆盖登录等无布局路由及业务布局路由。
- R2：统一海运表单默认值来源，检查全部字段组件调用方，保留新建默认值、编辑已有值、主分单切换和草稿恢复行为，避免只删除告警而丢失字段值。
- R3：核查可访问的本地服务响应头；仅在证据确认属于本仓库配置时修改源配置，否则记录来源证据或定位限制。
- 不屏蔽 console，不过滤框架告警；React DevTools 开发提示不作为缺陷处理。

## 验收标准
- AC1（R1）：首次加载无布局和业务路由时有合理加载占位，且不再产生缺少 HydrateFallback 告警。
- AC2（R2）：海运表单不再产生上述 initialValues 冲突告警，新建默认值与编辑已有值保持正确，相关切换与恢复路径通过针对性验证。
- AC3（R3）：记录响应头核查结果，明确已修复项及仓库外或尚无法定位的项，不声称未验证的告警已消除。
- AC4：相关测试、类型检查、变更文件检查与风险匹配的前端门禁通过，执行 git diff --check，并以中文 Conventional Commit 提交。

## 范围外与风险
不改业务契约、数据库、权限或浏览器设置，不建设通用初始化框架。Permissions-Policy 来源可能在仓库外，若本地无法复现则保留明确限制。

## 计划状态
轻量缺陷修复，采用 PRD 与研究记录。用户已明确批准最终计划，已进入实施；新建、详情和切换入口统一默认值属于 R2 必需范围。
