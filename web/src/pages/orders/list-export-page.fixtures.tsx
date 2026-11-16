import { cleanup } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { afterEach, beforeEach, vi } from 'vitest';
import OrderListPage from './list';

export { state };

const state = vi.hoisted(() => ({
  company: 'company-a' as string | undefined,
  canRead: true,
  canReadContainers: false,
  canReadAttachments: false,
  list: vi.fn(),
  export: vi.fn(),
  download: vi.fn(),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user',
        currentOrganization: { id: state.company, kind: 2 },
      },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: (_type: unknown, operation: string) =>
      operation === 'read'
        ? state.canRead
        : operation === 'container.read'
          ? state.canReadContainers
          : operation === 'attachment.read'
            ? state.canReadAttachments
            : false,
    canOperateOrganization: () => true,
  }),
}));
vi.mock('react-router', async (original) => ({
  ...(await original<typeof import('react-router')>()),
  useLocation: () => ({ pathname: '/orders/air-import' }),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListOrders: state.list,
  orderServiceExportOrders: state.export,
}));
vi.mock('./list-resources', () => ({
  useOrderListResources: () => ({
    ports: [],
    airports: [],
    customerMap: {},
    containerSpecOptions: [],
    containerSpecMap: {},
    searchCustomers: vi.fn().mockResolvedValue([]),
    searchLocations: vi.fn().mockResolvedValue([]),
    searchOrderPorts: vi.fn().mockResolvedValue([]),
    searchOrderCarriers: vi.fn().mockResolvedValue([]),
    searchOrderPersonnel: vi.fn().mockResolvedValue([]),
  }),
}));
vi.mock('@/utils/csv', async (original) => ({
  ...(await original<typeof import('@/utils/csv')>()),
  downloadCsv: state.download,
}));

export const response = (company = 'company-a') => ({
  data: [
    {
      id: 'order',
      orderNo: 'AI-CURRENT',
      organizationId: company,
      businessType: 4,
    },
  ],
  total: 1,
  success: true,
  generatedAt: '2026-11-16T12:30:00Z',
});
export const Page = () => (
  <App>
    <ConfigProvider theme={{ token: { motion: false } }}>
      <OrderListPage />
    </ConfigProvider>
  </App>
);
beforeEach(() => {
  vi.clearAllMocks();
  state.company = 'company-a';
  state.canRead = true;
  state.canReadContainers = false;
  state.canReadAttachments = false;
  state.list.mockResolvedValue(response());
});
afterEach(cleanup);
