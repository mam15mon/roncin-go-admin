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
