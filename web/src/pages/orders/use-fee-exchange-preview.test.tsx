import type { ProFormInstance } from '@ant-design/pro-components';
import { act, renderHook, waitFor } from '@testing-library/react';
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
});
