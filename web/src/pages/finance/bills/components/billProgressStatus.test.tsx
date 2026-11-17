import { cleanup, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import {
  FinanceBillInvoiceProgress as Invoice,
  FinanceBillSettlementProgress as Settlement,
} from '@/enums.generated';
import { billExportColumns } from '../ledgerExport';
import { getFinanceBillColumns } from './billColumns';
import { getBillProgressStatus } from './billProgressStatus';

const none = Invoice.FINANCE_BILL_INVOICE_PROGRESS_NONE;
const partial = Invoice.FINANCE_BILL_INVOICE_PROGRESS_PARTIAL;
const full = Invoice.FINANCE_BILL_INVOICE_PROGRESS_FULL;
const exempt = Invoice.FINANCE_BILL_INVOICE_PROGRESS_NOT_REQUIRED;
const unsettled = Settlement.FINANCE_BILL_SETTLEMENT_PROGRESS_UNSETTLED;
const partlySettled = Settlement.FINANCE_BILL_SETTLEMENT_PROGRESS_PARTIAL;
const settled = Settlement.FINANCE_BILL_SETTLEMENT_PROGRESS_SETTLED;

afterEach(cleanup);

describe('日常账单业务状态', () => {
  const cases: {
    name: string;
    row: API.FinanceBill;
    expected: string | undefined;
  }[] = [
    {
      name: '未开票与未核销并存时优先显示未开票，省略 completed 不算完成',
      row: { invoiceProgress: none, settlementProgress: unsettled },
      expected: '未开票',
    },
    {
      name: '部分开票与部分核销并存时优先显示部分开票',
      row: {
        invoiceProgress: partial,
        settlementProgress: partlySettled,
        verifiedAmount: '40.00000000',
      },
      expected: '部分开票',
    },
    {
      name: '先结清仍需完成开票',
      row: { invoiceProgress: partial, settlementProgress: settled },
      expected: '部分开票',
    },
    {
      name: '先开票仍需核销',
      row: { invoiceProgress: full, settlementProgress: unsettled },
      expected: '未核销',
    },
    {
      name: '票齐且已有部分核销时显示部分核销',
      row: {
        invoiceProgress: full,
        settlementProgress: partlySettled,
        verifiedAmount: '40.00000000',
      },
      expected: '部分核销',
    },
    {
      name: '无需开票仍需核销',
      row: { invoiceProgress: exempt, settlementProgress: unsettled },
      expected: '未核销',
    },
    {
      name: '仅发生部分对冲不冒充部分核销',
      row: {
        invoiceProgress: full,
        settlementProgress: partlySettled,
        verifiedAmount: '0.00000000',
      },
      expected: '未核销',
    },
    {
      name: '负数冲回的核销也展示部分核销',
      row: {
        invoiceProgress: full,
        settlementProgress: partlySettled,
        verifiedAmount: '-40.00000000',
      },
      expected: '部分核销',
    },
    {
      name: '结清且票齐后合并为已完成',
      row: {
        completed: true,
        invoiceProgress: full,
        settlementProgress: settled,
      },
      expected: '已完成',
    },
    {
      name: '免票或零额账单按服务端完成结果展示',
      row: {
        completed: true,
        invoiceProgress: exempt,
        settlementProgress: settled,
      },
      expected: '已完成',
    },
    {
      name: '先核销完但未开票仍显示未开票',
      row: { invoiceProgress: none, settlementProgress: settled },
      expected: '未开票',
    },
    { name: '未投影进度时不猜测状态', row: {}, expected: undefined },
  ];

  it.each(cases)('$name，导出与页面共用相同文案', ({ row, expected }) => {
    expect(getBillProgressStatus(row)?.text).toBe(expected);
    const exportStatus = billExportColumns.find((col) => col.header === '状态');
    expect(exportStatus?.value(row)).toBe(expected ?? '');
  });

  it('首屏状态列展示业务待办，不再显示正常', () => {
    const columns = getFinanceBillColumns({
      access: { canOperateOrganization: () => true },
      onOpenDetail: () => {},
      onOpenEdit: () => {},
      onCancelBill: () => {},
    });
    const column = columns.find((col) => col.dataIndex === 'status');
    const renderCell = column?.render as (
      value: unknown,
      row: API.FinanceBill,
    ) => ReactNode;
    render(
      renderCell(undefined, {
        invoiceProgress: partial,
        settlementProgress: unsettled,
      }),
    );
    expect(screen.getByText('部分开票')).toBeInTheDocument();
    expect(screen.queryByText('未核销')).not.toBeInTheDocument();
    expect(screen.queryByText('正常')).not.toBeInTheDocument();
  });
});
