import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { aeIdentity, awbFixture } from './model.fixtures';
import { fetchAwbBundle, useAwbChangeNotice } from './use-awb-data';

const state = vi.hoisted(() => ({
  order: {} as API.Order,
  read: vi.fn(),
  readMaster: vi.fn(),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: state.read,
}));
vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: state.readMaster,
}));

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  state.order = awbFixture().order as API.Order;
  state.read.mockImplementation(async () => ({
    success: true,
    data: structuredClone(state.order),
  }));
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});
function mountNotice() {
  const client = createTestQueryClient();
  const baseline = awbFixture();
  const hook = renderHook(
    ({ enabled, session }) =>
      useAwbChangeNotice(baseline, aeIdentity, session, enabled),
    {
      initialProps: { enabled: true, session: 0 },
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    },
  );
  return { ...hook, client };
}
async function tick(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
}

describe('航空制作变动探测与读取隔离', () => {
  it('前台15秒检查，发现变化仅持续提示，不替换打开基线', async () => {
    const hook = mountNotice();
    await tick(14_999);
    expect(state.read).not.toHaveBeenCalled();
    state.order.version = '5';
    await tick(2);
    expect(state.read).toHaveBeenCalledOnce();
    expect(hook.result.current.changed).toBe(true);
    state.order.version = '4';
    await tick(15_001);
    expect(hook.result.current.changed).toBe(true);
    hook.unmount();
    await tick(30_000);
    expect(state.read).toHaveBeenCalledTimes(2);
  });
  it('窗口blur和浏览器hidden暂停；恢复前台立即复核', async () => {
    mountNotice();
    fireEvent.blur(window);
    await tick(30_000);
    expect(state.read).not.toHaveBeenCalled();
    fireEvent.focus(window);
    await tick(1);
    expect(state.read).toHaveBeenCalledOnce();
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    fireEvent(document, new Event('visibilitychange'));
    await tick(30_000);
    expect(state.read).toHaveBeenCalledTimes(1);
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    fireEvent(document, new Event('visibilitychange'));
    await tick(1);
    expect(state.read).toHaveBeenCalledTimes(2);
  });
  it('离页enabled=false不轮询也不回焦请求，显式新会话独立', async () => {
    const hook = mountNotice();
    hook.rerender({ enabled: false, session: 0 });
    await tick(30_000);
    fireEvent.blur(window);
    fireEvent.focus(window);
    await tick(1);
    expect(state.read).not.toHaveBeenCalled();
    hook.rerender({ enabled: true, session: 1 });
    await tick(15_001);
    expect(state.read).toHaveBeenCalledOnce();
  });
  it('生成前验证同时比较公司抬头，不只比较Master.version', async () => {
    const hook = mountNotice();
    if (!state.order.airExport?.master) throw new Error('测试主单缺失');
    state.order.airExport.master.organizationNameEn = 'NEW REAL COMPANY';
    let verified = true;
    await act(async () => {
      verified = await hook.result.current.verifyCurrent();
    });
    await tick(1);
    expect(verified).toBe(false);
    expect(hook.result.current.changed).toBe(true);
  });
  it('错误wire公司缺失明确失败；错误读取不被当成保存快照', async () => {
    delete state.order.organizationId;
    await expect(fetchAwbBundle(aeIdentity)).rejects.toThrow('公司');
    const hook = mountNotice();
    let error: unknown;
    await act(async () => {
      try {
        await hook.result.current.verifyCurrent();
      } catch (caught) {
        error = caught;
      }
    });
    await tick(1);
    expect(error).toBeInstanceOf(Error);
    expect(hook.result.current.checkError).toBe(true);
  });
  it('A→B→A卸载重建后旧在途响应不会标记新实例', async () => {
    let finish: ((response: API.GetOrderResponse) => void) | undefined;
    state.read.mockImplementationOnce(
      () =>
        new Promise<API.GetOrderResponse>((resolve) => {
          finish = resolve;
        }),
    );
    const first = mountNotice();
    await tick(15_001);
    first.unmount();
    const second = mountNotice();
    const late = structuredClone(state.order);
    late.version = '99';
    await act(async () => finish?.({ success: true, data: late }));
    await tick(1);
    expect(second.result.current.changed).toBe(false);
  });
});
