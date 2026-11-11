import { renderWithApp } from '@root/tests/renderWithApp';
import { screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { partnerServiceListPartnerShippingPresets } from '@/services/roncin/partnerService';
import ShippingPresetSection from './ShippingPresetSection';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartnerShippingPresets: vi.fn(),
  partnerServiceCreatePartnerShippingPreset: vi.fn(),
  partnerServiceUpdatePartnerShippingPreset: vi.fn(),
}));
beforeEach(() => vi.clearAllMocks());

it.each([undefined, false, true])(
  '预设 enabled=%s 只展示实际启用项',
  async (enabled) => {
    vi.mocked(partnerServiceListPartnerShippingPresets).mockResolvedValue({
      data: [
        {
          id: 'preset-wire',
          title: '状态测试预设',
          presetType: 1,
          party: { companyName: '测试收发主体' },
          ...(enabled === undefined ? {} : { enabled }),
        },
      ],
    });
    renderWithApp(
      <ShippingPresetSection
        partnerId="partner"
        canCreate={false}
        canUpdate={false}
      />,
    );
    await waitFor(() =>
      expect(partnerServiceListPartnerShippingPresets).toHaveBeenCalledOnce(),
    );
    if (enabled === true) {
      expect(await screen.findByText('状态测试预设')).toBeInTheDocument();
    } else {
      expect(
        await screen.findByText('暂无常用单证预设，点击右上角按钮快捷录入'),
      ).toBeInTheDocument();
      expect(screen.queryByText('状态测试预设')).not.toBeInTheDocument();
    }
  },
);
