// 新建角色默认勾选的基础权限：没有它用户登录后没有任何可进页面。
export const ROLE_BASE_PERMISSION_KEY = 'system.platform.access';

export type RoleFormValues = {
  name?: string;
  permissionKeys?: string[];
  enabled?: boolean;
};

export type PermissionLeafNode = {
  key: string;
  title: string;
  name: string;
  group: string;
  description?: string;
  requires?: string[];
  isLeaf: boolean;
};

export type PermissionGroupNode = {
  key: string;
  title: string;
  groupName: string;
  path: string[];
  isLeaf: boolean;
  children: PermissionTreeNode[];
};

export type PermissionTreeNode = PermissionGroupNode | PermissionLeafNode;
