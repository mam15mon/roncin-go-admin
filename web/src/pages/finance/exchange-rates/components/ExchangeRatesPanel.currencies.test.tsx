import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import type { ReactNode } from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import { CURRENCY_QUERY_PREFIX } from '@/features/master-data/currencies';
import { ExchangeRatesPanel } from './ExchangeRatesPanel';

const api = vi.hoisted(() => ({ currencies: vi.fn(), rates: vi.fn() }));
vi.mock('@/app/access', () => ({ useAccess: () => ({}) }));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: api.currencies,
}));
vi.mock('@/services/roncin/exchangeRateService', () => ({
  exchangeRateServiceListExchangeRateSettings: api.rates,
  exchangeRateServiceCreateExchangeRateSetting: vi.fn(),
  exchangeRateServiceDisableExchangeRateSetting: vi.fn(),
  exchangeRateServiceUpdateExchangeRateSetting: vi.fn(),
}));
vi.mock('@/components/ui/column-settings', () => ({
  useColumnSettings: ({ columns }: { columns: unknown[] }) => ({
    columns,
    entry: null,
  }),
}));
vi.mock('./ExchangeRateImportModal', () => ({
  ExchangeRateImportModal: () => null,
}));
vi.mock('./ExchangeRateSyncModal', () => ({
  ExchangeRateSyncModal: () => null,
}));
vi.mock('@/components/ui', () => ({
  GuardedModalForm: () => null,
  ProFormSearchableSelect: () => null,
  AsyncPopconfirm: () => null,
  ResponsiveProTable: ({
    columns,
    request,
  }: {
    columns: Array<{
      dataIndex?: string;
      render?: (
        value: unknown,
        row: { fromCurrency: string; toCurrency: string },
      ) => ReactNode;
    }>;
    request: (params: {
      current: number;
      pageSize: number;
    }) => Promise<unknown>;
  }) => (
    <>
      <output aria-label="汇率币种名称">
        {columns
          .find((column) => column.dataIndex === 'fromCurrency')
          ?.render?.(undefined, { fromCurrency: 'USD', toCurrency: 'CNY' })}
      </output>
      <button
        type="button"
        onClick={() => void request({ current: 2, pageSize: 20 })}
      >
        刷新汇率表
      </button>
    </>
  ),
}));

beforeEach(() => {
  vi.clearAllMocks();
  api.currencies.mockResolvedValue({
    success: true,
    data: [
      { code: 'USD', name: '美元', enabled: true },
      { code: 'CNY', name: '人民币', enabled: true },
    ],
  });
  api.rates.mockResolvedValue({ success: true, baseCurrency: 'CNY' });
});

it('币种缓存失效更新汇率名称，表格刷新不再借用本地货币快照', async () => {
  const { queryClient } = renderWithClient(
    <App>
      <ExchangeRatesPanel />
    </App>,
  );
  await waitFor(() =>
    expect(screen.getByLabelText('汇率币种名称')).toHaveTextContent(
      'USD (美元) → CNY (人民币)',
    ),
  );
  fireEvent.click(screen.getByRole('button', { name: '刷新汇率表' }));
  await waitFor(() =>
    expect(api.rates).toHaveBeenCalledWith({ page: 2, pageSize: 20 }),
  );
  expect(api.currencies).toHaveBeenCalledOnce();
  api.currencies.mockResolvedValue({
    success: true,
    data: [
      { code: 'USD', name: '美元更新', enabled: true },
      { code: 'CNY', name: '人民币', enabled: true },
    ],
  });
  await act(async () => {
    await queryClient.invalidateQueries({ queryKey: CURRENCY_QUERY_PREFIX });
  });
  await waitFor(() =>
    expect(screen.getByLabelText('汇率币种名称')).toHaveTextContent(
      'USD (美元更新) → CNY (人民币)',
    ),
  );
  expect(api.rates).toHaveBeenCalledOnce();
});
