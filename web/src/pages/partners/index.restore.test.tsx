import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
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

describe('客户黑名单归类与恢复', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.pathname = '/partners/customers';
  });

  it.each([false, true])(
    '移出黑名单后按成功响应恢复客户分类，isCasual=%s',
    async (casual) => {
      const partner = {
        id: 'restore-customer',
        legalName: '待恢复客户',
        roles: [
          {
            type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
            enabled: true,
            blacklisted: true,
          },
        ],
      };
      vi.mocked(partnerServiceListPartners).mockImplementation(
        async (params) => ({
          data: params.blacklisted ? [partner] : [],
          total: params.blacklisted ? 1 : 0,
        }),
      );
      // false 在真实 ProtoJSON 响应中省略，不能据此误判为缺少合作类型。
      vi.mocked(partnerServiceSetPartnerRoleBlacklist).mockResolvedValue({
        success: true,
        data: { id: partner.id, ...(casual ? { isCasual: true } : {}) },
      });
      render(
        <App>
          <Partners />
        </App>,
      );
      fireEvent.click(screen.getByRole('radio', { name: '黑名单' }));
      fireEvent.click(await screen.findByRole('button', { name: /更多/ }));
      fireEvent.click(await screen.findByText('移出黑名单'));
      await waitFor(() =>
        expect(partnerServiceSetPartnerRoleBlacklist).toHaveBeenCalledWith(
          { id: partner.id },
          {
            id: partner.id,
            roleType: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
            blacklisted: false,
          },
        ),
      );
      await waitFor(() =>
        expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
          expect.objectContaining({
            blacklisted: false,
            isCasual: casual,
            page: 1,
          }),
        ),
      );
      expect(
        screen.getByRole('radio', { name: casual ? '散客' : '正式客户' }),
      ).toBeChecked();
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    },
  );

  it('黑名单同时包含正式客户与散客，导出不残留合作类型筛选', async () => {
    const partners = [
      {
        id: 'blacklisted-regular',
        legalName: '已拉黑正式客户',
        roles: [
          {
            type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
            enabled: true,
            blacklisted: true,
          },
        ],
      },
      {
        id: 'blacklisted-casual',
        legalName: '已拉黑散客',
        isCasual: true,
        roles: [
          {
            type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
            enabled: true,
            blacklisted: true,
          },
        ],
      },
    ];
    vi.mocked(partnerServiceListPartners).mockImplementation(
      async (params) => ({
        data: params.blacklisted ? partners : [],
        total: params.blacklisted ? 2 : 0,
      }),
    );
    vi.mocked(partnerServiceExportPartners).mockResolvedValue({
      data: partners,
    });
    render(
      <App>
        <Partners />
      </App>,
    );
    fireEvent.click(screen.getByRole('radio', { name: '散客' }));
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
        expect.objectContaining({ isCasual: true, blacklisted: false }),
      ),
    );
    fireEvent.click(screen.getByRole('radio', { name: '黑名单' }));
    expect(await screen.findByText('已拉黑正式客户')).toBeInTheDocument();
    expect(screen.getByText('已拉黑散客')).toBeInTheDocument();
    expect(partnerServiceListPartners).toHaveBeenLastCalledWith(
      expect.objectContaining({ blacklisted: true, isCasual: undefined }),
    );
    fireEvent.click(screen.getByRole('button', { name: /导出 Excel/ }));
    await waitFor(() =>
      expect(xlsxMock.writeFile).toHaveBeenCalledWith(
        expect.anything(),
        '黑名单客户档案列表.xlsx',
      ),
    );
    expect(partnerServiceExportPartners).toHaveBeenLastCalledWith(
      expect.objectContaining({ blacklisted: true, isCasual: undefined }),
    );
  });
});
