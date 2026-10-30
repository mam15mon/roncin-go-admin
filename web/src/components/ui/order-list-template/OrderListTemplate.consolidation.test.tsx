import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { ShipmentType } from '@/enums.generated';
import { OrderListTemplate } from './OrderListTemplate';
import type { OrderListItem } from './types';

beforeEach(() => localStorage.clear());
afterEach(cleanup);

async function openOrderMenu(
  transportMode: 'sea' | 'air',
  shipmentType?: number,
) {
  const order: OrderListItem = {
    id: 'order-consolidation',
    orderNo: 'ORDER-CONSOLIDATION',
    rawRecord: { shipmentType },
  };
  const onOpenConsolidations = vi.fn();
  render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <OrderListTemplate
        options={{ transportMode }}
        queryOrders={vi.fn().mockResolvedValue({ data: [order], total: 1 })}
        onOpenConsolidations={onOpenConsolidations}
      />
    </ConfigProvider>,
  );
  await screen.findByText(order.orderNo);
  fireEvent.mouseEnter(screen.getByRole('button', { name: /更多/ }));
  const feeAction = await screen.findByText('费用核算');
  await waitFor(() => expect(feeAction).toBeVisible());
  return { order, onOpenConsolidations };
}

it('海运拼箱订单显示自拼汇总并将当前行交给打开回调', async () => {
  const { order, onOpenConsolidations } = await openOrderMenu(
    'sea',
    ShipmentType.SHIPMENT_TYPE_LCL,
  );
  const action = screen.getByText('自拼汇总');
  expect(action).toBeVisible();
  fireEvent.click(action);
  expect(onOpenConsolidations).toHaveBeenCalledExactlyOnceWith(order);
});

it.each([
  {
    label: '海运整箱',
    transportMode: 'sea' as const,
    shipmentType: ShipmentType.SHIPMENT_TYPE_FCL,
  },
  {
    label: '海运托运类型未明确',
    transportMode: 'sea' as const,
    shipmentType: undefined,
  },
  {
    label: '空运拼箱标记',
    transportMode: 'air' as const,
    shipmentType: ShipmentType.SHIPMENT_TYPE_LCL,
  },
])(
  '$label 不显示自拼汇总，费用入口保持可见',
  async ({ transportMode, shipmentType }) => {
    const { onOpenConsolidations } = await openOrderMenu(
      transportMode,
      shipmentType,
    );
    expect(screen.queryByText('自拼汇总')).not.toBeInTheDocument();
    expect(screen.getByText('费用核算')).toBeVisible();
    expect(onOpenConsolidations).not.toHaveBeenCalled();
  },
);
