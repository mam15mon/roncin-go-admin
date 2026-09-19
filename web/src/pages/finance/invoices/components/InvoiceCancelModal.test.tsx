import { render, screen } from '@testing-library/react';
import { Form } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { FinanceInvoiceStatus } from '@/enums.generated';
import InvoiceCancelModal from './InvoiceCancelModal';
import {
  invoiceCancelSuccessText,
  invoiceDraftCancelTitle,
  invoiceVoidSuccessText,
  invoiceVoidTitle,
} from './invoiceConstants';

function CancelModalHarness({
  status,
  direction,
}: {
  status: number;
  direction?: string;
}) {
  const [form] = Form.useForm();
  return (
    <InvoiceCancelModal
      open
      submitting={false}
      cancelForm={form}
      cancelTarget={{
        id: 'inv-1',
        version: '1',
        status,
        direction,
        organizationName: '验收组织',
      }}
      onCancel={vi.fn()}
      onOk={vi.fn()}
    />
  );
}

describe('发票终态确认文案（INV-01）', () => {
  it('草稿呈现取消确认与释放说明，不出现线下税务提示', () => {
    render(
      <CancelModalHarness
        status={FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT}
        direction="RECEIVABLE"
      />,
    );
    expect(
      screen.getByText('取消 验收组织 的开票记录草稿并释放关联账单？'),
    ).toBeInTheDocument();
    expect(
      screen.getByText('取消草稿并释放关联账单，取消后不可恢复。'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('取消原因')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('请输入取消原因（必填）'),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '确认取消' }),
    ).toBeInTheDocument();
    expect(screen.queryByText('线下税务条件确认')).not.toBeInTheDocument();
  });

  it('已开票呈现作废确认并包含线下税务条件危险提示', () => {
    render(
      <CancelModalHarness
        status={FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED}
        direction="RECEIVABLE"
      />,
    );
    expect(
      screen.getByText('作废 验收组织 的已开具发票并释放关联账单？'),
    ).toBeInTheDocument();
    expect(screen.getByText('线下税务条件确认')).toBeInTheDocument();
    expect(
      screen.getByText(
        /仅当已于线下税控\/开票系统完成作废或满足当期作废条件时才能执行/,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/系统不判断税期，不会代替线下税务判断/),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('作废原因')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('请输入作废原因（必填）'),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '确认作废' }),
    ).toBeInTheDocument();
  });

  it('作废标题与成功提示保持销项/进项方向语义', () => {
    expect(invoiceVoidTitle('RECEIVABLE', '验收组织')).toBe(
      '作废 验收组织 的已开具发票并释放关联账单？',
    );
    expect(invoiceVoidTitle('PAYABLE', '验收组织')).toBe(
      '作废 验收组织 的已收票发票并释放关联账单？',
    );
    expect(invoiceDraftCancelTitle('PAYABLE')).toBe(
      '取消 所属公司未标识 的收票记录草稿并释放关联账单？',
    );
    expect(invoiceCancelSuccessText('RECEIVABLE')).toBe(
      '开票记录草稿已取消，关联账单已释放',
    );
    expect(invoiceVoidSuccessText('PAYABLE')).toBe(
      '收票记录已作废，关联账单已释放',
    );
  });
});
