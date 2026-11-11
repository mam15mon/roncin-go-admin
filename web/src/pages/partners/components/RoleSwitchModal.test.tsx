import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { partnerServiceUpdatePartnerRoles } from '@/services/roncin/partnerService';
import RoleSwitchModal from './RoleSwitchModal';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceUpdatePartnerRoles: vi.fn(),
}));

describe('RoleSwitchModal 业务角色转换弹窗', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it.each([undefined, false, true])(
    '原档案启停=%s，切换身份只发送角色集合',
    async (enabled) => {
      const mockPartner = {
        id: 'p-switch-1',
        code: 'CUST-888',
        legalName: '远洋国际货运代理有限公司',
        ...(enabled === undefined ? {} : { enabled }),
        isCasual: true,
        contacts: [{ name: '原联系人' }],
        aliases: [{ aliasName: '原别名' }],
        profile: { nameEn: 'Original Customer' },
        assignments: [{ userId: 'sales-1', role: 3 }],
        roles: [{ type: 1, enabled: true }],
      } as API.Partner;

      const onSuccess = vi.fn();
      const onClose = vi.fn();

      vi.mocked(partnerServiceUpdatePartnerRoles).mockResolvedValue({
        data: { id: 'p-switch-1' },
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
      expect(screen.getByRole('checkbox', { name: '客户' })).toBeChecked();
      expect(screen.queryByText('当前生效角色：')).not.toBeInTheDocument();
      expect(screen.queryByText('角色转换规则说明')).not.toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: '仅设为供应商' }),
      ).not.toBeInTheDocument();

      // 取消客户身份并勾选供应商，直接编辑最终身份。
      fireEvent.click(screen.getByRole('checkbox', { name: '客户' }));
      fireEvent.click(screen.getByRole('checkbox', { name: '供应商' }));

      // 点击保存
      const submitBtn = screen.getByRole('button', { name: /保\s*存/ });
      fireEvent.click(submitBtn);

      await waitFor(() => {
        expect(partnerServiceUpdatePartnerRoles).toHaveBeenCalledWith(
          { partnerId: 'p-switch-1' },
          { partnerId: 'p-switch-1', roleTypes: [2] },
        );
        expect(onSuccess).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
      });
    },
  );
});
