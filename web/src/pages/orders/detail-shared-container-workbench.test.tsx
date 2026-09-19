import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as sharedService from '@/services/roncin/seaSharedContainerService';
import OrderDetailPage from './detail';

const routeState = vi.hoisted(() => ({
  params: { kind: 'sea-export', id: 'ord-A' } as {
    kind: string;
    id?: string;
  },
}));

vi.mock('@umijs/max', () => ({
  useParams: () => routeState.params,
  useAccess: () => ({
    canOrder: () => true,
  }),
  history: { push: vi.fn() },
}));

vi.mock('./use-order-detail-data', () => ({
  useOrderDetailData: (orderId?: string) => ({
    loading: false,
    order: orderId
      ? {
          id: orderId,
          orderNo: `ORDER-${orderId}`,
          version: '1',
          businessType: 1,
          seaMasterBill: { transportExecutionId: 'TE-A' },
        }
      : undefined,
    shippingDocs: [],
    personnel: [],
    serviceTypeOptions: [],
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: vi.fn().mockResolvedValue([]),
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: [],
    loadData: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock('./use-order-lock-state', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('./use-order-lock-state')>();
  return {
    ...actual,
    useOrderLockState: () => ({
      state: null,
      loading: false,
      error: null,
      refresh: vi.fn().mockResolvedValue(null),
    }),
  };
});

vi.mock('@/components/ui/order-template/OrderFormTemplate', () => ({
  OrderFormTemplate: ({ header }: { header: React.ReactNode }) => header,
}));

vi.mock('./components/detail/OrderDetailHeader', () => ({
  default: (props: {
    moreMenuItems?: Array<{
      key?: React.Key;
      onClick?: () => void;
    }>;
  }) => (
    <div>
      <button
        type="button"
        onClick={() => {
          props.moreMenuItems
            ?.find((item) => item?.key === 'shared-container-workbench')
            ?.onClick?.();
        }}
      >
        打开共享箱工作台
      </button>
    </div>
  ),
}));

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceGetSeaOrderChangeActions: vi
    .fn()
    .mockResolvedValue({ data: { canSplit: true, canReassign: true } }),
}));

const listContainersSpy = vi.spyOn(
  sharedService,
  'seaSharedContainerServiceListSeaSharedContainers',
);
const candidatesSpy = vi.spyOn(
  sharedService,
  'seaSharedContainerServiceListSeaSharedContainerCandidates',
);

describe('订单详情共享箱工作台路由复用隔离', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    routeState.params = { kind: 'sea-export', id: 'ord-A' };
    listContainersSpy.mockResolvedValue({ data: [], total: 0 } as any);
    candidatesSpy.mockResolvedValue({ data: [], total: 0 } as any);
  });

  it('订单 A 打开工作台后切换到订单 B：抽屉立即关闭且不再以旧运输执行发起新请求', async () => {
    const { rerender } = render(
      <App>
        <OrderDetailPage />
      </App>,
    );

    fireEvent.click(screen.getByText('打开共享箱工作台'));

    await waitFor(() => {
      expect(
        screen.getByText('跨订单共享箱 / 客户拼货工作台'),
      ).toBeInTheDocument();
    });
    expect(listContainersSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        orderId: 'ord-A',
        transportExecutionId: 'TE-A',
      }),
    );
    expect(listContainersSpy.mock.calls.length).toBeGreaterThan(0);

    // 同一组件实例被复用到订单 B：抽屉必须关闭，旧 TE 状态清空
    routeState.params = { kind: 'sea-export', id: 'ord-B' };
    rerender(
      <App>
        <OrderDetailPage />
      </App>,
    );

    await waitFor(() => {
      expect(
        screen.queryByText('跨订单共享箱 / 客户拼货工作台'),
      ).not.toBeInTheDocument();
    });
    // 抽屉关闭后等待一个周期，确认没有以新订单或旧运输执行发起任何新请求
    await new Promise((resolve) => setTimeout(resolve, 120));
    for (const call of listContainersSpy.mock.calls) {
      expect(call[0]).toEqual(
        expect.objectContaining({
          orderId: 'ord-A',
          transportExecutionId: 'TE-A',
        }),
      );
    }
    expect(candidatesSpy.mock.calls.length).toBeGreaterThanOrEqual(1);
  });
});
