import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OrderListTemplate } from './OrderListTemplate';
import type { OrderListAcceptedSnapshot, OrderListFilterParams } from './types';

vi.mock('./OrderListSearchFilter', () => ({
  default: ({
    onSearch,
  }: {
    onSearch: (filters: OrderListFilterParams, labels: object) => void;
  }) => (
    <button
      type="button"
      onClick={() => onSearch({ numberKeyword: '新筛选' }, {})}
    >
      提交新筛选
    </button>
  ),
}));

type Page = {
  data: { id: string; orderNo: string }[];
  total: number;
  success?: boolean;
};
const page = (name: string, total = 250): Page => ({
  data: [{ id: name, orderNo: name }],
  total,
});

function setup(request = vi.fn().mockResolvedValue(page('已采纳'))) {
  const actionRef: { current: ActionType | undefined } = { current: undefined };
  const exported =
    vi.fn<
      (snapshot: OrderListAcceptedSnapshot, current: () => boolean) => void
    >();
  const view = render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <OrderListTemplate
        actionRef={actionRef}
        customColumns={[
          { title: '订单号', dataIndex: 'orderNo', sorter: true },
        ]}
        queryOrders={request}
        onExportTable={exported}
      />
    </ConfigProvider>,
  );
  return { ...view, request, actionRef, exported };
}

const exportButton = () =>
  screen.getByRole('button', { name: /导出完整筛选结果/ });

afterEach(cleanup);

describe('订单完整导出的官方采纳快照', () => {
  it('成功空列表也经官方onLoad采纳，未采纳时不能导出', async () => {
    let finish!: (value: Page) => void;
    const pending = new Promise<Page>((resolve) => {
      finish = resolve;
    });
    const { exported, request } = setup(vi.fn().mockReturnValue(pending));
    expect(exportButton()).toBeDisabled();
    await waitFor(() => expect(request).toHaveBeenCalledOnce());
    await act(async () => finish({ data: [], total: 0, success: true }));
    await waitFor(() => expect(exportButton()).toBeEnabled());
    fireEvent.click(exportButton());
    expect(exported).toHaveBeenCalledWith(
      expect.objectContaining({ total: 0, filters: {} }),
      expect.any(Function),
    );
    expect(exported.mock.calls[0][1]()).toBe(true);
  });

  it('提交筛选立即撤销原快照与在途导出资格，新查询失败不能采纳空结果', async () => {
    let fail!: (value: Page) => void;
    const pending = new Promise<Page>((resolve) => {
      fail = resolve;
    });
    const request = vi
      .fn()
      .mockResolvedValueOnce(page('原结果'))
      .mockReturnValue(pending);
    const { exported } = setup(request);
    await screen.findByText('原结果');
    fireEvent.click(exportButton());
    const isCurrent = exported.mock.calls[0][1];
    expect(isCurrent()).toBe(true);
    fireEvent.click(screen.getByRole('button', { name: /展开筛选/ }));
    fireEvent.click(screen.getByRole('button', { name: '提交新筛选' }));
    expect(exportButton()).toBeDisabled();
    expect(isCurrent()).toBe(false);
    await waitFor(() => expect(request).toHaveBeenCalledTimes(2));
    await act(async () => fail({ data: [], total: 0, success: false }));
    expect(exportButton()).toBeDisabled();
  });

  it('新筛选先返回、旧查询迟到时只导出官方已采纳的新筛选与总数', async () => {
    let finishOld!: (value: Page) => void;
    const old = new Promise<Page>((resolve) => {
      finishOld = resolve;
    });
    const request = vi
      .fn()
      .mockReturnValueOnce(old)
      .mockResolvedValue(page('新结果', 251));
    const { exported } = setup(request);
    await waitFor(() => expect(request).toHaveBeenCalledOnce());
    fireEvent.click(screen.getByRole('button', { name: /展开筛选/ }));
    fireEvent.click(screen.getByRole('button', { name: '提交新筛选' }));
    await screen.findByText('新结果');
    await act(async () => finishOld(page('迟到旧结果', 900)));
    expect(screen.queryByText('迟到旧结果')).not.toBeInTheDocument();
    await waitFor(() => expect(exportButton()).toBeEnabled());
    fireEvent.click(exportButton());
    expect(exported.mock.calls[0][0]).toMatchObject({
      filters: { numberKeyword: '新筛选' },
      total: 251,
    });
    expect(exported.mock.calls[0][0].filters).not.toHaveProperty('page');
    expect(exported.mock.calls[0][0].filters).not.toHaveProperty('pageSize');
  });

  it('排序立即禁旧导出，采纳排序后保留相同排序；卸载后旧导出失效', async () => {
    let finish!: (value: Page) => void;
    const pending = new Promise<Page>((resolve) => {
      finish = resolve;
    });
    const request = vi
      .fn()
      .mockResolvedValueOnce(page('初始'))
      .mockReturnValue(pending);
    const { exported, unmount } = setup(request);
    await screen.findByText('初始');
    fireEvent.click(exportButton());
    const prior = exported.mock.calls[0][1];
    fireEvent.click(screen.getByRole('columnheader', { name: /订单号/ }));
    expect(exportButton()).toBeDisabled();
    expect(prior()).toBe(false);
    await waitFor(() => expect(request).toHaveBeenCalledTimes(2));
    await act(async () => finish(page('升序')));
    await waitFor(() => expect(exportButton()).toBeEnabled());
    fireEvent.click(exportButton());
    expect(exported.mock.calls[1][0].filters).toMatchObject({
      sorterField: 'orderNo',
      sorterOrder: 'ascend',
    });
    const current = exported.mock.calls[1][1];
    unmount();
    expect(current()).toBe(false);
  });
});
