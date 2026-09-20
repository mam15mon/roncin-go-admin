import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FinanceCommissionStatus } from '@/enums.generated';
import { financeErrorReasons } from '@/errorReasons.generated';
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

vi.mock('@umijs/max', () => ({
  useAccess: () => accessState,
}));

const searchFilterProps = vi.hoisted(() => ({
  current: undefined as Record<string, any> | undefined,
}));

vi.mock('@/components/ui', () => ({
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
    status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT,
    baseCurrency: 'CNY',
    amount: '88.5',
    reason: '锁后费用补录自动冲减：漏录拖车费',
    version: '2',
    createdAt: '2026-09-10T10:00:00Z',
    sourceType: 'LOCKED_FEE_SUPPLEMENT',
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

  it('默认按 DECREASE + DRAFT + LOCKED_FEE_SUPPLEMENT 服务端过滤', async () => {
    renderPanel();
    await waitFor(() => expect(listAdjustments).toHaveBeenCalledTimes(1));
    expect(listAdjustments).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT,
      sourceType: 'LOCKED_FEE_SUPPLEMENT',
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
      sourceType: 'LOCKED_FEE_SUPPLEMENT',
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

    expect(await screen.findByText('待处理')).toBeInTheDocument();
    expect(screen.getByText('已确认')).toBeInTheDocument();
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

  it('确认冲减成功后刷新列表', async () => {
    confirmAdjustment.mockResolvedValue({
      success: true,
    } as Awaited<ReturnType<typeof confirmAdjustment>>);
    listAdjustments.mockResolvedValue({
      data: [adjustmentRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listAdjustments>>);
    renderPanel();

    fireEvent.click(await screen.findByText('确认冲减'));
    fireEvent.click(
      await screen.findByRole('button', { name: /确\s*认\s*冲\s*减/ }),
    );

    await waitFor(() => expect(confirmAdjustment).toHaveBeenCalledTimes(1));
    expect(confirmAdjustment.mock.calls[0][1]).toEqual({
      id: 'adj-1',
      expectedVersion: '2',
    });
    await waitFor(() => expect(listAdjustments).toHaveBeenCalledTimes(2));
  });

  it('确认冲减超限冲突时提示刷新且不静默修改金额', async () => {
    confirmAdjustment.mockRejectedValue(
      Object.assign(new Error('超出可冲减余额'), {
        data: {
          reason: financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS,
          message: '超出可冲减余额',
        },
      }),
    );
    listAdjustments.mockResolvedValue({
      data: [adjustmentRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listAdjustments>>);
    renderPanel();

    fireEvent.click(await screen.findByText('确认冲减'));
    fireEvent.click(
      await screen.findByRole('button', { name: /确\s*认\s*冲\s*减/ }),
    );

    expect(
      (await screen.findAllByText('冲减金额超出可冲减余额')).length,
    ).toBeGreaterThan(0);
    await waitFor(() => expect(listAdjustments).toHaveBeenCalledTimes(2));
  });

  it('忽略建议必须填写原因', async () => {
    cancelAdjustment.mockResolvedValue({
      success: true,
    } as Awaited<ReturnType<typeof cancelAdjustment>>);
    listAdjustments.mockResolvedValue({
      data: [adjustmentRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listAdjustments>>);
    renderPanel();

    fireEvent.click(await screen.findByText('忽略建议'));
    const reasonInput = await screen.findByPlaceholderText(
      '请输入忽略原因（必填）',
    );
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));
    expect(await screen.findByText('请输入忽略原因')).toBeInTheDocument();
    expect(cancelAdjustment).not.toHaveBeenCalled();

    fireEvent.change(reasonInput, { target: { value: '费用与该员工无关' } });
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    await waitFor(() => expect(cancelAdjustment).toHaveBeenCalledTimes(1));
    expect(cancelAdjustment.mock.calls[0][1]).toEqual({
      id: 'adj-1',
      expectedVersion: '2',
      reason: '费用与该员工无关',
    });
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
});
