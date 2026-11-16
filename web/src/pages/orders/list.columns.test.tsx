import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, within } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderServiceListOrders } from '@/services/roncin/orderService';
import OrderListPage from './list';

const locationState = vi.hoisted(() => ({ pathname: '/orders/sea-export' }));
vi.mock('react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router')>()),
  useLocation: () => ({ pathname: locationState.pathname }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOperateBusiness: true,
    canOrder: () => true,
  }),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '测试组织' },
      },
    },
  }),
}));
vi.mock('@/features/orders/options', () => ({
  getMasterDataOptions: vi.fn().mockResolvedValue([]),
  getCachedPorts: vi.fn().mockResolvedValue([]),
  getCachedAirports: vi.fn().mockResolvedValue([]),
}));
vi.mock('@/features/partners', () => ({
  searchPartnerOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('@/features/master-data/shipping-lines', () => ({
  searchShippingLineOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListPorts: vi
    .fn()
    .mockResolvedValue({ success: true, data: [], total: 0 }),
  masterDataServiceListAirports: vi
    .fn()
    .mockResolvedValue({ success: true, data: [], total: 0 }),
  masterDataServiceListItems: vi
    .fn()
    .mockResolvedValue({ success: true, data: [], total: 0 }),
  masterDataServiceSearchOptions: vi
    .fn()
    .mockResolvedValue({ success: true, data: [] }),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListPersonnelOptions: vi
    .fn()
    .mockResolvedValue({ success: true, data: [] }),
  orderServiceListOrders: vi.fn(),
}));
const mockListOrders = vi.mocked(orderServiceListOrders);

describe('订单品类列设置', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mockListOrders.mockResolvedValue({
      success: true,
      data: [{ id: 'column-order', orderNo: 'ORDER-COLUMNS' }],
      total: 1,
    });
  });

  it.each(['sea-export', 'air-export'])(
    '%s 使用独立列偏好，海运专属列仅在海运出现',
    async (kind) => {
      locationState.pathname = `/orders/${kind}`;
      localStorage.setItem(
        'roncin:column-settings:v1:orders:sea-export:anonymous:default',
        JSON.stringify({
          order: ['bookingNo', 'orderNo'],
          hidden: ['operatorName'],
          freeze: { left: ['bookingNo'], right: ['option'] },
        }),
      );
      const { container } = renderWithClient(
        <App>
          <OrderListPage />
        </App>,
      );
      await screen.findByText('ORDER-COLUMNS');
      const headers = within(
        container.querySelector('thead') as HTMLTableSectionElement,
      );
      expect(headers.queryByText('提成')).not.toBeInTheDocument();
      expect(headers.queryByText('目的地')).not.toBeInTheDocument();
      if (kind === 'sea-export') {
        expect(headers.getByText('订舱编号')).toBeInTheDocument();
        expect(headers.queryByText('操作人员')).not.toBeInTheDocument();
        expect(headers.getByText('ETD')).toBeInTheDocument();
        expect(headers.getByText('本票目的港')).toBeInTheDocument();
      } else {
        expect(headers.queryByText('订舱编号')).not.toBeInTheDocument();
        expect(headers.getByText('操作人员')).toBeInTheDocument();
        expect(headers.getByText('ETD')).toBeInTheDocument();
        expect(headers.getByText('主单 / 分单')).toBeInTheDocument();
        expect(headers.getByText('航空公司')).toBeInTheDocument();
        expect(headers.getByText('本票确认计费重')).toBeInTheDocument();
        expect(headers.getByText('委托总件数')).toBeInTheDocument();
        expect(headers.getByText('目的机场')).toBeInTheDocument();
      }
      fireEvent.click(screen.getByLabelText('列设置'));
      const siColumn = screen.queryByLabelText('截 SI 时间', {
        selector: 'input[type=checkbox]',
      });
      if (kind === 'sea-export') {
        expect(siColumn).not.toBeChecked();
        expect(
          screen.getByLabelText('共同段卸货港', {
            selector: 'input[type=checkbox]',
          }),
        ).not.toBeChecked();
      } else {
        expect(siColumn).not.toBeInTheDocument();
      }
      expect(
        screen.queryByLabelText('提成', { selector: 'input[type=checkbox]' }),
      ).not.toBeInTheDocument();
    },
  );
});
