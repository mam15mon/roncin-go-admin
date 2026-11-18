import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  feeRouteState,
  feeTestState,
  renderFeesPage,
  rerenderFeesPage,
} from './fees.fixtures';

describe('订单费用页关联账单追踪', () => {
  const getFeeLedgerOrderDetail = feeTestState.getFeeLedgerOrderDetail;

  beforeEach(() => {
    feeRouteState.params = { kind: 'sea-export', id: 'order-A' };
    feeTestState.lockState = { isLocked: false };
    feeTestState.canCreateFee = true;
    feeTestState.canReadFee = true;
    feeTestState.canLock = true;
    feeTestState.canReadFinanceFees = false;
    feeTestState.canOperate = true;
    feeTestState.hasPendingSupplement = false;
    vi.clearAllMocks();
  });

  it('无财务读取权限时不发起关联账单请求，也不向表格注入投影', async () => {
    renderFeesPage();
    await waitFor(() =>
      expect(screen.getByTestId('fee-table-state')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('fee-bill-tracking')).toHaveTextContent('none');
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(getFeeLedgerOrderDetail).not.toHaveBeenCalled();
  });

  it('有财务读取权限时按订单维度查询一次并投影活动账单号', async () => {
    feeTestState.canReadFinanceFees = true;
    getFeeLedgerOrderDetail.mockResolvedValue({
      data: {
        fees: [
          { id: 'fee-A-1', billNo: 'BILL-2026-001', financialProgress: 2 },
        ],
      },
    } as Awaited<ReturnType<typeof getFeeLedgerOrderDetail>>);

    renderFeesPage();
    await waitFor(() =>
      expect(getFeeLedgerOrderDetail).toHaveBeenCalledWith(
        { orderId: 'order-A' },
        expect.anything(),
      ),
    );
    await waitFor(() =>
      expect(screen.getByTestId('fee-bill-tracking')).toHaveTextContent(
        'ready|BILL-2026-001',
      ),
    );
    expect(getFeeLedgerOrderDetail).toHaveBeenCalledTimes(1);
  });

  it('行内保存通知后失效关联账单查询并重新拉取', async () => {
    feeTestState.canReadFinanceFees = true;
    getFeeLedgerOrderDetail.mockResolvedValue({
      data: { fees: [] },
    } as Awaited<ReturnType<typeof getFeeLedgerOrderDetail>>);

    renderFeesPage();
    await waitFor(() =>
      expect(getFeeLedgerOrderDetail).toHaveBeenCalledTimes(1),
    );

    fireEvent.click(screen.getByRole('button', { name: '通知行内保存' }));
    await waitFor(() =>
      expect(getFeeLedgerOrderDetail).toHaveBeenCalledTimes(2),
    );
  });

  it('切换订单后按新订单查询，不串用上一订单的关联信息', async () => {
    feeTestState.canReadFinanceFees = true;
    getFeeLedgerOrderDetail.mockImplementation(({ orderId }) =>
      Promise.resolve({
        data: {
          fees: [
            {
              id: orderId === 'order-A' ? 'fee-A-1' : 'fee-B-1',
              billNo: `BILL-${orderId}`,
              financialProgress: 7,
            },
          ],
        },
      } as any),
    );

    const { rerender } = renderFeesPage();
    await waitFor(() =>
      expect(screen.getByTestId('fee-bill-tracking')).toHaveTextContent(
        'ready|BILL-order-A',
      ),
    );

    feeRouteState.params = { kind: 'sea-export', id: 'order-B' };
    rerenderFeesPage(rerender);
    await waitFor(() =>
      expect(getFeeLedgerOrderDetail).toHaveBeenCalledWith(
        { orderId: 'order-B' },
        expect.anything(),
      ),
    );
    await waitFor(() =>
      expect(screen.getByTestId('fee-bill-tracking')).toHaveTextContent(
        'ready|BILL-order-B',
      ),
    );
    expect(screen.getByTestId('fee-bill-tracking')).not.toHaveTextContent(
      'BILL-order-A',
    );
  });
});
