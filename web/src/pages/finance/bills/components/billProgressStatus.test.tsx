import { cleanup, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import { FinanceBillBusinessStatus as Business } from '@/enums.generated';
import { billExportColumns } from '../ledgerExport';
import { getFinanceBillColumns } from './billColumns';
import { getBillProgressStatus } from './billProgressStatus';

afterEach(cleanup);

describe('日常账单业务状态', () => {
  const cases: {
    name: string;
    row: API.FinanceBill;
    expected: string | undefined;
  }[] = [
    {
      name: '未开票',
      row: { businessStatus: Business.FINANCE_BILL_BUSINESS_STATUS_UNINVOICED },
      expected: '未开票',
    },
    {
      name: '部分开票',
      row: {
        businessStatus:
          Business.FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_INVOICED,
      },
      expected: '部分开票',
    },
    {
      name: '未核销',
      row: { businessStatus: Business.FINANCE_BILL_BUSINESS_STATUS_UNVERIFIED },
      expected: '未核销',
    },
    {
      name: '部分核销',
      row: {
        businessStatus:
          Business.FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_VERIFIED,
      },
      expected: '部分核销',
    },
    {
      name: '完成投影优先于其他字段',
      row: {
        businessStatus: Business.FINANCE_BILL_BUSINESS_STATUS_COMPLETED,
        completed: false,
        invoiceProgress: 2,
      },
      expected: '已完成',
    },
    {
      name: '省略业务状态即使其他进度齐全也不猜测',
      row: { completed: true, invoiceProgress: 4, settlementProgress: 3 },
      expected: undefined,
    },
    {
      name: '显式零不投影',
      row: {
        businessStatus: Business.FINANCE_BILL_BUSINESS_STATUS_UNSPECIFIED,
      },
      expected: undefined,
    },
    {
      name: '未知枚举不猜测',
      row: { businessStatus: 99 },
      expected: undefined,
    },
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
      onCancelBill: () => {},
    });
    const column = columns.find((col) => col.dataIndex === 'status');
    const renderCell = column?.render as (
      value: unknown,
      row: API.FinanceBill,
    ) => ReactNode;
    render(
      renderCell(undefined, {
        businessStatus:
          Business.FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_INVOICED,
      }),
    );
    expect(screen.getByText('部分开票')).toBeInTheDocument();
    expect(screen.queryByText('未核销')).not.toBeInTheDocument();
    expect(screen.queryByText('正常')).not.toBeInTheDocument();
  });
});
