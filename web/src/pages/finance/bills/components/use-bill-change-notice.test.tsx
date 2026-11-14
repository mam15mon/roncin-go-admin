import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { settlementServiceGetBill } from '@/services/roncin/settlementService';
import { useBillChangeNotice } from './use-bill-change-notice';

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetBill: vi.fn(),
}));
const getBill = vi.mocked(settlementServiceGetBill);

/** 真实 wire 形态：uint64 版本为字符串。 */
function billVersion(version: string) {
  return { success: true, data: { id: 'bill-1', version } } as Awaited<
    ReturnType<typeof settlementServiceGetBill>
  >;
}

function mount(version = '3') {
  const client = createTestQueryClient();
  return renderHook(
    (props: { version: string; enabled: boolean }) =>
      useBillChangeNotice({ billId: 'bill-1', ...props }),
    {
      initialProps: { version, enabled: true },
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
  getBill.mockResolvedValue(billVersion('3'));
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('账单详情外部变动检查', () => {
  it('前台每 15 秒复查，版本未变不提醒；更新后持续提醒，检查失败不清除', async () => {
    const { result } = mount();
    await advance(0);
    expect(getBill).toHaveBeenCalledTimes(1);
    await advance(15_000);
    expect(getBill).toHaveBeenCalledTimes(2);
    expect(result.current.changed).toBe(false);
    getBill.mockResolvedValue(billVersion('4'));
    await advance(15_000);
    expect(result.current.changed).toBe(true);
    getBill.mockRejectedValue(new Error('网络错误'));
    await advance(15_000);
    expect(result.current.changed).toBe(true);
  });

  it('本人保存后详情版本前移，较旧或相同的探测不误报', async () => {
    const { result, rerender } = mount();
    await advance(0);
    getBill.mockResolvedValue(billVersion('4'));
    rerender({ version: '4', enabled: true });
    await advance(15_000);
    expect(result.current.changed).toBe(false);
  });

  it('页面隐藏时停止轮询，回到前台立即检查', async () => {
    mount();
    await advance(0);
    expect(getBill).toHaveBeenCalledTimes(1);
    vi.mocked(document.hasFocus).mockReturnValue(false);
    await act(async () => {
      window.dispatchEvent(new Event('blur'));
    });
    await advance(30_000);
    expect(getBill).toHaveBeenCalledTimes(1);
    vi.mocked(document.hasFocus).mockReturnValue(true);
    await act(async () => {
      window.dispatchEvent(new Event('focus'));
    });
    await advance(0);
    expect(getBill).toHaveBeenCalledTimes(2);
  });

  it('未启用（已取消账单）不发请求', async () => {
    const { result } = mount();
    cleanup();
    getBill.mockClear();
    const client = createTestQueryClient();
    renderHook(
      () =>
        useBillChangeNotice({ billId: 'bill-1', version: '3', enabled: false }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={client}>{children}</QueryClientProvider>
        ),
      },
    );
    await advance(30_000);
    expect(getBill).not.toHaveBeenCalled();
    expect(result.current.changed).toBe(false);
  });
});
