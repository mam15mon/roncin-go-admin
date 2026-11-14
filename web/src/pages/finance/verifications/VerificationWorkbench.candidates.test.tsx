import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import VerificationWorkbench from './VerificationWorkbench';

const api = vi.hoisted(() => ({
  cashflows: vi.fn(),
  bills: vi.fn(),
  create: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateFinanceVerifications: true,
    canOperateOrganization: (id: string) => id === 'company-a',
  }),
}));
vi.mock('@/features/finance/credit-control', () => ({
  useCreditLimitIntervention: () => false,
  disableCreditExceededOptions: (options: unknown) => options,
}));
vi.mock('@/features/master-data/currencies', async (original) => ({
  ...(await original<typeof import('@/features/master-data/currencies')>()),
  getCurrencyOptions: vi
    .fn()
    .mockResolvedValue([{ label: '人民币', value: 'CNY' }]),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFinanceSettlementPartyOptions: vi
    .fn()
    .mockResolvedValue({ data: [{ id: 'party-a', code: 'P', name: '远航' }] }),
  settlementServiceListVerificationCashflowCandidates: api.cashflows,
  settlementServiceListVerificationBillCandidates: api.bills,
  settlementServiceCreateVerification: api.create,
}));

beforeEach(() => {
  vi.clearAllMocks();
  api.cashflows.mockImplementation(async ({ page, keyword }) => ({
    success: true,
    total: keyword ? 0 : 205,
    data: keyword
      ? []
      : page === 1
        ? [
            {
              id: 'cash-1',
              flowNo: 'FLOW-1',
              unverifiedAmount: '60',
              organizationId: 'company-a',
            },
          ]
        : [
            {
              id: 'cash-2',
              flowNo: 'FLOW-2',
              unverifiedAmount: '40',
              organizationId: 'company-a',
            },
          ],
  }));
  api.bills.mockImplementation(async ({ page, keyword }) => ({
    success: true,
    total: keyword ? 0 : 205,
    data: keyword
      ? []
      : [
          {
            id: page === 1 ? 'bill-1' : 'bill-2',
            billNo: page === 1 ? 'BILL-1' : 'BILL-2',
            unverifiedAmount: '50',
            organizationId: 'company-a',
          },
        ],
  }));
  api.create.mockResolvedValue({
    success: true,
    data: { id: 'verification-a' },
  });
});

describe('核销候选官方分页与搜索', () => {
  it('分别请求第二页，跨页勾选和搜索后仅按明确选中的来源分配提交', async () => {
    const onCreated = vi.fn();
    renderWithApp(
      <VerificationWorkbench open onClose={vi.fn()} onCreated={onCreated} />,
      {
        user: {
          id: 'operator',
          currentOrganization: {
            id: 'company-a',
            kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
          },
        },
      },
    );
    expect(api.cashflows).not.toHaveBeenCalled();
    expect(api.bills).not.toHaveBeenCalled();
    fireEvent.mouseDown(screen.getByRole('combobox', { name: '结算单位' }));
    fireEvent.click(await screen.findByText('远航 (P)'));
    const cashPanel = (
      await screen.findByText('待核销资金')
    ).closest<HTMLElement>('.ant-card');
    const billPanel = screen
      .getByText('待核销账单')
      .closest<HTMLElement>('.ant-card');
    if (!cashPanel || !billPanel) throw new Error('未渲染候选面板');
    const cash1 = (await screen.findByText('FLOW-1')).closest('tr');
    const bill1 = (await screen.findByText('BILL-1')).closest('tr');
    if (!cash1 || !bill1) throw new Error('未渲染第一页候选');
    fireEvent.click(within(cash1).getByRole('checkbox'));
    fireEvent.click(within(bill1).getByRole('checkbox'));
    fireEvent.click(within(cashPanel).getByTitle('下一页'));
    fireEvent.click(within(billPanel).getByTitle('下一页'));
    const cash2 = (await screen.findByText('FLOW-2')).closest('tr');
    const bill2 = (await screen.findByText('BILL-2')).closest('tr');
    if (!cash2 || !bill2) throw new Error('未渲染第二页候选');
    fireEvent.click(within(cash2).getByRole('checkbox'));
    fireEvent.click(within(bill2).getByRole('checkbox'));
    expect(api.cashflows).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 2,
        pageSize: 20,
        organizationId: 'company-a',
      }),
      { skipErrorHandler: true },
    );
    expect(api.bills).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 2,
        pageSize: 20,
        organizationId: 'company-a',
      }),
      { skipErrorHandler: true },
    );
    const searchCash = within(cashPanel).getByPlaceholderText(
      '搜索流水号／单位／银行参考号',
    );
    fireEvent.change(searchCash, { target: { value: '不存在的候选' } });
    fireEvent.keyDown(searchCash, { key: 'Enter', keyCode: 13 });
    const searchBill = within(billPanel).getByPlaceholderText(
      '搜索账单号／单位／订单号',
    );
    fireEvent.change(searchBill, { target: { value: '不存在的候选' } });
    fireEvent.keyDown(searchBill, { key: 'Enter', keyCode: 13 });
    await waitFor(() =>
      expect(api.cashflows).toHaveBeenCalledWith(
        expect.objectContaining({ keyword: '不存在的候选', page: 1 }),
        { skipErrorHandler: true },
      ),
    );
    await waitFor(() =>
      expect(api.bills).toHaveBeenCalledWith(
        expect.objectContaining({ keyword: '不存在的候选', page: 1 }),
        { skipErrorHandler: true },
      ),
    );
    fireEvent.click(screen.getByRole('button', { name: /按余额自动分配/ }));
    fireEvent.click(screen.getByRole('button', { name: /提交核销/ }));
    await waitFor(() => expect(api.create).toHaveBeenCalledOnce());
    expect(api.create.mock.calls[0][0].allocations).toEqual([
      { cashflowId: 'cash-1', billId: 'bill-1', amount: '50.00000000' },
      { cashflowId: 'cash-1', billId: 'bill-2', amount: '10.00000000' },
      { cashflowId: 'cash-2', billId: 'bill-2', amount: '40.00000000' },
    ]);
    await waitFor(() => expect(onCreated).toHaveBeenCalledOnce());
  });
});
