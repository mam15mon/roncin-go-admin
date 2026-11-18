import { App } from 'antd';
import { useEffect, useRef } from 'react';
import { useBlocker, useLocation } from 'react-router';
import { resolveTabKey } from './routeUtils';
import {
  getTabGuardMessage,
  hasUnrecoverableTabChanges,
  isTabDepartureConfirmed,
} from './tabCloseGuard';

/** 工作区只为不能恢复的临时输入展示离开确认。 */
export function WorkspaceRouteLeavePrompt() {
  const location = useLocation();
  const tabKey = resolveTabKey(location.pathname);
  useRouteLeaveGuard(tabKey, () => hasUnrecoverableTabChanges(tabKey));
  return null;
}

export function useRouteLeaveGuard(tabKey: string, isDirty: () => boolean) {
  const { modal } = App.useApp();
  const location = useLocation();
  const dirtyRef = useRef(isDirty);
  dirtyRef.current = isDirty;
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      // 路由提交后，旧组件的 effect 可能尚未清理；它不能阻断新页面的导航。
      currentLocation.key === location.key &&
      nextLocation.pathname !== '/user/login' &&
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
      okText: '放弃并离开',
      cancelText: '继续编辑',
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
