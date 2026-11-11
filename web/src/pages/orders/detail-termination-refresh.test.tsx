import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { isTabDirty } from '@/components/layout/tabCloseGuard';
import { OrderAllowedAction, OrderTerminationStatus } from '@/enums.generated';
import {
  bundle,
  getOrderReadMock,
  getPortSaveState,
  renderDetail,
  resetPortSaveState,
} from './detail-port-save.fixtures';
import { confirmOrderTermination } from './order-detail-transitions';

vi.mock('./order-detail-transitions', () => ({
  confirmOrderTermination: vi.fn().mockReturnValue(true),
  confirmOrderFlow: vi.fn(),
  confirmOrderClosure: vi.fn(),
}));
const state = getPortSaveState();
const changedMessage = '这个订单已经发生变动，请重新刷新。';

beforeEach(() => {
  resetPortSaveState();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  state.order = {
    ...state.order,
    terminationStatus:
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
    allowedActions: [
      OrderAllowedAction.ORDER_ALLOWED_ACTION_COMPLETE_TERMINATION,
    ],
  };
});
afterEach(() => vi.restoreAllMocks());

async function openCompletion() {
  fireEvent.click(screen.getByRole('button', { name: '选择退关操作' }));
  fireEvent.click(await screen.findByRole('menuitem', { name: '完成退关' }));
  return vi.mocked(confirmOrderTermination).mock.calls[0][3];
}
function completedOrder(): API.Order {
  return {
    ...state.order,
    version: '5',
    terminationStatus:
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
    seaMasterBill: undefined,
    seaDocumentLinkVersion: undefined,
    seaDocumentStructure: undefined,
    allowedActions: [],
  };
}

it('完成退关后显示终态，不把本人操作当外部变更，并保留输入与原提交版本', async () => {
  const view = renderDetail();
  fireEvent.change(screen.getByLabelText('起运港'), {
    target: { value: 'unsaved-port' },
  });
  const onCompleted = await openCompletion();
  const written = completedOrder();
  state.loadData.mockImplementation(async () => {
    state.order = written;
    return bundle();
  });
  await act(async () => {
    await onCompleted(written);
  });
  expect(screen.getByText('已退关')).toBeVisible();
  expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
  expect(state.form?.getFieldsValue(true)).toMatchObject({
    version: '4',
    originLocationId: 'unsaved-port',
  });
  expect(screen.queryByText(changedMessage)).toBeNull();
  await act(async () => {
    await view.queryClient.refetchQueries({
      queryKey: ['orders', 'change-notice'],
    });
  });
  expect(screen.queryByText(changedMessage)).toBeNull();
  getOrderReadMock().mockResolvedValue({ data: { ...written, version: '6' } });
  await act(async () => {
    await view.queryClient.refetchQueries({
      queryKey: ['orders', 'change-notice'],
    });
  });
  expect(await screen.findByText(changedMessage)).toBeVisible();
  expect(state.form?.getFieldsValue(true)).toMatchObject({ version: '4' });
  expect(isTabDirty('/orders/sea-export/order-1')).toBe(true);
});

it('完成退关后重读失败保留输入、未保存状态和原有变动提醒', async () => {
  const view = renderDetail();
  fireEvent.change(screen.getByLabelText('起运港'), {
    target: { value: 'unsaved-port' },
  });
  getOrderReadMock().mockResolvedValue({
    data: { ...state.order, version: '5' },
  });
  await act(async () => {
    await view.queryClient.refetchQueries({
      queryKey: ['orders', 'change-notice'],
    });
  });
  expect(await screen.findByText(changedMessage)).toBeVisible();
  const onCompleted = await openCompletion();
  state.loadData.mockRejectedValueOnce(new Error('模拟单证读取失败'));
  await act(async () => {
    await onCompleted(completedOrder());
  });
  await waitFor(() =>
    expect(
      screen.getByText('状态已更新，但最新数据刷新失败，请手动刷新'),
    ).toBeInTheDocument(),
  );
  expect(screen.getByLabelText('起运港')).toHaveValue('unsaved-port');
  expect(state.form?.getFieldsValue(true)).toMatchObject({
    version: '4',
    originLocationId: 'unsaved-port',
  });
  expect(screen.getByText(changedMessage)).toBeVisible();
});

it('终态重新打开后不因保留主单正文产生伪变动提示', async () => {
  state.order = completedOrder();
  const view = renderDetail();
  expect(screen.getByText('已退关')).toBeVisible();
  expect(screen.queryByText(changedMessage)).toBeNull();
  await act(async () => {
    await view.queryClient.refetchQueries({
      queryKey: ['orders', 'change-notice'],
    });
  });
  expect(screen.queryByText(changedMessage)).toBeNull();
});

it('完成退关写入先被轮询发现，整包刷新完成后清除本人误报', async () => {
  const view = renderDetail();
  const onCompleted = await openCompletion();
  const written = completedOrder();
  const pending = Promise.withResolvers<ReturnType<typeof bundle>>();
  state.loadData.mockReturnValueOnce(pending.promise);
  let completion!: Promise<void>;
  await act(async () => {
    completion = Promise.resolve(onCompleted(written));
  });
  getOrderReadMock().mockResolvedValue({ data: written });
  await act(async () => {
    await view.queryClient.refetchQueries({
      queryKey: ['orders', 'change-notice'],
    });
  });
  expect(await screen.findByText(changedMessage)).toBeVisible();
  state.order = written;
  await act(async () => {
    pending.resolve(bundle());
    await completion;
  });
  expect(screen.queryByText(changedMessage)).toBeNull();
  expect(screen.getByText('已退关')).toBeVisible();
});

it('状态刷新已读到更晚外部版本时立即显示提醒', async () => {
  renderDetail();
  const onCompleted = await openCompletion();
  const written = completedOrder();
  state.loadData.mockImplementation(async () => {
    state.order = { ...written, version: '6' };
    return bundle();
  });
  await act(async () => {
    await onCompleted(written);
  });
  expect(screen.getByText(changedMessage)).toBeVisible();
});

it('确认成功回调晚于订单切换时不发起新订单刷新', async () => {
  const { default: OrderDetailPage } = await import('./detail');
  const view = renderDetail();
  const onCompleted = await openCompletion();
  const written = completedOrder();
  state.routeId = 'order-2';
  state.order = { ...state.order, id: 'order-2', orderNo: 'SE0002' };
  await act(async () => {
    view.rerender(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <OrderDetailPage />
        </App>
      </ConfigProvider>,
    );
  });
  await act(async () => {
    await onCompleted(written);
  });
  expect(state.loadData).not.toHaveBeenCalled();
  expect(screen.queryByText(changedMessage)).toBeNull();
});

it('状态刷新期间切换订单，旧刷新失败不向新订单弹出警告', async () => {
  const { default: OrderDetailPage } = await import('./detail');
  const view = renderDetail();
  const onCompleted = await openCompletion();
  const written = completedOrder();
  const pending = Promise.withResolvers<ReturnType<typeof bundle>>();
  state.loadData.mockReturnValueOnce(pending.promise);
  let completion!: Promise<void>;
  await act(async () => {
    completion = Promise.resolve(onCompleted(written));
  });
  expect(state.loadData).toHaveBeenCalledOnce();
  state.routeId = 'order-2';
  state.order = { ...state.order, id: 'order-2', orderNo: 'SE0002' };
  await act(async () => {
    view.rerender(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <OrderDetailPage />
        </App>
      </ConfigProvider>,
    );
  });
  await act(async () => {
    pending.reject(new Error('旧订单刷新失败'));
    await completion;
  });
  expect(
    screen.queryByText('状态已更新，但最新数据刷新失败，请手动刷新'),
  ).toBeNull();
  expect(screen.queryByText(changedMessage)).toBeNull();
});
