import { cleanup, render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import OrderDetailPage from './detail';
import OrderFeesPage from './fees';

const mockPush = vi.fn();
let mockParams = { kind: 'sea-export', id: 'ord-1' };

vi.mock('@umijs/max', () => ({
  useParams: () => mockParams,
  useAccess: () => ({
    canOrder: () => true,
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
  orderServiceGetOrder: vi.fn().mockResolvedValue({ data: undefined }),
  orderServiceListPersonnelOptions: vi.fn().mockResolvedValue({ data: [] }),
  orderServiceCheckOrderReference: vi.fn(),
  orderServiceCreateOrder: vi.fn(),
  orderServiceUpdateOrder: vi.fn(),
}));

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceAddFee: vi.fn(),
  orderFeeServiceConfirmFee: vi.fn(),
  orderFeeServiceRemoveFee: vi.fn(),
  orderFeeServiceReopenFee: vi.fn(),
  orderFeeServiceUpdateFee: vi.fn(),
}));

vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceGetOrderLockState: vi.fn().mockResolvedValue({
    data: {
      orderId: 'ord-1',
      businessType: 1,
      isLocked: false,
      orderVersion: '1',
    },
  }),
}));

vi.mock('./use-order-lock-state', () => ({
  useOrderLockState: () => ({
    state: null,
    loading: false,
    error: null,
    refresh: vi.fn(),
  }),
  getOrderBusinessWritePolicy: () => ({
    disabled: false,
    reason: undefined,
  }),
}));

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

describe('订单模块面包屑与异常路由校验', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('详情页遇到未知 kind 时显示 404 错误状态，不静默回退为海运出口', () => {
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
  });

  it('费用页遇到未知 kind 时显示 404 错误状态，不静默回退为海运出口', () => {
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
  });
});
