import { useCallback, useLayoutEffect, useRef } from 'react';
import { history } from '@/router/history';
import { resolveTabKey } from './routeUtils';
import { useTabCloseGuard } from './tabCloseGuard';

/** 仅由用户编辑事件标记；初始化、候选回填和失败请求不改变未保存状态。 */
export function useUnsavedChangesGuard(
  options: {
    enabled?: boolean;
    identity?: unknown;
    tabKey?: string;
    message?: string;
  } = {},
) {
  const { enabled = true, identity, message } = options;
  const tabKey = options.tabKey ?? resolveTabKey(history.location.pathname);
  const dirty = useRef(false);
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  const markDirty = useCallback(() => {
    if (enabledRef.current) dirty.current = true;
  }, []);
  const resetDirty = useCallback(() => {
    dirty.current = false;
  }, []);

  useLayoutEffect(() => {
    resetDirty();
  }, [enabled, identity, tabKey, resetDirty]);

  useTabCloseGuard({
    enabled,
    tabKey,
    isDirty: () => dirty.current,
    message,
  });
  return { markDirty, resetDirty };
}
