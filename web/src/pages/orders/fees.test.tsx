import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { feeCatalogServiceListTaxableServices } from '@/services/roncin/feeCatalogService';
import { orderFeeServiceListOrderFeeSupplementRequests } from '@/services/roncin/orderFeeService';
import OrderFeesPage from './fees';

let mockParams = { kind: 'sea-export', id: 'order-A' };

const feeTestState = vi.hoisted(() => ({
  resetPreview: vi.fn(),
  lockState: { isLocked: false } as Partial<API.OrderLockStateData>,
  canCreateFee: true,
  canOperate: true,
}));

vi.mock('@/router/history', () => ({
  history: { push: vi.fn() },
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => feeTestState.canOperate,
    canOperateBusiness: true,
    canCreateFinanceBills: true,
    canOrder: () => feeTestState.canCreateFee,
  }),
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useParams: () => mockParams,
  };
});

vi.mock('@/components/ui', () => ({
  FinanceSummaryBoard: ({ selectedRows, allRows }: any) => (
    <div data-testid="finance-summary-board">
      {selectedRows.map((item: API.OrderFee) => item.id).join(',')}|
      {allRows.map((item: API.OrderFee) => item.id).join(',')}
    </div>
  ),
  SectionCard: ({ title, children }: any) => (
    <div data-testid="fee-supplement-card">
      {title}
      {children}
    </div>
  ),
  ProFormSearchableSelect: (props: any) => (
    <div data-testid={`ui-select-${props.name}`}>{props.label}</div>
  ),
  ExchangeRatePreviewCard: () => <div data-testid="exchange-preview" />,
}));

vi.mock('@/features/finance/bill-creation', () => ({
  BillCreationWorkbench: ({
    open,
    initialFeeIds,
    initialOrganizationId,
    sourceLabel,
  }: any) => {
    const [workbenchFeeIds, setWorkbenchFeeIds] = React.useState<string[]>([]);
    React.useEffect(() => {
      if (open) setWorkbenchFeeIds(initialFeeIds);
    }, [initialFeeIds, open]);
    return (
      <div data-testid="bill-workbench">
        {String(open)}|{workbenchFeeIds.join(',')}|{initialOrganizationId}|
        {sourceLabel}
        {open && <button type="button">提交账单</button>}
      </div>
    );
  },
}));

vi.mock('@/services/roncin/feeCatalogService', () => ({
  feeCatalogServiceListTaxableServices: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceAddFee: vi.fn(),
  orderFeeServiceConfirmFee: vi.fn(),
  orderFeeServiceRemoveFee: vi.fn(),
  orderFeeServiceReopenFee: vi.fn(),
  orderFeeServiceUpdateFee: vi.fn(),
  orderFeeServiceListOrderFeeSupplementRequests: vi
    .fn()
    .mockResolvedValue({ data: { items: [], total: 0 }, success: true }),
  orderFeeServiceCreateOrderFeeSupplement: vi.fn(),
  orderFeeServiceApproveOrderFeeSupplement: vi.fn(),
  orderFeeServiceRejectOrderFeeSupplement: vi.fn(),
  orderFeeServiceWithdrawOrderFeeSupplement: vi.fn(),
  orderFeeServiceCancelApprovedOrderFeeSupplement: vi.fn(),
}));

vi.mock('./use-order-fee-options', () => ({
  useOrderFeeOptions: (orderId?: string) => ({
    loading: false,
    order: orderId
      ? {
          id: orderId,
          orderNo: orderId.replace('order-', 'ORDER-'),
          organizationId: `organization-${orderId}`,
          businessType: 1,
        }
      : undefined,
    currencies: [],
    settlementParties: [],
    setSettlementParties: vi.fn(),
    feeSettings: [],
    setFeeSettings: vi.fn(),
    billingUnits: [],
    financeLocked: false,
    financeLockReason: undefined,
    financeLockCommissionNos: [],
    customerName: '',
    loadData: vi.fn(),
  }),
}));

vi.mock('./use-order-lock-state', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('./use-order-lock-state')>();
  return {
    ...actual,
    useOrderLockState: () => ({
      state: feeTestState.lockState,
      loading: false,
      error: null,
      refresh: vi.fn(),
    }),
  };
});

vi.mock('./use-fee-exchange-preview', () => ({
  useFeeExchangePreview: () => ({
    totalPreview: undefined,
    exchangeRatePreview: undefined,
    exchangeRateStatus: 'idle',
    manualExchangeRate: false,
    setManualExchangeRate: vi.fn(),
    resetPreview: feeTestState.resetPreview,
    seedFromFee: vi.fn(),
    handleValuesChange: vi.fn(),
  }),
}));

vi.mock('./components/OrderPageHeader', () => ({
  default: ({ orderId, orderNo }: any) => (
    <div data-testid="order-page-header">
      {orderId}|{orderNo}
    </div>
  ),
}));

vi.mock('./components/fees/OrderFeeHeader', () => ({
  default: ({ receivableSummary, payableSummary }: any) => (
    <div data-testid="fee-summary">
      {receivableSummary.count}:{receivableSummary.totalAmount}|
      {payableSummary.count}:{payableSummary.totalAmount}
    </div>
  ),
}));

vi.mock('./components/fees/OrderFeeTableTabs', () => ({
  default: (props: any) => (
    <div>
      <div data-testid="fee-table-state">
        {props.orderId}|{props.selectedReceivableFeeIds.join(',')}|
        {props.selectedPayableFeeIds.join(',')}
      </div>
      <button
        type="button"
        onClick={() => {
          const receivableId = `receivable-${props.orderId}`;
          const payableId = `payable-${props.orderId}`;
          props.setSelectedReceivableFeeIds([receivableId]);
          props.setSelectedPayableFeeIds([payableId]);
          props.setAllReceivableItems([{ id: receivableId }]);
          props.setAllPayableItems([{ id: payableId }]);
          props.setReceivableSummary({ totalAmount: 120, count: 1 });
          props.setPayableSummary({ totalAmount: 20, count: 1 });
          props.onOpenBillWorkbench([receivableId]);
        }}
      >
        准备当前订单费用
      </button>
      <button
        type="button"
        onClick={() =>
          props.onOpenFeeModal(1, { id: `editing-${props.orderId}` })
        }
      >
        编辑当前订单费用
      </button>
    </div>
  ),
}));

vi.mock('./components/fees/orderFeeColumns', () => ({
  getOrderFeeTableColumns: () => [],
}));

vi.mock('./components/fees/FeeFormModal', () => ({
  default: ({
    open,
    editingFee,
    onOpenQuickAddFee,
    onOpenQuickAddPartner,
  }: any) => (
    <div data-testid="fee-form-modal">
      {String(open)}|{editingFee?.id || ''}
      <button type="button" onClick={onOpenQuickAddFee}>
        打开快捷费目
      </button>
      <button type="button" onClick={onOpenQuickAddPartner}>
        打开快捷往来单位
      </button>
    </div>
  ),
}));

vi.mock('./components/fees/QuickAddFeeModal', () => ({
  default: ({ open, taxableServices }: any) => (
    <div data-testid="quick-fee-modal">
      {String(open)}|
      {taxableServices.map((item: API.TaxableService) => item.id).join(',')}
    </div>
  ),
}));

vi.mock('./components/fees/QuickAddPartnerModal', () => ({
  default: ({ open }: any) => (
    <div data-testid="quick-partner-modal">{String(open)}</div>
  ),
}));

// 费用页已迁移 React Query（快捷费目候选项查询）：渲染必须包
// QueryClientProvider；rerender 需复用同一 client，保持在途请求语义。
function renderFeesPage() {
  return renderWithClient(
    <App>
      <OrderFeesPage />
    </App>,
  );
}

function rerenderFeesPage(
  rerender: (ui: React.ReactNode) => void,
  queryClient: ReturnType<typeof renderWithClient>['queryClient'],
) {
  rerender(
    <QueryClientProvider client={queryClient}>
      <App>
        <OrderFeesPage />
      </App>
    </QueryClientProvider>,
  );
}

describe('订单费用页跨订单状态隔离', () => {
  const listTaxableServices = vi.mocked(feeCatalogServiceListTaxableServices);

  beforeEach(() => {
    mockParams = { kind: 'sea-export', id: 'order-A' };
    feeTestState.lockState = { isLocked: false };
    feeTestState.canCreateFee = true;
    feeTestState.canOperate = true;
    vi.clearAllMocks();
  });

  it('A 的快捷费目迟到响应不得覆盖 B 当前弹窗数据', async () => {
    let resolveA!: (value: any) => void;
    let resolveB!: (value: any) => void;
    const responseA = new Promise<any>((resolve) => {
      resolveA = resolve;
    });
    const responseB = new Promise<any>((resolve) => {
      resolveB = resolve;
    });
    listTaxableServices
      .mockImplementationOnce(() => responseA)
      .mockImplementationOnce(() => responseB);

    const { rerender, queryClient } = renderFeesPage();
    fireEvent.click(screen.getByRole('button', { name: '打开快捷费目' }));
    await waitFor(() => expect(listTaxableServices).toHaveBeenCalledTimes(1));

    mockParams = { kind: 'sea-export', id: 'order-B' };
    rerenderFeesPage(rerender, queryClient);
    fireEvent.click(screen.getByRole('button', { name: '打开快捷费目' }));
    await waitFor(() => expect(listTaxableServices).toHaveBeenCalledTimes(2));

    resolveB({ data: [{ id: 'taxable-B' }] });
    await waitFor(() =>
      expect(screen.getByTestId('quick-fee-modal')).toHaveTextContent(
        'true|taxable-B',
      ),
    );

    resolveA({ data: [{ id: 'taxable-A' }] });
    await waitFor(() =>
      expect(screen.getByTestId('quick-fee-modal')).toHaveTextContent(
        'true|taxable-B',
      ),
    );
    expect(screen.getByTestId('quick-fee-modal')).not.toHaveTextContent(
      'taxable-A',
    );
  });

  it('同一页面实例从 A 切到 B 时关闭工作台并清空 A 的费用状态', async () => {
    const { rerender, queryClient } = renderFeesPage();

    fireEvent.click(screen.getByRole('button', { name: '准备当前订单费用' }));
    fireEvent.click(screen.getByRole('button', { name: '编辑当前订单费用' }));
    fireEvent.click(screen.getByRole('button', { name: '打开快捷费目' }));
    fireEvent.click(screen.getByRole('button', { name: '打开快捷往来单位' }));

    expect(screen.getByTestId('fee-table-state')).toHaveTextContent(
      'order-A|receivable-order-A|payable-order-A',
    );
    expect(screen.getByTestId('fee-summary')).toHaveTextContent('1:120|1:20');
    expect(screen.getByTestId('finance-summary-board')).toHaveTextContent(
      'receivable-order-A,payable-order-A|receivable-order-A,payable-order-A',
    );
    expect(screen.getByTestId('bill-workbench')).toHaveTextContent(
      'true|receivable-order-A|organization-order-A|订单 ORDER-A',
    );
    expect(
      screen.getByRole('button', { name: '提交账单' }),
    ).toBeInTheDocument();
    expect(screen.getByTestId('fee-form-modal')).toHaveTextContent(
      'true|editing-order-A',
    );
    expect(screen.getByTestId('quick-fee-modal')).toHaveTextContent('true');
    expect(screen.getByTestId('quick-partner-modal')).toHaveTextContent('true');

    mockParams = { kind: 'sea-export', id: 'order-B' };
    rerenderFeesPage(rerender, queryClient);

    await waitFor(() => {
      expect(screen.getByTestId('fee-table-state')).toHaveTextContent(
        'order-B||',
      );
      expect(screen.getByTestId('fee-summary')).toHaveTextContent('0:0|0:0');
      expect(screen.getByTestId('finance-summary-board')).toHaveTextContent(
        '|',
      );
      expect(screen.getByTestId('bill-workbench')).toHaveTextContent(
        'false|||订单 ORDER-B',
      );
      expect(
        screen.queryByRole('button', { name: '提交账单' }),
      ).not.toBeInTheDocument();
      expect(screen.getByTestId('fee-form-modal')).toHaveTextContent('false|');
      expect(screen.getByTestId('quick-fee-modal')).toHaveTextContent('false');
      expect(screen.getByTestId('quick-partner-modal')).toHaveTextContent(
        'false',
      );
    });
  });
});

describe('订单费用页锁后费用补录', () => {
  const listSupplements = vi.mocked(
    orderFeeServiceListOrderFeeSupplementRequests,
  );

  function renderPage() {
    return renderFeesPage();
  }
  beforeEach(() => {
    mockParams = { kind: 'sea-export', id: 'order-A' };
    feeTestState.lockState = { isLocked: false };
    feeTestState.canCreateFee = true;
    feeTestState.canOperate = true;
    vi.clearAllMocks();
    listSupplements.mockResolvedValue({
      data: { items: [], total: 0 },
      success: true,
    } as Awaited<ReturnType<typeof listSupplements>>);
  });

  it('订单锁定时普通费用写入口关闭，但按 fee.create 能力开放补录入口', async () => {
    feeTestState.lockState = { isLocked: true };
    renderPage();

    // 补录申请区不依赖 fee.read，申请列表照常读取。
    await waitFor(() =>
      expect(listSupplements).toHaveBeenCalledWith({
        orderId: 'order-A',
        page: 1,
        pageSize: 10,
      }),
    );
    expect(
      await screen.findByRole('button', { name: '补录费用' }),
    ).toBeInTheDocument();
    // 普通新增入口在锁定时保持禁用（feeWritesDisabled 传递为 true）。
    await waitFor(() => {
      const tabs = screen.getByTestId('fee-table-state');
      expect(tabs).toBeInTheDocument();
    });
    expect(screen.getByTestId('fee-form-modal')).toHaveTextContent('false|');
  });

  it('无 fee.create 能力时不展示补录入口，但申请列表仍可读取', async () => {
    feeTestState.lockState = { isLocked: true };
    feeTestState.canCreateFee = false;
    renderPage();

    await waitFor(() => expect(listSupplements).toHaveBeenCalled());
    expect(
      screen.queryByRole('button', { name: '补录费用' }),
    ).not.toBeInTheDocument();
  });

  it('订单未锁定时不展示补录入口', async () => {
    renderPage();

    await waitFor(() => expect(listSupplements).toHaveBeenCalled());
    expect(
      screen.queryByRole('button', { name: '补录费用' }),
    ).not.toBeInTheDocument();
  });
  it('跨公司只读订单有费用权限也不能补录，仍可查看申请列表', async () => {
    feeTestState.canOperate = false;
    feeTestState.lockState = { isLocked: true };
    renderPage();
    await waitFor(() => expect(listSupplements).toHaveBeenCalled());
    expect(
      screen.queryByRole('button', { name: '补录费用' }),
    ).not.toBeInTheDocument();
  });
});
