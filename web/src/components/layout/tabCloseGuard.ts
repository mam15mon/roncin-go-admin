import { Modal } from 'antd';
import { useEffect, useRef } from 'react';
import { getAppFeedback } from '@/utils/appFeedback';
import { hasTabDraft } from './formDraft';

export type TabCloseGuard = {
  isDirty: () => boolean;
  message?: string;
};

const guards = new Map<string, TabCloseGuard>();

/**
 * 注册页签关闭守卫。当页签试图被关闭时，如果 isDirty() 为 true，则会拦截并弹出确认对话框。
 */
export function registerTabCloseGuard(
  tabKey: string,
  guard: TabCloseGuard,
): () => void {
  guards.set(tabKey, guard);
  return () => {
    if (guards.get(tabKey) === guard) {
      guards.delete(tabKey);
    }
  };
}

/**
 * 检查指定 tabKey 是否处于已编辑未保存（dirty）状态
 * 同时检查实时状态与持久草稿，避免表单状态更新前的瞬间关闭漏掉已写入的草稿。
 */
export function isTabDirty(tabKey: string, draftScope?: string): boolean {
  const guard = guards.get(tabKey);
  return Boolean(guard?.isDirty()) || hasTabDraft(tabKey, draftScope);
}

/**
 * 获取指定 tabKey 注册的提示文案，默认返回标准文案
 */
export function getTabGuardMessage(tabKey: string): string {
  const guard = guards.get(tabKey);
  return guard?.message || '修改的信息尚未保存，您确定要离开吗？';
}

/**
 * 检查要关闭的 tabKeys 列表中是否存在未保存的脏数据。
 * 如果存在，弹出确认框；用户确认后执行 onConfirm，取消则不执行。
 * 如果不存在，直接执行 onConfirm。
 */
export function confirmIfTabsDirty(
  tabKeys: string[],
  onConfirm: () => void,
  customConfirmModal?: (props: Parameters<typeof Modal.confirm>[0]) => void,
  draftScope?: string,
): boolean {
  const dirtyTabKey = tabKeys.find((key) => isTabDirty(key, draftScope));
  if (!dirtyTabKey) {
    onConfirm();
    return true;
  }

  const message = getTabGuardMessage(dirtyTabKey);
  const confirmFn =
    customConfirmModal || getAppFeedback().modal?.confirm || Modal.confirm;

  confirmFn({
    title: '提示',
    content: message,
    okText: '确定离开',
    cancelText: '取消',
    centered: true,
    onOk: () => {
      onConfirm();
    },
  });

  return false;
}

/**
 * 供组件便捷声明关闭守卫的 React Hook
 */
export function useTabCloseGuard(options: {
  tabKey?: string;
  isDirty: boolean | (() => boolean);
  message?: string;
  enabled?: boolean;
}) {
  const { tabKey, isDirty, message, enabled = true } = options;
  const isDirtyRef = useRef(isDirty);
  isDirtyRef.current = isDirty;
  const messageRef = useRef(message);
  messageRef.current = message;

  useEffect(() => {
    if (!enabled || !tabKey) return;

    return registerTabCloseGuard(tabKey, {
      isDirty: () => {
        const check = isDirtyRef.current;
        return typeof check === 'function' ? check() : check;
      },
      message: messageRef.current,
    });
  }, [tabKey, enabled]);
}

/**
 * 清除所有页签关闭守卫（主要用于测试环境清理）
 */
export function _clearAllTabCloseGuards(): void {
  guards.clear();
}
