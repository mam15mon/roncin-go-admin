import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { _clearAllTabCloseGuards, isTabDirty } from './tabCloseGuard';
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard';

afterEach(_clearAllTabCloseGuards);

describe('useUnsavedChangesGuard', () => {
  it('仅用户动作同步标脏，保存清理，关闭和换记录不残留', () => {
    const { result, rerender, unmount } = renderHook(
      ({ enabled, identity }) =>
        useUnsavedChangesGuard({ enabled, identity, tabKey: '/admin' }),
      { initialProps: { enabled: true, identity: 'role-1' } },
    );
    expect(isTabDirty('/admin')).toBe(false);
    act(() => result.current.markDirty());
    expect(isTabDirty('/admin')).toBe(true);
    rerender({ enabled: true, identity: 'role-2' });
    expect(isTabDirty('/admin')).toBe(false);
    act(() => result.current.markDirty());
    act(() => result.current.resetDirty());
    expect(isTabDirty('/admin')).toBe(false);
    act(() => result.current.markDirty());
    rerender({ enabled: false, identity: 'role-2' });
    expect(isTabDirty('/admin')).toBe(false);
    act(() => result.current.markDirty());
    rerender({ enabled: true, identity: 'role-2' });
    expect(isTabDirty('/admin')).toBe(false);
    act(() => result.current.markDirty());
    unmount();
    expect(isTabDirty('/admin')).toBe(false);
  });
});
