import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { StrictMode, useRef } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import { OrganizationWorkspace } from './OrganizationWorkspace';
import {
  _clearAllTabCloseGuards,
  registerTabCloseGuard,
  useTabCloseGuard,
} from './tabCloseGuard';

vi.mock('./TagsView', () => ({ TagsView: () => null }));

function Editor() {
  const dirty = useRef(false);
  useTabCloseGuard({
    tabKey: '/orders/sea-export',
    isDirty: () => dirty.current,
  });
  return (
    <>
      <input
        aria-label="未保存订单输入"
        onChange={() => {
          dirty.current = true;
        }}
      />
      <button
        type="button"
        onClick={() => {
          dirty.current = false;
        }}
      >
        保存成功
      </button>
    </>
  );
}

function mountWorkspace(strict = false) {
  function tree(userId = 'user-a', companyId = 'company-a') {
    const workspace = (
      <appInitialStateContext.Provider
        value={{
          initialState: {
            currentUser: {
              id: userId,
              currentOrganization: { id: companyId },
            },
          },
          setInitialState: vi.fn(),
          refresh: vi.fn(),
          loading: false,
        }}
      >
        <OrganizationWorkspace>
          <Editor />
        </OrganizationWorkspace>
      </appInitialStateContext.Provider>
    );
    return strict ? <StrictMode>{workspace}</StrictMode> : workspace;
  }
  const view = render(tree());
  return {
    ...view,
    changeIdentity: (userId: string, companyId: string) =>
      view.rerender(tree(userId, companyId)),
  };
}

function requestBrowserDeparture() {
  const event = new Event('beforeunload', { cancelable: true });
  const preventDefault = vi.spyOn(event, 'preventDefault');
  window.dispatchEvent(event);
  return { event, preventDefault };
}

function editOrder() {
  fireEvent.change(screen.getByLabelText('未保存订单输入'), {
    target: { value: '未保存编号' },
  });
}

afterEach(() => {
  cleanup();
  _clearAllTabCloseGuards();
  vi.restoreAllMocks();
});

describe('工作区浏览器离开保护', () => {
  it('初始化不阻断，真实编辑后立即阻断，保存清理后立即放行', () => {
    mountWorkspace();
    expect(requestBrowserDeparture().event.defaultPrevented).toBe(false);

    editOrder();
    expect(requestBrowserDeparture().event.defaultPrevented).toBe(true);

    fireEvent.click(screen.getByRole('button', { name: '保存成功' }));
    expect(requestBrowserDeparture().event.defaultPrevented).toBe(false);
  });

  it.each([
    ['user-a', 'company-b'],
    ['user-b', 'company-a'],
  ])('身份切至 %s/%s 后不继承旧输入或守卫', (userId, companyId) => {
    const view = mountWorkspace();
    editOrder();
    expect(requestBrowserDeparture().event.defaultPrevented).toBe(true);

    view.changeIdentity(userId, companyId);
    expect(screen.getByLabelText('未保存订单输入')).toHaveValue('');
    expect(requestBrowserDeparture().event.defaultPrevented).toBe(false);

    editOrder();
    expect(requestBrowserDeparture().preventDefault).toHaveBeenCalledOnce();
  });

  it('StrictMode 只保留一个监听，卸载后即使另有dirty守卫也不阻断', () => {
    const view = mountWorkspace(true);
    editOrder();
    expect(requestBrowserDeparture().preventDefault).toHaveBeenCalledOnce();

    view.unmount();
    const unregister = registerTabCloseGuard('outside-unmounted-workspace', {
      isDirty: () => true,
    });
    expect(requestBrowserDeparture().preventDefault).not.toHaveBeenCalled();
    unregister();
  });
});
