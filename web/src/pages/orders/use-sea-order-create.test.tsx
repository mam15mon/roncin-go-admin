import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderServiceCreateOrder } from '@/services/roncin/orderService';
import { useSeaOrderCreate } from './use-sea-order-create';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceCreateOrder: vi.fn(),
}));
const create = vi.mocked(orderServiceCreateOrder);
const defaults: API.CreateOrderRequest = {
  customerId: 'customer',
  businessType: 1,
  tradeDirection: 1,
  paymentTerm: 1,
};
function setup(key = 'test-scope:sea') {
  const client = createTestQueryClient();
  return renderHook(() => useSeaOrderCreate(key), {
    wrapper: ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    ),
  });
}
beforeEach(() => {
  sessionStorage.clear();
  vi.resetAllMocks();
});
describe('海运新建待确认提交', () => {
  it('超时后修改传入值仍以原快照同键重试，刷新后也保留', async () => {
    create.mockRejectedValueOnce(new Error('timeout'));
    const first = setup();
    await act(async () => {
      await expect(
        first.result.current.submit({
          ...defaults,
          customerId: 'customer-A',
          notes: '原输入',
        }),
      ).rejects.toThrow('待确认');
    });
    expect(first.result.current.frozen).toBe(true);
    const sent = create.mock.calls[0][0];
    first.unmount();
    const second = setup();
    expect(second.result.current.status).toBe('unknown');
    create.mockResolvedValueOnce({ data: { id: 'created' } });
    await act(async () => {
      expect(
        await second.result.current.submit({
          ...defaults,
          customerId: 'customer-B',
        }),
      ).toBe('created');
    });
    expect(create.mock.calls[1][0]).toEqual(sent);
    expect(second.result.current.status).toBe('success');
    second.unmount();
    const third = setup();
    await act(async () => {
      expect(await third.result.current.submit()).toBe('created');
    });
    expect(create).toHaveBeenCalledTimes(2);
  });

  it('请求期间拦截重复提交，成功只写一张订单', async () => {
    let finish: (response: { data: { id: string } }) => void = () => {};
    create.mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const hook = setup();
    let pending: Promise<string | undefined>;
    act(() => {
      pending = hook.result.current.submit({ ...defaults, notes: '原输入' });
    });
    await act(async () => {
      expect(
        await hook.result.current.submit({ ...defaults, notes: '重复' }),
      ).toBeUndefined();
    });
    expect(hook.result.current.frozen).toBe(true);
    await act(async () => {
      finish({ data: { id: 'created' } });
      await pending;
    });
    expect(create).toHaveBeenCalledTimes(1);
  });

  it('业务参数拒绝后保留编辑能力，新意图重新生成键', async () => {
    create.mockRejectedValueOnce({
      data: {
        code: 400,
        reason: 'ORDER_INVALID_ARGUMENT',
        message: '缺少资料',
      },
    });
    const hook = setup();
    await act(async () => {
      await expect(
        hook.result.current.submit({ ...defaults, notes: '错误' }),
      ).rejects.toThrow('缺少资料');
    });
    expect(hook.result.current.frozen).toBe(false);
    create.mockResolvedValueOnce({ data: { id: 'created' } });
    await act(async () => {
      await hook.result.current.submit({ ...defaults, notes: '修正' });
    });
    expect(create.mock.calls[1][0].idempotencyKey).not.toBe(
      create.mock.calls[0][0].idempotencyKey,
    );
  });

  it.each([
    undefined,
    {},
    { success: false },
    { response: { status: 500 } },
    { response: { status: 409 } },
    {
      name: 'DuplicateSubmitError',
      response: { status: 409, data: { reason: 'DUPLICATE_SUBMIT' } },
    },
    {
      response: { status: 409, data: { reason: 'ORDER_IDEMPOTENCY_CONFLICT' } },
    },
  ])('空响应及不确定错误 %j 不解除冻结', async (response) => {
    if (response && 'response' in response)
      create.mockRejectedValueOnce(response);
    else create.mockResolvedValueOnce(response as never);
    const hook = setup();
    await act(async () => {
      await expect(
        hook.result.current.submit({ ...defaults, notes: '保留' }),
      ).rejects.toThrow();
    });
    expect(hook.result.current.status).toBe('unknown');
  });

  it('结果未知后确认遇到权限拒绝仍保留原请求键', async () => {
    create.mockRejectedValueOnce(new Error('timeout'));
    const hook = setup();
    await act(async () => {
      await hook.result.current.submit(defaults).catch(() => {});
    });
    create.mockRejectedValueOnce({
      response: {
        status: 403,
        data: { reason: 'FORBIDDEN', message: '无权访问' },
      },
    });
    await act(async () => {
      await hook.result.current.submit().catch(() => {});
    });
    expect(hook.result.current.status).toBe('unknown');
    expect(create.mock.calls[1][0]).toEqual(create.mock.calls[0][0]);
  });

  it('会话存储失败时不发送请求', async () => {
    const hook = setup();
    const storage = vi
      .spyOn(sessionStorage, 'setItem')
      .mockImplementation(() => {
        throw new Error('quota');
      });
    await act(async () => {
      await expect(
        hook.result.current.submit({ ...defaults, notes: '保留' }),
      ).rejects.toThrow('尚未发送');
    });
    expect(create).not.toHaveBeenCalled();
    storage.mockRestore();
  });

  it('用户、公司或来源路径不同不恢复其他意图', async () => {
    create.mockRejectedValueOnce(new Error('timeout'));
    const first = setup('userA:orgA:sourceA');
    await act(async () => {
      await first.result.current
        .submit({ ...defaults, notes: '保留' })
        .catch(() => {});
    });
    first.unmount();
    expect(setup('userA:orgB:sourceA').result.current.status).toBe('idle');
    expect(setup('userB:orgA:sourceA').result.current.status).toBe('idle');
    expect(setup('userA:orgA:sourceB').result.current.status).toBe('idle');
  });
});
