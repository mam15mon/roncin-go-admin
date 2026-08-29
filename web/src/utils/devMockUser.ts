/**
 * 本地离线开发免密体验用户（具备全量超管权限与全数据范围）
 */
export const DEV_MOCK_USER: API.CurrentUser = {
  id: 'dev-admin-user-001',
  username: 'admin',
  displayName: '系统管理员 (本地开发)',
  email: 'admin@roncin.com',
  currentOrganization: {
    id: 'org-root-001',
    name: 'Roncin 国际物流总部',
    code: 'RONCIN_HQ',
  },
  roleScopes: [
    {
      roleCode: 'super_admin',
      dataScope: 'all',
    },
  ],
  permissions: [
    'system.platform.access',
    'system.organization.read',
    'system.organization.create',
    'system.organization.update',
    'system.user.read',
    'system.user.create',
    'system.user.update',
    'system.user.delete',
    'system.user.authorize_wecom',
    'system.user.authorize_dingtalk',
    'system.user.reset_password',
    'system.role.read',
    'system.role.create',
    'system.role.update',
    'system.permission.read',
    'system.audit.read',
    'system.finance.exchange_rate.read',
    'system.finance.exchange_rate.create',
    'system.finance.exchange_rate.update',
    'system.finance.exchange_rate.disable',
    'system.finance.exchange_rate.override',
    'system.finance.fee_setting.read',
    'system.finance.fee_setting.create',
    'system.finance.fee_setting.update',
    'system.finance.fee.read',
    'system.finance.bill.read',
    'system.finance.bill.create',
    'system.finance.bill.update',
    'system.finance.bill.confirm',
    'system.finance.invoice.read',
    'system.finance.invoice.create',
    'system.finance.invoice.update',
    'system.finance.cashflow.read',
    'system.finance.cashflow.create',
    'system.finance.cashflow.update',
    'system.finance.verification.read',
    'system.finance.verification.create',
    'system.finance.verification.reverse',
    'system.finance.commission.read',
    'system.finance.commission.manage',
    'business.partner.read',
    'business.partner.create',
    'business.partner.update',
    'business.partner.blacklist',
    'business.partner.import',
    'business.partner.export',
    'business.partner.account.read',
    'business.partner.account.create',
    'business.partner.account.update',
    'business.partner.contract.read',
    'business.partner.contract.create',
    'business.partner.contract.update',
    'business.partner.settlement_rule.read',
    'business.partner.settlement_rule.create',
    'business.partner.settlement_rule.update',
    'business.partner.attachment.read',
    'business.partner.attachment.register',
    'business.partner.shipping_preset.read',
    'business.partner.shipping_preset.create',
    'business.partner.shipping_preset.update',
    'business.partner.audit.read',
    'business.partner.assignment_option.read',
    'system.master_data.currency.read',
    'system.master_data.administrative_region.read',
    'system.master_data.option.read',
    'system.master_data.item.read',
    'system.master_data.item.create',
    'system.master_data.item.update',
    'system.master_data.item.import',
    'system.master_data.port.read',
    'system.master_data.port.create',
    'system.master_data.port.update',
    'system.master_data.airport.read',
    'system.master_data.airport.create',
    'system.master_data.airport.update',
    'system.master_data.airline.read',
    'system.master_data.airline.create',
    'system.master_data.airline.update',
    'system.master_data.shipping_line.read',
    'system.master_data.shipping_line.create',
    'system.master_data.shipping_line.update',
    'system.master_data.number_rule.read',
    'system.master_data.number_rule.create',
    'system.master_data.number_rule.update',
    'system.task.read',
    'system.task.requeue',
    // 海运出口全部权限
    'business.order.se.read',
    'business.order.se.create',
    'business.order.se.update',
    'business.order.se.delete',
    'business.order.se.fee.read',
    'business.order.se.fee.create',
    'business.order.se.fee.update',
    'business.order.se.fee.delete',
    'business.order.se.release_pod.create',
    'business.order.se.abnormal_case.create',
    // 海运进口全部权限
    'business.order.si.read',
    'business.order.si.create',
    'business.order.si.update',
    'business.order.si.delete',
    'business.order.si.fee.read',
    'business.order.si.fee.create',
    'business.order.si.fee.update',
    'business.order.si.fee.delete',
    // 空运出口全部权限
    'business.order.ae.read',
    'business.order.ae.create',
    'business.order.ae.update',
    'business.order.ae.delete',
    'business.order.ae.fee.read',
    'business.order.ae.fee.create',
    // 空运进口全部权限
    'business.order.ai.read',
    'business.order.ai.create',
    'business.order.ai.update',
    'business.order.ai.delete',
    'business.order.ai.fee.read',
  ],
};

const DEV_MOCK_STORAGE_KEY = 'roncin_dev_mock_mode';

export function isDevMockEnabled(): boolean {
  if (typeof window === 'undefined') return false;
  return (
    process.env.NODE_ENV === 'development' &&
    window.localStorage.getItem(DEV_MOCK_STORAGE_KEY) === 'true'
  );
}

export function enableDevMock(): void {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(DEV_MOCK_STORAGE_KEY, 'true');
}

export function disableDevMock(): void {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem(DEV_MOCK_STORAGE_KEY);
}
