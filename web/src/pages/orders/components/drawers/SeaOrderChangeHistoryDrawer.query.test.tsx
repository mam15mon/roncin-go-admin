import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  SeaOrderChangeHistoryDrawer,
  SeaOrderChangeHistorySection,
} from './SeaOrderChangeHistoryDrawer';

const service = vi.hoisted(() => ({ list: vi.fn(), detail: vi.fn() }));
vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceListSeaOrderChangeEvents: service.list,
  seaOrderChangeServiceGetSeaOrderChangeEvent: service.detail,
}));
const row = {
  id: 'event-1',
  eventType: 'REASSIGNMENT',
  noteOrReason: '当前记录',
};
const drawer = (open: boolean) => (
  <SeaOrderChangeHistoryDrawer
    orderId="order-1"
    open={open}
    onClose={() => {}}
  />
);

beforeEach(() => vi.resetAllMocks());

describe('海运变更历史查询活动身份', () => {
  it('关闭取消列表请求，重开不采用忽略 signal 的旧请求', async () => {
    const old = Promise.withResolvers<{
      success: boolean;
      data: API.SeaOrderChangeEventSummary[];
      total: number;
    }>();
    service.list.mockReturnValueOnce(old.promise).mockResolvedValue({
      success: true,
      data: [row],
      total: 1,
    });
    const view = renderWithClient(drawer(true));
    await waitFor(() => expect(service.list).toHaveBeenCalledOnce());
    const signal = service.list.mock.calls[0][1].signal as AbortSignal;
    view.rerender(drawer(false));
    expect(signal.aborted).toBe(true);
    view.rerender(drawer(true));
    expect(await screen.findAllByText('当前记录')).toHaveLength(2);
    expect(service.list).toHaveBeenCalledTimes(2);
    await act(async () =>
      old.resolve({
        success: true,
        data: [{ ...row, noteOrReason: '旧请求内容' }],
        total: 1,
      }),
    );
    expect(screen.queryByText('旧请求内容')).not.toBeInTheDocument();
    expect(screen.getAllByText('当前记录')).toHaveLength(2);
  });

  it('关闭取消详情请求，重开详情由当前请求返回', async () => {
    const old = Promise.withResolvers<{
      success: boolean;
      data: API.SeaOrderChangeEventDetailData;
    }>();
    service.list.mockResolvedValue({ success: true, data: [row], total: 1 });
    service.detail.mockReturnValueOnce(old.promise).mockResolvedValue({
      success: true,
      data: { eventType: 'REASSIGNMENT', noteOrReason: '当前详情' },
    });
    const view = renderWithClient(drawer(true));
    fireEvent.click(await screen.findByRole('button', { name: '详情' }));
    await waitFor(() => expect(service.detail).toHaveBeenCalledOnce());
    const signal = service.detail.mock.calls[0][1].signal as AbortSignal;
    view.rerender(drawer(false));
    expect(signal.aborted).toBe(true);
    view.rerender(drawer(true));
    expect(await screen.findByText('当前详情')).toBeInTheDocument();
    await act(async () =>
      old.resolve({ success: true, data: { noteOrReason: '过期详情' } }),
    );
    expect(screen.queryByText('过期详情')).not.toBeInTheDocument();
  });

  it('预览首次失败不当成零条记录或上报零计数', async () => {
    service.list.mockResolvedValue(undefined);
    const onTotalChange = vi.fn();
    renderWithClient(
      <SeaOrderChangeHistorySection
        orderId="order-1"
        onOpenAll={() => {}}
        onTotalChange={onTotalChange}
      />,
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(
      '加载拆票与改配记录失败',
    );
    expect(screen.queryByText('暂无拆票或改配记录')).not.toBeInTheDocument();
    expect(onTotalChange).not.toHaveBeenCalledWith(0);
  });
});
