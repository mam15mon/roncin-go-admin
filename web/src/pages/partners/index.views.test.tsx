import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ListQueryStateProvider } from '@/components/ui/list-query-state';
import { PartnerRoleType } from '@/enums.generated';
import {
  partnerServiceExportPartners,
  partnerServiceListPartners,
  partnerServiceSetPartnerRoleBlacklist,
} from '@/services/roncin/partnerService';
import Partners from './index';

const routeState = vi.hoisted(() => ({
  pathname: '/partners/customers',
}));

const xlsxMock = vi.hoisted(() => ({
  utils: {
    aoa_to_sheet: vi.fn(() => ({})),
    book_new: vi.fn(() => ({})),
    book_append_sheet: vi.fn(),
  },
  writeFile: vi.fn(),
}));

vi.mock('xlsx', () => xlsxMock);

vi.mock('@/router/history', () => ({
  history: {
    get location() {
      return { ...window.location, pathname: routeState.pathname };
    },
    push: vi.fn(),
  },
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOperateBusiness: true,
    canManagePartners: true,
  }),
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useLocation: () => ({ pathname: routeState.pathname }),
  };
});

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartners: vi.fn(),
  partnerServiceExportPartners: vi.fn(),
  partnerServiceImportPartners: vi.fn(),
  partnerServiceSetPartnerRoleBlacklist: vi.fn(),
}));

describe('Partners 页签与黑名单交互', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.pathname = '/partners/customers';
  });

  it('散客页签在搜索、导出与重置时保持散客范围，切换其他页签后清除强制筛选', async () => {
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [],
      total: 0,
    });
    vi.mocked(partnerServiceExportPartners).mockResolvedValue({
      data: [{ code: 'CASUAL-1', legalName: '散客导出公司' }],
    });
    render(
      <App>
        <Partners />
      </App>,
    );
    await waitFor(() => expect(partnerServiceListPartners).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('radio', { name: '散客' }));
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({
          isCasual: true,
          blacklisted: undefined,
          page: 1,
        }),
      ),
    );
    expect(
      screen.queryByText('合作类型', { selector: '.ant-select-placeholder' }),
    ).not.toBeInTheDocument();

    fireEvent.change(
      screen.getByPlaceholderText('搜索单位编码、名称、拼音或税号...'),
      { target: { value: '单次合作' } },
    );
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({ keyword: '单次合作', isCasual: true }),
      ),
    );
    fireEvent.click(screen.getByRole('button', { name: /导出 Excel/ }));
    await waitFor(() =>
      expect(xlsxMock.writeFile).toHaveBeenCalledWith(
        expect.anything(),
        '散客客户档案列表.xlsx',
      ),
    );
    expect(partnerServiceExportPartners).toHaveBeenLastCalledWith({
      keyword: '单次合作',
      role: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
      enabled: undefined,
      isCasual: true,
      blacklisted: undefined,
    });

    fireEvent.click(screen.getByRole('button', { name: /重置/ }));
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({ keyword: undefined, isCasual: true }),
      ),
    );
    fireEvent.click(screen.getByRole('radio', { name: '黑名单客户' }));
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({ isCasual: undefined, blacklisted: true }),
      ),
    );
    fireEvent.click(screen.getByRole('radio', { name: '全部客户' }));
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({
          isCasual: undefined,
          blacklisted: undefined,
        }),
      ),
    );
  });

  it('返回列表恢复散客页签与已提交的查询条件', async () => {
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [],
      total: 0,
    });
    const page = (visible: boolean) => (
      <ListQueryStateProvider enabled>
        <App>{visible && <Partners />}</App>
      </ListQueryStateProvider>
    );
    const { rerender } = render(page(true));
    fireEvent.click(screen.getByRole('radio', { name: '散客' }));
    fireEvent.change(
      screen.getByPlaceholderText('搜索单位编码、名称、拼音或税号...'),
      { target: { value: '临时客户' } },
    );
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({ keyword: '临时客户', isCasual: true }),
      ),
    );
    rerender(page(false));
    vi.mocked(partnerServiceListPartners).mockClear();
    rerender(page(true));
    expect(screen.getByRole('radio', { name: '散客' })).toBeChecked();
    expect(
      screen.getByPlaceholderText('搜索单位编码、名称、拼音或税号...'),
    ).toHaveValue('临时客户');
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({ keyword: '临时客户', isCasual: true }),
      ),
    );
  });

  it('客户黑名单状态使用明确选项且无需说明即可加入黑名单', async () => {
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [
        {
          id: 'customer-1',
          legalName: '正常客户',
          roles: [
            { type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER, enabled: true },
          ],
        },
      ],
      total: 1,
    });
    vi.mocked(partnerServiceSetPartnerRoleBlacklist).mockResolvedValue({
      success: true,
      data: { id: 'customer-1' },
    });
    render(
      <App>
        <Partners />
      </App>,
    );
    fireEvent.click(await screen.findByRole('button', { name: /更多/ }));
    fireEvent.click(await screen.findByText('黑名单'));
    expect(
      await screen.findByRole('radio', { name: '正常合作' }),
    ).toBeChecked();
    expect(screen.queryByRole('switch')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('变更原因与说明')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('radio', { name: '已加入黑名单' }));
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));
    await waitFor(() =>
      expect(partnerServiceSetPartnerRoleBlacklist).toHaveBeenCalledWith(
        { id: 'customer-1' },
        {
          id: 'customer-1',
          roleType: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
          blacklisted: true,
        },
      ),
    );
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenCalledTimes(2),
    );
  });
});
