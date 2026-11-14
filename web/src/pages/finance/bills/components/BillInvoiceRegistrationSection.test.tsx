import { render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import BillInvoiceRegistrationSection from './BillInvoiceRegistrationSection';

const state = vi.hoisted(() => ({
  invoice: {} as API.FinanceInvoice,
  request: undefined as
    | undefined
    | ((params: { current: number; pageSize: number }) => Promise<unknown>),
  list: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canReadFinanceInvoices: true,
    canCreateFinanceInvoices: false,
    canOperateOrganization: (id: string) => id === 'company',
  }),
}));
vi.mock('@/components/ui', () => ({
  SectionCard: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  ResponsiveProTable: (props: {
    columns: {
      title?: string;
      render?: (unused: unknown, invoice: API.FinanceInvoice) => ReactNode;
    }[];
    request: (params: {
      current: number;
      pageSize: number;
    }) => Promise<unknown>;
  }) => {
    state.request = props.request;
    return (
      <div>
        {props.columns
          .filter((column) => column.render)
          .map((column) => (
            <div key={column.title}>
              {column.render?.(undefined, state.invoice)}
            </div>
          ))}
      </div>
    );
  },
}));
vi.mock('@/features/finance/invoice-registration', () => ({
  InvoiceRegistrationModal: () => null,
  InvoiceAllocationDetails: () => null,
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListInvoices: state.list,
}));
beforeEach(() => {
  state.list.mockReset();
  state.list.mockResolvedValue({ success: true, data: [], total: '0' });
  state.invoice = {
    id: 'invoice',
    currency: 'CNY',
    billLinks: [
      {
        id: 'old',
        billId: 'bill',
        billNo: 'BI-001',
        amount: '40',
        active: false,
      },
      {
        id: 'current',
        billId: 'bill',
        billNo: 'BI-001',
        amount: '70',
        active: true,
      },
    ],
  };
});
describe('账单侧发票登记投影', () => {
  it('修订后只显示当前有效分配，账单名称去重，查询精准固定源账单', async () => {
    render(
      <BillInvoiceRegistrationSection
        active
        bill={{ id: 'bill', version: '3', organizationId: 'company' }}
      />,
    );
    expect(screen.getByText('70.00 CNY')).toBeInTheDocument();
    expect(screen.queryByText('110.00 CNY')).not.toBeInTheDocument();
    expect(screen.getByText('BI-001')).toBeInTheDocument();
    await state.request?.({ current: 2, pageSize: 10 });
    expect(state.list).toHaveBeenCalledWith({
      organizationId: 'company',
      billId: 'bill',
      page: 2,
      pageSize: 10,
    });
  });
  it('只有已释放历史时明确释放口径，不把旧修订相加', () => {
    state.invoice.billLinks = state.invoice.billLinks?.map((link) => ({
      ...link,
      active: false,
    }));
    render(
      <BillInvoiceRegistrationSection
        active
        bill={{ id: 'bill', version: '4', organizationId: 'company' }}
      />,
    );
    expect(screen.getByText('已释放')).toBeInTheDocument();
    expect(screen.getByText('已释放，历史保留')).toBeInTheDocument();
    expect(screen.queryByText(/110.00/)).not.toBeInTheDocument();
  });
});
