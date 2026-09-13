import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import OrgCreateModal from './OrgCreateModal';

const { getCurrencyOptionsMock, adminServiceCreateOrganizationMock } =
  vi.hoisted(() => ({
    getCurrencyOptionsMock: vi.fn(),
    adminServiceCreateOrganizationMock: vi.fn(),
  }));

vi.mock('@/utils/options', () => ({
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

  it('新增公司（上级为总部）时渲染本币下拉框并默认选中 CNY', async () => {
    const parentOrg: API.AdminOrganization = {
      id: 'hq-1',
      code: 'HQ',
      name: '总部',
      kind: 1, // HQ -> child is Company (kind: 2)
    };

    render(
      <App>
        <OrgCreateModal
          open={true}
          onOpenChange={vi.fn()}
          parentOrg={parentOrg}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    expect(screen.getByText(/新增公司（所属上级：总部）/)).toBeInTheDocument();
    expect(screen.getByLabelText('组织编码')).toBeInTheDocument();
    expect(screen.getByLabelText('组织名称')).toBeInTheDocument();
    // 本币应为选择器（combobox），具备下拉选择属性
    const baseCurrencySelect = screen.getByRole('combobox', { name: '本币' });
    expect(baseCurrencySelect).toBeInTheDocument();

    await waitFor(() => {
      expect(getCurrencyOptionsMock).toHaveBeenCalled();
    });
  });
});
