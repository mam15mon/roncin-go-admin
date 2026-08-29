import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useOrderFeePanelExchangeRate } from './use-order-fee-panel-rate';

const messageError = vi.hoisted(() => vi.fn());
const resolveExchangeRate = vi.hoisted(() => vi.fn());

vi.mock('antd', () => ({
  App: { useApp: () => ({ message: { error: messageError } }) },
}));

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceResolveFeeExchangeRate: resolveExchangeRate,
}));

describe('useOrderFeePanelExchangeRate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('汇率未配置时允许手工录入', async () => {
    resolveExchangeRate.mockRejectedValueOnce({
      message: '汇率未配置',
      data: { reason: 'FEE_EXCHANGE_RATE_MISSING' },
    });
    const { result } = renderHook(() => useOrderFeePanelExchangeRate());

    act(() =>
      result.current.resolveExchangeRate('order-1', 1, 'USD', '2026-08-29'),
    );

    await waitFor(() =>
      expect(result.current.exchangeRateStatus).toBe('missing'),
    );
    expect(result.current.manualExchangeRate).toBe(true);
    expect(messageError).not.toHaveBeenCalled();
  });

  it('服务异常时不自动回退到手工汇率', async () => {
    resolveExchangeRate.mockRejectedValueOnce(new Error('服务不可用'));
    const { result } = renderHook(() => useOrderFeePanelExchangeRate());

    act(() =>
      result.current.resolveExchangeRate('order-1', 1, 'USD', '2026-08-29'),
    );

    await waitFor(() =>
      expect(result.current.exchangeRateStatus).toBe('error'),
    );
    expect(result.current.manualExchangeRate).toBe(false);
    expect(messageError).toHaveBeenCalledWith('服务不可用');
  });
});
