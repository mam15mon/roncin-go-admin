import { renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getFormDraftKey, getFormDraftScope } from './formDraft';
import {
  _clearAllTabCloseGuards,
  confirmIfAnyTabDirty,
  confirmIfTabsDirty,
  getAnyDirtyTabKey,
  getTabGuardMessage,
  isTabDirty,
  registerTabCloseGuard,
  useTabCloseGuard,
} from './tabCloseGuard';

describe('tabCloseGuard', () => {
  beforeEach(() => {
    _clearAllTabCloseGuards();
    sessionStorage.clear();
  });

  afterEach(() => {
    _clearAllTabCloseGuards();
    sessionStorage.clear();
  });

  it('未注册守卫的 tab 默认为非 dirty', () => {
    expect(isTabDirty('/orders/sea-export')).toBe(false);
  });

  it('注册守卫后可正确报告 dirty 状态与反注册', () => {
    let dirty = false;
    const unregister = registerTabCloseGuard('/orders/sea-export', {
      isDirty: () => dirty,
    });

    expect(isTabDirty('/orders/sea-export')).toBe(false);

    dirty = true;
    expect(isTabDirty('/orders/sea-export')).toBe(true);

    unregister();
    expect(isTabDirty('/orders/sea-export')).toBe(false);
  });

  it('未挂载但存在草稿的 tab 依然判定为 dirty', () => {
    const tabKey = '/orders/sea-export';
    const draftScope = getFormDraftScope('user-1', 'org-1');
    expect(isTabDirty(tabKey, draftScope)).toBe(false);

    sessionStorage.setItem(
      getFormDraftKey(tabKey, '/orders/sea-export/new', draftScope),
      JSON.stringify({ name: 'test' }),
    );
    expect(isTabDirty(tabKey, draftScope)).toBe(true);

    sessionStorage.clear();
    expect(isTabDirty(tabKey, draftScope)).toBe(false);
  });

  it('实时守卫尚未更新时，已写入的草稿仍能阻止页签关闭', () => {
    const tabKey = '/orders/sea-export';
    const draftScope = getFormDraftScope('user-1', 'org-1');
    registerTabCloseGuard(tabKey, { isDirty: () => false });
    sessionStorage.setItem(
      getFormDraftKey(tabKey, '/orders/sea-export/new', draftScope),
      JSON.stringify({ name: 'test' }),
    );

    expect(isTabDirty(tabKey, draftScope)).toBe(true);
  });

  it('返回默认提示文案，或自定义文案', () => {
    expect(getTabGuardMessage('/not-exist')).toBe(
      '修改的信息尚未保存，您确定要离开吗？',
    );

    registerTabCloseGuard('/custom-tab', {
      isDirty: () => true,
      message: '自定义离开提示',
    });

    expect(getTabGuardMessage('/custom-tab')).toBe('自定义离开提示');
  });

  it('只从已注册守卫中找到任意 dirty 表单，不扫描持久化草稿', () => {
    const draftScope = getFormDraftScope('user-1', 'org-1');
    sessionStorage.setItem(
      getFormDraftKey(
        '/orders/sea-export',
        '/orders/sea-export/new',
        draftScope,
      ),
      JSON.stringify({ name: 'draft only' }),
    );
    expect(getAnyDirtyTabKey()).toBeUndefined();

    registerTabCloseGuard('/partners/customers', {
      isDirty: () => true,
    });
    expect(getAnyDirtyTabKey()).toBe('/partners/customers');
  });

  describe('confirmIfAnyTabDirty', () => {
    it('没有已注册的 dirty 表单时立即切换', () => {
      const onConfirm = vi.fn();
      const modal = vi.fn();

      const result = confirmIfAnyTabDirty(onConfirm, modal);

      expect(result).toBe(true);
      expect(onConfirm).toHaveBeenCalledTimes(1);
      expect(modal).not.toHaveBeenCalled();
    });

    it('有 dirty 表单时由确认框控制继续或取消', () => {
      registerTabCloseGuard('/orders/sea-export', {
        isDirty: () => true,
        message: '订单尚未保存',
      });
      const onConfirm = vi.fn();
      const onCancel = vi.fn();
      const modal = vi.fn();

      const result = confirmIfAnyTabDirty(onConfirm, modal, onCancel);

      expect(result).toBe(false);
      expect(onConfirm).not.toHaveBeenCalled();
      expect(modal).toHaveBeenCalledTimes(1);
      const modalProps = modal.mock.calls[0][0];
      expect(modalProps.content).toBe('订单尚未保存');

      modalProps.onCancel();
      expect(onCancel).toHaveBeenCalledTimes(1);
      expect(onConfirm).not.toHaveBeenCalled();

      modalProps.onOk();
      expect(onConfirm).toHaveBeenCalledTimes(1);
    });
  });

  describe('confirmIfTabsDirty', () => {
    it('当没有任何 dirty tab 时，直接执行 onConfirm 并返回 true', () => {
      const onConfirm = vi.fn();
      const mockModal = vi.fn();

      const result = confirmIfTabsDirty(
        ['/orders/sea-export'],
        onConfirm,
        mockModal,
      );

      expect(result).toBe(true);
      expect(onConfirm).toHaveBeenCalledTimes(1);
      expect(mockModal).not.toHaveBeenCalled();
    });

    it('当存在 dirty tab 时，弹出 Modal.confirm 且暂不执行 onConfirm', () => {
      registerTabCloseGuard('/orders/sea-export', {
        isDirty: () => true,
      });

      const onConfirm = vi.fn();
      const mockModal = vi.fn();

      const result = confirmIfTabsDirty(
        ['/welcome', '/orders/sea-export'],
        onConfirm,
        mockModal,
      );

      expect(result).toBe(false);
      expect(onConfirm).not.toHaveBeenCalled();
      expect(mockModal).toHaveBeenCalledTimes(1);

      const callArg = mockModal.mock.calls[0][0];
      expect(callArg.title).toBe('提示');
      expect(callArg.content).toBe('修改的信息尚未保存，您确定要离开吗？');
      expect(callArg.okText).toBe('确定离开');
      expect(callArg.cancelText).toBe('取消');

      // 模拟用户点击「确定离开」
      callArg.onOk();
      expect(onConfirm).toHaveBeenCalledTimes(1);
    });
  });

  describe('useTabCloseGuard', () => {
    it('组件挂载时注册守卫，卸载时注销守卫', () => {
      let dirty = true;
      const { unmount, rerender } = renderHook(
        ({ isDirty }) =>
          useTabCloseGuard({
            tabKey: '/orders/sea-export',
            isDirty,
          }),
        {
          initialProps: { isDirty: () => dirty },
        },
      );

      expect(isTabDirty('/orders/sea-export')).toBe(true);

      dirty = false;
      rerender({ isDirty: () => dirty });
      expect(isTabDirty('/orders/sea-export')).toBe(false);

      unmount();
      expect(isTabDirty('/orders/sea-export')).toBe(false);
    });

    it('当 enabled 为 false 时不注册守卫', () => {
      renderHook(() =>
        useTabCloseGuard({
          tabKey: '/orders/sea-export',
          isDirty: true,
          enabled: false,
        }),
      );

      expect(isTabDirty('/orders/sea-export')).toBe(false);
    });
  });
});
