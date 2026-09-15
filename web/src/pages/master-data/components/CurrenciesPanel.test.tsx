import { cleanup, render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import CurrenciesPanel from './CurrenciesPanel';

const mockListCurrencies = vi.fn();
const mockSetCurrencyEnabled = vi.fn();

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: (params: any) => mockListCurrencies(params),
  masterDataServiceSetCurrencyEnabled: (params: any, body: any) =>
    mockSetCurrencyEnabled(params, body),
}));

vi.mock('@umijs/max', () => ({
  useAccess: () => ({
    isHeadquartersOrganization: false,
    canUpdateMasterDataCurrencies: true,
  }),
}));

describe('CurrenciesPanel Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正确展示货币列表、本位币锁定标识与启用状态', async () => {
    mockListCurrencies.mockResolvedValue({
      data: [
        {
          id: '1',
          code: 'CNY',
          name: '人民币',
          symbol: '¥',
          minorUnit: 2,
          enabled: true,
          isBaseCurrency: true,
        },
        {
          id: '2',
          code: 'USD',
          name: '美元',
          symbol: '$',
          minorUnit: 2,
          enabled: true,
          isBaseCurrency: false,
        },
        {
          id: '3',
          code: 'AFN',
          name: '阿富汗尼',
          symbol: '؋',
          minorUnit: 2,
          enabled: false,
          isBaseCurrency: false,
        },
      ],
    });

    render(<CurrenciesPanel />);

    await waitFor(() => {
      expect(mockListCurrencies).toHaveBeenCalledWith({ enabledOnly: false });
    });

    expect(await screen.findByText('人民币')).toBeInTheDocument();
    expect(screen.getByText('美元')).toBeInTheDocument();
    expect(screen.getByText('阿富汗尼')).toBeInTheDocument();

    // 本位币锁定标签
    expect(screen.getByText('本位币 (锁定)')).toBeInTheDocument();
  });
});
