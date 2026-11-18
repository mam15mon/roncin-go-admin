import { cleanup, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  FinanceBillFeeLockFact,
  FinanceBillInvoiceProgress,
  FinanceBillSettlementProgress,
  FinanceBillStatus,
} from '@/enums.generated';
import { getFinanceBillColumns } from './billColumns';

describe('getFinanceBillColumns', () => {
  afterEach(cleanup);
  const onCancelBill = vi.fn();
  const columns = getFinanceBillColumns({
    access: {
      canOperateOrganization: (id) => id === 'company-a',
      canUpdateFinanceBills: true,
    },
    onOpenDetail: () => {},
    onCancelBill,
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
          canCancel: true,
        }),
      );
      expect(screen.getByText('详情')).toBeInTheDocument();
      expect(screen.queryByText('编辑')).not.toBeInTheDocument();
      expect(screen.queryByText('取消')).not.toBeInTheDocument();
    },
  );

  it('当前公司的正常账单列表只保留详情与取消，编辑移入详情', () => {
    const actions = columns.find((col) => col.valueType === 'option')
      ?.render as any;
    const row = {
      organizationId: 'company-a',
      status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
      canCancel: true,
    };
    render(actions(undefined, row));
    expect(screen.queryByText('编辑')).not.toBeInTheDocument();
    screen.getByText('取消').click();
    expect(onCancelBill).toHaveBeenCalledWith(row);
    expect(screen.queryByText('确认')).not.toBeInTheDocument();
  });

  it.each([false, undefined])(
    '取消资格为 %s 的有效账单仅显示详情',
    (canCancel) => {
      const actions = columns.find((col) => col.valueType === 'option')
        ?.render as any;
      render(
        actions(undefined, {
          organizationId: 'company-a',
          status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
          canCancel,
        }),
      );
      expect(screen.getByText('详情')).toBeInTheDocument();
      expect(screen.queryByText('取消')).not.toBeInTheDocument();
    },
  );

  it('只有查看权限时，即使账单可取消也仅显示详情', () => {
    const readOnly = getFinanceBillColumns({
      access: {
        canOperateOrganization: () => true,
        canUpdateFinanceBills: false,
      },
      onOpenDetail: () => {},
      onCancelBill,
    });
    const actions = readOnly.find((col) => col.valueType === 'option')
      ?.render as any;
    render(
      actions(undefined, { organizationId: 'company-a', canCancel: true }),
    );
    expect(screen.getByText('详情')).toBeInTheDocument();
    expect(screen.queryByText('取消')).not.toBeInTheDocument();
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

describe('账单进度列', () => {
  afterEach(cleanup);
  const columns = getFinanceBillColumns({
    access: { canOperateOrganization: () => true },
    onOpenDetail: () => {},
    onCancelBill: () => {},
  });
  const renderColumn = (title: string, row: API.FinanceBill) => {
    const column = columns.find((col) => col.title === title);
    const renderCell = column?.render as any;
    return render(renderCell(undefined, row));
  };

  it('按服务端投影展示结算、开票与完成状态', () => {
    const row = {
      settlementProgress:
        FinanceBillSettlementProgress.FINANCE_BILL_SETTLEMENT_PROGRESS_SETTLED,
      invoiceProgress:
        FinanceBillInvoiceProgress.FINANCE_BILL_INVOICE_PROGRESS_NOT_REQUIRED,
      completed: true,
    };
    renderColumn('结算进度', row);
    renderColumn('开票进度', row);
    renderColumn('完成', row);
    expect(screen.getByText('已结清')).toBeInTheDocument();
    expect(screen.getByText('无需开票')).toBeInTheDocument();
    expect(screen.getByText('已完成')).toBeInTheDocument();
  });

  it('ProtoJSON 省略零值时：未完成、无锁定标记，未投影进度显示 -', () => {
    const row = { feeCount: 2 };
    renderColumn('完成', row);
    expect(screen.getByText('未完成')).toBeInTheDocument();
    cleanup();
    renderColumn('结算进度', row);
    expect(screen.getByText('-')).toBeInTheDocument();
    cleanup();
    renderColumn('费用数', row);
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.queryByText('锁定')).not.toBeInTheDocument();
  });

  it('存在锁定事实时在费用数旁标记锁定', () => {
    renderColumn('费用数', {
      feeCount: 1,
      feeLockFacts: [FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_INVOICE],
    });
    expect(screen.getByText('锁定')).toHaveAttribute(
      'title',
      '已有发票，费用不可增删',
    );
  });
});
