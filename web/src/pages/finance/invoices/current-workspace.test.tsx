import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act } from '@testing-library/react';
import { App, Form, type FormInstance } from 'antd';
import type { Dispatch, SetStateAction } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  settlementServiceCreateInvoice,
  settlementServiceListInvoices,
} from '@/services/roncin/settlementService';
import FinanceInvoicesPage from './index';

type LedgerProps = {
  topBar?: unknown;
  columns: { title?: unknown }[];
  request: (params: {
    current: number;
    pageSize: number;
    keyword?: string;
    organizationId?: string;
  }) => Promise<unknown>;
};
type CreateModalProps = {
  createForm: FormInstance;
  setSelectedIDs: Dispatch<SetStateAction<React.Key[]>>;
  setSelectedBills: Dispatch<SetStateAction<API.FinanceBill[]>>;
  onOk: () => Promise<false | undefined>;
};
const state = vi.hoisted(() => ({
  organizationId: 'workspace-A' as string | undefined,
  company: true,
  ledger: undefined as LedgerProps | undefined,
  createModal: undefined as CreateModalProps | undefined,
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user',
        currentOrganization: { id: state.organizationId },
      },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateFinanceInvoices: true,
    canOperateOrganization: (id?: string) =>
      state.company && id === state.organizationId,
  }),
}));
vi.mock('@/components/ui', () => ({
  ResponsiveTable: () => null,
  FinanceLedgerTemplate: (props: LedgerProps) => {
    state.ledger = props;
    return null;
  },
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCreateInvoice: vi
    .fn()
    .mockResolvedValue({ data: { id: 'invoice' } }),
  settlementServiceListInvoices: vi
    .fn()
    .mockResolvedValue({ data: [], total: '0', matchedSummary: {} }),
}));
vi.mock('./components/InvoiceCreateModal', () => ({
  default: (props: CreateModalProps) => {
    state.createModal = props;
    return (
      <Form form={props.createForm}>
        <Form.Item name="invoiceProfileId">
          <input />
        </Form.Item>
        <Form.Item name="invoiceType">
          <input />
        </Form.Item>
      </Form>
    );
  },
}));
vi.mock('./components/InvoiceDetailDrawer', () => ({ default: () => null }));

beforeEach(() => {
  vi.clearAllMocks();
  state.company = true;
  state.organizationId = 'workspace-A';
});
describe('invoices列表当前公司', () => {
  it('请求固定当前公司，分页搜索传来的外公司值不能覆盖；公司展示列保留', async () => {
    renderWithClient(
      <App>
        <FinanceInvoicesPage />
      </App>,
    );
    expect(state.ledger?.topBar).toBeUndefined();
    expect(
      state.ledger?.columns.some((column) => column.title === '所属公司'),
    ).toBe(true);
    await state.ledger?.request({
      current: 2,
      pageSize: 20,
      keyword: '条件',
      organizationId: 'other',
    });
    expect(settlementServiceListInvoices).toHaveBeenLastCalledWith(
      expect.objectContaining({
        organizationId: 'workspace-A',
        page: 2,
        pageSize: 20,
        keyword: '条件',
      }),
      { skipErrorHandler: true },
    );
  });
  it.each(['未就绪', '系统工作台'])('%s不请求经营数据', async (scenario) => {
    if (scenario === '未就绪') state.organizationId = undefined;
    else state.company = false;
    renderWithClient(
      <App>
        <FinanceInvoicesPage />
      </App>,
    );
    await expect(
      state.ledger?.request({ current: 1, pageSize: 20 }),
    ).resolves.toMatchObject({ success: false });
    expect(settlementServiceListInvoices).not.toHaveBeenCalled();
  });
  it('工作区重建后分页使用新公司', async () => {
    const first = renderWithClient(
      <App>
        <FinanceInvoicesPage />
      </App>,
    );
    await state.ledger?.request({ current: 1, pageSize: 20 });
    first.unmount();
    state.organizationId = 'workspace-B';
    renderWithClient(
      <App>
        <FinanceInvoicesPage />
      </App>,
    );
    await state.ledger?.request({ current: 3, pageSize: 40 });
    expect(settlementServiceListInvoices).toHaveBeenLastCalledWith(
      expect.objectContaining({ organizationId: 'workspace-B', page: 3 }),
      { skipErrorHandler: true },
    );
  });
  it.each(['workspace-A', 'other-company', undefined])(
    '开票只接受当前公司来源账单：%s',
    async (billOrganizationId) => {
      renderWithClient(
        <App>
          <FinanceInvoicesPage />
        </App>,
      );
      act(() => {
        state.createModal?.setSelectedIDs(['bill']);
        state.createModal?.setSelectedBills([
          { id: 'bill', organizationId: billOrganizationId },
        ]);
        state.createModal?.createForm.setFieldsValue({
          invoiceProfileId: 'profile',
          invoiceType: 'VAT_SPECIAL',
        });
      });
      await act(async () => {
        await state.createModal?.onOk();
      });
      if (billOrganizationId === 'workspace-A') {
        expect(settlementServiceCreateInvoice).toHaveBeenCalledWith(
          expect.objectContaining({
            billIds: ['bill'],
            invoiceProfileId: 'profile',
          }),
        );
        expect(
          vi.mocked(settlementServiceCreateInvoice).mock.lastCall?.[0],
        ).not.toHaveProperty('organizationId');
      } else {
        expect(settlementServiceCreateInvoice).not.toHaveBeenCalled();
      }
    },
  );
});
