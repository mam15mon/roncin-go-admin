import { App } from 'antd';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import React, { createRef } from 'react';
import { describe, expect, it, vi } from 'vitest';
import SeaCargoAllocationDrawer, {
  type SeaCargoAllocationDrawerRef,
} from './SeaCargoAllocationDrawer';
import * as service from '@/services/roncin/seaCargoAllocationService';

describe('SeaCargoAllocationDrawer', () => {
  it('打开抽屉能正常加载并展示三视角数据与操作按钮', async () => {
    const mockOrder: API.Order = {
      id: 'order-123',
      orderNo: 'ORD-SE-20260903',
      businessType: 1,
    };

    const mockAggregate: API.SeaCargoAllocationAggregate = {
      orderId: 'order-123',
      documentStructure: 3, // HOUSE
      shipmentType: 'FCL',
      allocationStatus: 1, // DRAFT
      allocationVersion: '3',
      cargoItems: [
        {
          id: 'ci-1',
          cargoName: '精密机械',
          packageCount: 100,
          grossWeightKg: 1000,
          volumeCbm: 10,
          version: '1',
        },
      ],
      containers: [
        {
          id: 'cntr-1',
          containerNo: 'TGHU1234567',
          packageCount: 100,
          grossWeightKg: 1000,
          volumeCbm: 10,
          version: '1',
        },
      ],
      houseBills: [
        {
          id: 'hb-1',
          houseNo: 'HBL-001',
          version: '1',
          content: {
            packageCount: 100,
            grossWeightKg: 1000,
            volumeCbm: 10,
          },
        },
      ],
      allocations: [
        {
          id: 'alloc-1',
          cargoItemId: 'ci-1',
          houseBillId: 'hb-1',
          containerId: 'cntr-1',
          packageCount: 100,
          grossWeightKg: '1000.000',
          volumeCbm: '10.000000',
        },
      ],
      allowedActions: [1, 2, 4], // SAVE_DRAFT, CONFIRM, APPLY_HBL
    };

    vi.spyOn(service, 'seaCargoAllocationServiceGetSeaCargoAllocation').mockResolvedValue({
      data: mockAggregate,
    } as any);
    vi.spyOn(service, 'seaCargoAllocationServiceSaveSeaCargoAllocationDraft').mockRejectedValue({
      message: '精密机械的件数已超分',
      metadata: { cargo_item_id: 'ci-1', object_type: 'cargo_item' },
    });

    const drawerRef = createRef<SeaCargoAllocationDrawerRef>();

    render(
      <App>
        <SeaCargoAllocationDrawer ref={drawerRef} canManage />
      </App>,
    );

    drawerRef.current?.open(mockOrder);

    await waitFor(() => {
      expect(screen.getByText('海运出口箱货定量分配')).toBeInTheDocument();
      expect(screen.getByText(/ORD-SE-20260903/)).toBeInTheDocument();
      expect(screen.getByText(/草稿 \(DRAFT\)/)).toBeInTheDocument();
      expect(screen.getByText('保存草稿')).toBeInTheDocument();
      expect(screen.getByText('确认分配 (守恒门禁)')).toBeInTheDocument();
      expect(screen.getByTestId('live-allocation-progress')).toHaveTextContent(
        '剩余 0 件 / 0.000 KG / 0.000000 CBM',
      );
    });

    fireEvent.click(screen.getByText('保存草稿'));
    await waitFor(() => {
      expect(screen.getAllByText('精密机械的件数已超分')).not.toHaveLength(0);
      const targetRow = screen.getByDisplayValue('100').closest('tr');
      expect(targetRow).toContainElement(document.activeElement as HTMLElement);
      expect(targetRow).toHaveClass(
        'sea-cargo-allocation-error-row',
      );
    });
  });
});
