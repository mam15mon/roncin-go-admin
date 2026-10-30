import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { useOrderChangeNotice } from './use-order-change-notice';

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: vi.fn(),
}));
const getOrder = vi.mocked(orderServiceGetOrder);
type Options = Parameters<typeof useOrderChangeNotice>[0];

function snapshot(id = 'order-1'): API.Order {
  return {
    id,
    version: '9007199254740993',
    seaMasterBill: {
      masterBillId: 'mbl-1',
      version: '2',
      transportExecutionId: 'te-1',
      transportExecutionVersion: '3',
    },
    seaDocumentLinkVersion: '4',
    seaDocumentStructure: 1,
  };
}

function mount(overrides: Partial<Options> = {}) {
  const editing = { current: snapshot() };
  const options: Options = {
    identity: 'sea-export:order-1',
    scope: 'user-1:org-1',
    orderId: 'order-1',
    initialOrder: snapshot(),
    enabled: true,
    paused: false,
    readEditingOrder: () => ({
      order: editing.current,
      masterContentVersion: editing.current.seaMasterBill?.version,
    }),
    ...overrides,
  };
  const client = createTestQueryClient();
  const result = renderHook((props) => useOrderChangeNotice(props), {
    initialProps: options,
    wrapper: ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    ),
  });
  return { ...result, editing, options, client };
}

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  getOrder.mockResolvedValue({ data: snapshot() });
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('订单变动轻量检查', () => {
  it('复用已加载快照，15秒才读取一次，结果不回填编辑快照', async () => {
    const latest = { ...snapshot(), version: '9007199254740994' };
    getOrder.mockResolvedValue({ data: latest });
    const { result, editing } = mount();
    await advance(14_999);
    expect(getOrder).not.toHaveBeenCalled();
    expect(result.current.changed).toBe(false);
    await advance(1);
    await advance(1);
    expect(getOrder).toHaveBeenCalledOnce();
    expect(getOrder).toHaveBeenCalledWith(
      { id: 'order-1' },
      { signal: expect.any(AbortSignal), skipErrorHandler: true },
    );
    expect(result.current.changed).toBe(true);
    expect(editing.current.version).toBe('9007199254740993');
    await advance(15_000);
    expect(getOrder).toHaveBeenCalledTimes(2);
    expect(result.current.changed).toBe(true);
  });

  it.each([
    ['主单', { seaMasterBill: { ...snapshot().seaMasterBill, version: '5' } }],
    [
      '航程',
      {
        seaMasterBill: {
          ...snapshot().seaMasterBill,
          transportExecutionVersion: '5',
        },
      },
    ],
    [
      '航程身份',
      {
        seaMasterBill: {
          ...snapshot().seaMasterBill,
          transportExecutionId: 'te-2',
        },
      },
    ],
    ['分单活动关联', { seaDocumentLinkVersion: '5' }],
    ['单证模式', { seaDocumentStructure: 2 }],
    ['解除主单关联', { seaMasterBill: undefined }],
  ] satisfies Array<[string, Partial<API.Order>]>)(
    '只修改%s也能发现变化',
    async (_label, change) => {
      getOrder.mockResolvedValue({ data: { ...snapshot(), ...change } });
      const { result } = mount();
      await advance(15_000);
      await advance(1);
      expect(result.current.changed).toBe(true);
    },
  );

  it('恢复旧草稿立即与已加载快照比较，不额外重复初始GET', async () => {
    const { result, editing } = mount({
      readEditingOrder: () => ({
        order: { ...snapshot(), version: '9007199254740992' },
        masterContentVersion: '2',
      }),
    });
    await advance(0);
    expect(result.current.changed).toBe(true);
    expect(getOrder).not.toHaveBeenCalled();
    expect(editing.current.version).toBe('9007199254740993');
  });

  it('检查失败保留已发现的变化，未变化时不提示', async () => {
    const { result } = mount();
    await advance(15_000);
    await advance(1);
    expect(result.current.changed).toBe(false);
    getOrder.mockResolvedValueOnce({
      data: { ...snapshot(), seaDocumentLinkVersion: '8' },
    });
    await advance(15_000);
    await advance(1);
    expect(result.current.changed).toBe(true);
    getOrder.mockRejectedValueOnce(new Error('离线'));
    await advance(15_000);
    expect(result.current.changed).toBe(true);
  });

  it('失焦或隐藏时停查，focus与visibility恢复只触发一次并复用在途请求', async () => {
    const pending = Promise.withResolvers<API.GetOrderResponse>();
    getOrder.mockReturnValue(pending.promise);
    mount();
    act(() => window.dispatchEvent(new Event('blur')));
    await advance(30_000);
    expect(getOrder).not.toHaveBeenCalled();
    const visibility = vi.spyOn(document, 'visibilityState', 'get');
    visibility.mockReturnValue('hidden');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await advance(30_000);
    expect(getOrder).not.toHaveBeenCalled();
    visibility.mockReturnValue('visible');
    act(() => {
      window.dispatchEvent(new Event('focus'));
      document.dispatchEvent(new Event('visibilitychange'));
      window.dispatchEvent(new Event('focus'));
    });
    await advance(0);
    expect(getOrder).toHaveBeenCalledOnce();
    act(() => {
      window.dispatchEvent(new Event('blur'));
      window.dispatchEvent(new Event('focus'));
    });
    await advance(0);
    expect(getOrder).toHaveBeenCalledOnce();
    await act(async () => pending.resolve({ data: snapshot() }));
  });

  it('本人保存期间暂停，成功接受快照后丢弃旧检查的晚到结果', async () => {
    const pending = Promise.withResolvers<API.GetOrderResponse>();
    getOrder.mockReturnValueOnce(pending.promise);
    const { result, rerender, editing, options } = mount();
    await advance(15_000);
    const signal = getOrder.mock.calls[0][1]?.signal as AbortSignal;
    rerender({ ...options, paused: true });
    expect(signal.aborted).toBe(true);
    await advance(30_000);
    expect(getOrder).toHaveBeenCalledOnce();
    const saved = { ...snapshot(), version: '9007199254740994' };
    editing.current = saved;
    act(() => result.current.acceptSnapshot(saved));
    rerender({ ...options, paused: false });
    await act(async () => pending.resolve({ data: snapshot() }));
    await advance(0);
    expect(result.current.changed).toBe(false);
    getOrder.mockResolvedValue({ data: saved });
    await advance(15_000);
    expect(result.current.changed).toBe(false);
  });

  it.each(['订单往返', '切换组织'])(
    '%s不消费旧检查或旧重置回调',
    async (change) => {
      const pending = Promise.withResolvers<API.GetOrderResponse>();
      getOrder.mockReturnValueOnce(pending.promise);
      const { result, rerender, editing, options } = mount();
      const oldAccept = result.current.acceptSnapshot;
      await advance(15_000);
      await advance(1);
      const next =
        change === '订单往返'
          ? {
              ...options,
              identity: 'sea-export:order-2',
              orderId: 'order-2',
              initialOrder: snapshot('order-2'),
            }
          : { ...options, scope: 'user-1:org-2' };
      editing.current = next.initialOrder ?? snapshot();
      rerender(next);
      if (change === '订单往返') {
        editing.current = snapshot();
        rerender(options);
      }
      await act(async () =>
        pending.resolve({ data: { ...snapshot(), version: '999' } }),
      );
      await advance(0);
      expect(result.current.changed).toBe(false);
      getOrder.mockResolvedValue({
        data: { ...editing.current, version: '1000' },
      });
      await advance(15_000);
      await advance(1);
      expect(result.current.changed).toBe(true);
      act(() => oldAccept(snapshot()));
      expect(result.current.changed).toBe(true);
    },
  );

  it('主单身份基线已更新但正文草稿旧时也提前提醒', async () => {
    const { result } = mount({
      readEditingOrder: () => ({
        order: snapshot(),
        masterContentVersion: '1',
      }),
    });
    await advance(0);
    expect(result.current.changed).toBe(true);
    expect(getOrder).not.toHaveBeenCalled();
  });

  it.each([
    { scope: undefined },
    { identity: undefined },
    { orderId: 'other-order' },
    { enabled: false },
  ])('返回前台也不绕过未就绪身份%j', async (overrides) => {
    mount(overrides);
    act(() => {
      window.dispatchEvent(new Event('blur'));
      window.dispatchEvent(new Event('focus'));
    });
    await advance(30_000);
    expect(getOrder).not.toHaveBeenCalled();
  });
});
