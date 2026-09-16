import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { partnerServiceUpdatePartner } from '@/services/roncin/partnerService';
import RoleSwitchModal from './RoleSwitchModal';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceUpdatePartner: vi.fn(),
}));

describe('RoleSwitchModal 业务角色转换弹窗', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('展示当前单位信息与已生效角色，并支持勾选切换角色提交', async () => {
    const mockPartner = {
      id: 'p-switch-1',
      code: 'CUST-888',
      legalName: '远洋国际货运代理有限公司',
      enabled: true,
      roles: [{ type: 1, enabled: true }],
    } as API.Partner;

    const onSuccess = vi.fn();
    const onClose = vi.fn();

    vi.mocked(partnerServiceUpdatePartner).mockResolvedValue({
      id: 'p-switch-1',
      success: true,
    } as never);

    render(
      <App>
        <RoleSwitchModal
          open={true}
          partner={mockPartner}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      </App>,
    );

    expect(screen.getByText('远洋国际货运代理有限公司')).toBeInTheDocument();
    expect(screen.getByText('CUST-888')).toBeInTheDocument();

    // 点击快捷设置：仅设为供应商
    const supplierBtn = screen.getByRole('button', { name: '仅设为供应商' });
    fireEvent.click(supplierBtn);

    // 点击确认转换
    const submitBtn = screen.getByRole('button', { name: '确认转换' });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(partnerServiceUpdatePartner).toHaveBeenCalledWith(
        { id: 'p-switch-1' },
        expect.objectContaining({
          id: 'p-switch-1',
          roles: [{ type: 2, enabled: true }],
        }),
      );
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });
});
