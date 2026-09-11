import { render, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import BillDetailDrawer from './BillDetailDrawer';

describe('BillDetailDrawer 结算账户快照', () => {
  it('只展示账单 DTO 固化的账户快照，不请求往来单位账户主数据', () => {
    render(
      <BillDetailDrawer
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
  });

  it('展示固化的预计开票币种、预计开票汇率和预计开票金额', () => {
    render(
      <BillDetailDrawer
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
    expect(screen.getByText('712.34567900 CNY')).toBeInTheDocument();
  });
});
