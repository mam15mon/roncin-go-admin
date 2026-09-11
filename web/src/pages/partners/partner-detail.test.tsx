import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  partnerServiceGetPartner,
  partnerServiceListPartnerSettlementRules,
} from '@/services/roncin/partnerService';
import PartnerDetailPage from './partner-detail';

const routeState = vi.hoisted(() => ({
  params: { id: 'create' } as { id?: string },
  pathname: '/partners/customers/create',
  search: '?legalName=%E4%B8%8A%E6%B5%B7%20%E5%AE%89%E5%8F%AF',
}));

vi.mock('@umijs/max', () => ({
  history: { push: vi.fn() },
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  useLocation: () => ({
    pathname: routeState.pathname,
    search: routeState.search,
  }),
  useParams: () => routeState.params,
  useSearchParams: () => [new URLSearchParams(routeState.search)],
  useAccess: () => ({
    canReadPartnerAccounts: true,
    canCreatePartnerAccounts: true,
    canUpdatePartnerAccounts: true,
  }),
}));

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
  partnerServiceUpdatePartner: vi.fn(),
}));

vi.mock('@/utils/options', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));

describe('PartnerDetailPage 创建预填', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.params = { id: 'create' };
    routeState.pathname = '/partners/customers/create';
    routeState.search = '?legalName=%E4%B8%8A%E6%B5%B7%20%E5%AE%89%E5%8F%AF';
    vi.mocked(partnerServiceGetPartner).mockReset();
    vi.mocked(partnerServiceListPartnerSettlementRules).mockReset();
  });

  it('创建模式按当前查询参数预填，并在同组件地址变化时重建默认值', async () => {
    const { rerender } = render(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    expect(await screen.findByLabelText('公司抬头')).toHaveValue('上海 安可');

    routeState.search = '?legalName=%E6%96%B0%26%E5%85%AC%E5%8F%B8';
    rerender(
      <App>
        <PartnerDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('新&公司'),
    );

    routeState.search = '';
    rerender(
      <App>
        <PartnerDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue(''),
    );
    expect(partnerServiceGetPartner).not.toHaveBeenCalled();
  });

  it('编辑模式忽略 legalName 查询参数并使用服务端档案名称', async () => {
    routeState.params = { id: 'partner-1' };
    routeState.search = '?legalName=%E4%B8%8D%E5%BA%94%E8%A6%86%E7%9B%96';
    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: 'partner-1',
        legalName: '服务端伙伴名称',
        enabled: true,
      } as never,
    });
    vi.mocked(partnerServiceListPartnerSettlementRules).mockResolvedValue({
      data: [],
    } as never);

    render(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('服务端伙伴名称'),
    );
    expect(screen.getByLabelText('公司抬头')).not.toHaveValue('不应覆盖');
    expect(partnerServiceGetPartner).toHaveBeenCalledWith({ id: 'partner-1' });
  });

  it('编辑散客档案时页头展示散客标签且单次合作开关开启', async () => {
    routeState.params = { id: 'partner-casual' };
    routeState.search = '';
    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: 'partner-casual',
        code: 'P00000001',
        legalName: '单次合作散客公司',
        enabled: true,
        isCasual: true,
      } as never,
    });
    vi.mocked(partnerServiceListPartnerSettlementRules).mockResolvedValue({
      data: [],
    } as never);

    render(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('单次合作散客公司'),
    );
    const headerTags = document.querySelector('.roncin-page-header-tags');
    expect(headerTags).toHaveTextContent('散客');
    expect(
      screen.getByRole('switch', { name: '单次合作 (散客)' }),
    ).toBeChecked();
  });
});
