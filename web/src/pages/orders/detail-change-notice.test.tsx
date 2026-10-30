import type { QueryClient } from '@tanstack/react-query';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getFormDraft, hasFormDraft } from '@/components/layout/formDraft';
import {
  advanceVersion,
  bundle,
  draftKey,
  getOrderReadMock,
  getPortSaveState,
  renderDetail,
  resetPortSaveState,
} from './detail-port-save.fixtures';

const state = getPortSaveState();
const changedMessage = '这个订单已经发生变动，请重新刷新。';

async function click(name: string) {
  const button = screen
    .getByText(name, { normalizer: (value) => value.replace(/\s/g, '') })
    .closest('button');
  expect(button).toBeVisible();
  await act(async () => {
    fireEvent.click(button as HTMLButtonElement);
  });
}

async function confirmRefresh() {
  const title = await screen.findByText('刷新订单资料', {
    selector: '.ant-modal-confirm-title',
  });
  const dialog = title.closest('[role="dialog"]') as HTMLElement;
  const button = within(dialog)
    .getByText('确定刷新')
    .closest('button') as HTMLButtonElement;
  await waitFor(() => expect(button).toBeVisible());
  await act(async () => {
    fireEvent.click(button);
  });
}

async function exposeExternalChange(queryClient: QueryClient) {
  getOrderReadMock().mockResolvedValue({
    data: { ...state.order, version: '5' },
  });
  await act(async () => {
    await queryClient.refetchQueries({ queryKey: ['orders', 'change-notice'] });
  });
  expect(await screen.findByText(changedMessage)).toBeVisible();
}

beforeEach(() => {
  resetPortSaveState();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
});
afterEach(() => vi.restoreAllMocks());

describe('海运详情变动提醒与真实刷新确认', () => {
  it('轻量检查发现外部修改时持续提醒，并保留表单与原草稿版本', async () => {
    const view = renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    await exposeExternalChange(view.queryClient);
    expect(getOrderReadMock()).toHaveBeenCalledOnce();
    expect(state.loadData).not.toHaveBeenCalled();
    expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
    expect(getFormDraft(draftKey)).toMatchObject({
      version: '4',
      seaMasterBillExpectedCandidateVersion: '4',
      originLocationId: 'unsaved-port',
    });
    await act(async () => {
      await view.queryClient.refetchQueries({
        queryKey: ['orders', 'change-notice'],
      });
    });
    await waitFor(() => expect(getOrderReadMock()).toHaveBeenCalledTimes(2));
    expect(screen.getAllByText(changedMessage)).toHaveLength(1);
  });

  it('提醒和菜单刷新都先确认，取消不读取或丢弃输入', async () => {
    const view = renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    await exposeExternalChange(view.queryClient);
    for (const entry of ['刷新', '刷新数据']) {
      await click(entry);
      const title = await screen.findByText('刷新订单资料', {
        selector: '.ant-modal-confirm-title',
      });
      const dialog = title.closest('[role="dialog"]') as HTMLElement;
      await waitFor(() =>
        expect(
          within(dialog).getByText(
            '当前修改尚未保存，刷新将覆盖这些内容。确定刷新吗？',
          ),
        ).toBeVisible(),
      );
      const cancel = within(dialog)
        .getByText(/^取\s*消$/)
        .closest('button') as HTMLButtonElement;
      expect(cancel).toBeVisible();
      await act(async () => {
        fireEvent.click(cancel);
      });
      await waitFor(() => expect(dialog).not.toBeInTheDocument());
      expect(state.loadData).not.toHaveBeenCalled();
      expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
      expect(hasFormDraft(draftKey)).toBe(true);
      expect(screen.getByText(changedMessage)).toBeVisible();
    }
  });

  it('确认刷新后冻结输入，成功才重置完整快照并清除草稿和提醒', async () => {
    let finishRefresh!: (value: ReturnType<typeof bundle>) => void;
    state.loadData.mockImplementation(
      () =>
        new Promise<ReturnType<typeof bundle>>((resolve) => {
          finishRefresh = resolve;
        }),
    );
    const view = renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    await exposeExternalChange(view.queryClient);
    await click('刷新');
    await confirmRefresh();
    expect(state.loadData).toHaveBeenCalledOnce();
    expect(screen.getByLabelText('起运港')).toBeDisabled();
    expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
    expect(hasFormDraft(draftKey)).toBe(true);
    advanceVersion('5');
    state.order.originLocationId = 'latest-port';
    await act(async () => {
      finishRefresh(bundle());
    });
    await waitFor(() => {
      expect(screen.getByLabelText('起运港')).toHaveValue('latest-port');
      expect(screen.getByLabelText('起运港')).toBeEnabled();
    });
    expect(hasFormDraft(draftKey)).toBe(false);
    expect(screen.queryByText(changedMessage)).toBeNull();
  });

  it('刷新失败保留输入、草稿和提醒，并恢复编辑能力', async () => {
    state.loadData.mockRejectedValue(new Error('模拟订单读取失败'));
    const view = renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'unsaved-port' },
    });
    await exposeExternalChange(view.queryClient);
    await click('刷新数据');
    await confirmRefresh();
    const errorMessage = await screen.findByText(
      '刷新失败，输入已保留，请稍后重试。',
    );
    // Message 浮层的尺寸与动画由组件负责；业务持续提醒仍需真实可见。
    expect(errorMessage).toBeInTheDocument();
    expect(screen.getByLabelText('起运港')).toBeEnabled();
    expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
    expect(hasFormDraft(draftKey)).toBe(true);
    expect(screen.getByText(changedMessage)).toBeVisible();
    expect(state.loadData).toHaveBeenCalledOnce();
  });

  it('确认框打开后切换订单，旧确认不能读取或重置新订单', async () => {
    const { default: OrderDetailPage } = await import('./detail');
    const view = renderDetail();
    fireEvent.change(screen.getByLabelText('起运港'), {
      target: { value: 'order-one-draft' },
    });
    await click('刷新数据');
    const title = await screen.findByText('刷新订单资料', {
      selector: '.ant-modal-confirm-title',
    });
    const oldDialog = title.closest('[role="dialog"]') as HTMLElement;
    const oldConfirm = within(oldDialog)
      .getByText('确定刷新')
      .closest('button') as HTMLButtonElement;
    state.routeId = 'order-2';
    state.order = {
      ...state.order,
      id: 'order-2',
      orderNo: 'SE0002',
      originLocationId: 'order-two-port',
    };
    await act(async () => {
      view.rerender(
        <ConfigProvider theme={{ token: { motion: false } }}>
          <App>
            <OrderDetailPage />
          </App>
        </ConfigProvider>,
      );
    });
    await waitFor(() => expect(oldDialog).not.toBeInTheDocument());
    await act(async () => {
      fireEvent.click(oldConfirm);
    });
    expect(state.loadData).not.toHaveBeenCalled();
    expect(screen.getByLabelText('起运港')).toHaveValue('order-two-port');
    expect(hasFormDraft(draftKey)).toBe(true);
  });

  it('无未保存内容时手动刷新直接同步，无需弹出确认', async () => {
    renderDetail();
    advanceVersion('5');
    state.order.originLocationId = 'latest-port';
    await click('刷新数据');
    await waitFor(() => expect(state.loadData).toHaveBeenCalledOnce());
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(screen.getByLabelText('起运港')).toHaveValue('latest-port');
    expect(hasFormDraft(draftKey)).toBe(false);
    expect(screen.queryByText(changedMessage)).toBeNull();
  });
});
