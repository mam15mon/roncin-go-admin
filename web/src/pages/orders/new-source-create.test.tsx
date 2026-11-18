import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { getFormDraftKey, hasFormDraft } from '@/components/layout/formDraft';
import {
  createOrder,
  deferred,
  getOrder,
  mount,
  navigate,
  navigation,
  ready,
  reference,
} from './new-source.fixtures';

describe('连续新增分单的提交与迟到结果', () => {
  it('失败重试沿用幂等键，来源隔离；后台来源成功只清自己的已提交草稿', async () => {
    const { router } = mount();
    await ready();
    fireEvent.change(reference(), { target: { value: 'A草稿' } });
    fireEvent.click(
      screen.getAllByRole('button', { name: /创建订单|确认创建结果/ })[0],
    );
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(
        screen.getAllByRole('button', { name: /创建订单|确认创建结果/ })[0],
      ).not.toHaveClass('ant-btn-loading'),
    );
    fireEvent.click(
      screen.getAllByRole('button', { name: /创建订单|确认创建结果/ })[0],
    );
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(2));
    const pending = deferred<API.CreateOrderResponse>();
    createOrder.mockReturnValueOnce(pending.promise);
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    fireEvent.change(reference(), { target: { value: 'B草稿' } });
    fireEvent.click(
      screen.getAllByRole('button', { name: /创建订单|确认创建结果/ })[0],
    );
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(3));
    await navigate(router, '/orders/sea-export/new?sourceOrderId=A');
    expect(reference()).toHaveValue('A草稿');
    await act(async () => pending.resolve({ data: { id: 'late-B' } }));
    expect(navigation.push).not.toHaveBeenCalled();
    expect(screen.queryByText('创建订单成功')).not.toBeInTheDocument();
    fireEvent.click(
      screen.getAllByRole('button', { name: /创建订单|确认创建结果/ })[0],
    );
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(4));
    const keys = createOrder.mock.calls.map(
      ([payload]) => payload.idempotencyKey,
    );
    expect(keys[0]).toBeTruthy();
    expect(keys[0]).toBe(keys[1]);
    expect(keys[2]).not.toBe(keys[0]);
    expect(keys[3]).toBe(keys[0]);
    expect(keys[3]).not.toBe(keys[2]);
    expect(reference()).toHaveValue('A草稿');
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    expect(
      hasFormDraft(
        getFormDraftKey(
          '/orders/sea-export/new',
          '/orders/sea-export/new?sourceOrderId=B',
          'user-1:org-1',
        ),
      ),
    ).toBe(false);
    // 已成功的 B 输入已提交，独立待确认记录继续保留原幂等键供核对。
    const pendingB = sessionStorage.getItem(
      `roncin:pending-order-create:${getFormDraftKey('/orders/sea-export/new', '/orders/sea-export/new?sourceOrderId=B', 'user-1:org-1')}`,
    );
    expect(JSON.parse(pendingB ?? '{}').payload?.idempotencyKey).toBe(keys[2]);
    expect(reference()).not.toHaveValue('B草稿');
  });

  it('切换来源后旧创建失败不反馈错误，当前创建失败仍保留输入与重试能力', async () => {
    const pending = deferred<API.CreateOrderResponse>();
    createOrder.mockReturnValueOnce(pending.promise);
    const { router } = mount();
    await ready();
    fireEvent.click(
      screen.getAllByRole('button', { name: /创建订单|确认创建结果/ })[0],
    );
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(1));
    expect(createOrder.mock.calls[0][1]).toEqual({ skipErrorHandler: true });
    await navigate(router, '/orders/sea-export/new?sourceOrderId=B');
    fireEvent.change(reference(), { target: { value: 'B未保存输入' } });
    await act(async () => pending.reject(new Error('A迟到失败')));
    expect(screen.queryByText('A迟到失败')).not.toBeInTheDocument();
    createOrder.mockRejectedValueOnce({
      response: {
        status: 400,
        data: { message: '当前创建失败', reason: 'ORDER_INVALID_ARGUMENT' },
      },
    });
    fireEvent.click(
      screen.getAllByRole('button', { name: /创建订单|确认创建结果/ })[0],
    );
    await screen.findByText('当前创建失败');
    expect(screen.getAllByText('当前创建失败')).toHaveLength(1);
    expect(reference()).toHaveValue('B未保存输入');
    expect(navigation.push).not.toHaveBeenCalled();
    createOrder.mockResolvedValueOnce(undefined as never);
    fireEvent.click(
      screen.getAllByRole('button', { name: /创建订单|确认创建结果/ })[0],
    );
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(3));
    expect(createOrder.mock.calls[2][0].idempotencyKey).not.toBe(
      createOrder.mock.calls[1][0].idempotencyKey,
    );
    expect(reference()).toHaveValue('B未保存输入');
  });
  it('真实表单提交中冻结输入，刷新且来源读取失败后仍可同键同载荷确认', async () => {
    const pending = deferred<API.CreateOrderResponse>();
    createOrder.mockReturnValueOnce(pending.promise);
    mount();
    await ready();
    fireEvent.change(reference(), { target: { value: '待确认输入' } });
    fireEvent.click(screen.getAllByRole('button', { name: /创建订单/ })[0]);
    await waitFor(() => expect(createOrder).toHaveBeenCalledTimes(1));
    expect(reference()).toBeDisabled();
    expect(screen.getAllByRole('button', { name: /创建订单/ })).toHaveLength(1);
    await act(async () => pending.reject(new Error('超时')));
    const firstPayload = createOrder.mock.calls[0][0];
    cleanup();
    getOrder.mockRejectedValueOnce(new Error('来源不可用'));
    createOrder.mockResolvedValueOnce({ data: { id: 'recovered-created' } });
    mount();
    await screen.findByText('本次创建结果待确认');
    fireEvent.click(screen.getByRole('button', { name: '确认创建结果' }));
    await waitFor(() =>
      expect(navigation.push).toHaveBeenCalledWith(
        '/orders/sea-export/recovered-created',
      ),
    );
    expect(createOrder.mock.calls[1][0]).toEqual(firstPayload);
    expect(sessionStorage.length).toBe(0);
  });
});
