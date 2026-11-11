# 验收记录

## 已通过
- 前端7个定向测试文件52例：列表wire映射（缺省/零值）、页面海空运隔离、默认列与已有偏好、海运字段日期金额真实渲染、附件未授权与0区分、主分单和审核相邻路径。
- 前端tsc、修改文件Biome、架构扫描、git diff --check；独立trellis-check未发现遗留问题。
- 后端biz/service定向TestOrderList：非法排序、资源读取权限/跨组织与optional零保留。
- 专用PostgreSQL真实测试三项：TestOrderListResourcesPostgres、TestOrderListSortPostgres、TestOrderListProjectionNamesPostgres，3 PASS / 0 SKIP。
- 主会话另跑专用PostgreSQL相邻提成回归：TestCommissionResolutionPostgres、TestWorkbenchListMyCommissionsPostgres，2 PASS / 0 SKIP（含子用例0 FAIL / 0 SKIP）。
- make API、Wire、OpenAPI与前端客户端/枚举源生成已完成。

## 最终门禁
`pnpm run check:fast`通过（总耗时233秒）：前端436个测试文件通过、1个跳过，2367例通过、11例跳过；后端默认测试、vet、分层/重复扫描、Proto lint与可达漏洞扫描通过。普通全量Go门禁不提供集成库环境变量，不能将其计作全量真实库覆盖；本次真实库覆盖仅上述定向集合。

## 验证边界
未做真实浏览器验收；未跑生产构建（无构建配置、依赖或生产入口变更）。本次新建专用测试库，五项定向真实库回归结束后已删除本任务测试库/角色及临时凭据；无开发数据修改、无Schema变更。

## 交付
实现提交：`38513a05`；规划提交：`582dcba7`。未推送远端。
