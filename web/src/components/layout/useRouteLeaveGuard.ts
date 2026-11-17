import { App } from 'antd';
import { useEffect, useRef } from 'react';
import { useBlocker, useLocation } from 'react-router';
import { clearTabDrafts } from './formDraft';
import { getTabGuardMessage, isTabDepartureConfirmed } from './tabCloseGuard';

/**
 * 真实路由离开复用页签确认，取消时保留当前表单。
 * 传入 draftScope 的页面（新建订单等持久草稿）确认「放弃并离开」即清当前页签草稿，
 * 与关闭页签口径一致，避免下次进入恢复已放弃的输入。
 */
export function useRouteLeaveGuard(
  tabKey: string,
  isDirty: () => boolean,
  draftScope?: string,
) {
  const { modal } = App.useApp();
  const location = useLocation();
  const dirtyRef = useRef(isDirty);
  dirtyRef.current = isDirty;
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      // 路由提交后，旧组件的 effect 可能尚未清理；它不能阻断新页面的导航。
      currentLocation.key === location.key &&
      dirtyRef.current() &&
      !isTabDepartureConfirmed(tabKey) &&
      (currentLocation.pathname !== nextLocation.pathname ||
        currentLocation.search !== nextLocation.search),
  );
  const confirmationRef = useRef<ReturnType<typeof modal.confirm> | undefined>(
    undefined,
  );
  const abandonedRef = useRef(false);
  const scopeRef = useRef(draftScope);
  scopeRef.current = draftScope;

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
        abandonedRef.current = true;
        clearTabDrafts(tabKey, draftScope);
        blocker.proceed();
      },
      onCancel: () => {
        confirmationRef.current = undefined;
        blocker.reset();
      },
    });
  }, [blocker, modal, tabKey, draftScope]);

  // 路由过渡期间旧页面仍挂载，草稿快照可能在确认后再写入；卸载时以最终状态再清一次。
  useEffect(
    () => () => {
      if (abandonedRef.current) clearTabDrafts(tabKey, scopeRef.current);
    },
    [tabKey],
  );

  useEffect(() => () => confirmationRef.current?.destroy(), []);
}
