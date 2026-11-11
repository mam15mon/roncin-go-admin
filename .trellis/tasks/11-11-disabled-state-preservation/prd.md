# 停用状态往返保护

## 目标与需求

对应父任务 R3：已成功返回的实体非 optional enabled 省略应解释为 false，普通编辑不能悄悄启用。新建默认启用及 optional 存在性保持原契约，真实必填标识缺失仍失败。

## 问题证据

web/src/pages/admin/components/org/OrgEditModal.tsx:43；web/src/pages/partners/partner-detail.tsx:393；RoleSwitchModal.tsx:75；同根候选含 organization-tree、organizations、UserMembershipModal、enterprise-resources、ShippingPresetSection、AbnormalCasesPanel，须逐一先核对 Proto 后纳入。

## 验收

- [ ] 省略 false、显式 false、true 的编辑往返。
- [ ] 组织树与详情一致。
- [ ] 新建仍启用。
- [ ] 成员关系停用不恢复权限。
- [ ] optional 未设置保持未设置。
- [ ] 加载/失败不伪造业务状态。
- [ ] 缺必填 ID 显式失败。。

## 交付边界

客商分支在 R1 之后整合；资源页面同 R2 负责人协调，禁止互相覆盖。
不重置数据库、不自动纠正已有业务金额；保留无关工作。按父任务授权完成独立 review、合并主分支和 push。
