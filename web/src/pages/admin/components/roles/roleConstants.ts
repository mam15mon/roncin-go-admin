import { AdminDataScope } from '@/enums.generated';
// 新建角色默认勾选的基础权限：没有它用户登录后没有任何可进页面。
export const ROLE_BASE_PERMISSION_KEY = 'system.platform.access';

export const dataScopeOptions = [
  {
    label: '全部组织',
    value: AdminDataScope.DATA_SCOPE_ALL,
    color: 'purple',
    description: '可跨越组织边界访问全平台业务与管理数据',
  },
  {
    label: '当前组织',
    value: AdminDataScope.DATA_SCOPE_ORGANIZATION,
    color: 'orange',
    description: '仅能访问用户当前所在组织的业务数据',
  },
  {
    label: '组织树',
    value: AdminDataScope.DATA_SCOPE_ORGANIZATION_TREE,
    color: 'cyan',
    description: '可访问当前组织及所有直属或深层下级组织的业务数据',
  },
];

export const dataScopeMap = new Map<
  number,
  { label: string; value: number; color: string; description: string }
>(
  [
    ...dataScopeOptions,
    {
      label: '仅本人（已停用，需调整）',
      value: AdminDataScope.DATA_SCOPE_SELF,
      color: 'default',
      description: '此范围不再授予权限，请管理员明确选择新的范围',
    },
  ].map((item) => [item.value, item]),
);

export type RoleFormValues = {
  name?: string;
  dataScope?: number;
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
