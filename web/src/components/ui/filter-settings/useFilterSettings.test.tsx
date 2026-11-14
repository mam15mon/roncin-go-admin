import { act, cleanup, renderHook } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import { useFilterSettings } from './useFilterSettings';

const errorSpy = vi.hoisted(() => vi.fn());
vi.mock('antd', async (orig) => {
  const mod = await orig<typeof import('antd')>();
  return {
    ...mod,
    App: Object.assign(mod.App, {
      useApp: () => ({ message: { error: errorSpy } }),
    }),
  };
});

const fields = [
  { name: 'a', label: 'A', defaultVisible: true },
  { name: 'b', label: 'B', defaultVisible: true },
];
function wrapperFor(userId?: string, orgId?: string) {
  return ({ children }: { children: React.ReactNode }) => (
    <App>
      <appInitialStateContext.Provider
        value={
          {
            initialState: {
              currentUser: userId
                ? { id: userId, currentOrganization: { id: orgId } }
                : undefined,
            },
          } as never
        }
      >
        {children}
      </appInitialStateContext.Provider>
    </App>
  );
}

describe('useFilterSettings', () => {
  const spies: { mockRestore: () => void }[] = [];
  const track = <T extends { mockRestore: () => void }>(spy: T) => {
    spies.push(spy);
    return spy;
  };
  afterEach(() => {
    for (const spy of spies.splice(0)) spy.mockRestore();
    cleanup();
    window.localStorage.clear();
    vi.restoreAllMocks();
    errorSpy.mockClear();
  });

  it('设置持久化并在同用户公司页面恢复，其他用户/公司/页面不串', () => {
    const first = renderHook(
      () => useFilterSettings({ pageKey: 'p', fields }),
      { wrapper: wrapperFor('u1', 'o1') },
    );
    act(() => first.result.current.setVisible(['a']));
    expect(first.result.current.visibleNames).toEqual(['a']);
    const same = renderHook(() => useFilterSettings({ pageKey: 'p', fields }), {
      wrapper: wrapperFor('u1', 'o1'),
    });
    expect(same.result.current.visibleNames).toEqual(['a']);
    for (const [pageKey, u, o] of [
      ['q', 'u1', 'o1'],
      ['p', 'u2', 'o1'],
      ['p', 'u1', 'o2'],
    ]) {
      const other = renderHook(() => useFilterSettings({ pageKey, fields }), {
        wrapper: wrapperFor(u, o),
      });
      expect(other.result.current.visibleNames).toEqual(['a', 'b']);
    }
    act(() => same.result.current.reset());
    expect(same.result.current.visibleNames).toEqual(['a', 'b']);
  });

  it('存储失败只提示一次且当次设置仍生效', () => {
    track(vi.spyOn(window.localStorage, 'setItem')).mockImplementation(() => {
      throw new Error('denied');
    });
    const { result } = renderHook(
      () => useFilterSettings({ pageKey: 'p', fields }),
      { wrapper: wrapperFor('u1', 'o1') },
    );
    act(() => result.current.setVisible(['b']));
    act(() => result.current.setVisible(['a', 'b']));
    act(() => result.current.setVisible(['a']));
    expect(errorSpy).toHaveBeenCalledTimes(1);
    expect(result.current.visibleNames).toEqual(['a']);
    expect(result.current.persistFailed).toBe(true);
  });

  it('身份未就绪不写存储也不提示，仅内存生效', () => {
    const set = track(vi.spyOn(window.localStorage, 'setItem'));
    set.mockClear();
    const { result } = renderHook(
      () => useFilterSettings({ pageKey: 'p', fields }),
      { wrapper: wrapperFor() },
    );
    act(() => result.current.setVisible(['a']));
    expect(result.current.visibleNames).toEqual(['a']);
    expect(set).not.toHaveBeenCalled();
    expect(errorSpy).not.toHaveBeenCalled();
  });
});
