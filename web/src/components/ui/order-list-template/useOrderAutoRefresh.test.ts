import type { ActionType } from '@ant-design/pro-components';
import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useOrderAutoRefresh } from './useOrderAutoRefresh';

let visibility: DocumentVisibilityState;
beforeEach(() => {
  vi.useFakeTimers();
  visibility = 'visible';
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockImplementation(
    () => visibility,
  );
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});
function mount(enabled = true) {
  const reload = vi.fn();
  const actionRef = {
    current: { reload } as unknown as ActionType | undefined,
  };
  const busyRef = { current: false };
  const result = renderHook(
    ({ active }) => useOrderAutoRefresh(active, actionRef, busyRef),
    { initialProps: { active: enabled } },
  );
  return { ...result, reload, actionRef, busyRef };
}
function tick(ms = 60_000) {
  act(() => vi.advanceTimersByTime(ms));
}
function event(target: Window | Document, type: string) {
  act(() => target.dispatchEvent(new Event(type)));
}

describe('订单前台自动刷新', () => {
  it('关闭时无计时器，开启60秒刷新且忙碌tick跳过', () => {
    const view = mount(false);
    tick(120_000);
    expect(view.reload).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
    view.rerender({ active: true });
    tick(59_999);
    expect(view.reload).not.toHaveBeenCalled();
    tick(1);
    expect(view.reload).toHaveBeenCalledTimes(1);
    view.busyRef.current = true;
    tick();
    expect(view.reload).toHaveBeenCalledTimes(1);
    view.busyRef.current = false;
    tick();
    expect(view.reload).toHaveBeenCalledTimes(2);
    view.rerender({ active: false });
    expect(vi.getTimerCount()).toBe(0);
    tick();
    expect(view.reload).toHaveBeenCalledTimes(2);
    view.unmount();
  });

  it('blur及隐藏停止，恢复前台仅建立一个计时器且不立即请求', () => {
    const view = mount();
    event(window, 'blur');
    expect(vi.getTimerCount()).toBe(0);
    tick(120_000);
    expect(view.reload).not.toHaveBeenCalled();
    event(window, 'focus');
    event(window, 'focus');
    expect(vi.getTimerCount()).toBe(1);
    expect(view.reload).not.toHaveBeenCalled();
    tick();
    expect(view.reload).toHaveBeenCalledTimes(1);
    visibility = 'hidden';
    event(document, 'visibilitychange');
    expect(vi.getTimerCount()).toBe(0);
    tick();
    expect(view.reload).toHaveBeenCalledTimes(1);
    visibility = 'visible';
    event(document, 'visibilitychange');
    event(document, 'visibilitychange');
    expect(vi.getTimerCount()).toBe(1);
    tick();
    expect(view.reload).toHaveBeenCalledTimes(2);
    view.unmount();
    expect(vi.getTimerCount()).toBe(0);
    event(window, 'focus');
    event(document, 'visibilitychange');
    tick();
    expect(view.reload).toHaveBeenCalledTimes(2);
  });

  it('初始失焦或隐藏不建立轮询，action稍后挂载可刷新', () => {
    vi.mocked(document.hasFocus).mockReturnValue(false);
    const view = mount();
    expect(vi.getTimerCount()).toBe(0);
    visibility = 'hidden';
    event(window, 'focus');
    expect(vi.getTimerCount()).toBe(0);
    view.actionRef.current = undefined;
    visibility = 'visible';
    event(document, 'visibilitychange');
    tick();
    expect(view.reload).not.toHaveBeenCalled();
    view.actionRef.current = { reload: view.reload } as unknown as ActionType;
    tick();
    expect(view.reload).toHaveBeenCalledTimes(1);
    view.unmount();
  });
});
