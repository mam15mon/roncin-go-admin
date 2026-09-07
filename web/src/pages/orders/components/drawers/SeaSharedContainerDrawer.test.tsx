import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as service from '@/services/roncin/seaSharedContainerService';
import SeaSharedContainerDrawer from './SeaSharedContainerDrawer';

describe('SeaSharedContainerDrawer', () => {
  const mockContainers: API.SeaSharedContainer[] = [
    {
      id: 'cntr-shared-1',
      organizationId: 'org-1',
      transportExecutionId: 'te-123',
      containerNo: 'TGHU1234567',
      containerSpecId: 'spec-40gp',
      containerSpecName: '40GP',
      sealNo: 'SEAL123',
      packageCount: 100,
      grossWeightKg: '2000.000',
      volumeCbm: '15.000000',
      status: 1, // DRAFT
      version: '1',
      allocations: [
        {
          id: 'alloc-1',
          sharedContainerId: 'cntr-shared-1',
          orderId: 'order-1',
          orderNo: 'SE20260903001',
          houseBillId: 'hb-1',
          houseNo: 'HBL001',
          cargoItemId: 'cargo-1',
          cargoName: '机械零件',
          packageCount: 60,
          grossWeightKg: '1200.000',
          volumeCbm: '9.000000',
          version: '1',
        },
      ],
      progress: {
        allocatedPackageCount: 60,
        allocatedGrossWeightKg: '1200.000',
        allocatedVolumeCbm: '9.000000',
        remainingPackageCount: 40,
        remainingGrossWeightKg: '800.000',
        remainingVolumeCbm: '6.000000',
        containerBalanced: false,
        cargoBalanced: true,
      },
    },
  ];

  const mockCandidates: API.SeaSharedContainerCandidateOrder[] = [
    {
      orderId: 'order-1',
      orderNo: 'SE20260903001',
      houseBillId: 'hb-1',
      houseNo: 'HBL001',
      orderVersion: '1',
      linkVersion: '1',
      houseBillVersion: '1',
      cargoItems: [
        {
          id: 'cargo-1',
          cargoName: '机械零件',
          packageCount: 100,
          grossWeightKg: '2000.000',
          volumeCbm: '15.000000',
          version: '1',
        },
      ],
    },
    {
      orderId: 'order-2',
      orderNo: 'SE20260903002',
      houseBillId: 'hb-2',
      houseNo: 'HBL002',
      orderVersion: '1',
      linkVersion: '1',
      houseBillVersion: '1',
      cargoItems: [
        {
          id: 'cargo-2',
          cargoName: '塑料制品',
          packageCount: 50,
          grossWeightKg: '800.000',
          volumeCbm: '6.000000',
          version: '1',
        },
      ],
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(
      service,
      'seaSharedContainerServiceListSeaSharedContainers',
    ).mockResolvedValue({
      data: mockContainers,
      total: 1,
    } as any);
    vi.spyOn(
      service,
      'seaSharedContainerServiceListSeaSharedContainerCandidates',
    ).mockResolvedValue({
      data: mockCandidates,
    } as any);
  });

  it('正确渲染共享箱工作台并展示选定物理箱与跨订单货物分配表格', async () => {
    render(
      <App>
        <SeaSharedContainerDrawer
          open={true}
          onClose={() => {}}
          transportExecutionId="te-123"
          orderId="order-1"
          orderNo="SE20260903001"
        />
      </App>,
    );

    await waitFor(() => {
      expect(
        screen.getByText('跨订单共享箱 / 客户拼货工作台'),
      ).toBeInTheDocument();
      expect(screen.getAllByText('TGHU1234567').length).toBeGreaterThan(0);
      expect(screen.getByText('机械零件')).toBeInTheDocument();
      expect(screen.getByText('塑料制品')).toBeInTheDocument();
      expect(screen.getByText('保存草稿')).toBeInTheDocument();
      expect(screen.getByText('确认分配')).toBeInTheDocument();
    });
  });

  it('未关联实际运输执行时展示阻断提示', async () => {
    render(
      <App>
        <SeaSharedContainerDrawer
          open={true}
          onClose={() => {}}
          transportExecutionId={undefined}
        />
      </App>,
    );

    expect(
      screen.getByText('当前订单未关联实际运输执行'),
    ).toBeInTheDocument();
  });
});
