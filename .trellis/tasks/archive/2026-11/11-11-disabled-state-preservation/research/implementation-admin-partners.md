# R3 管理与客商范围实施记录

## 实现

已核对 Proto 中 `AdminOrganization.enabled`、`AdminUserMembership.enabled`、`AdminRole.enabled`、`MasterDataItem.enabled`、`Partner.enabled`、`PartnerShippingPreset.enabled` 均为非 optional bool。成功实体省略该字段时按 false 解释。

- 组织树、拓扑节点与组织编辑表单保持同一启停语义，编辑提交不使用新建默认 true。
- 成员关系和角色编辑初始化、提交保留停用；新增默认仍启用。
- 异常配置编辑的启停与排序非 optional 零值分别为 false/0，新建仍 true/100。
- 客商详情仅显示启停，无注册启停控件，保存必须直接保留已加载的 `partner.enabled`，不能取 `onFinish.values.enabled`。已以真实页面三态回归验证。
- R1 专用角色载荷不再包含 enabled，因此身份操作不会启用停用客商。
- 常用单证预设列表仅显示明确 enabled=true 的项，省略 false 不再混入；编辑保留停用，新建默认 true。
- 未改 optional 查询字段、加载状态、全局请求配置或无关模块。

## 修改文件

`web/src/pages/admin/organization-tree.ts`、`organizations.tsx`、`components/org/OrgEditModal.tsx`、`components/users/UserMembershipModal.tsx`、`components/roles/RoleFormModal.tsx`、`components/AbnormalCasesPanel.tsx` 与对应测试；`web/src/pages/partners/partner-detail.tsx`（已交接 R5 信用修改）、新 `partner-detail-enabled.test.tsx`；`ShippingPresetSection.tsx` 与新 `ShippingPresetSection.enabled.test.tsx`。资源页部分由其他子代理负责。

## 验证命令

```bash
pnpm --dir web exec vitest run src/pages/partners/components/RoleSwitchModal.test.tsx src/pages/partners/components/RoleSwitchModal.feedback.test.tsx src/pages/admin/components/org/OrgEditModal.test.tsx src/pages/admin/components/users/UserMembershipModal.test.tsx src/pages/admin/organization-tree.test.ts src/pages/admin/components/roles/RoleFormModal.test.tsx src/pages/partners/partner-detail-enabled.test.tsx src/pages/partners/partner-detail-save.test.tsx src/pages/partners/partner-detail.test.tsx src/pages/partners/components/ShippingPresetSection.enabled.test.tsx src/pages/admin/components/AbnormalCasesPanel.enabled.test.tsx src/features/audit/audit-presentation.test.ts src/pages/partners/components/AuditLogSection.test.tsx
```

结果：13 文件、76 例全部通过。`pnpm --dir web tsc` 通过。负责修改的 20 个前端文件 Biome lint 通过；两个关键组件本地 antd lint 无问题；`git diff --check` 通过。

新增重型页面用例拆到独立 `partner-detail-enabled.test.tsx`，未向旧保存恢复 suite 堆叠。没有执行全量门禁（由主会话统一）、没有提交或 push。
