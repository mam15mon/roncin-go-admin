import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App, Button } from 'antd';
import { StrictMode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import { useModalOwnerSignal } from '@/components/layout/useModalOwnerSignal';
import { confirmWithReason } from './confirmWithReason';
import { makeVersionActions } from './versionActions';

afterEach(_clearAllTabCloseGuards);

describe('原因确认框归属与未保存保护', () => {
  it('真实输入登记保护，失败保留，取消注销；StrictMode 的当前信号仍可打开', async () => {
    const submit = vi.fn().mockResolvedValue(false);
    const result = vi.fn();
    function Owner() {
      const app = App.useApp();
      const getSignal = useModalOwnerSignal();
      return (
        <Button
          onClick={() =>
            void confirmWithReason(app, '撤回账单', submit, {
              signal: getSignal(),
            }).then(result)
          }
        >
          填写原因
        </Button>
      );
    }
    renderWithApp(
      <StrictMode>
        <Owner />
      </StrictMode>,
    );
    fireEvent.click(screen.getByRole('button', { name: '填写原因' }));
    const input = await screen.findByPlaceholderText('请输入操作原因（必填）');
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.change(input, { target: { value: '需要重新确认' } });
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: '确 认' }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith('需要重新确认'));
    expect(input).toHaveValue('需要重新确认');
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: '取 消' }));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
    expect(result).toHaveBeenCalledWith(false);
  });

  it('业务组件卸载会销毁 App 中的旧组织确认框并注销守卫', async () => {
    const submit = vi.fn().mockResolvedValue(true);
    function Owner() {
      const app = App.useApp();
      const getSignal = useModalOwnerSignal();
      return (
        <Button
          onClick={() =>
            void confirmWithReason(app, '处理记录', submit, {
              signal: getSignal(),
            })
          }
        >
          填写原因
        </Button>
      );
    }
    const view = renderWithApp(<Owner />);
    fireEvent.click(screen.getByRole('button', { name: '填写原因' }));
    fireEvent.change(
      await screen.findByPlaceholderText('请输入操作原因（必填）'),
      { target: { value: '旧组织输入' } },
    );
    expect(getAnyDirtyTabKey()).toBeDefined();
    await act(async () => view.rerender(<div>新工作区</div>));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
    const closingDialog = screen.queryByRole('dialog');
    if (closingDialog) fireEvent.transitionEnd(closingDialog);
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    expect(submit).not.toHaveBeenCalled();
  });

  it('版本动作透传所属组件当前信号，卸载后旧原因守卫不残留且不能提交', async () => {
    const submit = vi.fn().mockResolvedValue(true);
    function Owner() {
      const app = App.useApp();
      const getSignal = useModalOwnerSignal();
      const actions = makeVersionActions(app, getSignal);
      return (
        <Button
          onClick={() =>
            actions.confirm({ id: 'bill-1', version: '2' }, '取消账单', submit)
          }
        >
          版本动作原因
        </Button>
      );
    }
    const view = renderWithApp(
      <StrictMode>
        <Owner />
      </StrictMode>,
    );
    fireEvent.click(screen.getByRole('button', { name: '版本动作原因' }));
    const input = await screen.findByPlaceholderText('请输入操作原因（必填）');
    fireEvent.change(input, { target: { value: '旧组织未提交原因' } });
    expect(getAnyDirtyTabKey()).toBeDefined();
    await act(async () => view.rerender(<div>新工作区</div>));
    await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
    expect(submit).not.toHaveBeenCalled();
  });
});
