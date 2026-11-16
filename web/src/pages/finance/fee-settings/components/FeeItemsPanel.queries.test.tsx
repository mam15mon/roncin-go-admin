import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { CURRENCY_QUERY_PREFIX } from '@/features/master-data/currencies';
import { FeeItemsPanel } from './FeeItemsPanel';

const api = vi.hoisted(() => ({
  currencies: vi.fn(),
  units: vi.fn(),
  taxable: vi.fn(),
  items: vi.fn(),
  fees: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canCreateFeeSettings: true, canUpdateFeeSettings: true }),
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: api.currencies,
  masterDataServiceListOptions: api.items,
}));
vi.mock('@/services/roncin/feeCatalogService', () => ({
  feeCatalogServiceListBillingUnits: api.units,
  feeCatalogServiceListTaxableServices: api.taxable,
  feeCatalogServiceListFeeSettings: api.fees,
  feeCatalogServiceCreateFeeSetting: vi.fn(),
  feeCatalogServiceUpdateFeeSetting: vi.fn(),
}));
vi.mock('@/components/ui', () => ({
  SettingTableTemplate: ({
    query,
    renderFormItems,
  }: {
    query: (params: { current: number }) => Promise<unknown>;
    renderFormItems: () => ReactNode;
  }) => (
    <>
      <button type="button" onClick={() => void query({ current: 2 })}>
        刷新费用列表
      </button>
      {renderFormItems()}
    </>
  ),
}));
vi.mock('./FeeSettingFields', () => ({
  default: ({
    currencies,
    billingUnits,
  }: {
    currencies: API.Currency[];
    billingUnits: API.BillingUnit[];
  }) => (
    <>
      <output aria-label="费用币种候选">
        {currencies.map((row) => row.code).join(',')}
      </output>
      <output aria-label="计费单位候选">
        {billingUnits.map((row) => row.name).join(',')}
      </output>
    </>
  ),
}));

beforeEach(() => {
  vi.clearAllMocks();
  api.currencies.mockResolvedValue({
    success: true,
    data: [{ code: 'USD', enabled: true }],
  });
  api.units.mockResolvedValue({
    success: true,
    data: [{ id: 'unit', name: '票', enabled: true }],
  });
  api.taxable.mockResolvedValue({ success: true });
  api.items.mockResolvedValue({ success: true });
  api.fees.mockResolvedValue({ success: true });
});

describe('费用设置列表与候选查询分离', () => {
  it('表格刷新不重复加载候选，币种失效独立更新候选', async () => {
    const { queryClient } = renderWithClient(<FeeItemsPanel />);
    await waitFor(() =>
      expect(screen.getByLabelText('费用币种候选')).toHaveTextContent('USD'),
    );
    expect(screen.getByLabelText('计费单位候选')).toHaveTextContent('票');
    fireEvent.click(screen.getByRole('button', { name: '刷新费用列表' }));
    await waitFor(() =>
      expect(api.fees).toHaveBeenCalledWith({ page: 2, pageSize: 20 }),
    );
    expect(api.currencies).toHaveBeenCalledOnce();
    expect(api.units).toHaveBeenCalledOnce();
    expect(api.taxable).toHaveBeenCalledOnce();
    expect(api.items).toHaveBeenCalledOnce();
    api.currencies.mockResolvedValue({
      success: true,
      data: [{ code: 'EUR', enabled: true }],
    });
    await queryClient.invalidateQueries({ queryKey: CURRENCY_QUERY_PREFIX });
    await waitFor(() =>
      expect(screen.getByLabelText('费用币种候选')).toHaveTextContent('EUR'),
    );
    expect(api.fees).toHaveBeenCalledOnce();
    expect(api.units).toHaveBeenCalledOnce();
  });

  it('候选失败保留明确错误状态，不伪装为空成功', async () => {
    api.units.mockResolvedValueOnce(undefined);
    const { queryClient } = renderWithClient(<FeeItemsPanel />);
    await waitFor(() =>
      expect(
        queryClient.getQueryState([
          'finance',
          'fee-settings-reference-options',
          {},
        ])?.error?.message,
      ).toBe('计费单位加载失败'),
    );
    expect(screen.getByLabelText('费用币种候选')).toHaveTextContent('USD');
  });
});
