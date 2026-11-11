# 资源覆盖导入修复与验证

## 实现

- service 预览与提交均校验当前公司 create 权限；overwriteConflicts=true 时额外校验同公司 update 权限，在进入仓储前失败，混合批次无部分写入。
- 预览 overwriteAllowed 同时依赖冲突可消解与实际 update 权限；前端弹窗与提交守卫复用 canUpdateEnterpriseResources，并区分无权限和歧义冲突提示。
- 没有更改权限 Manifest、Proto 或生成物。

## 修改文件

- server/internal/service/enterprise_resource.go
- server/internal/service/enterprise_resource_import_test.go
- web/src/pages/enterprise-resources/index.tsx
- web/src/pages/enterprise-resources/components/ResourceImportModal.tsx
- web/src/pages/enterprise-resources/components/ResourceImportModal.permission.test.tsx
- web/src/pages/enterprise-resources/components/ResourceImportModal.unsaved.test.tsx

## 验证

- `go -C server test ./internal/service -run 'Test(PreviewEnterpriseResourceImport|CommitEnterpriseResourceImport|EnterpriseResourceImport)' -count=1` 通过。
- 权限回归覆盖 create-only 新增、create-only 伪造覆盖拒绝且仓储零调用、create+update 覆盖、update-only/无权限拒绝、预览编辑权限、歧义冲突、混合批次拒绝、系统工作区拒绝。
- `pnpm --dir web exec vitest run src/pages/enterprise-resources/components/ResourceImportModal.permission.test.tsx src/pages/enterprise-resources/components/ResourceImportModal.unsaved.test.tsx src/pages/enterprise-resources/resource-types.test.tsx src/pages/enterprise-resources/resource-save.test.tsx` 首轮 16 项通过（当时包含新增停用测试，随后已拆分）。
- 最后定向三文件 `resource-types.test.tsx`、`resource-enabled.test.tsx`、`ResourceImportModal.permission.test.tsx` 11 项通过，最慢文件 14.29 秒。
- 修改文件 Biome、`git diff --check` 通过；Ant Design CLI 对 ResourceImportModal 的 lint 0 问题。

## 限制

未对真实经营数据执行导入；服务端权限失败是在仓储前断言，没有冒充真实库覆盖验收。独立 review、统一 tsc 与最终门禁由主会话继续。
