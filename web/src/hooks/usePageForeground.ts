import { useEffect, useRef, useState } from 'react';

/** 页面可见且窗口聚焦时才视为前台；恢复前台只通知一次，不持有业务数据。 */
export function usePageForeground(onResume?: () => void): boolean {
  const onResumeRef = useRef(onResume);
  onResumeRef.current = onResume;
  const [foreground, setForeground] = useState(
    () => document.visibilityState === 'visible' && document.hasFocus(),
  );
  useEffect(() => {
    let focused = document.hasFocus();
    let active = document.visibilityState === 'visible' && focused;
    setForeground(active);
    const synchronize = () => {
      const next = document.visibilityState === 'visible' && focused;
      if (next === active) return;
      active = next;
      setForeground(next);
      if (next) onResumeRef.current?.();
    };
    const onFocus = () => {
      focused = true;
      synchronize();
    };
    const onBlur = () => {
      focused = false;
      synchronize();
    };
    const onVisibility = () => {
      focused = document.hasFocus();
      synchronize();
    };
    window.addEventListener('focus', onFocus);
    window.addEventListener('blur', onBlur);
    document.addEventListener('visibilitychange', onVisibility);
    return () => {
      window.removeEventListener('focus', onFocus);
      window.removeEventListener('blur', onBlur);
      document.removeEventListener('visibilitychange', onVisibility);
    };
  }, []);
  return foreground;
}
