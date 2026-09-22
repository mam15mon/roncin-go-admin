import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import OrgCreateModal from './OrgCreateModal';

const { getCurrencyOptionsMock, adminServiceCreateOrganizationMock } =
  vi.hoisted(() => ({
    getCurrencyOptionsMock: vi.fn(),
    adminServiceCreateOrganizationMock: vi.fn(),
  }));

vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: getCurrencyOptionsMock,
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceCreateOrganization: adminServiceCreateOrganizationMock,
}));

describe('OrgCreateModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getCurrencyOptionsMock.mockResolvedValue([
      { label: 'CNY - 人民币', value: 'CNY' },
      { label: 'USD - 美元', value: 'USD' },
    ]);
  });

  afterEach(() => {
    cleanup();
  });

  it('新增独立公司时渲染本币下拉框并默认选中 CNY', async () => {
    render(
      <App>
        <OrgCreateModal
          open={true}
          onOpenChange={vi.fn()}
          parentOrg={null}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    expect(screen.getByText(/新增公司/)).toBeInTheDocument();
    expect(screen.getByLabelText('组织编码')).toBeInTheDocument();
    expect(screen.getByLabelText('组织名称')).toBeInTheDocument();
    adminServiceCreateOrganizationMock.mockResolvedValue({
      data: { id: 'company-new' },
    });
    fireEvent.change(screen.getByLabelText('组织编码'), {
      target: { value: 'COMPANY_A' },
    });
    fireEvent.change(screen.getByLabelText('组织名称'), {
      target: { value: '测试公司' },
    });
    // 本币应为选择器（combobox），具备下拉选择属性
    const baseCurrencySelect = screen.getByRole('combobox', { name: '本币' });
    expect(baseCurrencySelect).toBeInTheDocument();

    await waitFor(() => {
      expect(getCurrencyOptionsMock).toHaveBeenCalled();
    });
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    await waitFor(() =>
      expect(adminServiceCreateOrganizationMock).toHaveBeenCalledWith({
        code: 'COMPANY_A',
        name: '测试公司',
        parentId: undefined,
        kind: 2,
        baseCurrency: 'CNY',
      }),
    );
  });
});
