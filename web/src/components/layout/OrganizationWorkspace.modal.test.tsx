import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import { OrganizationWorkspace } from './OrganizationWorkspace';

vi.mock('./TagsView', () => ({ TagsView: () => null }));

function ConfirmAction({
  title,
  onConfirm,
}: {
  title: string;
  onConfirm: () => void;
}) {
  const { modal } = App.useApp();
  return (
    <button
      type="button"
      onClick={() =>
        modal.confirm({ title, onOk: onConfirm, okText: '执行操作' })
      }
    >
      打开{title}
    </button>
  );
}

function mount() {
  const businessAction = vi.fn();
  const shellAction = vi.fn();
  function tree(organizationId: string) {
    return (
      <App>
        <ConfirmAction title="组织切换确认" onConfirm={shellAction} />
        <appInitialStateContext.Provider
          value={{
            initialState: {
              currentUser: {
                id: 'admin',
                currentOrganization: { id: organizationId },
              },
            },
            setInitialState: vi.fn(),
            refresh: vi.fn(),
            loading: false,
          }}
        >
          <OrganizationWorkspace>
            <ConfirmAction
              title="原组织业务删除确认"
              onConfirm={businessAction}
            />
          </OrganizationWorkspace>
        </appInitialStateContext.Provider>
      </App>
    );
  }
  const view = render(tree('beijing'));
  return {
    ...view,
    businessAction,
    shellAction,
    change: (id: string) => view.rerender(tree(id)),
  };
}

afterEach(cleanup);

describe('组织工作区的业务确认框所有权', () => {
  it('重建工作区销毁普通业务确认框，根组织确认框保持有效', async () => {
    const view = mount();
    fireEvent.click(
      screen.getByRole('button', { name: '打开原组织业务删除确认' }),
    );
    await waitFor(() =>
      expect(
        screen.getByText('原组织业务删除确认', {
          selector: '.ant-modal-confirm-title',
        }),
      ).toBeInTheDocument(),
    );
    fireEvent.click(screen.getByRole('button', { name: '打开组织切换确认' }));
    await waitFor(() =>
      expect(
        screen.getByText('组织切换确认', {
          selector: '.ant-modal-confirm-title',
        }),
      ).toBeInTheDocument(),
    );
    view.change('chengdu');
    await waitFor(() =>
      expect(
        screen.queryByText('原组织业务删除确认', {
          selector: '.ant-modal-confirm-title',
        }),
      ).not.toBeInTheDocument(),
    );
    expect(view.businessAction).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '执行操作' }));
    await waitFor(() => expect(view.shellAction).toHaveBeenCalledOnce());
    expect(view.businessAction).not.toHaveBeenCalled();
  });

  it('身份不变时原业务确认框继续可用，失败切换不提前销毁', async () => {
    const view = mount();
    fireEvent.click(
      screen.getByRole('button', { name: '打开原组织业务删除确认' }),
    );
    await waitFor(() =>
      expect(
        screen.getByText('原组织业务删除确认', {
          selector: '.ant-modal-confirm-title',
        }),
      ).toBeInTheDocument(),
    );
    view.change('beijing');
    expect(
      screen.getByText('原组织业务删除确认', {
        selector: '.ant-modal-confirm-title',
      }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '执行操作' }));
    await waitFor(() => expect(view.businessAction).toHaveBeenCalledOnce());
    expect(view.shellAction).not.toHaveBeenCalled();
  });
});
