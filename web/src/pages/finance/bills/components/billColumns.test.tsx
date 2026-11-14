import { cleanup, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import { FinanceBillStatus } from '@/enums.generated';
import { getFinanceBillColumns } from './billColumns';

describe('getFinanceBillColumns', () => {
  afterEach(cleanup);
  const columns = getFinanceBillColumns({
    access: {
      canOperateOrganization: (id) => id === 'company-a',
      canUpdateFinanceBills: true,
    },
    onOpenDetail: () => {},
    onOpenEdit: () => {},
    onCancelBill: () => {},
  });

  it.each(['company-b', undefined])(
    '外公司或缺失归属 %s 仅显示详情',
    (organizationId) => {
      const actions = columns.find((col) => col.valueType === 'option')
        ?.render as any;
      render(
        actions(undefined, {
          organizationId,
          status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
        }),
      );
      expect(screen.getByText('详情')).toBeInTheDocument();
      expect(screen.queryByText('编辑')).not.toBeInTheDocument();
      expect(screen.queryByText('取消')).not.toBeInTheDocument();
    },
  );

  it('当前公司的正常账单保留编辑与取消入口，不再提供确认', () => {
    const actions = columns.find((col) => col.valueType === 'option')
      ?.render as any;
    render(
      actions(undefined, {
        organizationId: 'company-a',
        status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
      }),
    );
    expect(screen.getByText('编辑')).toBeInTheDocument();
    expect(screen.getByText('取消')).toBeInTheDocument();
    expect(screen.queryByText('确认')).not.toBeInTheDocument();
  });

  it('已取消账单仅显示详情', () => {
    const actions = columns.find((col) => col.valueType === 'option')
      ?.render as any;
    render(
      actions(undefined, {
        organizationId: 'company-a',
        status: FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED,
      }),
    );
    expect(screen.getByText('详情')).toBeInTheDocument();
    expect(screen.queryByText('编辑')).not.toBeInTheDocument();
    expect(screen.queryByText('取消')).not.toBeInTheDocument();
  });

  const dueDateCol = columns.find((col) => col.dataIndex === 'dueDate');

  it('到期日为空时渲染 -', () => {
    expect(dueDateCol).toBeDefined();
    const renderFn = dueDateCol?.render as any;
    const result = renderFn(undefined, { dueDate: undefined });
    expect(result).toBe('-');
  });

  it('未逾期时仅渲染到期日', () => {
    const renderFn = dueDateCol?.render as any;
    const result = renderFn('2026-09-30', {
      dueDate: '2026-09-30',
      overdueDays: 0,
    });
    expect(result).toBe('2026-09-30');
  });

  it('逾期时渲染到期日与逾期天数 Tag', () => {
    const renderFn = dueDateCol?.render as any;
    const result = renderFn('2026-09-01', {
      dueDate: '2026-09-01',
      overdueDays: 9,
    });
    render(result as React.ReactElement);
    expect(screen.getByText('2026-09-01')).toBeInTheDocument();
    expect(screen.getByText('逾期 9 天')).toBeInTheDocument();
  });
});
