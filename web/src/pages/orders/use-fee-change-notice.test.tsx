import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { orderLockServiceGetOrderLockState } from '@/services/roncin/orderLockService';
import { useFeeChangeNotice } from './use-fee-change-notice';

vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceGetOrderLockState: vi.fn(),
}));
const getLockState = vi.mocked(orderLockServiceGetOrderLockState);

/** 真实 wire 形态：未锁定时 protojson 省略 isLocked，lockGeneration 为字符串。 */
function lockState(feeContentVersion: string, lockGeneration = '0') {
  return {
    success: true,
    data: { orderId: 'order-1', feeContentVersion, lockGeneration },
  } as Awaited<ReturnType<typeof orderLockServiceGetOrderLockState>>;
}

function mount(options: { enabled?: boolean } = {}) {
  const client = createTestQueryClient();
  return renderHook(
    (props: { enabled: boolean }) =>
      useFeeChangeNotice({ orderId: 'order-1', enabled: props.enabled }),
    {
      initialProps: { enabled: options.enabled ?? true },
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    },
  );
}

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
  // 探测结果经 React Query 通知队列异步派发，补推进几轮使其落入渲染。
  for (let i = 0; i < 3; i += 1) {
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
  }
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  getLockState.mockResolvedValue(lockState('5'));
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('费用页外部变动检查', () => {
  it('首次探测建立参考点；前台每 15 秒复查，版本未变不提示', async () => {
    const { result } = mount();
    await advance(0);
    expect(getLockState).toHaveBeenCalledTimes(1);
    expect(getLockState).toHaveBeenCalledWith(
      { orderId: 'order-1' },
      { signal: expect.any(AbortSignal), skipErrorHandler: true },
    );
    await advance(14_000);
    expect(getLockState).toHaveBeenCalledTimes(1);
    await advance(1_000);
    expect(getLockState).toHaveBeenCalledTimes(2);
    expect(result.current.changed).toBe(false);
  });

  it('费用内容版本变化后持续提示，后续检查失败也不清除提醒', async () => {
    const { result } = mount();
    await advance(0);
    getLockState.mockResolvedValue(lockState('6'));
    await advance(15_000);
    expect(result.current.changed).toBe(true);
    getLockState.mockRejectedValue(new Error('network'));
    await advance(15_000);
    expect(result.current.changed).toBe(true);
  });

  it('审核锁状态变化同样视为变动（未锁定到已锁定）', async () => {
    const { result } = mount();
    await advance(0);
    getLockState.mockResolvedValue({
      success: true,
      data: {
        orderId: 'order-1',
        feeContentVersion: '5',
        lockGeneration: '1',
        isLocked: true,
      },
    } as Awaited<ReturnType<typeof orderLockServiceGetOrderLockState>>);
    await advance(15_000);
    expect(result.current.changed).toBe(true);
  });

  it('隐藏页签不检查，切回前台立即检查', async () => {
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    const { result } = mount();
    await advance(45_000);
    expect(getLockState).not.toHaveBeenCalled();
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'));
    });
    await advance(0);
    expect(getLockState).toHaveBeenCalledTimes(1);
    expect(result.current.changed).toBe(false);
  });

  it('未启用（新建、只读或加载中）不发请求', async () => {
    mount({ enabled: false });
    await advance(45_000);
    expect(getLockState).not.toHaveBeenCalled();
  });

  it('本人保存按成功响应版本同步参考点，不误报；显式刷新后同步到最新', async () => {
    const { result } = mount();
    await advance(0);
    getLockState.mockResolvedValue(lockState('6'));
    await act(async () => {
      await result.current.acceptOwnSave('6');
    });
    await advance(15_000);
    expect(result.current.changed).toBe(false);

    getLockState.mockResolvedValue(lockState('8'));
    await advance(15_000);
    expect(result.current.changed).toBe(true);
    await act(async () => {
      await result.current.rebaseline();
    });
    expect(result.current.changed).toBe(false);
  });
});

// 应收独立保存不会刷新应付，已发现的应付外部变动须持续提示。
it('本侧保存保留已发现的另一侧变动，显式刷新后才解除', async () => {
  const { result } = mount();
  await advance(0);
  getLockState.mockResolvedValue(lockState('6'));
  await advance(15_000);
  expect(result.current.changed).toBe(true);
  getLockState.mockResolvedValue(lockState('7'));
  await act(async () => {
    await result.current.acceptOwnSave('7');
  });
  await advance(0);
  expect(result.current.changed).toBe(true);
  await advance(15_000);
  expect(result.current.changed).toBe(true);
  getLockState.mockRejectedValue(new Error('network'));
  await act(async () => {
    await result.current.rebaseline();
  });
  expect(result.current.changed).toBe(true);
  getLockState.mockResolvedValue(lockState('7'));
  await act(async () => {
    await result.current.rebaseline();
  });
  await advance(0);
  expect(result.current.changed).toBe(false);
});
