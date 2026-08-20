const permissions = {
  platformAccess: 'system.platform.access',
  organizationManage: 'system.organization.manage',
  userManage: 'system.user.manage',
  roleManage: 'system.role.manage',
  auditRead: 'system.audit.read',
  partnerRead: 'business.partner.read',
  partnerManage: 'business.partner.manage',
} as const;

export default function access(
  initialState: { currentUser?: API.CurrentUser } | undefined,
) {
  const granted = new Set(initialState?.currentUser?.permissions ?? []);
  const has = (permission: string) => granted.has(permission);

  return {
    isAuthenticated: Boolean(initialState?.currentUser),
    canAccessPlatform: has(permissions.platformAccess),
    canManageOrganizations: has(permissions.organizationManage),
    canManageUsers: has(permissions.userManage),
    canManageRoles: has(permissions.roleManage),
    canReadAudit: has(permissions.auditRead),
    canReadPartners: has(permissions.partnerRead),
    canManagePartners: has(permissions.partnerManage),
  };
}
