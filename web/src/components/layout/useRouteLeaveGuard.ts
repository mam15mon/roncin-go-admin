import { App } from 'antd';
import { useEffect, useRef } from 'react';
import { useBlocker } from 'react-router';
import { getTabGuardMessage, isTabDepartureConfirmed } from './tabCloseGuard';

/** 客商无持久草稿：真实路由离开复用页签确认，并保留取消后的表单。 */
export function useRouteLeaveGuard(tabKey: string, isDirty: () => boolean) {
  const { modal } = App.useApp();
  const dirtyRef = useRef(isDirty);
  dirtyRef.current = isDirty;
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      dirtyRef.current() &&
      !isTabDepartureConfirmed(tabKey) &&
      (currentLocation.pathname !== nextLocation.pathname ||
        currentLocation.search !== nextLocation.search),
  );
  const confirmationRef = useRef<ReturnType<typeof modal.confirm> | undefined>(
    undefined,
  );

  useEffect(() => {
    if (blocker.state !== 'blocked' || confirmationRef.current) return;
    confirmationRef.current = modal.confirm({
      title: '提示',
      content: getTabGuardMessage(tabKey),
      okText: '确定离开',
      cancelText: '取消',
      centered: true,
      onOk: () => {
        confirmationRef.current = undefined;
        blocker.proceed();
      },
      onCancel: () => {
        confirmationRef.current = undefined;
        blocker.reset();
      },
    });
  }, [blocker, modal, tabKey]);

  useEffect(() => () => confirmationRef.current?.destroy(), []);
}
