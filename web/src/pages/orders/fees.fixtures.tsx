import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { App } from 'antd';
import React from 'react';
import { beforeEach, vi } from 'vitest';
import { OrderClosureStatus } from '@/enums.generated';
import OrderFeesPage from './fees';

export const feeRouteState = { params: { kind: 'sea-export', id: 'order-A' } };

const feeTestState = vi.hoisted(() => ({
  listTaxableServices: vi.fn().mockResolvedValue({ data: [] }),
  listCorrections: vi
    .fn()
    .mockResolvedValue({ success: true, data: [], total: 0 }),
  getFeeLedgerOrderDetail: vi.fn(),
  lockState: { isLocked: false } as Partial<API.OrderLockStateData> | null,
  canCreateFee: true,
  canReadFee: true,
  canLock: true,
  canReadFinanceFees: false,
  feeOptions: vi.fn(),
  canOperate: true,
  hasPendingSupplement: false,
  closureStatus: 1,
  feeChanged: false,
  rebaseline: vi.fn(),
  acceptOwnSave: vi.fn(),
  refreshLockState: vi.fn(),
  lockStateLoading: false,
  lockStateRefreshing: false,
  lockStateError: null as Error | null,
}));

beforeEach(() => {
  feeTestState.closureStatus = OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN;
  feeTestState.feeChanged = false;
  feeTestState.rebaseline.mockReset();
  feeTestState.acceptOwnSave.mockReset();
  feeTestState.refreshLockState.mockReset();
  feeTestState.lockStateLoading = false;
  feeTestState.lockStateRefreshing = false;
  feeTestState.lockStateError = null;
});

// 本文件验证费用页已有编辑与跨订单状态；审核流程由独立组件用例验证。
vi.mock('@/features/orders/fee-review', async (original) => ({
  ...(await original<typeof import('@/features/orders/fee-review')>()),
  useOrderFeeReview: () => ({
    data: {
      canEdit: !feeTestState.lockState?.isLocked,
      canBill: !feeTestState.hasPendingSupplement,
      hasPendingSupplement: feeTestState.hasPendingSupplement,
      hasPendingChange: feeTestState.hasPendingSupplement,
      status: feeTestState.lockState?.isLocked ? 'APPROVED' : 'UNSUBMITTED',
    },
    refetch: vi.fn(),
  }),
}));
vi.mock('@/router/history', () => ({
  history: {
    get location() {
      return window.location;
    },
    push: vi.fn(),
  },
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => feeTestState.canOperate,
    canOperateBusiness: true,
    canCreateFinanceBills: true,
    canReadFinanceFees: feeTestState.canReadFinanceFees,
    canReviewOrderFees: feeTestState.canLock,
    canOrder: (_: unknown, operation: string) =>
      operation === 'fee.read'
        ? feeTestState.canReadFee
        : operation === 'lock'
          ? feeTestState.canLock
          : feeTestState.canCreateFee,
  }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user-1', currentOrganization: { id: 'org-1' } },
    },
  }),
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useParams: () => feeRouteState.params,
  };
});

vi.mock('@/components/ui', async () => ({
  GuardedModalForm: (await import('@/components/ui/GuardedModalForm'))
    .GuardedModalForm,
  ResponsiveProTable: (await import('@ant-design/pro-components')).ProTable,
  FinanceSummaryBoard: ({
    selected,
    summary,
  }: import('@/components/ui/finance-ledger-template').FinanceSummaryBoardProps) => (
    <div data-testid="finance-summary-board">
      {String(selected)}|{summary?.count}|
      {summary?.amounts.map((item) => item.amount).join(',')}
    </div>
  ),
  SectionCard: (await import('@/components/ui/page-shell/SectionCard'))
    .SectionCard,
  ProFormSearchableSelect: (props: {
    name?: string;
    label?: React.ReactNode;
  }) => <div data-testid={`ui-select-${props.name}`}>{props.label}</div>,
  ExchangeRatePreviewCard: () => <div data-testid="exchange-preview" />,
}));

vi.mock('@/features/finance/bill-creation', () => ({
  BillCreationWorkbench: ({
    open,
    initialFeeIds,
    initialOrganizationId,
    sourceLabel,
  }: React.ComponentProps<
    typeof import('@/features/finance/bill-creation').BillCreationWorkbench
  >) => {
    const [workbenchFeeIds, setWorkbenchFeeIds] = React.useState<string[]>([]);
    React.useEffect(() => {
      if (open) setWorkbenchFeeIds(initialFeeIds ?? []);
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
  feeCatalogServiceListTaxableServices: feeTestState.listTaxableServices,
}));

vi.mock('@/services/roncin/orderFeeService', async (original) => ({
  ...(await original<typeof import('@/services/roncin/orderFeeService')>()),
  orderFeeServiceListOrderFeeCorrectionRequests: feeTestState.listCorrections,
  orderFeeServicePreviewOrderFeeReview: vi.fn().mockResolvedValue({
    data: { canEdit: true, canBill: true, status: 'UNSUBMITTED' },
  }),
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
  useOrderFeeOptions: (orderId?: string) => {
    feeTestState.feeOptions(orderId);
    return {
      loading: false,
      order: orderId
        ? {
            id: orderId,
            orderNo: orderId.replace('order-', 'ORDER-'),
            organizationId: `organization-${orderId}`,
            businessType: 1,
            closureStatus: feeTestState.closureStatus,
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
    };
  },
}));

vi.mock('./use-order-lock-state', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('./use-order-lock-state')>();
  return {
    ...actual,
    useOrderLockState: () => ({
      state: feeTestState.lockState,
      loading: feeTestState.lockStateLoading,
      refreshing: feeTestState.lockStateRefreshing,
      error: feeTestState.lockStateError,
      refresh: feeTestState.refreshLockState,
    }),
  };
});

vi.mock('./use-fee-change-notice', () => ({
  useFeeChangeNotice: () => ({
    changed: feeTestState.feeChanged,
    rebaseline: feeTestState.rebaseline,
    acceptOwnSave: feeTestState.acceptOwnSave,
  }),
}));

vi.mock('./components/OrderPageHeader', () => ({
  default: ({
    orderId,
    orderNo,
    extra,
  }: React.ComponentProps<
    typeof import('./components/OrderPageHeader').default
  >) => (
    <div data-testid="order-page-header">
      {orderId}|{orderNo}
      {extra}
    </div>
  ),
}));

vi.mock('./components/fees/OrderFeeHeader', () => ({
  default: ({
    receivableSummary,
    payableSummary,
  }: React.ComponentProps<
    typeof import('./components/fees/OrderFeeHeader').default
  >) => (
    <div data-testid="fee-summary">
      {receivableSummary?.count ?? 0}:{receivableSummary?.gross ?? 0}|
      {payableSummary?.count ?? 0}:{payableSummary?.gross ?? 0}
    </div>
  ),
}));

vi.mock('./components/fees/OrderFeeTableTabs', () => ({
  default: (
    props: React.ComponentProps<
      typeof import('./components/fees/OrderFeeTableTabs').default
    >,
  ) => (
    <div>
      <div data-testid="fee-table-state">
        {props.orderId}|{props.selectedReceivableFeeIds.join(',')}|
        {props.selectedPayableFeeIds.join(',')}
      </div>
      <div data-testid="fee-write-disabled">
        {String(props.feeWritesDisabled)}
      </div>
      <div data-testid="fee-rows-locked">{String(props.feeRowsLocked)}</div>
      <div data-testid="fee-bill-tracking">
        {props.feeBillTracking
          ? `${props.feeBillTracking.state}|${
              props.feeBillTracking.byFeeId[
                props.orderId === 'order-A' ? 'fee-A-1' : 'fee-B-1'
              ]?.billNo ?? '-'
            }`
          : 'none'}
      </div>
      <button type="button" onClick={() => props.onFeeSaved?.()}>
        通知行内保存
      </button>
      <button type="button" onClick={() => props.onFeeSaved?.('11')}>
        通知整表保存
      </button>
      <div data-testid="fee-reload-token">{props.reloadToken}</div>
      <button type="button" onClick={() => props.onDraftChange?.(true)}>
        产生费用草稿
      </button>
      <button
        type="button"
        onClick={() => {
          const receivableId = `receivable-${props.orderId}`;
          const payableId = `payable-${props.orderId}`;
          props.setSelectedReceivableFeeIds([receivableId]);
          props.setSelectedPayableFeeIds([payableId]);
          props.setAllReceivableItems([
            {
              id: receivableId,
              direction: 1,
              currency: 'CNY',
              totalAmount: '120',
              baseCurrency: 'CNY',
              baseCurrencyAmount: '120',
            },
          ]);
          props.setAllPayableItems([
            {
              id: payableId,
              direction: 2,
              currency: 'CNY',
              totalAmount: '20',
              baseCurrency: 'CNY',
              baseCurrencyAmount: '20',
            },
          ]);
          props.setReceivableSummary({
            count: 1,
            gross: '120',
            net: '120',
            unconvertedCount: 0,
            currencies: [],
          });
          props.setPayableSummary({
            count: 1,
            gross: '20',
            net: '20',
            unconvertedCount: 0,
            currencies: [],
          });
          props.onOpenBillWorkbench([receivableId]);
        }}
      >
        准备当前订单费用
      </button>
      <button type="button" onClick={() => props.onOpenQuickAddFee?.()}>
        打开快捷费目
      </button>
      <button type="button" onClick={() => props.onOpenQuickAddPartner?.(2)}>
        打开快捷往来单位
      </button>
    </div>
  ),
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetFeeLedgerOrderDetail:
    feeTestState.getFeeLedgerOrderDetail,
}));

vi.mock('./components/fees/QuickAddFeeModal', () => ({
  default: ({
    open,
    taxableServices,
  }: React.ComponentProps<
    typeof import('./components/fees/QuickAddFeeModal').default
  >) => (
    <div data-testid="quick-fee-modal">
      {String(open)}|
      {(taxableServices ?? [])
        .map((item: API.TaxableService) => item.id)
        .join(',')}
      {open ? <input aria-label="快捷费目草稿" /> : null}
    </div>
  ),
}));

vi.mock('./components/fees/QuickAddPartnerModal', () => ({
  default: ({
    open,
    defaultRole,
  }: React.ComponentProps<
    typeof import('./components/fees/QuickAddPartnerModal').default
  >) => (
    <div data-testid="quick-partner-modal">
      {String(open)}|{String(defaultRole)}
      {open ? <input aria-label="快捷往来草稿" /> : null}
    </div>
  ),
}));

// 费用页已迁移 React Query（快捷费目候选项查询）：渲染必须包
// QueryClientProvider；rerender 需复用同一 client，保持在途请求语义。
export function renderFeesPage() {
  return renderWithClient(
    <App>
      <OrderFeesPage />
    </App>,
  );
}

export function rerenderFeesPage(rerender: (ui: React.ReactNode) => void) {
  rerender(
    <App>
      <OrderFeesPage />
    </App>,
  );
}

export { feeTestState };
