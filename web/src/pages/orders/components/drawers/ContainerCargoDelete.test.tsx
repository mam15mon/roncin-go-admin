import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React, { createRef } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CargoItemDrawer, { type CargoItemDrawerRef } from './CargoItemDrawer';
import ContainerDrawer, { type ContainerDrawerRef } from './ContainerDrawer';

const containerService = vi.hoisted(() => ({
  list: vi.fn(),
  remove: vi.fn(),
}));
const cargoService = vi.hoisted(() => ({
  list: vi.fn(),
  remove: vi.fn(),
}));

vi.mock('@/services/roncin/orderContainerService', () => ({
  orderContainerServiceAddContainer: vi.fn(),
  orderContainerServiceListContainers: containerService.list,
  orderContainerServiceRemoveContainer: containerService.remove,
  orderContainerServiceUpdateContainer: vi.fn(),
}));

vi.mock('@/services/roncin/orderCargoItemService', () => ({
  orderCargoItemServiceAddCargoItem: vi.fn(),
  orderCargoItemServiceListCargoItems: cargoService.list,
  orderCargoItemServiceRemoveCargoItem: cargoService.remove,
  orderCargoItemServiceUpdateCargoItem: vi.fn(),
}));

async function confirmDelete() {
  fireEvent.click(screen.getByRole('button', { name: '删除' }));
  await waitFor(() => {
    expect(screen.getByText(/确定移除/)).toBeInTheDocument();
  });
  const confirmButtons = screen.getAllByRole('button', { name: '确 定' });
  // 点击确认后，onConfirm 异步链（message 提示、列表刷新、弹层关闭）必须
  // 在 act 作用域内落地，否则静态 message 独立根与 ProTable 刷新会迟到更新。
  await act(async () => {
    fireEvent.click(confirmButtons.at(-1) as HTMLElement);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
}

// 删除确认后的收尾流（message 提示、列表刷新、弹层关闭）为异步链，
// 在 act 内冲刷微任务，确保用例结束前全部落地，避免迟到 setState 触发 act 警告。
async function flushDeleteAftermath() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe('箱货删除版本', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    containerService.remove.mockResolvedValue({ success: true });
    cargoService.remove.mockResolvedValue({ success: true });
  });

  it('集装箱删除发送记录真实版本', async () => {
    containerService.list.mockResolvedValue({
      data: [{ id: 'container-1', containerNo: 'TGHU1234567', version: '7' }],
    });
    const ref = createRef<ContainerDrawerRef>();
    render(
      <App>
        <ContainerDrawer
          ref={ref}
          canCreate={false}
          canUpdate={false}
          canRemove
          containerSpecOptions={[]}
          containerSpecMap={{}}
        />
      </App>,
    );
    await act(async () => {
      ref.current?.open({ id: 'order-1', orderNo: 'SE001' });
    });
    await screen.findByText('TGHU1234567');

    await confirmDelete();

    await waitFor(() => {
      expect(containerService.remove).toHaveBeenCalledWith({
        orderId: 'order-1',
        id: 'container-1',
        expectedVersion: '7',
      });
    });
    await flushDeleteAftermath();
  });

  it('货物删除发送记录真实版本', async () => {
    cargoService.list.mockResolvedValue({
      data: [{ id: 'cargo-1', cargoName: '精密机械', version: '9' }],
    });
    const ref = createRef<CargoItemDrawerRef>();
    render(
      <App>
        <CargoItemDrawer
          ref={ref}
          canCreate={false}
          canUpdate={false}
          canRemove
        />
      </App>,
    );
    await act(async () => {
      ref.current?.open({ id: 'order-2', orderNo: 'SE002' });
    });
    await screen.findByText('精密机械');

    await confirmDelete();

    await waitFor(() => {
      expect(cargoService.remove).toHaveBeenCalledWith({
        orderId: 'order-2',
        id: 'cargo-1',
        expectedVersion: '9',
      });
    });
    await flushDeleteAftermath();
  });

  it('版本缺失或为零时不调用删除接口', async () => {
    containerService.list.mockResolvedValue({
      data: [{ id: 'container-1', containerNo: 'TGHU0000000' }],
    });
    const ref = createRef<ContainerDrawerRef>();
    render(
      <App>
        <ContainerDrawer
          ref={ref}
          canCreate={false}
          canUpdate={false}
          canRemove
          containerSpecOptions={[]}
          containerSpecMap={{}}
        />
      </App>,
    );
    await act(async () => {
      ref.current?.open({ id: 'order-3', orderNo: 'SE003' });
    });
    await screen.findByText('TGHU0000000');

    await confirmDelete();

    await waitFor(() => {
      expect(containerService.remove).not.toHaveBeenCalled();
    });
    await flushDeleteAftermath();
  });

  it('货物版本为零时不调用删除接口', async () => {
    cargoService.list.mockResolvedValue({
      data: [{ id: 'cargo-1', cargoName: '待补录货物', version: '0' }],
    });
    const ref = createRef<CargoItemDrawerRef>();
    render(
      <App>
        <CargoItemDrawer
          ref={ref}
          canCreate={false}
          canUpdate={false}
          canRemove
        />
      </App>,
    );
    await act(async () => {
      ref.current?.open({ id: 'order-4', orderNo: 'SE004' });
    });
    await screen.findByText('待补录货物');

    await confirmDelete();

    await waitFor(() => {
      expect(cargoService.remove).not.toHaveBeenCalled();
    });
    await flushDeleteAftermath();
  });
});
