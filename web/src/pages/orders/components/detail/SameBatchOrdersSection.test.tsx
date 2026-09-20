import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import SameBatchOrdersSection from './SameBatchOrdersSection';

const { listSameBatchOrdersMock, historyPushMock } = vi.hoisted(() => ({
  listSameBatchOrdersMock: vi.fn(),
  historyPushMock: vi.fn(),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListSameBatchOrders: listSameBatchOrdersMock,
}));

vi.mock('@/router/history', () => ({
  history: { push: historyPushMock },
}));

describe('SameBatchOrdersSection', () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('去重并展示匹配来源，点击后进入同类订单详情', async () => {
    listSameBatchOrdersMock.mockResolvedValue({
      data: [
        { orderId: 'current', orderNo: 'SE-CURRENT' },
        {
          orderId: 'related',
          orderNo: 'SE-RELATED',
          matchSources: ['CUSTOMER_REFERENCE', 'BOOKING', 'MASTER'],
        },
        { orderId: 'related', orderNo: 'SE-DUPLICATE' },
      ],
    });

    render(<SameBatchOrdersSection orderId="current" orderKind="sea-export" />);

    await waitFor(() => {
      expect(screen.getByText('SE-RELATED')).toBeInTheDocument();
    });
    expect(screen.queryByText('SE-CURRENT')).not.toBeInTheDocument();
    expect(screen.queryByText('SE-DUPLICATE')).not.toBeInTheDocument();
    expect(screen.getByText('同客户业务号')).toBeInTheDocument();
    expect(screen.getByText('同订舱号')).toBeInTheDocument();
    expect(screen.getByText('同 MBL')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /SE-RELATED/ }));
    expect(historyPushMock).toHaveBeenCalledWith('/orders/sea-export/related');
  });

  it('没有关联订单时展示空状态', async () => {
    listSameBatchOrdersMock.mockResolvedValue({ data: [] });

    render(<SameBatchOrdersSection orderId="current" orderKind="sea-export" />);

    expect(await screen.findByText('暂无同批订单')).toBeInTheDocument();
  });
});
