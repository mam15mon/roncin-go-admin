import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { expect, it, vi } from 'vitest';
import * as service from '@/services/roncin/seaSharedContainerService';
import SeaSharedContainerDrawer from './SeaSharedContainerDrawer';
import type { CargoAllocationItem } from './seaSharedContainerModels';

// 在候选区首次可交互的提交阶段立即触发填入，稳定覆盖被动初始化覆盖用户操作的窗口。
vi.mock('./SeaSharedContainerAllocationTable', () => ({
  default: ({
    flatCargoList,
    onFillAllCargo,
  }: {
    flatCargoList: CargoAllocationItem[];
    onFillAllCargo: (record: CargoAllocationItem) => void;
  }) => {
    const filled = React.useRef(false);
    React.useEffect(() => {
      if (!filled.current && flatCargoList[0]) {
        filled.current = true;
        onFillAllCargo(flatCargoList[0]);
      }
    }, [flatCargoList, onFillAllCargo]);
    return (
      <div>
        <button
          type="button"
          onClick={() => {
            if (flatCargoList[0])
              onFillAllCargo({ ...flatCargoList[0], totalPackageCount: 7 });
            document.querySelectorAll('button').forEach((button) => {
              if (button.textContent === '保存草稿') button.click();
            });
          }}
        >
          快速填入并保存
        </button>
        已填入件数：{flatCargoList[0]?.draft.packageCount ?? '-'}
      </div>
    );
  },
}));

it('首次候选行填入发生在被动效果阶段也不会被草稿初始化覆盖', async () => {
  const container: API.SeaSharedContainer = {
    id: 'container-1',
    containerNo: 'TEST1234567',
    transportExecutionId: 'te-1',
    packageCount: 12,
    grossWeightKg: '12',
    volumeCbm: '1',
    status: 1,
    version: '1',
  };
  vi.spyOn(
    service,
    'seaSharedContainerServiceListSeaSharedContainers',
  ).mockResolvedValue({ success: true, data: [container] });
  vi.spyOn(
    service,
    'seaSharedContainerServiceListSeaSharedContainerCandidates',
  ).mockResolvedValue({
    success: true,
    data: [
      {
        orderId: 'order-1',
        orderNo: 'SE001',
        houseBillId: 'hbl-1',
        orderVersion: '1',
        linkVersion: '1',
        houseBillVersion: '1',
        cargoItems: [
          {
            id: 'cargo-1',
            version: '1',
            packageCount: 12,
            grossWeightKg: '12',
            volumeCbm: '1',
          },
        ],
      },
    ],
    total: 1,
  });
  const save = vi
    .spyOn(
      service,
      'seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft',
    )
    .mockResolvedValue({ success: true, data: container });
  renderWithClient(
    <App>
      <SeaSharedContainerDrawer
        open
        orderId="order-1"
        transportExecutionId="te-1"
        onClose={() => {}}
        canUpdate
      />
    </App>,
  );
  await screen.findByText('已填入件数：12');
  fireEvent.click(screen.getByText('保存草稿'));
  await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
  expect(save.mock.calls[0][1].allocations).toEqual([
    expect.objectContaining({
      cargoItemId: 'cargo-1',
      orderId: 'order-1',
      houseBillId: 'hbl-1',
      packageCount: 12,
      expectedCargoItemVersion: '1',
    }),
  ]);
});

it('同一事件批次填入后立即保存使用最新草稿快照', async () => {
  const container: API.SeaSharedContainer = {
    id: 'container-1',
    containerNo: 'TEST1234567',
    transportExecutionId: 'te-1',
    packageCount: 12,
    grossWeightKg: '12',
    volumeCbm: '1',
    status: 1,
    version: '1',
  };
  vi.spyOn(
    service,
    'seaSharedContainerServiceListSeaSharedContainers',
  ).mockResolvedValue({ success: true, data: [container] });
  vi.spyOn(
    service,
    'seaSharedContainerServiceListSeaSharedContainerCandidates',
  ).mockResolvedValue({
    success: true,
    data: [
      {
        orderId: 'order-1',
        orderNo: 'SE001',
        houseBillId: 'hbl-1',
        orderVersion: '1',
        linkVersion: '1',
        houseBillVersion: '1',
        cargoItems: [
          {
            id: 'cargo-1',
            version: '1',
            packageCount: 12,
            grossWeightKg: '12',
            volumeCbm: '1',
          },
        ],
      },
    ],
    total: 1,
  });
  const save = vi
    .spyOn(
      service,
      'seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft',
    )
    .mockResolvedValue({ success: true, data: container });
  renderWithClient(
    <App>
      <SeaSharedContainerDrawer
        open
        orderId="order-1"
        transportExecutionId="te-1"
        onClose={() => {}}
        canUpdate
      />
    </App>,
  );
  await screen.findByText('已填入件数：12');
  await act(async () => {
    fireEvent.click(screen.getByText('快速填入并保存'));
  });
  await waitFor(() => expect(save).toHaveBeenCalled());
  expect(save.mock.lastCall?.[1].allocations?.[0].packageCount).toBe(7);
});
