import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FinanceCommissionStatus } from '@/enums.generated';
import {
  settlementServiceCancelCommissionAdjustment,
  settlementServiceConfirmCommissionAdjustment,
  settlementServiceListCommissionAdjustments,
  settlementServiceMarkCommissionAdjustmentPaid,
} from '@/services/roncin/settlementService';
import PendingDecreasePanel from './PendingDecreasePanel';

const accessState = vi.hoisted(() => ({
  canOperateOrganization: () => true,
  canConfigureFinanceCommissions: false,
  canManageFinanceCommissions: true,
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState,
}));

const searchFilterProps = vi.hoisted(() => ({
  current: undefined as Record<string, any> | undefined,
}));

vi.mock('@/components/ui', async () => ({
  ResponsiveProTable: (await import('@ant-design/pro-components')).ProTable,
  SearchFilterTemplate: (props: Record<string, any>) => {
    searchFilterProps.current = props;
    return (
      <div data-testid="decrease-search">
        {props.extraRight}
        <button
          type="button"
          onClick={() => props.onSearch({ keyword: ' SE-100 ', status: 2 })}
        >
          提交筛选
        </button>
        <button type="button" onClick={() => props.onReset()}>
          重置筛选
        </button>
      </div>
    );
  },
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListCommissionAdjustments: vi.fn(),
  settlementServiceConfirmCommissionAdjustment: vi.fn(),
  settlementServiceCancelCommissionAdjustment: vi.fn(),
  settlementServiceMarkCommissionAdjustmentPaid: vi.fn(),
}));

const listAdjustments = vi.mocked(settlementServiceListCommissionAdjustments);
const confirmAdjustment = vi.mocked(
  settlementServiceConfirmCommissionAdjustment,
);
const cancelAdjustment = vi.mocked(settlementServiceCancelCommissionAdjustment);
const markPaid = vi.mocked(settlementServiceMarkCommissionAdjustmentPaid);

function adjustmentRow(
  overrides: Partial<API.FinanceCommissionAdjustment> = {},
): API.FinanceCommissionAdjustment {
  return {
    id: 'adj-1',
    adjustmentNo: 'COM-ADJ-001',
    commissionId: 'commission-1',
    commissionNo: 'COM-2026-001',
    orderId: 'order-1',
    orderNo: 'SE20260901001',
    employeeName: '张三',
    direction: 'DECREASE',
    status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
    baseCurrency: 'CNY',
    amount: '88.5',
    reason: '锁后费用补录自动冲减：漏录拖车费',
    version: '2',
    createdAt: '2026-09-10T10:00:00Z',
    sourceType: 'LOCKED_FEE_SUPPLEMENT',
    canExecute: true,
    ...overrides,
  };
}

function renderPanel() {
  return render(
    <App>
      <PendingDecreasePanel onOpenCommissionDetail={vi.fn()} />
    </App>,
  );
}

describe('PendingDecreasePanel 待处理冲减视图', () => {
  beforeEach(() => {
    accessState.canManageFinanceCommissions = true;
    vi.clearAllMocks();
    listAdjustments.mockResolvedValue({
      data: [],
      total: '0',
      success: true,
    } as Awaited<ReturnType<typeof listAdjustments>>);
  });

  it('默认按已确认调整服务端过滤，包含更正补付与扣回', async () => {
    renderPanel();
    await waitFor(() => expect(listAdjustments).toHaveBeenCalledTimes(1));
    expect(listAdjustments).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
      keyword: undefined,
    });
  });

  it('提交筛选后列表使用同一份规范化条件', async () => {
    renderPanel();
    await waitFor(() => expect(listAdjustments).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByRole('button', { name: '提交筛选' }));
    await waitFor(() => expect(listAdjustments).toHaveBeenCalledTimes(2));
    expect(listAdjustments).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 20,
      status: 2,
      keyword: 'SE-100',
    });
  });

  it('状态文案严格区分待处理/已确认/已扣回/已取消', async () => {
    listAdjustments.mockResolvedValue({
      data: [
        adjustmentRow(),
        adjustmentRow({
          id: 'adj-2',
          adjustmentNo: 'COM-ADJ-002',
          status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
        }),
        adjustmentRow({
          id: 'adj-3',
          adjustmentNo: 'COM-ADJ-003',
          status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID,
        }),
      ],
      total: '3',
      success: true,
    } as Awaited<ReturnType<typeof listAdjustments>>);
    renderPanel();

    expect(await screen.findAllByText('已确认')).toHaveLength(2);

    expect(screen.getByText('已扣回')).toBeInTheDocument();
    // 列表默认不包含已取消；也不把待处理渲染成「冲减草稿」等旧文案。
    expect(screen.queryByText('冲减草稿')).not.toBeInTheDocument();
  });

  it('无 commission.manage 权限时隐藏确认/忽略动作', async () => {
    accessState.canManageFinanceCommissions = false;
    listAdjustments.mockResolvedValue({
      data: [adjustmentRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listAdjustments>>);
    renderPanel();

    await waitFor(() => expect(listAdjustments).toHaveBeenCalled());
    expect(screen.queryByText('确认冲减')).not.toBeInTheDocument();
    expect(screen.queryByText('忽略建议')).not.toBeInTheDocument();
  });

  it('补录冲减已在审批时确认，不再提供确认或忽略入口', async () => {
    listAdjustments.mockResolvedValue({
      data: [
        adjustmentRow({
          status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
        }),
      ],
      total: '1',
      success: true,
    });
    renderPanel();
    expect(await screen.findByText('张三')).toBeInTheDocument();
    expect(screen.queryByText('确认冲减')).not.toBeInTheDocument();
    expect(screen.queryByText('忽略建议')).not.toBeInTheDocument();
    expect(confirmAdjustment).not.toHaveBeenCalled();
    expect(cancelAdjustment).not.toHaveBeenCalled();
  });

  it('父提成未发时显示随父净额执行，不允许独立扣回', async () => {
    listAdjustments.mockResolvedValue({
      data: [
        adjustmentRow({
          status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
          canExecute: false,
          executionBlockedReason: '父提成尚未发放',
        }),
      ],
      total: '1',
      success: true,
    });
    renderPanel();
    expect(await screen.findByText('随父提成净额发放')).toBeInTheDocument();
    expect(screen.queryByText('标记已扣回')).not.toBeInTheDocument();
  });

  it('已确认建议沿用标记已扣回动作', async () => {
    markPaid.mockResolvedValue({
      success: true,
    } as Awaited<ReturnType<typeof markPaid>>);
    listAdjustments.mockResolvedValue({
      data: [
        adjustmentRow({
          status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
        }),
      ],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listAdjustments>>);
    renderPanel();

    fireEvent.click(await screen.findByText('标记已扣回'));
    fireEvent.click(
      await screen.findByRole('button', { name: /标\s*记\s*已\s*扣\s*回/ }),
    );

    await waitFor(() => expect(markPaid).toHaveBeenCalledTimes(1));
    expect(markPaid.mock.calls[0][1]).toEqual({
      id: 'adj-1',
      expectedVersion: '2',
    });
  });
  it('费用更正增提显示补付动作和正金额', async () => {
    listAdjustments.mockResolvedValue({
      data: [
        adjustmentRow({
          direction: 'INCREASE',
          sourceType: 'FEE_CORRECTION',
          amount: '90',
          status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
        }),
      ],
      total: '1',
      success: true,
    });
    renderPanel();
    expect(await screen.findByText('+90.00 CNY')).toBeInTheDocument();
    expect(screen.getByText('标记已补付')).toBeInTheDocument();
    expect(screen.queryByText('标记已扣回')).not.toBeInTheDocument();
  });
});
