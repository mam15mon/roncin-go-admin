import { cleanup, renderHook } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import { appInitialStateContext } from '@/app/AppProvider';
import { saveColumnSettingsPreference } from './preference';
import { useColumnSettings } from './useColumnSettings';

const columns = [
  { key: 'name', title: '姓名' },
  { key: 'email', title: '邮箱' },
  { key: 'action', title: '操作', fixed: 'right' as const },
];

function authUser(id: string, organizationId: string) {
  return { id, currentOrganization: { id: organizationId } } as API.CurrentUser;
}

describe('列设置认证作用域', () => {
  afterEach(() => {
    cleanup();
    window.localStorage.clear();
  });

  it('读取真实认证用户/组织，切换时当前帧不暴露原偏好', () => {
    saveColumnSettingsPreference(
      'test:scope',
      { userId: 'u1', organizationId: 'o1' },
      {
        order: ['email', 'name'],
        hidden: ['name'],
        freeze: { left: ['email'], right: [] },
      },
    );
    saveColumnSettingsPreference(
      'test:scope',
      { userId: 'u1', organizationId: 'o2' },
      {
        order: ['name', 'email'],
        hidden: ['email'],
        freeze: { left: [], right: ['action'] },
      },
    );
    let currentUser = authUser('u1', 'o1');
    const renders: string[][] = [];
    const { result, rerender } = renderHook(
      () => {
        const settings = useColumnSettings<(typeof columns)[number]>({
          tableKey: 'test:scope',
          columns,
        });
        renders.push(settings.columns.map((column) => column.key));
        return settings;
      },
      {
        wrapper: ({ children }) => (
          <App>
            <appInitialStateContext.Provider
              value={{
                initialState: { currentUser },
                setInitialState: () => {},
                refresh: () => {},
                loading: false,
              }}
            >
              {children}
            </appInitialStateContext.Provider>
          </App>
        ),
      },
    );
    expect(result.current.columns.map((column) => column.key)).toEqual([
      'email',
      'action',
    ]);
    renders.length = 0;
    currentUser = authUser('u1', 'o2');
    rerender();
    expect(
      renders.every(
        (keys) => JSON.stringify(keys) === JSON.stringify(['name', 'action']),
      ),
    ).toBe(true);
    expect(result.current.columns.at(-1)?.fixed).toBe('right');
    renders.length = 0;
    currentUser = authUser('u2', 'o2');
    rerender();
    expect(
      renders.every(
        (keys) =>
          JSON.stringify(keys) === JSON.stringify(['name', 'email', 'action']),
      ),
    ).toBe(true);
  });

  it('显式scope优先于认证上下文，表格标识切换立即读独立配置', () => {
    const scope = { userId: 'explicit', organizationId: 'explicit-org' };
    saveColumnSettingsPreference('test:one', scope, {
      order: ['email', 'name'],
      hidden: ['name'],
      freeze: { left: ['email'], right: [] },
    });
    const { result, rerender } = renderHook(
      ({ tableKey }) =>
        useColumnSettings<(typeof columns)[number]>({
          tableKey,
          columns,
          scope,
        }),
      {
        initialProps: { tableKey: 'test:one' },
        wrapper: ({ children }) => (
          <App>
            <appInitialStateContext.Provider
              value={{
                initialState: { currentUser: authUser('auth', 'auth-org') },
                setInitialState: () => {},
                refresh: () => {},
                loading: false,
              }}
            >
              {children}
            </appInitialStateContext.Provider>
          </App>
        ),
      },
    );
    expect(result.current.columns.map((column) => column.key)).toEqual([
      'email',
      'action',
    ]);
    expect(result.current.columns.at(-1)?.fixed).toBe(false);
    rerender({ tableKey: 'test:two' });
    expect(result.current.columns.map((column) => column.key)).toEqual([
      'name',
      'email',
      'action',
    ]);
    expect(result.current.columns.at(-1)?.fixed).toBe('right');
  });
});
