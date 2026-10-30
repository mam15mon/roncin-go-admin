# 定位记录

路由入口为 web/src/router/index.ts，配置由 adaptRoutes.tsx 构建；页面和 AppLayout 均使用 lazy。现有适配测试会加载全部模块，可扩展真实首次进入路径验证。

海运条款默认值集中在 sea-document-form.ts，字段组件仍有重复 initialValue。实施前检查独立调用方，验证付款方式、主分单条款、编辑值与恢复输入。

全仓搜索 Permissions-Policy 及五项功能名称没有命中业务或配置文件。实施阶段检查本地响应头，避免猜测修改。

验证采用实现后补充的针对性回归、修改文件 Biome、tsc，以及最终 check:web；无需生产构建。

## 实际响应头核查

绕过当前进程代理（curl --noproxy '*'）直接访问本地服务：
- http://127.0.0.1:8001/：200，未返回 Permissions-Policy。
- http://127.0.0.1:8001/orders/sea-export/create：200，未返回 Permissions-Policy。
- http://127.0.0.1:8000/health/live：200，未返回 Permissions-Policy。
- http://127.0.0.1:8001/api/v1/auth/me：404，未返回 Permissions-Policy；该探针不代表业务接口可用性验证。

目前证据只能排除上述本地响应，无法从粘贴日志确定具体告警所属请求，不能断言来自某个代理或第三方。故不修改项目安全响应头配置；最终需说明此项未复现、来源未定。

## 浏览器基线与调用链补充

使用已安装 @playwright/test 的 Chromium，无身份访问本地 8001/，最终进入 /user/login，实际捕获 `No HydrateFallback element provided to render during initial hydration`；未捕获 Permissions-Policy。

实施排查确认 buildSeaExportCreateDefaults 原先没有付款方式与完整主分单条款，新建仍依赖字段 initialValue。故须先在新建默认值、模式切换及详情初始化入口统一设置条款，再删除字段默认值，避免新建字段变空。

## 实施与定向验证

- 顶层裸路由和业务壳配置同步 HydrateFallback，复用 QuarterRing。
- sea-document-form.ts 导出 seaBillContentFormValues；新建、详情加载和模式切换共用初始化入口，移除字段重复默认值。
- 实施代理定向运行 12 个测试文件、68 用例通过；tsc、diff --check、修改文件 Biome 与 Antd 字段检查通过。Biome 保留原有未使用测试变量警告。
- 修复后真实 Chromium 分别首次进入 /user/login 和 /orders/sea-export/new，未捕获 HydrateFallback、initialValues、Permissions-Policy 告警。业务入口在无身份下回到登录页，因此海运字段内容正确性以真实表单组件测试验证，未宣称已完成登录后的浏览器业务验收。

## 独立审查与最终门禁

Trellis 检查代理审查无新增缺陷，无需自修复。`pnpm run check:web` 退出码 0：335 测试文件通过、1 文件跳过；1806 用例通过、12 跳过；Vitest 66.74 秒。权限键、Proto 生成物、架构、重复函数、Biome、tsc 均通过。Biome 有既存 15 warning/7 info，测试有 localhost:3000 连接拒绝/重置日志，不影响门禁通过；未为通过检查关闭规则或屏蔽告警。

AC1、AC2、AC4 已满足；AC3 已记录实际响应头核查及未复现限制。Permissions-Policy 未定位具体请求来源，不声明已修复。
