import { render, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it } from 'vitest';
import { getFinanceBillColumns } from './billColumns';

describe('getFinanceBillColumns', () => {
  const columns = getFinanceBillColumns({
    access: { canUpdateFinanceBills: true, canConfirmFinanceBills: true },
    onOpenDetail: () => {},
    onOpenEdit: () => {},
    onConfirmBill: () => {},
    onCancelBill: () => {},
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
