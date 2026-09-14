export type UserFormValues = {
  username?: string;
  displayName?: string;
  password?: string;
  email?: string;
  enabled?: boolean;
  roleIds?: string[];
  organizationId?: string;
};

export type UserMembershipFormValues = {
  organizationId?: string;
  roleIds?: string[];
  enabled?: boolean;
  primary?: boolean;
};

export const organizationKindLabels: Record<number, string> = {
  1: '总部',
  2: '公司',
  3: '部门',
  4: '组',
};

export function pendingExternalProvider(
  user?: API.AdminUser,
): 'wecom' | 'dingtalk' | undefined {
  if (user?.status !== 2) return undefined;
  if (user.wecomUserid) return 'wecom';
  if (user.dingtalkUnionid) return 'dingtalk';
  return undefined;
}

// 解析用户编辑表单的锚定组织：优先取 primary 成员关系所在组织，其次第一条启用
// 关系，最后退回列表第一条。后端按「工作台范围锚定成员关系」决定角色归属组织，
// 前端在已加载的成员关系数据上解析同一锚定组织，用于拉取角色选项；保存传出的
// roleIds 即该组织的角色，与后端按锚定组织的角色校验一致。
export function resolveAnchorOrganizationId(
  memberships: API.AdminUserMembership[],
): string | undefined {
  const primary = memberships.find((membership) => membership.primary);
  if (primary?.organizationId) return primary.organizationId;
  const enabled = memberships.find((membership) => membership.enabled);
  if (enabled?.organizationId) return enabled.organizationId;
  return memberships[0]?.organizationId;
}
