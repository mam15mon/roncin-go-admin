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
import { orderServiceCreateOrder } from '@/services/roncin/orderService';
import NewOrderPage from './new';

const routeState = vi.hoisted(() => ({
  params: { kind: 'sea-export' },
}));

vi.mock('@umijs/max', () => ({
  useParams: () => routeState.params,
  useAccess: () => ({ canOrder: () => true }),
  useModel: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        displayName: '测试用户',
        currentOrganization: { id: 'org-1', name: '总公司' },
      },
    },
  }),
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
  default: () => <div />,
}));

vi.mock('@/components/ui/order-template/OrderFormTemplate', () => ({
  OrderFormTemplate: (props: {
    onFinish?: (values: unknown) => Promise<boolean>;
    submitText?: string;
  }) => (
    <button
      type="button"
      onClick={() => props.onFinish?.({ customerId: 'c-1' })}
    >
      {props.submitText ?? 'submit'}
    </button>
  ),
}));

const mockCreateOrder = vi.mocked(orderServiceCreateOrder);

describe('订单新建页创建幂等键', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.params = { kind: 'sea-export' };
  });

  it('每次请求携带幂等键；失败重试沿用同键，成功后重新生成', async () => {
    mockCreateOrder.mockRejectedValueOnce(new Error('创建失败'));
    mockCreateOrder.mockResolvedValueOnce({} as never);
    mockCreateOrder.mockResolvedValueOnce({} as never);

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
});
