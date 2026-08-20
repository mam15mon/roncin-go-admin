const permissions = {
  platformAccess: 'system.platform.access',
  organizationManage: 'system.organization.manage',
  userManage: 'system.user.manage',
  roleManage: 'system.role.manage',
  auditRead: 'system.audit.read',
  partnerRead: 'business.partner.read',
  partnerManage: 'business.partner.manage',
  masterDataRead: 'system.master_data.read',
  masterDataManage: 'system.master_data.manage',
} as const;

export default function access(
  initialState: { currentUser?: API.CurrentUser } | undefined,
) {
  const granted = new Set(initialState?.currentUser?.permissions ?? []);
  const roleScopes = initialState?.currentUser?.roleScopes ?? [];
  const has = (permission: string) => granted.has(permission);
  const hasScope = (minimum: string) => {
    const rank: Record<string, number> = {
      self: 1,
      organization: 2,
      organization_tree: 3,
      all: 4,
    };
    return roleScopes.some((scope) => (rank[scope.dataScope ?? ''] ?? 0) >= (rank[minimum] ?? 0));
  };

  return {
    isAuthenticated: Boolean(initialState?.currentUser),
    canAccessPlatform: has(permissions.platformAccess),
    canManageOrganizations: has(permissions.organizationManage) && hasScope('all'),
    canManageUsers: has(permissions.userManage) && hasScope('organization'),
    canManageRoles: has(permissions.roleManage) && hasScope('organization'),
    canReadAudit: has(permissions.auditRead) && hasScope('organization'),
    canReadPartners: has(permissions.partnerRead) && hasScope('organization'),
    canManagePartners: has(permissions.partnerManage) && hasScope('organization'),
    canReadMasterData: has(permissions.masterDataRead) && hasScope('organization'),
    canManageMasterData: has(permissions.masterDataManage) && hasScope('organization'),
  };
}
