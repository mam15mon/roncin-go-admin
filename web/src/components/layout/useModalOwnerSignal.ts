import { useCallback, useLayoutEffect, useRef } from 'react';

/** 事件打开的 App modal 归属于当前业务组件；卸载时销毁，StrictMode 重挂生成新信号。 */
export function useModalOwnerSignal() {
  const controller = useRef<AbortController | undefined>(undefined);
  useLayoutEffect(() => {
    const current = new AbortController();
    controller.current = current;
    return () => current.abort();
  }, []);
  return useCallback(() => {
    if (!controller.current) throw new Error('确认框所属页面尚未挂载');
    return controller.current.signal;
  }, []);
}
