import { AdminOrganizationKind } from '@/enums.generated';

export type UserFormValues = {
  username?: string;
  displayName?: string;
  password?: string;
  email?: string;
  enabled?: boolean;
  roleIds?: string[];
};

export type UserMembershipFormValues = {
  organizationId?: string;
  roleIds?: string[];
  enabled?: boolean;
};

export const organizationKindLabels: Record<number, string> = {
  1: '系统管理',
  2: '公司',
  3: '部门',
  4: '组',
};

/**
 * 判定后端组织 kind 是否为工作台节点（系统管理/公司）。
 * 角色库只归属工作台维护，部门与团队共享其所属工作台的角色库，
 * 判定口径与后端 `internal/data/auth.go` 的 isWorkspaceKind 保持一致。
 * 枚举值取自生成常量，不复制裸数字作为第二套真相。
 */
export function isWorkspaceKindValue(kind?: number): boolean {
  return (
    kind === AdminOrganizationKind.ORGANIZATION_KIND_SYSTEM ||
    kind === AdminOrganizationKind.ORGANIZATION_KIND_COMPANY
  );
}

/**
 * 格式化组织层级名称：
 * 若组织为部门或组，向上溯源所属公司，拼接为 `${公司名} / ${部门名}`；
 * 若组织为公司或系统管理，直接展示自身名称。
 */
export function formatOrganizationHierarchyName(
  orgIdOrOrg: string | API.AdminOrganization | undefined,
  organizations: API.AdminOrganization[],
): string {
  if (!orgIdOrOrg) return '';
  const org =
    typeof orgIdOrOrg === 'string'
      ? organizations.find((item) => item.id === orgIdOrOrg)
      : orgIdOrOrg;
  if (!org) {
    return typeof orgIdOrOrg === 'string' ? '' : orgIdOrOrg.name || '';
  }
  // 如果是系统管理或公司，直接展示自身名称
  if (isWorkspaceKindValue(org.kind)) {
    return org.name || '';
  }
  // 部门/团队向上查找父级公司
  const parts: string[] = [org.name || ''];
  let current: API.AdminOrganization | undefined = org;
  const visited = new Set<string>([org.id || '']);
  while (current?.parentId) {
    if (visited.has(current.parentId)) break;
    visited.add(current.parentId);
    const parent: API.AdminOrganization | undefined = organizations.find(
      (item) => item.id === current?.parentId,
    );
    if (!parent) break;
    parts.unshift(parent.name || '');
    // 溯源到公司或系统管理即停
    if (isWorkspaceKindValue(parent.kind)) {
      break;
    }
    current = parent;
  }
  return parts.join(' / ');
}
