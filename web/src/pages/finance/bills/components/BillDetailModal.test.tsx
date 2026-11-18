import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  FinanceBillBusinessStatus,
  FinanceBillStatus,
} from '@/enums.generated';
import { partnerServiceListPartnerAccounts } from '@/services/roncin/partnerService';
import BillDetailModal from './BillDetailModal';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartnerAccounts: vi.fn(),
}));

describe('BillDetailModal 账单详情', () => {
  it('有效账单状态与列表共用业务投影，取消保留终态，未投影不回退正常', async () => {
    const detail: API.FinanceBill = {
      id: 'bill-status',
      billNo: 'BILL-STATUS',
      status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
      businessStatus:
        FinanceBillBusinessStatus.FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_INVOICED,
      lines: [],
    };
    const onClose = vi.fn();
    const view = renderWithApp(
      <BillDetailModal
        open
        loading={false}
        onClose={onClose}
        detail={detail}
      />,
    );
    expect(screen.getByText('部分开票', { exact: true })).toBeInTheDocument();
    expect(screen.queryByText('正常', { exact: true })).not.toBeInTheDocument();
    view.rerender(
      <BillDetailModal
        open
        loading={false}
        onClose={onClose}
        detail={{
          ...detail,
          status: FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED,
        }}
      />,
    );
    expect(screen.getByText('已取消', { exact: true })).toBeInTheDocument();
    expect(
      screen.queryByText('部分开票', { exact: true }),
    ).not.toBeInTheDocument();
    view.rerender(
      <BillDetailModal
        open
        loading={false}
        onClose={onClose}
        detail={{ ...detail, businessStatus: undefined }}
      />,
    );
    expect(
      screen.queryByText('已取消', { exact: true }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('部分开票', { exact: true }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('正常', { exact: true })).not.toBeInTheDocument();
    await act(async () => {});
  });

  it('采纳费用变更响应后显示新的创建日期，首次创建时间不代替业务日期', async () => {
    const detail: API.FinanceBill = {
      id: 'bill-date',
      billNo: 'BILL-DATE',
      billDate: '2026-11-16',
      createdAt: '2026-11-01T08:00:00Z',
      exchangeRateDate: '2026-11-15',
      lines: [],
    };
    const onClose = vi.fn();
    const { rerender } = renderWithApp(
      <BillDetailModal
        open
        loading={false}
        onClose={onClose}
        detail={detail}
      />,
    );
    expect(screen.getByText('2026-11-16')).toBeInTheDocument();
    rerender(
      <BillDetailModal
        open
        loading={false}
        onClose={onClose}
        detail={{ ...detail, billDate: '2026-11-17' }}
      />,
    );
    expect(screen.getByText('创建日期')).toBeInTheDocument();
    expect(screen.getByText('2026-11-17')).toBeInTheDocument();
    expect(screen.queryByText('2026-11-16')).not.toBeInTheDocument();
    expect(screen.queryByText('2026-11-01')).not.toBeInTheDocument();
    await act(async () => {});
  });

  it('只读明细只展示有效费用，不显示已释放行及关联状态列', async () => {
    renderWithApp(
      <BillDetailModal
        open
        loading={false}
        onClose={vi.fn()}
        detail={{
          id: 'bill-lines',
          billNo: 'BILL-LINES',
          lines: [
            { id: 'active', feeName: '当前费用', active: true },
            { id: 'released', feeName: '已移除费用', active: false },
            { id: 'omitted', feeName: '省略零值的已移除费用' },
          ],
        }}
      />,
    );
    expect(screen.getByText('当前费用')).toBeInTheDocument();
    expect(screen.queryByText('已移除费用')).not.toBeInTheDocument();
    expect(screen.queryByText('省略零值的已移除费用')).not.toBeInTheDocument();
    expect(screen.queryByText('关联状态')).not.toBeInTheDocument();
    expect(screen.queryByText('已释放')).not.toBeInTheDocument();
    await act(async () => {});
  });

  it('只展示账单 DTO 固化的账户快照，不请求往来单位账户主数据', async () => {
    renderWithApp(
      <BillDetailModal
        open
        loading={false}
        onClose={vi.fn()}
        detail={{
          id: 'bill-1',
          billNo: 'BILL-1',
          settlementPartyName: '结算单位甲',
          settlementAccountName: '人民币账户',
          settlementAccountHolder: '单位甲',
          settlementBankName: '测试银行',
          settlementBankAccount: '62220001',
          settlementAccountCurrency: 'CNY',
          settlementSwiftCode: 'TESTCNBJ',
          lines: [],
        }}
      />,
    );
    expect(screen.getByText('人民币账户')).toBeInTheDocument();
    expect(screen.getByText('单位甲')).toBeInTheDocument();
    expect(screen.getByText('测试银行')).toBeInTheDocument();
    expect(screen.getByText('62220001')).toBeInTheDocument();
    expect(screen.getByText('TESTCNBJ')).toBeInTheDocument();
    expect(partnerServiceListPartnerAccounts).not.toHaveBeenCalled();
    await act(async () => {});
  });

  it('展示固化的预计开票币种、预计开票汇率和预计开票金额', async () => {
    renderWithApp(
      <BillDetailModal
        open
        loading={false}
        onClose={vi.fn()}
        detail={{
          id: 'bill-2',
          billNo: 'BILL-2',
          currency: 'USD',
          totalAmount: '100.00000000',
          estimatedInvoiceCurrency: 'CNY',
          estimatedInvoiceRate: '7.12345679',
          estimatedInvoiceAmount: '712.34567900',
          lines: [],
        }}
      />,
    );
    expect(screen.getByText('CNY')).toBeInTheDocument();
    expect(screen.getByText('7.12345679')).toBeInTheDocument();
    expect(screen.getByText('712.35 CNY')).toBeInTheDocument();
    await act(async () => {});
  });

  it('有编辑权限时详情内提供编辑账单信息入口，无权限则不显示', async () => {
    const onEdit = vi.fn();
    const detail = { id: 'bill-3', billNo: 'BILL-3', lines: [] };
    const view = renderWithApp(
      <BillDetailModal
        open
        loading={false}
        onClose={vi.fn()}
        detail={detail}
        canEdit
        onEdit={onEdit}
      />,
    );
    fireEvent.click(screen.getByText('编辑账单信息'));
    expect(onEdit).toHaveBeenCalledWith(detail);
    view.rerender(
      <BillDetailModal
        open
        loading={false}
        onClose={vi.fn()}
        detail={detail}
        onEdit={onEdit}
      />,
    );
    expect(screen.queryByText('编辑账单信息')).not.toBeInTheDocument();
    await act(async () => {});
  });
});
