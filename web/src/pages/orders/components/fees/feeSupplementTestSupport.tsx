import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render as originalRender } from '@testing-library/react';
import React from 'react';
import { beforeEach, vi } from 'vitest';
import {
  orderFeeServiceApproveOrderFeeSupplement,
  orderFeeServiceCancelApprovedOrderFeeSupplement,
  orderFeeServiceCreateOrderFeeSupplement,
  orderFeeServiceListOrderFeeSupplementRequests,
  orderFeeServicePreviewOrderFeeSupplementApproval,
  orderFeeServiceRejectOrderFeeSupplement,
  orderFeeServiceWithdrawOrderFeeSupplement,
} from '@/services/roncin/orderFeeService';
import type { FeeSupplementFormValues } from './FeeSupplementModal';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
function render(ui: React.ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return originalRender(ui, {
    wrapper: ({ children }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    ),
  });
}

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListOrderFeeSupplementRequests: vi.fn(),
  orderFeeServicePreviewOrderFeeSupplementApproval: vi.fn(),
  orderFeeServiceCreateOrderFeeSupplement: vi.fn(),
  orderFeeServiceApproveOrderFeeSupplement: vi.fn(),
  orderFeeServiceRejectOrderFeeSupplement: vi.fn(),
  orderFeeServiceWithdrawOrderFeeSupplement: vi.fn(),
  orderFeeServiceCancelApprovedOrderFeeSupplement: vi.fn(),
}));

// 弹窗以桩件替换，专注测试提交后服务调用与反馈；表单校验单独用真实弹窗覆盖。
vi.mock('./FeeSupplementModal', () => ({
  default: ({
    open,
    onSubmit,
    initialRequest,
  }: {
    open: boolean;
    onSubmit: (values: FeeSupplementFormValues) => Promise<boolean>;
    initialRequest?: API.OrderFeeSupplementRequestData;
  }) => (
    <div data-testid="supplement-modal" data-prefill-id={initialRequest?.id}>
      {String(open)}
      <button
        type="button"
        onClick={() =>
          onSubmit({
            feeSettingId: 'fs-1',
            settlementPartyId: 'sp-1',
            billingUnitId: 'bu-1',
            quantity: '2',
            unitPrice: '50',
            currency: 'CNY',
            expenseDate: '2026-09-01',
            reason: '漏录拖车费',
          })
        }
      >
        提交补录
      </button>
    </div>
  ),
}));

const listSupplements = vi.mocked(
  orderFeeServiceListOrderFeeSupplementRequests,
);
const createSupplement = vi.mocked(orderFeeServiceCreateOrderFeeSupplement);
const approveSupplement = vi.mocked(orderFeeServiceApproveOrderFeeSupplement);
const previewSupplement = vi.mocked(
  orderFeeServicePreviewOrderFeeSupplementApproval,
);
const rejectSupplement = vi.mocked(orderFeeServiceRejectOrderFeeSupplement);
const withdrawSupplement = vi.mocked(orderFeeServiceWithdrawOrderFeeSupplement);
const cancelApproved = vi.mocked(
  orderFeeServiceCancelApprovedOrderFeeSupplement,
);

function supplementRow(
  overrides: Partial<API.OrderFeeSupplementRequestData> = {},
): API.OrderFeeSupplementRequestData {
  return {
    id: 'sup-1',
    orderId: 'order-1',
    lockBasis: 'BOTH',
    status: 'PENDING',
    version: '3',
    direction: 2,
    feeCode: 'OCEAN',
    feeName: '海运费',
    quantity: '2',
    unitPrice: '50',
    totalAmount: '100',
    currency: 'CNY',
    expenseDate: '2026-09-01',
    reason: '漏录拖车费',
    requestedBy: 'user-a',
    requestedByName: '张发起',
    requestedAt: '2026-09-01T10:00:00Z',
    canApprove: false,
    canWithdraw: false,
    canCancel: false,
    approverAvailable: true,
    ...overrides,
  };
}

function listResponse(items: API.OrderFeeSupplementRequestData[]) {
  return {
    success: true,
    data: { items, total: items.length, page: 1, pageSize: 10 },
  } as Awaited<ReturnType<typeof listSupplements>>;
}

function conflictError(reason: string, message: string) {
  const error = new Error(message) as Error & {
    data?: { reason: string; message: string };
  };
  error.data = { reason, message };
  return error;
}

function makeProps(orderId: string) {
  return {
    orderId,
    canCreate: true,
    feeReviewApproved: true,
    hasPendingSupplement: false,
    feeSettings: [],
    settlementParties: [],
    currencies: [],
    billingUnits: [],
    onFeeTablesReload: vi.fn(),
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  listSupplements.mockResolvedValue(listResponse([]));
  previewSupplement.mockResolvedValue({
    data: {
      baseCurrency: 'USD',
      previewToken: 'supplement-preview',
      currentReceivable: '1000.00000000',
      currentPayable: '600.00000000',
      currentProfit: '400.00000000',
      currentProfitRate: '40.0000',
      supplementCost: '100.00000000',
      projectedReceivable: '1000.00000000',
      projectedPayable: '700.00000000',
      projectedProfit: '300.00000000',
      projectedProfitRate: '30.0000',
      profitChange: '-100.00000000',
    },
  } as Awaited<ReturnType<typeof previewSupplement>>);
});

export {
  approveSupplement,
  cancelApproved,
  conflictError,
  createSupplement,
  listResponse,
  listSupplements,
  makeProps,
  previewSupplement,
  rejectSupplement,
  render,
  supplementRow,
  withdrawSupplement,
};
