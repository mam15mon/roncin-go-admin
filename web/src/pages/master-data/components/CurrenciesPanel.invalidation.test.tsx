import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { useQuery } from '@tanstack/react-query';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  currencyOptionsQueryOptions,
  getCurrencies,
} from '@/features/master-data/currencies';
import CurrenciesPanel from './CurrenciesPanel';

const fixtures = vi.hoisted(() => ({ enabled: false }));
const update = vi.hoisted(() => vi.fn());
const unrelated = vi.hoisted(() => vi.fn(async () => '其他主数据'));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    isSystemWorkspace: false,
    canUpdateMasterDataCurrencies: true,
  }),
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: async ({
    enabledOnly,
  }: {
    enabledOnly: boolean;
  }) => ({
    success: true,
    data:
      enabledOnly && !fixtures.enabled
        ? []
        : [{ id: 'usd', code: 'USD', name: '美元', enabled: fixtures.enabled }],
  }),
  masterDataServiceSetCurrencyEnabled: update,
}));
vi.mock('@/components/ui/master-data-template/MasterDataTemplate', () => ({
  MasterDataTemplate: ({
    items,
    onToggleActive,
  }: {
    items: Array<{ id: string; enabled: boolean }>;
    onToggleActive: (record: unknown) => Promise<void>;
  }) => (
    <button
      type="button"
      onClick={() => void onToggleActive(items[0])}
      disabled={!items.length}
    >
      启用美元
    </button>
  ),
}));

function Consumers() {
  const direct = useQuery(currencyOptionsQueryOptions());
  const aggregate = useQuery({
    queryKey: ['orders', 'currency-dependent-fixture'],
    queryFn: async () =>
      (await getCurrencies()).map((row) => row.code).join(','),
    meta: { currencyDependent: true },
  });
  useQuery({
    queryKey: ['master-data', 'unrelated-fixture'],
    queryFn: unrelated,
  });
  return (
    <>
      <output aria-label="直接候选">
        {direct.data?.map((item) => item.value).join(',') ?? '加载中'}
      </output>
      <output aria-label="订单聚合候选">{aggregate.data ?? '加载中'}</output>
    </>
  );
}

beforeEach(() => {
  fixtures.enabled = false;
  unrelated.mockClear();
  update.mockReset().mockImplementation(async () => {
    fixtures.enabled = true;
    return { success: true, data: { id: 'usd' } };
  });
});

describe('币种启停后的工作区查询失效', () => {
  it('启用后直接候选与显式依赖的订单聚合同时更新，其他主数据不重取', async () => {
    renderWithClient(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <CurrenciesPanel />
          <Consumers />
        </App>
      </ConfigProvider>,
    );
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '启用美元' })).toBeEnabled(),
    );
    await waitFor(() =>
      expect(screen.getByLabelText('直接候选')).toBeEmptyDOMElement(),
    );
    expect(screen.getByLabelText('订单聚合候选')).toBeEmptyDOMElement();
    expect(unrelated).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: '启用美元' }));
    await waitFor(() => {
      expect(screen.getByLabelText('直接候选')).toHaveTextContent('USD');
      expect(screen.getByLabelText('订单聚合候选')).toHaveTextContent('USD');
    });
    expect(update).toHaveBeenCalledExactlyOnceWith(
      { code: 'USD' },
      { code: 'USD', enabled: true },
      { skipErrorHandler: true },
    );
    expect(unrelated).toHaveBeenCalledOnce();
  });
});
