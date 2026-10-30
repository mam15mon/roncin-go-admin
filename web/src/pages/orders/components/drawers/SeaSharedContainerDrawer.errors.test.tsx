import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import * as service from '@/services/roncin/seaSharedContainerService';
import SeaSharedContainerDrawer from './SeaSharedContainerDrawer';

const list = vi.spyOn(
  service,
  'seaSharedContainerServiceListSeaSharedContainers',
);
const candidates = vi.spyOn(
  service,
  'seaSharedContainerServiceListSeaSharedContainerCandidates',
);
const save = vi.spyOn(
  service,
  'seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft',
);
const container: API.SeaSharedContainer = {
  id: 'shared-1',
  containerNo: 'TEST1234567',
  transportExecutionId: 'te-1',
  packageCount: 100,
  grossWeightKg: '100',
  volumeCbm: '10',
  version: '1',
  status: 1,
  allocations: [],
};
const candidate: API.SeaSharedContainerCandidateOrder = {
  orderId: 'order-1',
  orderNo: 'SE001',
  houseBillId: 'hbl-1',
  orderVersion: '1',
  houseBillVersion: '1',
  linkVersion: '1',
  cargoItems: [
    {
      id: 'cargo-1',
      cargoName: '测试货物',
      packageCount: 100,
      grossWeightKg: '100',
      volumeCbm: '10',
      version: '1',
    },
  ],
};
function setup() {
  const client = createTestQueryClient();
  const ui = (open: boolean, orderId = 'order-1') => (
    <QueryClientProvider client={client}>
      <App>
        <SeaSharedContainerDrawer
          open={open}
          orderId={orderId}
          transportExecutionId="te-1"
          onClose={() => {}}
          canCreate
          canUpdate
        />
      </App>
    </QueryClientProvider>
  );
  const view = render(ui(true));
  return { ...view, ui, client };
}
beforeEach(() => {
  vi.resetAllMocks();
  list.mockResolvedValue({ data: [container] } as never);
  candidates.mockResolvedValue({ data: [candidate], total: 1 } as never);
  save.mockResolvedValue({ data: container } as never);
});

it.each([
  ['403', Object.assign(new Error('forbidden'), { response: { status: 403 } })],
  [
    'business403',
    Object.assign(new Error('forbidden'), { data: { code: 403 } }),
  ],
  ['5xx', Object.assign(new Error('server'), { response: { status: 500 } })],
  ['network', new Error('network')],
  ['undefined', undefined],
] as const)(
  '共享箱查询 %s 显示失败而非 0 空态，重试后才可创建',
  async (_kind, failure) => {
    if (failure) list.mockRejectedValueOnce(failure);
    else list.mockResolvedValueOnce(undefined as never);
    setup();
    await screen.findByRole('button', { name: '重试共享箱' });
    expect(
      screen.queryByText('当前航次暂无共享物理箱'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/本航次共享物理箱 \(0\)/),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('新建共享物理箱')).not.toBeInTheDocument();
    expect(screen.queryByText('立即创建共享箱')).not.toBeInTheDocument();
    if (_kind === '403' || _kind === 'business403')
      expect(
        screen.getByText('无权读取共享箱，请联系管理员核对权限'),
      ).toBeInTheDocument();
    expect(list).toHaveBeenCalledWith(
      expect.objectContaining({ orderId: 'order-1' }),
      { skipErrorHandler: true },
    );
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: '重试共享箱' })),
    );
    await screen.findByText('新建共享物理箱');
    expect(screen.getAllByText('TEST1234567').length).toBeGreaterThan(0);
  },
);

it.each([[], undefined])(
  '成功列表 data=%j 按 ProtoJSON 空集合语义显示正常空态',
  async (data) => {
    list.mockResolvedValue({ success: true, data } as never);
    candidates.mockResolvedValue({ success: true, data, total: 0 } as never);
    setup();
    await screen.findByText('当前航次暂无共享物理箱');
    expect(screen.getByText('立即创建共享箱')).toBeInTheDocument();
    expect(screen.queryByText('重试共享箱')).not.toBeInTheDocument();
  },
);

it('候选分页失败及重试不清空已读共享箱与人工分配草稿', async () => {
  candidates
    .mockResolvedValueOnce({ data: [candidate], total: 21 } as never)
    .mockRejectedValueOnce(new Error('network'))
    .mockResolvedValue({ data: [candidate], total: 21 } as never);
  setup();
  await screen.findByText('测试货物');
  fireEvent.change(screen.getAllByRole('spinbutton')[0], {
    target: { value: '37' },
  });
  await act(async () => fireEvent.click(screen.getByTitle('2')));
  await screen.findByRole(
    'button',
    { name: '重试候选订单' },
    { timeout: 5000 },
  );
  expect(screen.getAllByText('TEST1234567').length).toBeGreaterThan(0);
  expect(screen.queryByText('共 0 票订单')).not.toBeInTheDocument();
  await act(async () =>
    fireEvent.click(screen.getByRole('button', { name: '重试候选订单' })),
  );
  await screen.findByText('测试货物');
  expect(screen.getAllByRole('spinbutton')[0]).toHaveValue('37');
  fireEvent.click(screen.getByText('保存草稿'));
  await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
  expect(save.mock.calls[0][1].allocations?.[0].packageCount).toBe(37);
  await waitFor(() => expect(list).toHaveBeenCalledTimes(2));
});

it('关闭后迟到查询不污染重新打开的工作台', async () => {
  let finish: (response: unknown) => void = () => {};
  list.mockReturnValueOnce(
    new Promise((resolve) => {
      finish = resolve;
    }) as never,
  );
  const view = setup();
  view.rerender(view.ui(false));
  view.rerender(view.ui(true));
  await waitFor(() =>
    expect(screen.getAllByText('TEST1234567').length).toBeGreaterThan(0),
  );
  await act(async () =>
    finish({ data: [{ ...container, containerNo: 'STALE1234567' }] }),
  );
  expect(screen.queryByText('STALE1234567')).not.toBeInTheDocument();
});
