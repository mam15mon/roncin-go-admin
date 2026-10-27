import type { AccessState } from './routeTypes';

type WorkspaceTab = {
  key: string;
  visible: (access: AccessState) => boolean;
};

/** 企业资源页与导航共同使用的功能子页签，影像启用状态仍由页面接口控制。 */
export const enterpriseResourceTabs = [
  { key: 'addresses', label: '地址管理', type: 1 },
  { key: 'remarks', label: '备注管理', type: 2 },
  { key: 'images', label: '图片管理', type: 3 },
  { key: 'tags', label: '标签管理', type: 4 },
  { key: 'consignees', label: '收货人管理', type: 6 },
  { key: 'shippers', label: '发货人管理', type: 5 },
  { key: 'notify-parties', label: '通知人管理', type: 7 },
] as const;

/** 页面展示与组织切换共用子页签的可读条件，不用菜单可见性扩大权限。 */
export const workspaceTabs = {
  '/enterprise-resources/config': enterpriseResourceTabs.map(({ key }) => ({
    key,
    visible: (access: AccessState) => access.canReadEnterpriseResources,
  })),
  '/admin': [
    { key: 'organizations', visible: (access) => access.canReadOrganizations },
    { key: 'users', visible: (access) => access.canReadUsers },
    { key: 'roles', visible: (access) => access.canReadRoles },
    {
      key: 'number-rules',
      visible: (access) => access.canReadMasterDataNumberRules,
    },
    {
      key: 'abnormal-cases',
      visible: (access) => access.canReadMasterDataItems,
    },
    { key: 'audit', visible: (access) => access.canReadAudit },
    { key: 'background-tasks', visible: (access) => access.canReadTasks },
    { key: 'permissions', visible: (access) => access.canReadPermissions },
  ],
  '/master-data': [
    { key: 'ports', visible: (access) => access.canReadMasterDataPorts },
    { key: 'airports', visible: (access) => access.canReadMasterDataAirports },
    { key: 'airlines', visible: (access) => access.canReadMasterDataAirlines },
    {
      key: 'shipping-lines',
      visible: (access) => access.canReadMasterDataShippingLines,
    },
    {
      key: 'shipping-agents',
      visible: (access) => access.canReadMasterDataItems,
    },
    { key: 'countries', visible: (access) => access.canReadMasterDataItems },
    {
      key: 'cities',
      visible: (access) => access.canReadMasterDataAdministrativeRegions,
    },
    {
      key: 'currencies',
      visible: (access) => access.canReadMasterDataCurrencies,
    },
  ],
  '/finance/fee-settings': [
    { key: 'fee-settings', visible: (access) => access.canReadFeeSettings },
    { key: 'billing-units', visible: (access) => access.canReadFeeSettings },
    {
      key: 'taxable-services',
      visible: (access) =>
        access.canOperateBusiness && access.canReadFeeSettings,
    },
    {
      key: 'custom-settings',
      visible: (access) =>
        access.canOperateBusiness && access.canReadFinanceBills,
    },
  ],
} satisfies Record<string, WorkspaceTab[]>;

export type WorkspaceTabPath = keyof typeof workspaceTabs;

export function isWorkspaceTabPath(path: string): path is WorkspaceTabPath {
  return Object.hasOwn(workspaceTabs, path);
}

export function isWorkspaceTabVisible(
  path: WorkspaceTabPath,
  key: string,
  access: AccessState,
): boolean {
  return workspaceTabs[path].some(
    (tab) => tab.key === key && tab.visible(access),
  );
}
