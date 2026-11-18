import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import PartnerDetailPage from './partner-detail';

const routeState = vi.hoisted(() => ({
  params: { id: 'create' } as { id?: string },
  pathname: '/partners/customers/create',
  search: '?legalName=%E4%B8%8A%E6%B5%B7%20%E5%AE%89%E5%8F%AF',
}));

const accessState = vi.hoisted(() => ({
  canOperateOrganization: vi.fn(() => true),
  canOperateBusiness: true,
  canManagePartners: false,
  canReadPartners: true,
  canCreatePartners: true,
  canUpdatePartners: true,
  canReadPartnerAccounts: true,
  canCreatePartnerAccounts: true,
  canUpdatePartnerAccounts: true,
  canReadPartnerContracts: true,
  canCreatePartnerContracts: true,
  canUpdatePartnerContracts: true,
  canReadPartnerSettlementRules: true,
  canCreatePartnerSettlementRules: true,
  canUpdatePartnerSettlementRules: true,
  canReadPartnerShippingPresets: true,
  canCreatePartnerShippingPresets: true,
  canUpdatePartnerShippingPresets: true,
  canReadPartnerAudit: true,
}));

vi.mock('@/components/layout/useRouteLeaveGuard', () => ({
  useRouteLeaveGuard: vi.fn(),
}));

vi.mock('@/router/history', () => ({
  history: {
    get location() {
      return { ...window.location, pathname: routeState.pathname };
    },
    push: vi.fn(),
    replace: vi.fn(),
  },
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'draft-user',
        currentOrganization: { id: 'draft-company', baseCurrency: 'CNY' },
      },
    },
  }),
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState,
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useLocation: () => ({
      pathname: routeState.pathname,
      search: routeState.search,
    }),
    useParams: () => routeState.params,
    useSearchParams: () => [new URLSearchParams(routeState.search)],
  };
});

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListOrganizations: vi.fn().mockResolvedValue({ data: [] }),
  adminServiceListUsers: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
  partnerServiceGetPartner: vi.fn(),
  partnerServiceListPartnerAccounts: vi.fn().mockResolvedValue({ data: [] }),
  partnerServiceListPartnerAssignmentOptions: vi
    .fn()
    .mockResolvedValue({ data: [] }),
  partnerServiceListPartnerSettlementRules: vi.fn(),
  partnerServiceListPartnerContracts: vi.fn().mockResolvedValue({ data: [] }),
  partnerServiceListPartnerShippingPresets: vi
    .fn()
    .mockResolvedValue({ data: [] }),
  partnerServiceListPartnerAuditLogs: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0 }),
  partnerServiceUpdatePartner: vi.fn(),
}));

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: vi.fn().mockResolvedValue({ success: true }),
}));

beforeEach(() => {
  sessionStorage.clear();
  vi.clearAllMocks();
});

describe('客商页面草稿恢复', () => {
  it('切换再进入保留新建档案的字段和别名，并保留显式清空', async () => {
    const first = renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '跨页签客户名称' },
    });
    const alias = screen
      .getByText('公司别名')
      .closest('.ant-form-item')
      ?.querySelector('input');
    if (!alias) throw new Error('缺少别名输入框');
    fireEvent.change(alias, { target: { value: '客户别名' } });
    fireEvent.keyDown(alias, { key: 'Enter', code: 'Enter' });
    await screen.findByText('客户别名');
    first.unmount();

    const second = renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('跨页签客户名称'),
    );
    expect(screen.getByText('客户别名')).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('公司抬头'), {
      target: { value: '' },
    });
    second.unmount();

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue(''),
    );
    expect(screen.getByText('客户别名')).toBeInTheDocument();
    expect(partnerServiceGetPartner).not.toHaveBeenCalled();
  });
});
