import {
  adminServiceListOrganizationRoles,
  adminServiceListRoles,
} from '@/services/roncin/adminService';
import { unwrapList } from '@/utils/api';
import { buildOrgTree } from '../../organization-tree';

/** 新建邀请的默认有效期（小时）：7 天。 */
export const INVITATION_DEFAULT_TTL_HOURS = 168;

/** 邀请有效期候选（小时），服务端允许 1-720。 */
export const INVITATION_TTL_OPTIONS = [
  { label: '24 小时', value: 24 },
  { label: '3 天', value: 72 },
  { label: '7 天（推荐）', value: INVITATION_DEFAULT_TTL_HOURS },
  { label: '30 天', value: 720 },
];

/** 国内手机号输入校验（允许 +86 前缀，服务端会先归一化 +86/0086 前缀）。 */
export const CHINA_MOBILE_PATTERN = /^(?:\+?86)?1[3-9]\d{9}$/;

export type SelectOption = { label: string; value: string };

/** 组织下拉选项，与用户管理一致使用「名称 (编码)」展示。 */
export function organizationSelectOptions(
  organizations: API.AdminOrganization[],
): SelectOption[] {
  return organizations
    .filter((organization) => organization.id)
    .map((organization) => ({
      label: `${organization.name ?? '-'} (${organization.code ?? '-'})`,
      value: organization.id as string,
    }));
}

/** 角色下拉选项。 */
export function roleSelectOptions(roles: API.AdminRole[]): SelectOption[] {
  return roles
    .filter((role) => role.id)
    .map((role) => ({
      label: `${role.name ?? '-'} (${role.code ?? '-'})`,
      value: role.id as string,
    }));
}

/**
 * 解析组织树根节点 ID：总部兜底注册的审批路由组织是注册收口组织（总部根）。
 * 复用 organization-tree 的 buildOrgTree 根判定（parentId 为空或指向不存在
 * 组织的节点作为根），列表为空时返回 undefined，由调用方回退到当前组织。
 */
export function resolveRootOrganizationId(
  organizations: API.AdminOrganization[],
): string | undefined {
  if (organizations.length === 0) return undefined;
  return buildOrgTree(organizations).treeData[0]?.key;
}

/**
 * 按目标组织加载角色：目标组织是当前组织时走组织内 ListRoles，
 * 其他组织走 ListOrganizationRoles（需要全局角色读取权限，由调用方按
 * canReadRoles 门控，服务端是最终权限裁决方）。
 */
export async function fetchRolesForOrganization(
  organizationId: string | undefined,
  currentOrganizationId?: string,
): Promise<API.AdminRole[]> {
  if (!organizationId) return [];
  if (organizationId === currentOrganizationId) {
    return unwrapList(await adminServiceListRoles());
  }
  return unwrapList(
    await adminServiceListOrganizationRoles({ organizationId }),
  );
}
