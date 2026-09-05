import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import OrderDetailPage from './detail';

const routeState = vi.hoisted(() => ({
  params: { kind: 'sea-export', id: 'ord-A' },
}));

const detailTestState = vi.hoisted(() => ({
  loadData: vi.fn<(orderId?: string) => Promise<void>>(),
}));

vi.mock('@umijs/max', () => ({
  useParams: () => routeState.params,
  useAccess: () => ({ canOrder: () => true }),
  history: { push: vi.fn() },
}));

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceGetSeaOrderChangeActions: vi.fn(),
}));

vi.mock('./use-order-detail-data', () => ({
  useOrderDetailData: (orderId?: string) => ({
    loading: false,
    order: orderId
      ? { id: orderId, orderNo: `ORDER-${orderId}`, version: '1' }
      : undefined,
    shippingDocs: [],
    personnel: [],
    serviceTypeOptions: [],
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: vi.fn().mockResolvedValue([]),
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: [],
    loadData: () => detailTestState.loadData(orderId),
  }),
}));

vi.mock('./use-order-lock-state', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('./use-order-lock-state')>();
  return {
    ...actual,
    useOrderLockState: () => ({
      state: null,
      loading: false,
      error: null,
      refresh: vi.fn().mockResolvedValue(null),
    }),
  };
});

vi.mock('@/components/ui/order-template/OrderFormTemplate', () => ({
  OrderFormTemplate: ({ header }: { header: React.ReactNode }) => header,
}));

vi.mock('./components/detail/OrderDetailHeader', () => ({
  default: (props: {
    splitDisabled?: boolean;
    splitBlockedReasons?: string[];
    reassignDisabled?: boolean;
    reassignBlockedReasons?: string[];
    moreMenuItems?: Array<{
      key?: React.Key;
      onClick?: () => void;
    }>;
    onSynchronizeLockChange?: () => Promise<void>;
  }) => (
    <div>
      <button
        type="button"
        data-testid="split-action"
        disabled={props.splitDisabled}
      >
        拆票
      </button>
      <span data-testid="split-reasons">
        {props.splitBlockedReasons?.join('；')}
      </span>
      <button
        type="button"
        data-testid="reassign-action"
        disabled={props.reassignDisabled}
      >
        改配
      </button>
      <span data-testid="reassign-reasons">
        {props.reassignBlockedReasons?.join('；')}
      </span>
      <button
        type="button"
        onClick={() =>
          props.moreMenuItems
            ?.find((item) => item?.key === 'reload-data')
            ?.onClick?.()
        }
      >
        刷新动作资格
      </button>
      <button
        type="button"
        onClick={() => void props.onSynchronizeLockChange?.()}
      >
        同步锁单状态
      </button>
    </div>
  ),
}));

const mockGetChangeActions = vi.mocked(
  seaOrderChangeServiceGetSeaOrderChangeActions,
);

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe('订单详情页拆票与改配动作隔离', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.params = { kind: 'sea-export', id: 'ord-A' };
    detailTestState.loadData.mockResolvedValue(undefined);
  });

  it('A 与 B 响应逆序返回时，仅展示当前订单 B 的动作资格', async () => {
    const requestA = deferred<any>();
    const requestB = deferred<any>();
    mockGetChangeActions
      .mockImplementationOnce(() => requestA.promise)
      .mockImplementationOnce(() => requestB.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => {
      expect(mockGetChangeActions).toHaveBeenCalledWith({ orderId: 'ord-A' });
    });

    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => {
      expect(mockGetChangeActions).toHaveBeenCalledWith({ orderId: 'ord-B' });
    });

    await act(async () => {
      requestB.resolve({
        data: {
          canSplit: false,
          splitBlockedReasons: ['B 不允许拆票'],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      });
    });
    await waitFor(() => {
      expect(screen.getByTestId('split-action')).toBeDisabled();
      expect(screen.getByTestId('split-reasons')).toHaveTextContent(
        'B 不允许拆票',
      );
      expect(screen.getByTestId('reassign-action')).toBeEnabled();
    });

    await act(async () => {
      requestA.resolve({
        data: {
          canSplit: true,
          splitBlockedReasons: [],
          canReassign: false,
          reassignBlockedReasons: ['A 不允许改配'],
        },
      });
    });

    expect(screen.getByTestId('split-action')).toBeDisabled();
    expect(screen.getByTestId('split-reasons')).toHaveTextContent(
      'B 不允许拆票',
    );
    expect(screen.getByTestId('reassign-action')).toBeEnabled();
    expect(screen.getByTestId('reassign-reasons')).toBeEmptyDOMElement();
  });

  it('切换到 B 后立即清空 A 的动作资格，且 B 失败时不恢复 A 的状态', async () => {
    const requestB = deferred<any>();
    mockGetChangeActions
      .mockResolvedValueOnce({
        data: {
          canSplit: true,
          splitBlockedReasons: [],
          canReassign: false,
          reassignBlockedReasons: ['A 不允许改配'],
        },
      } as any)
      .mockImplementationOnce(() => requestB.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() => {
      expect(screen.getByTestId('split-action')).toBeEnabled();
      expect(screen.getByTestId('reassign-reasons')).toHaveTextContent(
        'A 不允许改配',
      );
    });

    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );

    expect(screen.getByTestId('split-action')).toBeDisabled();
    expect(screen.getByTestId('split-reasons')).toBeEmptyDOMElement();
    expect(screen.getByTestId('reassign-action')).toBeDisabled();
    expect(screen.getByTestId('reassign-reasons')).toBeEmptyDOMElement();

    await act(async () => {
      requestB.reject(new Error('B 动作资格加载失败'));
    });

    await waitFor(() => {
      expect(screen.getByTestId('split-action')).toBeDisabled();
      expect(screen.getByTestId('reassign-action')).toBeDisabled();
    });
    expect(screen.getByTestId('reassign-reasons')).toBeEmptyDOMElement();
  });

  it('页面手工刷新会重新获取当前订单的动作资格', async () => {
    mockGetChangeActions
      .mockResolvedValueOnce({
        data: {
          canSplit: true,
          splitBlockedReasons: [],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      } as any)
      .mockResolvedValueOnce({
        data: {
          canSplit: false,
          splitBlockedReasons: ['刷新后不可拆票'],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      } as any);

    render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getByTestId('split-action')).toBeEnabled(),
    );

    fireEvent.click(screen.getByRole('button', { name: '刷新动作资格' }));

    await waitFor(() => {
      expect(mockGetChangeActions).toHaveBeenCalledTimes(2);
      expect(screen.getByTestId('split-action')).toBeDisabled();
      expect(screen.getByTestId('split-reasons')).toHaveTextContent(
        '刷新后不可拆票',
      );
    });
  });

  it('订单 A 的旧同步任务结束后，不得使订单 B 当前动作响应失效', async () => {
    const loadOrderA = deferred<void>();
    const requestB = deferred<any>();
    detailTestState.loadData.mockImplementation((orderId) =>
      orderId === 'ord-A' ? loadOrderA.promise : Promise.resolve(),
    );
    mockGetChangeActions
      .mockResolvedValueOnce({
        data: {
          canSplit: true,
          splitBlockedReasons: [],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      } as any)
      .mockImplementationOnce(() => requestB.promise);

    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getByTestId('split-action')).toBeEnabled(),
    );

    fireEvent.click(screen.getByRole('button', { name: '同步锁单状态' }));
    await waitFor(() =>
      expect(detailTestState.loadData).toHaveBeenCalledWith('ord-A'),
    );

    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(mockGetChangeActions).toHaveBeenCalledWith({ orderId: 'ord-B' }),
    );

    await act(async () => {
      loadOrderA.resolve();
    });
    expect(mockGetChangeActions).toHaveBeenCalledTimes(2);

    await act(async () => {
      requestB.resolve({
        data: {
          canSplit: false,
          splitBlockedReasons: ['B 当前不可拆票'],
          canReassign: true,
          reassignBlockedReasons: [],
        },
      });
    });

    await waitFor(() => {
      expect(screen.getByTestId('split-action')).toBeDisabled();
      expect(screen.getByTestId('split-reasons')).toHaveTextContent(
        'B 当前不可拆票',
      );
    });
  });
});
