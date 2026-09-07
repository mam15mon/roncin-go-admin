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

  const candidatesSpy = vi.spyOn(
    service,
    'seaSharedContainerServiceListSeaSharedContainerCandidates',
  );

  beforeEach(() => {
    vi.resetAllMocks();
    vi.spyOn(
      service,
      'seaSharedContainerServiceListSeaSharedContainers',
    ).mockResolvedValue({
      data: mockContainers,
      total: 1,
    } as any);
    candidatesSpy.mockResolvedValue({
      data: mockCandidates,
      total: 2,
      page: 1,
      pageSize: 20,
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
          canCreate={true}
          canUpdate={true}
          canDelete={true}
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
    // 候选订单走服务端关键字与分页参数
    expect(candidatesSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        transportExecutionId: 'te-123',
        page: 1,
        pageSize: 20,
      }),
    );
    expect(screen.getByText('共 2 票')).toBeInTheDocument();
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

  it('无对应权限时不展示创建、分配编辑与删除入口', async () => {
    render(
      <App>
        <SeaSharedContainerDrawer
          open={true}
          onClose={() => {}}
          transportExecutionId="te-123"
          canCreate={false}
          canUpdate={false}
          canDelete={false}
        />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getAllByText('TGHU1234567').length).toBeGreaterThan(0);
    });
    expect(screen.queryByText('新建共享物理箱')).not.toBeInTheDocument();
    expect(screen.queryByText('保存草稿')).not.toBeInTheDocument();
    expect(screen.queryByText('确认分配')).not.toBeInTheDocument();
    expect(screen.queryByText('删除')).not.toBeInTheDocument();
  });

  it('保存草稿携带期望版本并保留不在当前候选页的既有分配', async () => {
    // 第二条分配所属订单不在候选页，保存时必须原样保留
    const containersWithOffPage = [
      {
        ...mockContainers[0],
        version: '3',
        allocations: [
          ...(mockContainers[0].allocations ?? []),
          {
            id: 'alloc-2',
            sharedContainerId: 'cntr-shared-1',
            orderId: 'order-9',
            orderNo: 'SE20260903099',
            houseBillId: 'hb-9',
            houseNo: 'HBL009',
            cargoItemId: 'cargo-9',
            cargoName: '跨页货物',
            packageCount: 20,
            grossWeightKg: '400.000',
            volumeCbm: '3.000000',
            version: '1',
            orderVersion: '2',
            linkVersion: '2',
            houseBillVersion: '1',
            cargoItemVersion: '2',
          },
        ],
      },
    ];
    vi.spyOn(
      service,
      'seaSharedContainerServiceListSeaSharedContainers',
    ).mockResolvedValue({
      data: containersWithOffPage,
      total: 1,
    } as any);
    const saveSpy = vi
      .spyOn(
        service,
        'seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft',
      )
      .mockResolvedValue({ data: containersWithOffPage[0] } as any);

    render(
      <App>
        <SeaSharedContainerDrawer
          open={true}
          onClose={() => {}}
          transportExecutionId="te-123"
          canUpdate={true}
        />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText('保存草稿')).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText('保存草稿'));

    await waitFor(() => {
      expect(saveSpy).toHaveBeenCalledTimes(1);
    });
    const payload = saveSpy.mock.calls[0][1];
    expect(payload.expectedVersion).toBe('3');
    const preserved = (payload.allocations || []).find(
      (a: API.SeaSharedContainerAllocationInput) => a.orderId === 'order-9',
    );
    expect(preserved?.packageCount).toBe(20);
    expect(preserved?.expectedOrderVersion).toBe('2');
  });

  it('保存草稿遇到版本冲突时展示后端错误信息', async () => {
    render(
      <App>
        <SeaSharedContainerDrawer
          open={true}
          onClose={() => {}}
          transportExecutionId="te-123"
          canUpdate={true}
        />
      </App>,
    );
    await waitFor(() => {
      expect(screen.getByText('保存草稿')).toBeInTheDocument();
    });
    vi.spyOn(
      service,
      'seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft',
    ).mockRejectedValue(new Error('共享箱已被更新，请刷新后重试'));

    fireEvent.click(screen.getByText('保存草稿'));

    await waitFor(() => {
      expect(
        document.body.textContent?.includes('共享箱已被更新，请刷新后重试'),
      ).toBe(true);
    });
  });
});
