import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
} from '@/features/orders/options';
import { searchPartnerOptions } from '@/features/partners';
import { history } from '@/router/history';
import {
  orderServiceListOrders,
  orderServiceListPersonnelOptions,
} from '@/services/roncin/orderService';
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

const accessControl = vi.hoisted(() => ({
  canCreate: true,
  canUpdate: true,
  canLock: true,
  canReadFees: true,
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOperateBusiness: true,
    canOrder: (_businessType: number | string, operation: string) => {
      if (operation === 'create') return accessControl.canCreate;
      if (operation === 'update') return accessControl.canUpdate;
      if (operation === 'lock') return accessControl.canLock;
      if (operation === 'fee.read') return accessControl.canReadFees;
      return true;
    },
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
  history: {
    get location() {
      return window.location;
    },
    push: vi.fn(),
  },
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
  masterDataServiceListPorts: vi.fn().mockResolvedValue({ data: [], total: 0 }),
  masterDataServiceListAirports: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0 }),
  masterDataServiceListItems: vi.fn().mockResolvedValue({ data: [], total: 0 }),
  masterDataServiceSearchOptions: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListPersonnelOptions: vi.fn().mockResolvedValue({ data: [] }),
  orderServiceListOrders: vi.fn().mockResolvedValue({ data: [], total: 0 }),
}));

vi.mock('./components/fees/fee-review/OrderFeeReview', () => ({
  OrderFeeReviewDrawer: ({ orderId }: { orderId?: string }) =>
    orderId ? <div>锁定订单：{orderId}</div> : null,
}));

const mockGetMasterDataOptions = vi.mocked(getMasterDataOptions);
const mockGetPorts = vi.mocked(getCachedPorts);
const mockGetAirports = vi.mocked(getCachedAirports);
const mockSearchPartners = vi.mocked(searchPartnerOptions);
const mockListPersonnelOptions = vi.mocked(orderServiceListPersonnelOptions);
const mockListOrders = vi.mocked(orderServiceListOrders);

describe('订单列表页未知业务类型 fail-closed', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    locationState.pathname = '/orders/sea-import';
  });

  it('未知 kind 展示 404 且不发起任何主数据或人员请求', () => {
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
    expect(mockListPersonnelOptions).not.toHaveBeenCalled();
  });
});

describe('订单列表新建入口权限收口', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    locationState.pathname = '/orders/sea-export';
    accessControl.canCreate = true;
    accessControl.canUpdate = true;
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

  it('无 update 权限时不渲染批量维护标签按钮', () => {
    accessControl.canUpdate = false;
    renderWithClient(
      <App>
        <OrderListPage />
      </App>,
    );

    expect(screen.queryByText('批量维护标签')).not.toBeInTheDocument();
  });
});

describe('订单列表费用入口', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    accessControl.canCreate = true;
    accessControl.canUpdate = true;
  });

  it.each(['sea-export', 'air-export'])(
    '%s 费用核算进入对应订单费用页面',
    async (kind) => {
      locationState.pathname = `/orders/${kind}`;
      mockListOrders.mockResolvedValue({
        data: [{ id: 'order-fees-1', orderNo: 'ORDER-FEES-1' }],
        total: 1,
      });
      renderWithClient(
        <App>
          <OrderListPage />
        </App>,
      );

      await screen.findByText('ORDER-FEES-1');
      fireEvent.mouseEnter(screen.getByRole('button', { name: /更多/ }));
      fireEvent.click(await screen.findByText('费用核算'));

      expect(history.push).toHaveBeenCalledWith(
        `/orders/${kind}/order-fees-1/fees`,
      );
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    },
  );
});

describe('订单列表审核入口', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    locationState.pathname = '/orders/sea-export';
    accessControl.canLock = true;
    accessControl.canReadFees = true;
    mockListOrders.mockResolvedValue({
      data: [
        {
          id: 'review-order',
          orderNo: 'SE-REVIEW',
          businessType: 1,
          feeReview: { hasPendingCorrection: true },
        },
        {
          id: 'locked-order',
          orderNo: 'SE-LOCKED',
          businessType: 1,
          lockedAt: '2026-11-13T02:00:00Z',
        },
      ],
      total: 2,
    });
  });

  it('锁定状态区分已锁定和未锁定，保留补审提示与审核入口', async () => {
    renderWithClient(
      <App>
        <OrderListPage />
      </App>,
    );
    const orderNumber = await screen.findByText('SE-REVIEW');
    expect(
      screen.getByRole('columnheader', { name: '锁定状态' }),
    ).toBeInTheDocument();
    const row = within(orderNumber.closest('tr') as HTMLTableRowElement);
    expect(row.getByText('有补录待补审')).toBeInTheDocument();
    const lockedRow = within(
      screen.getByText('SE-LOCKED').closest('tr') as HTMLTableRowElement,
    );
    expect(lockedRow.getByText('已锁定')).toBeInTheDocument();
    expect(screen.queryByText('未审核')).not.toBeInTheDocument();
    expect(screen.queryByText('已审核')).not.toBeInTheDocument();
    expect(row.queryByText('详情')).not.toBeInTheDocument();
    fireEvent.click(orderNumber);
    expect(history.push).toHaveBeenCalledWith(
      '/orders/sea-export/review-order',
    );
    expect(row.queryByText('审核')).not.toBeInTheDocument();
    const review = row.getByText('未锁定').closest('button');
    await waitFor(() => expect(review).toBeVisible());
    expect(review).toBeEnabled();
    fireEvent.click(review as HTMLButtonElement);
    expect(await screen.findByText('锁定订单：review-order')).toBeVisible();
  });

  it('无审核及费用读取权限时，审核状态入口不可打开审核', async () => {
    accessControl.canLock = false;
    accessControl.canReadFees = false;
    renderWithClient(
      <App>
        <OrderListPage />
      </App>,
    );
    const orderNumber = await screen.findByText('SE-REVIEW');
    const row = within(orderNumber.closest('tr') as HTMLTableRowElement);
    expect(row.queryByText('审核')).not.toBeInTheDocument();
    const button = row.getByText('未锁定').closest('button');
    expect(button).toBeDisabled();
    fireEvent.click(button as HTMLButtonElement);
    expect(
      screen.queryByText('锁定订单：review-order'),
    ).not.toBeInTheDocument();
  });
});

describe('订单品类列设置', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mockListOrders.mockResolvedValue({
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
        expect(headers.queryByText('ETD')).not.toBeInTheDocument();
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
