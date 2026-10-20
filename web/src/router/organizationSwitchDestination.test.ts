import { describe, expect, it } from 'vitest';
import access from '@/access';
import { AuthOrganizationKind } from '@/enums.generated';
import { resolveOrganizationSwitchDestination } from './organizationSwitchDestination';

function capabilities(keys: string[], system = false) {
  return access({
    currentUser: {
      id: 'admin',
      currentOrganization: {
        id: 'new-org',
        kind: system
          ? AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM
          : AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
      },
      permissionCapabilities: keys.map((key) => ({
        key,
        dataScope: system ? 'all' : 'organization',
      })),
    },
  });
}
function destination(url: string, state: ReturnType<typeof access>) {
  const parsed = new URL(url, 'https://example.test');
  return resolveOrganizationSwitchDestination(parsed, state);
}
const companyAccess = capabilities([
  'system.platform.access',
  'system.role.read',
  'system.user.read',
  'business.partner.read',
  'business.order.se.read',
  'system.finance.fee.read',
  'system.finance.fee_setting.read',
  'system.finance.bill.read',
]);

describe('组织切换功能落点', () => {
  it('角色子页签保留，所有旧筛选、分页、记录参数和hash被清理', () => {
    expect(
      destination('/admin?tab=roles&page=9&roleId=old#edit', companyAccess),
    ).toEqual({ path: '/admin?tab=roles', accessDenied: false });
    expect(
      destination('/finance/bills?page=3&billId=old#detail', companyAccess),
    ).toEqual({ path: '/finance/bills', accessDenied: false });
  });

  it.each([
    ['/orders/sea-export/old?tab=fees', '/orders/sea-export'],
    ['/orders/sea-export/new?copyFrom=old', '/orders/sea-export'],
    ['/orders/sea-export/old/split', '/orders/sea-export'],
    ['/orders/sea-export/old/fees', '/orders/sea-export'],
    ['/partners/customers/old?tab=contacts', '/partners/customers'],
    ['/partners/customers/create', '/partners/customers'],
    ['/partners/suppliers/create', '/partners/suppliers'],
    ['/partners/foreign-agents/old', '/partners/foreign-agents'],
    ['/finance/fees/detail/old', '/finance/fees'],
  ])('记录路径 %s 返回实际列表 %s', (url, target) => {
    expect(destination(url, companyAccess)).toEqual({
      path: target,
      accessDenied: false,
    });
  });

  it('目标路由用新身份检查，Any订单能力不放行另一品类列表', () => {
    expect(
      destination(
        '/orders/sea-export/old',
        capabilities(['business.order.ae.read']),
      ),
    ).toEqual({ path: '/welcome', accessDenied: true });
    expect(
      destination(
        '/orders/sea-export/old',
        capabilities(['business.order.se.read'], true),
      ),
    ).toEqual({ path: '/welcome', accessDenied: true });
    expect(destination('/orders/air-export/old', companyAccess)).toEqual({
      path: '/welcome',
      accessDenied: true,
    });
    expect(destination('/orders/air-export/old', capabilities(['business.order.ae.read']))).toEqual({
      path: '/orders/air-export',
      accessDenied: false,
    });
  });

  it('父路由可进入也不能保留无权限的角色子页签', () => {
    expect(
      destination(
        '/admin?tab=roles',
        capabilities(['system.platform.access', 'system.user.read']),
      ),
    ).toEqual({ path: '/welcome', accessDenied: true });
  });

  it('不存在的子页签及缺省页签选择当前真实可见的第一项', () => {
    expect(
      destination('/admin?tab=old-record&roleId=old', companyAccess),
    ).toEqual({ path: '/admin?tab=users', accessDenied: false });
    expect(destination('/admin', companyAccess)).toEqual({
      path: '/admin?tab=users',
      accessDenied: false,
    });
    expect(
      destination('/admin', capabilities(['system.platform.access'])),
    ).toEqual({ path: '/welcome', accessDenied: true });
  });

  it('费用设置按页面共同条件保留合法页签，系统工作台不保留经营专属页签', () => {
    expect(
      destination(
        '/finance/fee-settings?tab=custom-settings&billId=old',
        companyAccess,
      ),
    ).toEqual({
      path: '/finance/fee-settings?tab=custom-settings',
      accessDenied: false,
    });
    expect(
      destination(
        '/finance/fee-settings?tab=taxable-services',
        capabilities(['system.finance.fee_setting.read'], true),
      ),
    ).toEqual({ path: '/welcome', accessDenied: true });
    expect(
      destination(
        '/finance/fee-settings?tab=billing-units',
        capabilities(['system.finance.fee_setting.read'], true),
      ),
    ).toEqual({
      path: '/finance/fee-settings?tab=billing-units',
      accessDenied: false,
    });
  });

  it('企业资源的地址、影像等功能子页签保持，但清除旧资源参数', () => {
    const state = capabilities(['business.enterprise_resource.read']);
    expect(
      destination(
        '/enterprise-resources/config?tab=images&resourceId=old&page=3#preview',
        state,
      ),
    ).toEqual({
      path: '/enterprise-resources/config?tab=images',
      accessDenied: false,
    });
    expect(
      destination('/enterprise-resources/config?tab=addresses', state),
    ).toEqual({
      path: '/enterprise-resources/config?tab=addresses',
      accessDenied: false,
    });
    expect(
      destination('/enterprise-resources/config?tab=unknown', state),
    ).toEqual({
      path: '/enterprise-resources/config?tab=addresses',
      accessDenied: false,
    });
  });

  it('主数据只保留新组织有权限的页签', () => {
    const state = capabilities(['system.master_data.port.read'], true);
    expect(destination('/master-data?tab=ports&page=2', state)).toEqual({
      path: '/master-data?tab=ports',
      accessDenied: false,
    });
    expect(destination('/master-data?tab=currencies', state)).toEqual({
      path: '/welcome',
      accessDenied: true,
    });
  });

  it.each([
    '/unknown/page',
    '/commission-adjustments/old/my-supplement-source',
    '/user/login',
  ])('没有对应列表的路径 %s 回工作台', (url) => {
    expect(destination(url, companyAccess)).toEqual({
      path: '/welcome',
      accessDenied: false,
    });
  });
});
