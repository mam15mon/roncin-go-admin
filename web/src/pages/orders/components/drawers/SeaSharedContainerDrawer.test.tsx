import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as service from '@/services/roncin/seaSharedContainerService';
import SeaSharedContainerDrawer from './SeaSharedContainerDrawer';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

describe('SeaSharedContainerDrawer', () => {
  const balancedContainer: API.SeaSharedContainer = {
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
        packageCount: 100,
        grossWeightKg: '2000.000',
        volumeCbm: '15.000000',
        version: '1',
        orderVersion: '1',
        linkVersion: '1',
        houseBillVersion: '1',
        cargoItemVersion: '1',
      },
    ],
    progress: {
      allocatedPackageCount: 100,
      allocatedGrossWeightKg: '2000.000',
      allocatedVolumeCbm: '15.000000',
      remainingPackageCount: 0,
      remainingGrossWeightKg: '0.000',
      remainingVolumeCbm: '0.000000',
      containerBalanced: true,
      cargoBalanced: true,
    },
  };

  const candidatesPage1: API.SeaSharedContainerCandidateOrder[] = [
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
      orderId: 'order-multi',
      orderNo: 'SE20260903002',
      houseBillId: 'hb-m',
      houseNo: 'HBL002',
      orderVersion: '1',
      linkVersion: '1',
      houseBillVersion: '1',
      cargoItems: [
        {
          id: 'cargo-m1',
          cargoName: '多货物行一',
          packageCount: 30,
          grossWeightKg: '600.000',
          volumeCbm: '4.500000',
          version: '1',
        },
        {
          id: 'cargo-m2',
          cargoName: '多货物行二',
          packageCount: 20,
          grossWeightKg: '400.000',
          volumeCbm: '3.000000',
          version: '1',
        },
      ],
    },
  ];

  const candidatesPage2: API.SeaSharedContainerCandidateOrder[] = [
    {
      orderId: 'order-2',
      orderNo: 'SE20260903009',
      houseBillId: 'hb-2',
      houseNo: 'HBL009',
      orderVersion: '1',
      linkVersion: '1',
      houseBillVersion: '1',
      cargoItems: [
        {
          id: 'cargo-2',
          cargoName: '第二页货物',
          packageCount: 50,
          grossWeightKg: '800.000',
          volumeCbm: '6.000000',
          version: '1',
        },
      ],
    },
  ];

  const listContainersSpy = vi.spyOn(
    service,
    'seaSharedContainerServiceListSeaSharedContainers',
  );
  const candidatesSpy = vi.spyOn(
    service,
    'seaSharedContainerServiceListSeaSharedContainerCandidates',
  );

  const renderDrawer = (props: Record<string, unknown> = {}) =>
    render(
      <App>
        <SeaSharedContainerDrawer
          open={true}
          onClose={() => {}}
          transportExecutionId="te-123"
          orderId="order-1"
          canUpdate={true}
          {...props}
        />
      </App>,
    );

  beforeEach(() => {
    vi.resetAllMocks();
    listContainersSpy.mockResolvedValue({
      data: [balancedContainer],
      total: 1,
    } as any);
    candidatesSpy.mockResolvedValue({
      data: candidatesPage1,
      total: 21,
      page: 1,
      pageSize: 20,
    } as any);
  });

  it('正确渲染共享箱工作台，候选按服务端订单分页并携带授权锚点', async () => {
    renderDrawer({ canCreate: true, canDelete: true });

    await waitFor(() => {
      expect(
        screen.getByText('跨订单共享箱 / 客户拼货工作台'),
      ).toBeInTheDocument();
      expect(screen.getAllByText('TGHU1234567').length).toBeGreaterThan(0);
      expect(screen.getByText('保存草稿')).toBeInTheDocument();
      expect(screen.getByText('确认分配')).toBeInTheDocument();
    });
    expect(candidatesSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        orderId: 'order-1',
        transportExecutionId: 'te-123',
        page: 1,
        pageSize: 20,
      }),
    );
    expect(listContainersSpy).toHaveBeenCalledWith(
      expect.objectContaining({ orderId: 'order-1', transportExecutionId: 'te-123' }),
    );
  });

  it('未关联实际运输执行时展示阻断提示', () => {
    render(
      <App>
        <SeaSharedContainerDrawer
          open={true}
          onClose={() => {}}
          transportExecutionId={undefined}
          orderId="order-1"
        />
      </App>,
    );

    expect(
      screen.getByText('当前订单未关联实际运输执行'),
    ).toBeInTheDocument();
  });

  it('无对应权限时不展示创建、分配编辑与删除入口', async () => {
    renderDrawer({ canCreate: false, canUpdate: false, canDelete: false });

    await waitFor(() => {
      expect(screen.getAllByText('TGHU1234567').length).toBeGreaterThan(0);
    });
    expect(screen.queryByText('新建共享物理箱')).not.toBeInTheDocument();
    expect(screen.queryByText('保存草稿')).not.toBeInTheDocument();
    expect(screen.queryByText('确认分配')).not.toBeInTheDocument();
    expect(screen.queryByText('删除')).not.toBeInTheDocument();
  });

  it('一页订单内的全部货物行完整展示，Table 不做本地二次分页', async () => {
    renderDrawer();

    // 第一页 2 张订单共 3 条货物行（其中一张订单有 2 条货物），全部可见
    await waitFor(() => {
      expect(screen.getByText('机械零件')).toBeInTheDocument();
    });
    expect(screen.getByText('多货物行一')).toBeInTheDocument();
    expect(screen.getByText('多货物行二')).toBeInTheDocument();
    expect(screen.getByText('共 21 票订单')).toBeInTheDocument();
  });

  it('翻页后展示第二页订单的货物并隐藏第一页货物', async () => {
    candidatesSpy.mockImplementation((async (params: any) => {
      if (params?.page === 2) {
        return { data: candidatesPage2, total: 21, page: 2, pageSize: 20 };
      }
      return { data: candidatesPage1, total: 21, page: 1, pageSize: 20 };
    }) as any);

    renderDrawer();
    await waitFor(() => {
      expect(screen.getByText('多货物行一')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTitle('2'));

    await waitFor(() => {
      expect(screen.getByText('第二页货物')).toBeInTheDocument();
    });
    expect(screen.queryByText('多货物行一')).not.toBeInTheDocument();
    expect(screen.queryByText('机械零件')).not.toBeInTheDocument();
    expect(candidatesSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 2 }),
    );
  });

  it('跨页新录入的分配在切换页面后保存时不丢失', async () => {
    candidatesSpy.mockImplementation((async (params: any) => {
      if (params?.page === 2) {
        return { data: candidatesPage2, total: 21, page: 2, pageSize: 20 };
      }
      return { data: candidatesPage1, total: 21, page: 1, pageSize: 20 };
    }) as any);
    const saveSpy = vi
      .spyOn(
        service,
        'seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft',
      )
      .mockResolvedValue({ data: balancedContainer } as any);

    renderDrawer();
    await waitFor(() => {
      expect(screen.getByText('多货物行一')).toBeInTheDocument();
    });

    // 第一页：为 order-multi 的第一条货物录入 30 件（表格首行是 order-1，第二行是 cargo-m1）
    const packageInputs = screen.getAllByRole('spinbutton');
    fireEvent.change(packageInputs[1], { target: { value: '30' } });
    fireEvent.blur(packageInputs[1]);

    // 未保存直接翻到第二页
    fireEvent.click(screen.getByTitle('2'));
    await waitFor(() => {
      expect(screen.getByText('第二页货物')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('保存草稿'));
    await waitFor(() => {
      expect(saveSpy).toHaveBeenCalledTimes(1);
    });
    const payload = saveSpy.mock.calls[0][1];
    const preserved = (payload.allocations || []).find(
      (a: API.SeaSharedContainerAllocationInput) =>
        a.cargoItemId === 'cargo-m1',
    );
    expect(preserved?.packageCount).toBe(30);
    expect(preserved?.expectedCargoItemVersion).toBe('1');
    expect(payload.orderId).toBe('order-1');
  });

  it('确认分配失败时展示错误并强制重新加载共享箱状态', async () => {
    // 初始分配恰好守恒，确认按钮可用
    const confirmSpy = vi
      .spyOn(service, 'seaSharedContainerServiceConfirmSeaSharedContainer')
      .mockRejectedValue(new Error('共享箱已被更新，请刷新后重试'));

    renderDrawer();
    await waitFor(() => {
      expect(screen.getByText('确认分配')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText('确认分配'));

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalledTimes(1);
    });
    const request = confirmSpy.mock.calls[0][1];
    // 单请求确认：携带全部草稿分配与授权锚点
    expect(request.orderId).toBe('order-1');
    expect(request.allocations?.length).toBe(1);
    expect(request.allocations?.[0].cargoItemId).toBe('cargo-1');

    await waitFor(() => {
      expect(
        document.body.textContent?.includes('共享箱已被更新，请刷新后重试'),
      ).toBe(true);
    });
    // 失败后必须重新加载，避免停留在过期版本
    await waitFor(() => {
      expect(listContainersSpy.mock.calls.length).toBeGreaterThan(1);
    });
  });

  it('上下文从 A 切换到 B 时迟到响应不得覆盖新上下文数据', async () => {
    const containerA: API.SeaSharedContainer = {
      ...balancedContainer,
      transportExecutionId: 'te-a',
      containerNo: 'AAAA1111111',
      allocations: [],
      progress: undefined,
    };
    const containerB: API.SeaSharedContainer = {
      ...balancedContainer,
      transportExecutionId: 'te-b',
      containerNo: 'BBBB2222222',
      allocations: [],
      progress: undefined,
    };

    const listDeferred = [
      deferred<{ data: API.SeaSharedContainer[]; total: number }>(),
      deferred<{ data: API.SeaSharedContainer[]; total: number }>(),
    ];
    let listCall = 0;
    listContainersSpy.mockImplementation(
      () => listDeferred[listCall++]?.promise ?? Promise.resolve({ data: [], total: 0 }),
    );
    const candidatesDeferred = [
      deferred<{ data: API.SeaSharedContainerCandidateOrder[]; total: number }>(),
      deferred<{ data: API.SeaSharedContainerCandidateOrder[]; total: number }>(),
    ];
    let candidateCall = 0;
    candidatesSpy.mockImplementation(
      () =>
        candidatesDeferred[candidateCall++]?.promise ??
        Promise.resolve({ data: [], total: 0 }),
    );

    const { rerender } = renderDrawer();
    // 切换到上下文 B
    rerender(
      <App>
        <SeaSharedContainerDrawer
          open={true}
          onClose={() => {}}
          transportExecutionId="te-b"
          orderId="order-b"
          canUpdate={true}
        />
      </App>,
    );

    // B 响应先返回
    candidatesDeferred[1].resolve({ data: candidatesPage1, total: 2 });
    listDeferred[1].resolve({ data: [containerB], total: 1 });
    await waitFor(() => {
      expect(screen.getAllByText('BBBB2222222').length).toBeGreaterThan(0);
    });

    // A 响应迟到，必须被丢弃
    candidatesDeferred[0].resolve({ data: candidatesPage2, total: 2 });
    listDeferred[0].resolve({ data: [containerA], total: 1 });

    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(screen.queryByText('AAAA1111111')).not.toBeInTheDocument();
    expect(screen.queryByText('第二页货物')).not.toBeInTheDocument();
    expect(screen.getAllByText('BBBB2222222').length).toBeGreaterThan(0);
  });
});
