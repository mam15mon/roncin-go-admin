import type { ActionType } from '@ant-design/pro-components';
import { type RefObject, useEffect } from 'react';

export function useOrderAutoRefresh(
  enabled: boolean,
  actionRef: RefObject<ActionType | undefined>,
  busyRef: RefObject<boolean>,
) {
  useEffect(() => {
    if (!enabled) return;
    let timer: ReturnType<typeof setInterval> | undefined;
    let focused = document.hasFocus();
    const stop = () => {
      if (timer) clearInterval(timer);
      timer = undefined;
    };
    const synchronize = () => {
      stop();
      if (focused && document.visibilityState === 'visible') {
        timer = setInterval(() => {
          if (!busyRef.current) actionRef.current?.reload();
        }, 60_000);
      }
    };
    const onFocus = () => {
      focused = true;
      synchronize();
    };
    const onBlur = () => {
      focused = false;
      stop();
    };
    synchronize();
    window.addEventListener('focus', onFocus);
    window.addEventListener('blur', onBlur);
    document.addEventListener('visibilitychange', synchronize);
    return () => {
      stop();
      window.removeEventListener('focus', onFocus);
      window.removeEventListener('blur', onBlur);
      document.removeEventListener('visibilitychange', synchronize);
    };
  }, [enabled, actionRef, busyRef]);
}
