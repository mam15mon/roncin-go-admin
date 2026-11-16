import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import {
  act,
  cleanup,
  fireEvent,
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
  history: {
    get location() {
      return window.location;
    },
    push: historyPushMock,
  },
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
  it('切换订单后首帧隐藏旧记录且旧请求迟到不覆盖新订单', async () => {
    let resolveOld!: (value: { data: API.SameBatchOrderSummary[] }) => void;
    listSameBatchOrdersMock.mockImplementation(({ id }: { id: string }) =>
      id === 'old'
        ? new Promise((resolve) => {
            resolveOld = resolve;
          })
        : Promise.resolve({
            success: true,
            data: [{ orderId: 'new-related', orderNo: 'NEW-RELATED' }],
          }),
    );
    const view = render(
      <SameBatchOrdersSection orderId="old" orderKind="sea-export" />,
    );
    view.rerender(
      <SameBatchOrdersSection orderId="new" orderKind="sea-export" />,
    );
    expect(await screen.findByText('NEW-RELATED')).toBeVisible();
    await act(async () =>
      resolveOld({
        data: [{ orderId: 'old-related', orderNo: 'OLD-RELATED' }],
      }),
    );
    expect(screen.queryByText('OLD-RELATED')).not.toBeInTheDocument();
    expect(screen.getByText('NEW-RELATED')).toBeVisible();
  });

  it.each([undefined, { success: false }, { data: {} }])(
    '无效列表响应显示失败而非空列表：%j',
    async (response) => {
      listSameBatchOrdersMock.mockResolvedValue(response);
      render(
        <SameBatchOrdersSection orderId="current" orderKind="sea-export" />,
      );
      expect(await screen.findByRole('alert')).toHaveTextContent(
        '同批订单加载失败',
      );
      expect(screen.queryByText('暂无同批订单')).not.toBeInTheDocument();
    },
  );

  it('成功且省略空 repeated 时展示合法空列表', async () => {
    listSameBatchOrdersMock.mockResolvedValue({ success: true });
    render(<SameBatchOrdersSection orderId="current" orderKind="sea-export" />);
    expect(await screen.findByText('暂无同批订单')).toBeVisible();
  });
});
