# 实施与验证计划

1. 用户明确批准最新方案后 task.py start；实现子代理加载 trellis-before-dev、roncin-web-stack 及相关 Ant Design 规范。
2. 修复 RolesPanel 的字典候选派生，角色树/表单/统计/全选共用当前有效能力交集；若触及请求链则改用 React Query。
3. 添加候选范围说明与编辑不可授予项的明确保护，保留后端边界校验。
4. 功能完成后补回归用例，禁止 TDD：公司管理员全选创建、系统工作台、普通用户、缺能力、编辑保留、身份变化。
5. 运行修改文件 Biome、角色页/表单/字典/access.workspace 定向 Vitest、前端 tsc；后端边界定向测试继续通过。若新增集成验证，使用专用 PostgreSQL 并确认无 SKIP。
6. 检查子代理复核并自修，最终运行风险匹配的 pnpm run check:web。无构建入口与依赖变化不例行 build。
7. 主会话更新权限候选规范；git diff --check，核对未改业务库授权与生成物；Conventional Commit 中文描述提交并完成归档。

主要文件：web/src/pages/admin/roles.tsx、components/roles/RoleFormModal.tsx、按需 PermissionMatrixHeader.tsx 及对应测试；access.ts 最小导出有效能力判定供角色页复用，避免复制 scope 规则。服务端 admin_role.go 实现不改，仅在专用库补充最大合法权限创建/编辑及非法权限拒绝的集成验证。

执行门槛：PRD/设计/计划及 implement/check JSONL 已准备；用户在最终方案后明确批准，才进入实施。
