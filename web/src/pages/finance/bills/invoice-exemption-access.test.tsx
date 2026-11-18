import { renderWithApp } from '@root/tests/renderWithApp';
import { act, waitFor } from '@testing-library/react';
import type { MutableRefObject } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind, FinanceBillStatus } from '@/enums.generated';
import type { BillInvoiceExemptionActions } from './components/BillInvoiceExemptionSection';
import FinanceBillsPage from './index';

const state = vi.hoisted(() => ({
  openDetail: undefined as
    | ((bill: API.FinanceBill) => Promise<void>)
    | undefined,
  drawer: undefined as
    | {
        detail?: API.FinanceBill;
        invoiceExemptionActions?: BillInvoiceExemptionActions;
      }
    | undefined,
  getBill: vi.fn(),
  reload: vi.fn(),
}));
vi.mock('@/components/ui', async (original) => ({
  ...(await original<typeof import('@/components/ui')>()),
  FinanceLedgerTemplate: ({
    actionRef,
  }: {
    actionRef: MutableRefObject<{ reload: () => void } | undefined>;
  }) => {
    actionRef.current = { reload: state.reload };
    return null;
  },
}));
vi.mock('@/features/finance/bill-creation', () => ({
  BillCreationWorkbench: () => null,
}));
vi.mock('./components/BillEditModal', () => ({ default: () => null }));
vi.mock('./components/BillDetailModal', () => ({
  default: (props: NonNullable<typeof state.drawer>) => {
    state.drawer = props;
    return null;
  },
}));
vi.mock('./components/billColumns', () => ({
  getFinanceBillColumns: ({
    onOpenDetail,
  }: {
    onOpenDetail: NonNullable<typeof state.openDetail>;
  }) => {
    state.openDetail = onOpenDetail;
    return [];
  },
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetBill: state.getBill,
}));

beforeEach(() => vi.clearAllMocks());

describe('账单页免票操作独立授权', () => {
  it.each([
    { organizationId: 'company-a', permission: true, enabled: true },
    { organizationId: 'company-b', permission: true, enabled: false },
    { organizationId: undefined, permission: true, enabled: false },
    { organizationId: 'company-a', permission: false, enabled: false },
  ])(
    '来源$organizationId、免票权限$permission时操作资格为$enabled',
    async ({ organizationId, permission, enabled }) => {
      const detail: API.FinanceBill = {
        id: 'bill-a',
        organizationId,
        version: '3',
        status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
      };
      state.getBill.mockResolvedValue({ success: true, data: detail });
      renderWithApp(<FinanceBillsPage />, {
        user: {
          id: 'user',
          currentOrganization: {
            id: 'company-a',
            kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
          },
          permissionCapabilities: permission
            ? [
                {
                  key: 'system.finance.invoice.exempt',
                  dataScope: 'organization',
                },
              ]
            : [],
        },
      });
      await act(async () => {
        await state.openDetail?.({ id: 'bill-a' });
      });
      await waitFor(() => expect(state.getBill).toHaveBeenCalledOnce());
      await waitFor(() => expect(state.drawer?.detail?.id).toBe('bill-a'));
      expect(Boolean(state.drawer?.invoiceExemptionActions)).toBe(enabled);
      if (!enabled) return;
      act(() =>
        state.drawer?.invoiceExemptionActions?.onUpdated({
          ...detail,
          version: '4',
        }),
      );
      await waitFor(() => expect(state.drawer?.detail?.version).toBe('4'));
      expect(state.reload).toHaveBeenCalledTimes(1);
      await act(async () => {
        state.drawer?.invoiceExemptionActions?.onRefresh();
      });
      await waitFor(() => expect(state.getBill).toHaveBeenCalledTimes(2));
    },
  );
});
