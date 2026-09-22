import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/features/orders/options';
import { searchPartnerOptions } from '@/features/partners';
import {
  orderServiceListOrders,
  orderServiceListPersonnelOptions,
} from '@/services/roncin/orderService';
import { orderTagServiceListOrderTagOptions } from '@/services/roncin/orderTagService';
import OrderListPage from './list';

const locationState = vi.hoisted(() => ({
  pathname: '/orders/sea-import',
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useLocation: () => ({ pathname: locationState.pathname }),
  };
});

const accessControl = vi.hoisted(() => ({ canCreate: true }));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOperateBusiness: true,
    canOrder: (_businessType: number | string, operation: string) =>
      operation === 'create' ? accessControl.canCreate : true,
    canCreateEnterpriseResources: true,
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

vi.mock('@/router/history', () => ({
  history: { push: vi.fn() },
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

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListPersonnelOptions: vi.fn().mockResolvedValue({ data: [] }),
  orderServiceListOrders: vi.fn().mockResolvedValue({ data: [], total: 0 }),
}));

vi.mock('@/services/roncin/orderTagService', () => ({
  orderTagServiceListOrderTagOptions: vi.fn().mockResolvedValue({ tags: [] }),
}));

const mockGetMasterDataOptions = vi.mocked(getMasterDataOptions);
const mockGetPorts = vi.mocked(getCachedPorts);
const mockGetAirports = vi.mocked(getCachedAirports);
const mockSearchPartners = vi.mocked(searchPartnerOptions);
const mockListTagOptions = vi.mocked(orderTagServiceListOrderTagOptions);
const mockListPersonnelOptions = vi.mocked(orderServiceListPersonnelOptions);
const mockListOrders = vi.mocked(orderServiceListOrders);

describe('订单列表页未知业务类型 fail-closed', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    locationState.pathname = '/orders/sea-import';
  });

  it('未知 kind 展示 404 且不发起任何主数据、标签或人员请求', () => {
    renderWithClient(
      <App>
        <OrderListPage />
      </App>,
    );

    expect(screen.getByText('未知的业务类型')).toBeInTheDocument();
    expect(mockGetMasterDataOptions).not.toHaveBeenCalled();
    expect(mockGetPorts).not.toHaveBeenCalled();
    expect(mockGetAirports).not.toHaveBeenCalled();
    expect(mockSearchPartners).not.toHaveBeenCalled();
    expect(mockListTagOptions).not.toHaveBeenCalled();
    expect(mockListPersonnelOptions).not.toHaveBeenCalled();
  });
});

describe('订单列表新建入口权限收口', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    locationState.pathname = '/orders/sea-export';
    accessControl.canCreate = true;
    mockListOrders.mockResolvedValue({ data: [], total: 0 });
  });

  it('有 create 权限时渲染新增订单按钮', () => {
    renderWithClient(
      <App>
        <OrderListPage />
      </App>,
    );

    expect(screen.getByText('新增海运出口订单')).toBeInTheDocument();
  });

  it('无 create 权限时不渲染新增订单按钮（如系统管理只读角色）', () => {
    accessControl.canCreate = false;
    renderWithClient(
      <App>
        <OrderListPage />
      </App>,
    );

    expect(screen.queryByText('新增海运出口订单')).not.toBeInTheDocument();
  });
});
