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
import { history } from '@/router/history';
import { orderServiceCreateOrder } from '@/services/roncin/orderService';
import NewOrderPage from './new';

const routeState = vi.hoisted(() => ({
  params: { kind: 'sea-export' },
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useParams: () => routeState.params,
  };
});

vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOrder: () => true }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        displayName: '测试用户',
        currentOrganization: { id: 'org-1', name: '总公司' },
      },
    },
  }),
}));

vi.mock('@/router/history', () => ({
  history: { push: vi.fn() },
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceCheckOrderReference: vi.fn(),
  orderServiceCreateOrder: vi.fn(),
}));

vi.mock('./use-order-create-options', () => ({
  useOrderCreateOptions: () => ({
    loading: false,
    error: null,
    retry: vi.fn(),
    serviceTypeOptions: [],
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: vi.fn().mockResolvedValue([]),
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: [],
  }),
}));

vi.mock('./components/OrderPageHeader', () => ({
  default: ({ actions }: { actions?: React.ReactNode }) => <div>{actions}</div>,
}));

vi.mock('@/components/ui/order-template/OrderFormTemplate', () => ({
  OrderFormTemplate: (props: {
    onFinish?: (values: unknown) => Promise<boolean>;
    header?: React.ReactNode;
    formRef?: React.MutableRefObject<{ submit: () => void } | undefined>;
  }) => {
    React.useEffect(() => {
      if (!props.formRef) return;
      props.formRef.current = {
        submit: () => {
          void props.onFinish?.({ customerId: 'c-1' });
        },
      };
      return () => {
        if (props.formRef) props.formRef.current = undefined;
      };
    }, [props.formRef, props.onFinish]);

    return <>{props.header}</>;
  },
}));

const mockCreateOrder = vi.mocked(orderServiceCreateOrder);

describe('订单新建页创建幂等键', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.params = { kind: 'sea-export' };
  });

  it('每次请求携带幂等键；失败重试沿用同键，成功后重新生成', async () => {
    mockCreateOrder.mockRejectedValueOnce(new Error('创建失败'));
    mockCreateOrder.mockResolvedValueOnce({
      data: { id: 'order-created-1' },
    } as never);
    mockCreateOrder.mockResolvedValueOnce({
      data: { id: 'order-created-2' },
    } as never);

    render(
      <App>
        <NewOrderPage />
      </App>,
    );

    const submit = () =>
      act(async () => {
        fireEvent.click(screen.getByText('创建订单'));
      });

    await submit();
    await submit();
    await submit();

    expect(mockCreateOrder).toHaveBeenCalledTimes(3);
    const sentKeys = mockCreateOrder.mock.calls.map(
      (call) => (call[0] as { idempotencyKey?: string }).idempotencyKey,
    );
    // 每次请求都必须携带幂等键。
    expect(sentKeys[0]).toBeTruthy();
    // 首次失败后的重试沿用同一键，由后端重放语义兜底超时场景。
    expect(sentKeys[1]).toBe(sentKeys[0]);
    // 创建成功后重新生成，下一次提交意图使用新键。
    await waitFor(() => expect(sentKeys[2]).toBeTruthy());
    expect(sentKeys[2]).not.toBe(sentKeys[1]);
  });

  it('创建成功读取服务端返回订单 ID 并直接进入对应详情页', async () => {
    const historyPush = vi.mocked(history.push);
    mockCreateOrder.mockResolvedValueOnce({
      data: { id: 'order-created-detail', orderNo: 'SE1' },
    } as never);

    render(
      <App>
        <NewOrderPage />
      </App>,
    );

    await act(async () => {
      fireEvent.click(screen.getByText('创建订单'));
    });

    await waitFor(() =>
      expect(historyPush).toHaveBeenCalledWith(
        '/orders/sea-export/order-created-detail',
      ),
    );
    // 不再返回订单列表。
    expect(historyPush).not.toHaveBeenCalledWith('/orders/sea-export');
  });

  it('空响应或缺 ID 不提示成功、不跳转、不轮换幂等键', async () => {
    const historyPush = vi.mocked(history.push);
    mockCreateOrder.mockResolvedValue({} as never);

    render(
      <App>
        <NewOrderPage />
      </App>,
    );

    await act(async () => {
      fireEvent.click(screen.getByText('创建订单'));
    });
    await act(async () => {
      fireEvent.click(screen.getByText('创建订单'));
    });

    const sentKeys = mockCreateOrder.mock.calls.map(
      (call) => (call[0] as { idempotencyKey?: string }).idempotencyKey,
    );
    // 未确认创建结果前沿用同键，供同键重试取回已创建订单。
    expect(sentKeys[1]).toBe(sentKeys[0]);
    expect(historyPush).not.toHaveBeenCalled();
  });
});
