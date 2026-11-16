import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  flush,
  order,
  renderPanel,
  response,
  services,
  setupTests,
} from './ImportHandlingPanel.fixtures';

setupTests();
async function openEditor() {
  const open = await screen.findByRole('button', {
    name: '维护承接与完成情况',
  });
  await waitFor(() => expect(open).toBeEnabled());
  fireEvent.click(open);
  return screen.findByRole('dialog');
}

describe('进口服务编辑基线与输入保护', () => {
  it('独立读取发现更晚版本时持续提示并保留输入，刷新取消或失败不替换旧稿', async () => {
    const view = renderPanel();
    const dialog = await openEditor();
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '承接清关' }));
    const notes = await within(dialog).findByRole('textbox', {
      name: '清关备注',
    });
    fireEvent.change(notes, { target: { value: '用户尚未保存内容' } });
    services.get.mockResolvedValue(response('8'));
    await act(async () => {
      await view.queryClient.invalidateQueries({
        queryKey: ['orders', 'import-handling'],
      });
    });
    await waitFor(() =>
      expect(
        within(dialog).getByText('这个订单已经发生变动，请重新刷新。'),
      ).toBeVisible(),
    );
    expect(
      within(dialog).getByRole('button', { name: '保存服务资料' }),
    ).toBeDisabled();
    expect(notes).toHaveValue('用户尚未保存内容');
    fireEvent.click(within(dialog).getByRole('button', { name: '重新读取' }));
    const confirm = await screen.findAllByText('重新读取服务资料？');
    expect(confirm.length).toBeGreaterThan(0);
    const dialogs = screen.getAllByRole('dialog');
    fireEvent.click(
      within(dialogs[dialogs.length - 1]).getByRole('button', {
        name: '取 消',
      }),
    );
    await flush();
    expect(notes).toHaveValue('用户尚未保存内容');
    services.get.mockResolvedValue(undefined);
    await act(async () => {
      await view.queryClient.invalidateQueries({
        queryKey: ['orders', 'import-handling'],
      });
    });
    expect(
      within(dialog).getByText('这个订单已经发生变动，请重新刷新。'),
    ).toBeVisible();
    expect(notes).toHaveValue('用户尚未保存内容');
  });

  it('保存等待期间继续输入时保留新稿，迟到成功不关闭或给旧稿推进版本', async () => {
    let finish!: (value: API.GetImportHandlingResponse) => void;
    services.save.mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    const view = renderPanel();
    const dialog = await openEditor();
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '承接提货' }));
    const notes = await within(dialog).findByRole('textbox', {
      name: '提货备注',
    });
    fireEvent.change(notes, { target: { value: '本次保存内容' } });
    fireEvent.click(
      within(dialog).getByRole('button', { name: '保存服务资料' }),
    );
    await waitFor(() => expect(services.save).toHaveBeenCalledTimes(1));
    fireEvent.change(notes, { target: { value: '提交后的新输入' } });
    const saved = response('8');
    services.get.mockResolvedValue(saved);
    await act(async () => finish(saved));
    await waitFor(() => expect(view.onSaved).toHaveBeenCalledTimes(1));
    expect(notes).toHaveValue('提交后的新输入');
    expect(dialog).toBeVisible();
    expect(
      within(dialog).getByRole('button', { name: '保存服务资料' }),
    ).toBeDisabled();
  });

  it('父聚合重读等待期间重新读取新服务稿，旧保存回调不得关闭新稿', async () => {
    let finishParent!: () => void;
    const onSaved = () =>
      new Promise<void>((resolve) => {
        finishParent = resolve;
      });
    const view = renderPanel({ onSaved });
    const dialog = await openEditor();
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '承接清关' }));
    services.save.mockResolvedValue(response('8'));
    fireEvent.click(
      within(dialog).getByRole('button', { name: '保存服务资料' }),
    );
    await waitFor(() => expect(finishParent).toBeTypeOf('function'));
    services.get.mockResolvedValue(response('8'));
    view.rerender(view.ui('8'));
    await within(dialog).findByText('这个订单已经发生变动，请重新刷新。');
    fireEvent.click(within(dialog).getByRole('button', { name: '重新读取' }));
    const confirmation = screen
      .getAllByRole('dialog')
      .find(
        (item) => within(item).queryAllByText('重新读取服务资料？').length > 0,
      );
    expect(confirmation).toBeDefined();
    fireEvent.click(
      within(confirmation as HTMLElement).getByRole('button', {
        name: '确 定',
      }),
    );
    await waitFor(() => expect(dialog).not.toBeInTheDocument());
    const next = await waitFor(() => {
      const current = screen
        .getAllByRole('dialog')
        .find((item) =>
          within(item).queryByRole('checkbox', { name: '承接提货' }),
        );
      expect(current).toBeDefined();
      return current as HTMLElement;
    });
    fireEvent.click(within(next).getByRole('checkbox', { name: '承接提货' }));
    const notes = await within(next).findByRole('textbox', {
      name: '提货备注',
    });
    fireEvent.change(notes, { target: { value: '新会话不可被旧保存关闭' } });
    await act(async () => finishParent());
    expect(next).toBeVisible();
    expect(notes).toHaveValue('新会话不可被旧保存关闭');
  });

  it('当前公司不符或缺读取权限时不请求，父提交基线变化不自动借新服务版本', async () => {
    const denied = renderPanel({
      order: { ...order, organizationId: 'other-company' },
    });
    await flush();
    expect(services.get).not.toHaveBeenCalled();
    denied.unmount();
    const view = renderPanel();
    await openEditor();
    view.rerender(view.ui('8'));
    const currentDialog = await screen.findByRole('dialog');
    await waitFor(() =>
      expect(
        within(currentDialog).getByText('这个订单已经发生变动，请重新刷新。'),
      ).toBeVisible(),
    );
    expect(
      within(currentDialog).getByRole('button', { name: '保存服务资料' }),
    ).toBeDisabled();
    expect(services.save).not.toHaveBeenCalled();
  });
});
