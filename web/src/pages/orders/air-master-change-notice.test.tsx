import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, cleanup, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useAirMasterChangeNotice } from './air-master-change-notice';

const getMaster = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: getMaster,
}));

const baseline: API.GetAirMasterWaybillResponse = {
  data: { id: 'master', version: '1' },
};

function setup() {
  const queryClient = createTestQueryClient();
  return renderHook(
    (props: {
      enabled: boolean;
      session: number;
      baseline: API.GetAirMasterWaybillResponse;
    }) => useAirMasterChangeNotice({ ...props, organizationId: 'org' }),
    {
      initialProps: { enabled: true, session: 1, baseline },
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    },
  );
}

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1);
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  getMaster.mockReset().mockResolvedValue(baseline);
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('航空主单前台变动检查', () => {
  it('15秒前台检查，隐藏暂停，恢复立即检查，失去当前页资格后不再请求', async () => {
    const view = setup();
    await advance(14_998);
    expect(getMaster).not.toHaveBeenCalled();
    await advance(1);
    expect(getMaster).toHaveBeenCalledTimes(1);
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await advance(45_000);
    expect(getMaster).toHaveBeenCalledTimes(1);
    getMaster.mockResolvedValue({ data: { id: 'master', version: '2' } });
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    act(() => document.dispatchEvent(new Event('visibilitychange')));
    await advance(0);
    expect(getMaster).toHaveBeenCalledTimes(2);
    expect(view.result.current).toBe(true);
    view.rerender({ enabled: false, session: 1, baseline });
    act(() => window.dispatchEvent(new Event('blur')));
    act(() => window.dispatchEvent(new Event('focus')));
    await advance(30_000);
    expect(getMaster).toHaveBeenCalledTimes(2);
  });

  it('检查失败仍保留提醒，仅新编辑基线成功采纳才清除', async () => {
    const view = setup();
    getMaster.mockResolvedValue({ data: { id: 'master', version: '2' } });
    await advance(15_000);
    expect(view.result.current).toBe(true);
    getMaster.mockResolvedValue(undefined);
    await advance(15_000);
    expect(view.result.current).toBe(true);
    view.rerender({
      enabled: true,
      session: 2,
      baseline: { data: { id: 'master', version: '2' } },
    });
    expect(view.result.current).toBe(false);
  });
});
