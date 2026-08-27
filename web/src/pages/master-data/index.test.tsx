import { cleanup, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MasterDataPage from './index';

// Mock umi access & hooks
vi.mock('@umijs/max', () => ({
  useAccess: () => ({
    canReadMasterDataPorts: true,
    canReadMasterDataAirports: true,
    canReadMasterDataAirlines: true,
    canReadMasterDataShippingLines: true,
    canReadMasterDataItems: true,
    canReadMasterDataAdministrativeRegions: true,
    canReadMasterDataCurrencies: true,
  }),
  history: {
    replace: vi.fn(),
  },
  useLocation: () => ({
    pathname: '/master-data',
    search: '',
  }),
}));

// Mock sub-panels to isolate test
vi.mock('./components/PortsPanel', () => ({
  default: () => <div data-testid="ports-panel">海运港口面板</div>,
}));
vi.mock('./components/AirportsPanel', () => ({
  default: () => <div data-testid="airports-panel">空运机场面板</div>,
}));
vi.mock('./components/AirlinesPanel', () => ({
  default: () => <div data-testid="airlines-panel">航空公司面板</div>,
}));
vi.mock('./components/ShippingLinesPanel', () => ({
  default: () => <div data-testid="shipping-lines-panel">船公司面板</div>,
}));
vi.mock('./components/CountriesPanel', () => ({
  default: () => <div data-testid="countries-panel">国家与地区面板</div>,
}));
vi.mock('./components/CitiesPanel', () => ({
  default: () => <div data-testid="cities-panel">城市与区划面板</div>,
}));
vi.mock('./components/CurrenciesPanel', () => ({
  default: () => <div data-testid="currencies-panel">货币与币种面板</div>,
}));

describe('MasterDataPage (主数据中心多Tab模板)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正确渲染主数据中心标题、副标题与7个业务Tab', () => {
    render(<MasterDataPage />);

    expect(screen.getByText('货代主数据管理中心')).toBeInTheDocument();
    expect(
      screen.getByText(
        '统一维护全球港口五字码、机场三字码、航司二字码、船司 SCAC、国家城市及货币币种基础资料',
      ),
    ).toBeInTheDocument();

    expect(screen.getByText('海运港口 (UN/LOCODE)')).toBeInTheDocument();
    expect(screen.getByText('空运机场 (IATA)')).toBeInTheDocument();
    expect(screen.getByText('航空公司 (Airlines)')).toBeInTheDocument();
    expect(screen.getByText('船公司 (Shipping Lines)')).toBeInTheDocument();
    expect(screen.getByText('国家与地区 (Countries)')).toBeInTheDocument();
    expect(screen.getByText('城市与区划 (Cities)')).toBeInTheDocument();
    expect(screen.getByText('货币与币种 (Currencies)')).toBeInTheDocument();

    // 默认展示 ports
    expect(screen.getByTestId('ports-panel')).toBeInTheDocument();
  });
});
