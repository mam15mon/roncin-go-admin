import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import {
  partnerServiceExportPartners,
  partnerServiceListPartners,
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

describe('企业资源往来单位的互斥分类', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.pathname = '/partners/customers';
  });

  it.each([
    ['/partners/suppliers', PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER],
    [
      '/partners/foreign-agents',
      PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
    ],
  ] as const)(
    '%s 正常合作和黑名单使用互斥筛选，导出范围与列表一致',
    async (pathname, role) => {
      routeState.pathname = pathname;
      vi.mocked(partnerServiceListPartners).mockResolvedValue({
        data: [],
        total: 0,
      });
      vi.mocked(partnerServiceExportPartners).mockResolvedValue({ data: [] });
      render(
        <App>
          <Partners />
        </App>,
      );
      await waitFor(() =>
        expect(partnerServiceListPartners).toHaveBeenCalled(),
      );
      expect(screen.getByRole('radio', { name: '正常合作' })).toBeChecked();
      expect(
        screen.queryByRole('radio', { name: '散客' }),
      ).not.toBeInTheDocument();
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({
          role,
          blacklisted: false,
          isCasual: undefined,
        }),
      );
      fireEvent.click(screen.getByRole('button', { name: /导出 Excel/ }));
      await waitFor(() =>
        expect(partnerServiceExportPartners).toHaveBeenLastCalledWith(
          expect.objectContaining({
            role,
            blacklisted: false,
            isCasual: undefined,
          }),
        ),
      );
      await waitFor(() =>
        expect(
          screen.getByRole('button', { name: /导出 Excel/ }),
        ).not.toHaveClass('ant-btn-loading'),
      );
      fireEvent.click(screen.getByRole('radio', { name: '黑名单' }));
      await waitFor(() =>
        expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
          expect.objectContaining({
            role,
            blacklisted: true,
            isCasual: undefined,
            page: 1,
          }),
        ),
      );
      fireEvent.click(screen.getByRole('button', { name: /导出 Excel/ }));
      await waitFor(() =>
        expect(partnerServiceExportPartners).toHaveBeenLastCalledWith(
          expect.objectContaining({
            role,
            blacklisted: true,
            isCasual: undefined,
          }),
        ),
      );
      await waitFor(() =>
        expect(
          screen.getByRole('button', { name: /导出 Excel/ }),
        ).not.toHaveClass('ant-btn-loading'),
      );
      fireEvent.click(screen.getByRole('radio', { name: '正常合作' }));
      await waitFor(() =>
        expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
          expect.objectContaining({
            role,
            blacklisted: false,
            isCasual: undefined,
          }),
        ),
      );
    },
  );
});
