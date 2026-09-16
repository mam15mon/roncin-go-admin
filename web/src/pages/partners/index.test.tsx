import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import Partners from './index';

const routeState = vi.hoisted(() => ({
  pathname: '/partners/customers',
}));

vi.mock('@umijs/max', () => ({
  history: { push: vi.fn() },
  useLocation: () => ({ pathname: routeState.pathname }),
  useAccess: () => ({
    canManagePartners: true,
  }),
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartners: vi.fn(),
  partnerServiceExportPartners: vi.fn(),
  partnerServiceImportPartners: vi.fn(),
  partnerServiceSetSupplierBlacklist: vi.fn(),
}));

describe('Partners 列表页', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('表格渲染合作类型列与快捷筛选，并正确展示散客与正式标签', async () => {
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [
        {
          id: 'p-1',
          code: 'CUST-CASUAL',
          legalName: '单次合作散客公司',
          isCasual: true,
          enabled: true,
          roles: [{ type: 1, enabled: true }],
        },
        {
          id: 'p-2',
          code: 'CUST-REGULAR',
          legalName: '长期正式合作公司',
          isCasual: false,
          enabled: true,
          roles: [{ type: 1, enabled: true }],
        },
      ],
      total: 2,
    } as never);

    render(
      <App>
        <Partners />
      </App>,
    );

    expect(await screen.findByText('单次合作散客公司')).toBeInTheDocument();
    expect(screen.getByText('长期正式合作公司')).toBeInTheDocument();

    // 合作类型表头
    expect(
      screen.getByRole('columnheader', { name: '合作类型' }),
    ).toBeInTheDocument();

    // 行内散客与正式 Tag
    expect(screen.getByText('散客')).toBeInTheDocument();
    expect(screen.getByText('正式')).toBeInTheDocument();

    // 快捷筛选包含合作类型
    const placeholders = screen.getAllByText('合作类型');
    expect(placeholders.length).toBeGreaterThanOrEqual(1);
  });

  it('国外代理列表页隐藏合作类型列与散客筛选', async () => {
    routeState.pathname = '/partners/foreign-agents';
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [
        {
          id: 'fa-1',
          code: 'AGT-001',
          legalName: 'Apex Global Logistics Ltd.',
          enabled: true,
          roles: [{ type: 3, enabled: true }],
        },
      ],
      total: 1,
    } as never);

    render(
      <App>
        <Partners />
      </App>,
    );

    expect(
      await screen.findByText('Apex Global Logistics Ltd.'),
    ).toBeInTheDocument();

    // 国外代理不应有合作类型表头与快捷筛选
    expect(
      screen.queryByRole('columnheader', { name: '合作类型' }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('合作类型')).not.toBeInTheDocument();
  });
});
