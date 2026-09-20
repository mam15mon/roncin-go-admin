import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import { orderTagServiceListOrderTagOptions } from '@/services/roncin/orderTagService';
import { searchPartnerOptions } from '@/utils/options';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/utils/order-options-cache';
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

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOperateBusiness: true,
    canOrder: () => true,
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

vi.mock('@/utils/order-options-cache', () => ({
  getMasterDataOptions: vi.fn().mockResolvedValue([]),
  getCachedPorts: vi.fn().mockResolvedValue([]),
  getCachedAirports: vi.fn().mockResolvedValue([]),
}));

vi.mock('@/utils/options', () => ({
  searchPartnerOptions: vi.fn().mockResolvedValue([]),
  searchShippingLineOptions: vi.fn().mockResolvedValue([]),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListPersonnelOptions: vi.fn().mockResolvedValue({ data: [] }),
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

describe('订单列表页未知业务类型 fail-closed', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    locationState.pathname = '/orders/sea-import';
  });

  it('未知 kind 展示 404 且不发起任何主数据、标签或人员请求', () => {
    render(
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
