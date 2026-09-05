import { App } from 'antd';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import React, { createRef } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
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
  fireEvent.click(confirmButtons.at(-1) as HTMLElement);
}

describe('箱货删除版本', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    containerService.remove.mockResolvedValue({ success: true });
    cargoService.remove.mockResolvedValue({ success: true });
  });

  afterEach(() => cleanup());

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
  });
});
