import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import OrderDetailPage from './detail';
import OrderFeesPage from './fees';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { orderFeeServiceListFeeOptions } from '@/services/roncin/orderFeeService';

const mockPush = vi.fn();
let mockParams = { kind: 'sea-export', id: 'ord-1' };

vi.mock('@umijs/max', () => ({
  useParams: () => mockParams,
  useAccess: () => ({
    canOrder: () => true,
    canCreateFee: () => true,
    canEditFee: () => true,
    canDeleteFee: () => true,
    canConfirmFee: () => true,
  }),
  useModel: () => ({
    initialState: {
      currentUser: { id: 'user-1' },
    },
  }),
  history: {
    push: (path: string) => mockPush(path),
  },
  Link: ({ to, children, ...rest }: any) => (
    <a href={to} {...rest}>
      {children}
    </a>
  ),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: vi.fn(),
  orderServiceListPersonnelOptions: vi.fn().mockResolvedValue({ data: [] }),
  orderServiceCheckOrderReference: vi.fn(),
  orderServiceCreateOrder: vi.fn(),
  orderServiceUpdateOrder: vi.fn(),
}));

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFeeOptions: vi.fn(),
  orderFeeServiceAddFee: vi.fn(),
  orderFeeServiceConfirmFee: vi.fn(),
  orderFeeServiceRemoveFee: vi.fn(),
  orderFeeServiceReopenFee: vi.fn(),
  orderFeeServiceUpdateFee: vi.fn(),
  orderFeeServiceListFees: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderShippingDocumentService', () => ({
  orderShippingDocumentServiceListShippingDocuments: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderContainerService', () => ({
  orderContainerServiceListContainers: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderCargoItemService', () => ({
  orderCargoItemServiceListCargoItems: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderMilestoneService', () => ({
  orderMilestoneServiceListMilestones: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderPersonnelService', () => ({
  orderPersonnelServiceListPersonnel: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('./common', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./common')>();
  return {
    ...actual,
    fetchOrderMasterData: vi.fn().mockResolvedValue({
      serviceTypeOptions: actual.seaServiceTypes.map(({ code, name }) => ({
        code,
        label: name,
        value: code,
      })),
      cargoCategoryOptions: [],
      seaLocationOptions: [],
      airLocationOptions: [],
      currencyOptions: [],
      masterOptions: [],
    }),
  };
});

const mockUseOrderLockState = vi.fn((_orderId?: string) => ({
  state: null,
  loading: false,
  error: null,
  refresh: vi.fn(),
}));

vi.mock('./use-order-lock-state', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./use-order-lock-state')>();
  return {
    ...actual,
    useOrderLockState: (orderId?: string) => mockUseOrderLockState(orderId),
  };
});

vi.mock('./use-fee-exchange-preview', () => ({
  useFeeExchangePreview: () => ({
    totalPreview: null,
    exchangeRatePreview: null,
    exchangeRateStatus: 'ok',
    manualExchangeRate: false,
    setManualExchangeRate: vi.fn(),
    resetPreview: vi.fn(),
    seedFromFee: vi.fn(),
    handleValuesChange: vi.fn(),
  }),
}));

const mockGetOrder = vi.mocked(orderServiceGetOrder);
const mockListFeeOptions = vi.mocked(orderFeeServiceListFeeOptions);

describe('订单模块面包屑与异常路由校验', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetOrder.mockResolvedValue({ data: undefined } as any);
    mockListFeeOptions.mockResolvedValue({
      currencies: [],
      settlementParties: [],
      feeSettings: [],
      billingUnits: [],
      financeLocked: false,
    } as any);
  });

  afterEach(() => {
    cleanup();
  });

  it('详情页遇到未知 kind 时显示 404 且不调用订单与锁状态接口', () => {
    mockParams = { kind: 'unknown-freight', id: 'ord-1' };

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(screen.getByText('业务类型不存在')).toBeInTheDocument();
    expect(
      screen.getByText('未知的业务类型路径 "unknown-freight"，请选择有效业务入口。'),
    ).toBeInTheDocument();
    expect(screen.getByText('返回海运出口订单')).toBeInTheDocument();

    // 严禁发送数据请求
    expect(mockGetOrder).not.toHaveBeenCalled();
    expect(mockUseOrderLockState).toHaveBeenCalledWith(undefined);
  });

  it('费用页遇到未知 kind 时显示 404 且不调用订单与费用接口', () => {
    mockParams = { kind: 'invalid-air', id: 'ord-1' };

    render(
      <App>
        <OrderFeesPage />
      </App>,
    );

    expect(screen.getByText('业务类型不存在')).toBeInTheDocument();
    expect(
      screen.getByText('未知的业务类型路径 "invalid-air"，请选择有效业务入口。'),
    ).toBeInTheDocument();
    expect(screen.getByText('返回海运出口订单')).toBeInTheDocument();

    // 严禁发送数据请求
    expect(mockGetOrder).not.toHaveBeenCalled();
    expect(mockListFeeOptions).not.toHaveBeenCalled();
    expect(mockUseOrderLockState).toHaveBeenCalledWith(undefined);
  });

  it('详情页在加载中与订单不存在状态下均正确渲染公共页头且无多余重复返回按钮', async () => {
    let resolveOrder!: (value: any) => void;
    mockGetOrder.mockImplementation(
      () =>
        new Promise((res) => {
          resolveOrder = res;
        }),
    );

    mockParams = { kind: 'sea-export', id: 'ord-loading' };

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    // 1. 加载中状态
    expect(screen.getByText('正在加载订单详情...')).toBeInTheDocument();
    expect(screen.getByText('订单管理')).toBeInTheDocument();
    expect(screen.getByText('海运出口')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /返回列表/ })).toBeInTheDocument();

    // 2. 订单不存在（空结果）
    resolveOrder({ data: undefined });
    await waitFor(() => {
      expect(screen.getByText('未找到对应的订单档案')).toBeInTheDocument();
    });

    // 验证：OrderPageHeader 的“返回列表”存在，但卡片内部无重复的“返回订单列表”按钮
    expect(screen.getByRole('button', { name: /返回列表/ })).toBeInTheDocument();
    expect(screen.queryByText('返回订单列表')).not.toBeInTheDocument();
  });

  it('费用页在加载中与订单不存在状态下均正确渲染公共页头且无多余重复返回按钮', async () => {
    let resolveOrder!: (value: any) => void;
    mockGetOrder.mockImplementation(
      () =>
        new Promise((res) => {
          resolveOrder = res;
        }),
    );

    mockParams = { kind: 'sea-export', id: 'ord-fee-loading' };

    render(
      <App>
        <OrderFeesPage />
      </App>,
    );

    // 1. 加载中状态
    expect(screen.getByText('正在加载费用工作台...')).toBeInTheDocument();
    expect(screen.getByText('订单管理')).toBeInTheDocument();
    expect(screen.getByText('海运出口')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /返回订单详情/ })).toBeInTheDocument();

    // 2. 订单不存在（空结果）
    resolveOrder({ data: undefined });
    await waitFor(() => {
      expect(screen.getByText('未找到对应的订单档案')).toBeInTheDocument();
    });

    // 验证：保留页头返回按钮，卡片内部无重复返回按钮
    expect(screen.getByRole('button', { name: /返回订单详情/ })).toBeInTheDocument();
    expect(screen.queryByText('返回订单列表')).not.toBeInTheDocument();
  });

  it('正常详情页正确接入公共页头面包屑层级', async () => {
    mockParams = { kind: 'sea-export', id: 'ord-detail-1' };
    mockGetOrder.mockResolvedValue({
      data: {
        id: 'ord-detail-1',
        orderNo: 'SE20260905001',
        version: '1',
      },
    } as any);

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getAllByText('SE20260905001').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('订单管理')).toBeInTheDocument();
    expect(screen.getByText('海运出口')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /返回列表/ })).toBeInTheDocument();
  });
});
