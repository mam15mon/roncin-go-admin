import type { ProFormInstance } from '@ant-design/pro-components';
import { act, renderHook, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderFeeServiceResolveFeeExchangeRate } from '@/services/roncin/orderFeeService';
import type { FeeFormValues } from './components/fees/FeeFormModal';
import { useFeeExchangePreview } from './use-fee-exchange-preview';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const resolveExchangeRate = vi.mocked(orderFeeServiceResolveFeeExchangeRate);

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

function wrapper({ children }: { children: React.ReactNode }) {
  return <App>{children}</App>;
}

describe('useFeeExchangePreview 请求隔离', () => {
  beforeEach(() => {
    resolveExchangeRate.mockReset();
  });

  it('切单重置后忽略上一订单迟到的汇率响应', async () => {
    const response = deferred<any>();
    resolveExchangeRate.mockReturnValue(response.promise);
    const formRef = {
      current: {
        getFieldsValue: () => ({
          direction: 1,
          currency: 'USD',
          expenseDate: '2026-09-05',
          quantity: '1',
          unitPrice: '100',
        }),
        setFieldValue: vi.fn(),
      } as unknown as ProFormInstance<FeeFormValues>,
    };
    const { result } = renderHook(
      () => useFeeExchangePreview('order-A', formRef),
      { wrapper },
    );

    act(() => result.current.handleValuesChange());
    await waitFor(() =>
      expect(result.current.exchangeRateStatus).toBe('loading'),
    );

    act(() => result.current.resetPreview());
    await act(async () => {
      response.resolve({ exchangeRate: '7.1234' });
    });

    expect(result.current.exchangeRateStatus).toBe('idle');
    expect(result.current.exchangeRatePreview).toBeUndefined();
  });
  it('确认数量只更新金额，保留手工汇率且不额外解析；普通变更仍解析汇率', async () => {
    resolveExchangeRate.mockResolvedValue({
      exchangeRate: '7.1',
      exchangeRateSource: 'SYSTEM',
    });
    const setFieldValue = vi.fn();
    const formRef = {
      current: {
        getFieldsValue: () => ({
          direction: 1,
          currency: 'USD',
          expenseDate: '2026-10-20',
          quantity: '8',
          unitPrice: '100',
          exchangeRateOverride: '7.2',
        }),
        setFieldValue,
      } as unknown as ProFormInstance<FeeFormValues>,
    };
    const { result } = renderHook(
      () => useFeeExchangePreview('order-A', formRef),
      { wrapper },
    );
    act(() => result.current.setManualExchangeRate(true));
    act(() => result.current.handleValuesChange({ amountOnly: true }));
    expect(result.current.totalPreview).toBe('800.00');
    expect(result.current.manualExchangeRate).toBe(true);
    expect(setFieldValue).not.toHaveBeenCalled();
    expect(resolveExchangeRate).not.toHaveBeenCalled();
    act(() => result.current.handleValuesChange());
    await waitFor(() =>
      expect(result.current.exchangeRateStatus).toBe('resolved'),
    );
    expect(result.current.manualExchangeRate).toBe(false);
    expect(setFieldValue).toHaveBeenCalledWith(
      'exchangeRateOverride',
      undefined,
    );
    expect(resolveExchangeRate).toHaveBeenCalledTimes(1);
  });

  it.each([
    [
      'ORDER_ETD_MISSING',
      '请先填写订单 ETD 后再解析汇率',
      { reason: 'ORDER_ETD_MISSING', message: '出口订单按 ETD 取汇率' },
    ],
    [
      'ORDER_ETA_MISSING',
      '请先填写订单 ETA 后再解析汇率',
      { reason: 'ORDER_ETA_MISSING', message: '进口订单按 ETA 取汇率' },
    ],
    [
      'FEE_EXCHANGE_RATE_MISSING',
      /应付费用 USD 在 2026-09-21 起的自然周未维护汇率/,
      {
        reason: 'FEE_EXCHANGE_RATE_MISSING',
        message: '订单取汇率日期所在周未维护该币种汇率，请先维护汇率',
        metadata: {
          currency: 'USD',
          week_start: '2026-09-21',
          direction: 'PAYABLE',
        },
      },
    ],
  ])('%s：状态为缺失并向用户展示对应提示', async (_name, expected, data) => {
    resolveExchangeRate.mockRejectedValue(
      Object.assign(new Error('Request failed'), {
        response: { status: 400, data },
      }),
    );
    const formRef = {
      current: {
        getFieldsValue: () => ({
          direction: 2,
          currency: 'USD',
          quantity: '1',
          unitPrice: '100',
        }),
        setFieldValue: vi.fn(),
      } as unknown as ProFormInstance<FeeFormValues>,
    };
    const { result } = renderHook(
      () => useFeeExchangePreview('order-A', formRef),
      { wrapper },
    );
    act(() => result.current.handleValuesChange());
    await waitFor(() =>
      expect(result.current.exchangeRateStatus).toBe('missing'),
    );
    expect(await screen.findByText(expected)).toBeInTheDocument();
  });
});
