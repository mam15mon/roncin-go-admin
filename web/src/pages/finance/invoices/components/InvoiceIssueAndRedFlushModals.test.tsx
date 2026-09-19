import { render, screen } from '@testing-library/react';
import { Form } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { FinanceInvoiceStatus } from '@/enums.generated';
import {
  InvoiceIssueModal,
  InvoiceRedFlushModal,
} from './InvoiceIssueAndRedFlushModals';
import {
  invoiceIssueActionText,
  invoiceIssueDateLabel,
  invoiceIssueVerb,
  invoiceRecordNoun,
  invoiceStateText,
} from './invoiceConstants';

function IssueModalHarness({ direction }: { direction?: string }) {
  const [form] = Form.useForm();
  return (
    <InvoiceIssueModal
      open
      submitting={false}
      issueForm={form}
      issueTarget={{
        id: 'inv-1',
        version: '1',
        direction,
        organizationName: '验收组织',
      }}
      onCancel={vi.fn()}
      onOk={vi.fn()}
    />
  );
}

describe('发票方向感知文案（AC3）', () => {
  it('销项（应收）呈现确认开具与开票日期', () => {
    render(<IssueModalHarness direction="RECEIVABLE" />);
    expect(screen.getByText('确认开具 验收组织 的发票')).toBeInTheDocument();
    expect(screen.getByText('开票日期')).toBeInTheDocument();
  });

  it('进项（应付）呈现确认收票与收票日期', () => {
    render(<IssueModalHarness direction="PAYABLE" />);
    expect(screen.getByText('确认收票 验收组织 的发票')).toBeInTheDocument();
    expect(screen.getByText('收票日期')).toBeInTheDocument();
  });

  it('同一 ISSUED 状态按方向分别呈现已开具与已收票', () => {
    expect(
      invoiceStateText(
        FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED,
        'RECEIVABLE',
      ),
    ).toBe('已开具');
    expect(
      invoiceStateText(
        FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED,
        'PAYABLE',
      ),
    ).toBe('已收票');
  });

  it('动词、名词与日期标签的方向映射', () => {
    expect(invoiceIssueActionText('RECEIVABLE')).toBe('确认开具');
    expect(invoiceIssueActionText('PAYABLE')).toBe('确认收票');
    expect(invoiceIssueVerb('RECEIVABLE')).toBe('开具');
    expect(invoiceIssueVerb('PAYABLE')).toBe('收票');
    expect(invoiceRecordNoun('RECEIVABLE')).toBe('开票记录');
    expect(invoiceRecordNoun('PAYABLE')).toBe('收票记录');
    expect(invoiceIssueDateLabel('RECEIVABLE')).toBe('开票日期');
    expect(invoiceIssueDateLabel('PAYABLE')).toBe('收票日期');
  });

  it('红冲弹窗标题保持方向中性', () => {
    function RedFlushModalHarness() {
      const [form] = Form.useForm();
      return (
        <InvoiceRedFlushModal
          open
          submitting={false}
          redFlushTarget={{
            id: 'inv-2',
            version: '1',
            direction: 'PAYABLE',
            organizationName: '验收组织',
            taxInvoiceNo: 'T-001',
          }}
          redFlushForm={form}
          onCancel={vi.fn()}
          onOk={vi.fn()}
        />
      );
    }
    render(<RedFlushModalHarness />);
    expect(screen.getByText('红冲 验收组织 的发票 T-001')).toBeInTheDocument();
    expect(screen.getByText('红冲日期')).toBeInTheDocument();
  });
});
