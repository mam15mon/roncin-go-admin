export const dataScopeOptions = [
  {
    label: '全部组织',
    value: 1,
    color: 'purple',
    description: '可跨越组织边界访问全平台业务与管理数据',
  },
  {
    label: '当前组织',
    value: 2,
    color: 'orange',
    description: '仅能访问用户当前所在组织的业务数据',
  },
  {
    label: '组织树',
    value: 3,
    color: 'cyan',
    description: '可访问当前组织及所有直属或深层下级组织的业务数据',
  },
  {
    label: '仅本人',
    value: 4,
    color: 'default',
    description: '仅能访问本人创建或直接参与指派的单据与业务数据',
  },
];

export const dataScopeMap = new Map(
  dataScopeOptions.map((item) => [item.value, item]),
);

export type RoleFormValues = {
  code?: string;
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
  isLeaf: boolean;
};

export type PermissionGroupNode = {
  key: string;
  title: string;
  groupName: string;
  isLeaf: boolean;
  children: PermissionLeafNode[];
};

export type OrderOrganizationAccess = {
  organizationId: string;
  writable: boolean;
};
