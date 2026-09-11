import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import { partnerServiceCreatePartner } from '@/services/roncin/partnerService';
import QuickAddPartnerModal from './QuickAddPartnerModal';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
}));

describe('QuickAddPartnerModal', () => {
  beforeEach(() => {
    vi.mocked(partnerServiceCreatePartner).mockReset();
  });

  it('以启用角色创建伙伴并回填，不强制统一社会信用代码', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: {
        id: 'partner-1',
        legalName: '费用测试单位',
        code: 'P00000001',
      } as never,
    });
    const onSuccess = vi.fn();

    render(
      <App>
        <QuickAddPartnerModal open onCancel={vi.fn()} onSuccess={onSuccess} />
      </App>,
    );

    fireEvent.change(screen.getByLabelText('单位全称'), {
      target: { value: '  费用测试单位  ' },
    });
    fireEvent.mouseDown(screen.getByLabelText('客商类型'));
    fireEvent.click(await screen.findByText('客户 (委托单位/收发通)'));
    fireEvent.click(screen.getByRole('button', { name: '保存并选用' }));

    await waitFor(() =>
      expect(partnerServiceCreatePartner).toHaveBeenCalledWith({
        legalName: '费用测试单位',
        roles: [
          {
            type: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
            enabled: true,
          },
        ],
      }),
    );
    expect(onSuccess).toHaveBeenCalledWith({
      id: 'partner-1',
      name: '费用测试单位',
      code: 'P00000001',
    });
  });
});
