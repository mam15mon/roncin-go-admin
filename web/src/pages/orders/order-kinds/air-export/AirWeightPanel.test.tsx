import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  airWeightServiceConfirmAirWeight,
  airWeightServicePreviewAirWeight,
} from '@/services/roncin/airWeightService';
import AirWeightPanel from './AirWeightPanel';

vi.mock('@/services/roncin/airWeightService', () => ({
  airWeightServicePreviewAirWeight: vi.fn(),
  airWeightServiceConfirmAirWeight: vi.fn(),
  airWeightServiceListAirWeightConfirmations: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0 }),
}));
const preview = vi.mocked(airWeightServicePreviewAirWeight);
const confirm = vi.mocked(airWeightServiceConfirmAirWeight);

describe('空运重量参考与人工确认', () => {
  beforeEach(() => {
    preview.mockReset();
    confirm.mockReset();
    preview.mockResolvedValue({
      data: {
        scope: 'ORDER',
        targetId: 'order',
        source: 'ACTUAL',
        inputFingerprint: 'fp-new',
        available: true,
        grossWeightKg: '100',
        volumeCbm: '1.2',
        volumetricWeightKg: '200',
        bubbleDifferenceKg: '100',
        sharedBubbleWeightKg: '100',
        suggestedChargeableWeightKg: '200',
        currentConfirmation: {
          id: 'old',
          version: '2',
          confirmedWeightKg: '125',
        },
        confirmationStale: true,
      },
    });
  });

  it('输入变化保留旧确认值与过期提示，调整确认携带当前指纹和确认版本', async () => {
    confirm.mockResolvedValue({ success: true, data: { id: 'new' } });
    renderWithApp(<AirWeightPanel scope="ORDER" targetId="order" canUpdate />);
    expect(await screen.findByText('确认依据已变化')).toBeInTheDocument();
    expect(screen.getByText('125')).toBeInTheDocument();
    expect(screen.getAllByText('200').length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '确认或调整计费重' }));
    fireEvent.change(await screen.findByLabelText('最终确认重量（KG）'), {
      target: { value: '130' },
    });
    fireEvent.change(screen.getByLabelText('确认依据 / 调整原因'), {
      target: { value: '客户确认' },
    });
    fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
    await waitFor(() =>
      expect(confirm).toHaveBeenCalledWith(
        expect.objectContaining({
          inputFingerprint: 'fp-new',
          expectedVersion: '2',
          confirmedWeightKg: '130',
          reason: '客户确认',
          scope: 'ORDER',
          source: 'ACTUAL',
        }),
      ),
    );
  });

  it('重查失败隐藏旧参考，不能继续确认缓存结果', async () => {
    renderWithApp(<AirWeightPanel scope="ORDER" targetId="order" canUpdate />);
    await screen.findByRole('button', { name: '确认或调整计费重' });
    preview.mockRejectedValueOnce(new Error('读取失败'));
    fireEvent.click(screen.getByRole('button', { name: '重新核对' }));
    await screen.findByText('重量参考加载失败');
    expect(
      screen.queryByRole('button', { name: '确认或调整计费重' }),
    ).not.toBeInTheDocument();
    expect(confirm).not.toHaveBeenCalled();
  });
});
