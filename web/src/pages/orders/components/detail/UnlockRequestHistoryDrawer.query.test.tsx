import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import UnlockRequestHistoryDrawer from './UnlockRequestHistoryDrawer';

const list = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/orderLockService', () => ({
  orderLockServiceListOrderUnlockRequests: list,
}));

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  list.mockReset();
});
describe('撤销审核历史查询', () => {
  it('只在前台轮询，隐藏停止，恢复立即检查并在关闭后停止', async () => {
    vi.useFakeTimers();
    let visible = true;
    vi.spyOn(document, 'hasFocus').mockReturnValue(true);
    vi.spyOn(document, 'visibilityState', 'get').mockImplementation(() =>
      visible ? 'visible' : 'hidden',
    );
    list.mockResolvedValue({
      success: true,
      data: { items: [{ id: 'request-1', status: 'PENDING_APPROVAL' }] },
    });
    const view = renderWithClient(
      <UnlockRequestHistoryDrawer open orderId="order-1" onClose={() => {}} />,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5);
    });
    expect(list).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(list).toHaveBeenCalledTimes(2);
    await act(async () => {
      visible = false;
      document.dispatchEvent(new Event('visibilitychange'));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(list).toHaveBeenCalledTimes(2);
    await act(async () => {
      visible = true;
      document.dispatchEvent(new Event('visibilitychange'));
      await vi.advanceTimersByTimeAsync(5);
    });
    expect(list).toHaveBeenCalledTimes(3);
    view.rerender(
      <UnlockRequestHistoryDrawer
        open={false}
        orderId="order-1"
        onClose={() => {}}
      />,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(list).toHaveBeenCalledTimes(3);
    view.unmount();
  });

  it('关闭取消在途审批查询，重开不复用旧会话结果', async () => {
    const old = Promise.withResolvers<{
      success: boolean;
      data: { items: API.OrderUnlockRequestData[] };
    }>();
    list
      .mockReturnValueOnce(old.promise)
      .mockResolvedValue({ success: true, data: { items: [] } });
    const view = renderWithClient(
      <UnlockRequestHistoryDrawer open orderId="order-1" onClose={() => {}} />,
    );
    await waitFor(() => expect(list).toHaveBeenCalledOnce());
    const signal = list.mock.calls[0][1].signal as AbortSignal;
    view.rerender(
      <UnlockRequestHistoryDrawer
        open={false}
        orderId="order-1"
        onClose={() => {}}
      />,
    );
    expect(signal.aborted).toBe(true);
    view.rerender(
      <UnlockRequestHistoryDrawer open orderId="order-1" onClose={() => {}} />,
    );
    expect(await screen.findByText('暂无撤销审核申请')).toBeInTheDocument();
    expect(list).toHaveBeenCalledTimes(2);
    await act(async () =>
      old.resolve({
        success: true,
        data: {
          items: [
            {
              id: 'old-request',
              reason: '过期审批原因',
              status: 'PENDING_APPROVAL',
            },
          ],
        },
      }),
    );
    expect(screen.queryByText('过期审批原因')).not.toBeInTheDocument();
    expect(screen.getByText('暂无撤销审核申请')).toBeInTheDocument();
  });

  it('undefined 响应保留失败展示，不当成无申请', async () => {
    list.mockResolvedValue(undefined);
    renderWithClient(
      <UnlockRequestHistoryDrawer open orderId="order-1" onClose={() => {}} />,
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(
      '加载撤销审核审批记录失败',
    );
    expect(screen.queryByText('暂无撤销审核申请')).not.toBeInTheDocument();
  });
});
