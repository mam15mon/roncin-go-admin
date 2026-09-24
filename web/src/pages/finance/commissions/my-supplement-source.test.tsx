import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FinanceCommissionStatus } from '@/enums.generated';
import { settlementServiceGetMyFeeSupplementAdjustmentSource } from '@/services/roncin/settlementService';
import MySupplementSourcePage from './my-supplement-source';

const mockParams = vi.hoisted(() => ({
  id: undefined as string | undefined,
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return { ...actual, useParams: () => mockParams };
});

vi.mock('@/app/access', () => ({
  // 员工本人落地页不要求组织级 commission.read：不注入任何权限。
  useAccess: () => ({}),
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetMyFeeSupplementAdjustmentSource: vi.fn(),
}));

const getSource = vi.mocked(
  settlementServiceGetMyFeeSupplementAdjustmentSource,
);

describe('MySupplementSourcePage 员工本人冲减来源详情', () => {
  beforeEach(() => {
    mockParams.id = 'adj-1';
    vi.clearAllMocks();
  });

  it('只展示后端白名单字段，不提供管理动作', async () => {
    getSource.mockResolvedValue({
      data: {
        adjustmentId: 'adj-1',
        adjustmentNo: 'COM-ADJ-001',
        orderNo: 'SE20260901001',
        commissionNo: 'COM-2026-001',
        status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT,
        suggestedAmount: '88.5',
        baseCurrency: 'CNY',
        createdAt: '2026-09-10T10:00:00Z',
        feeCode: 'TRUCK',
        feeName: '拖车费',
        feeCurrency: 'CNY',
        feeTotalAmount: '500',
        feeBaseCurrency: 'CNY',
        feeBaseCurrencyAmount: '500',
        feeExpenseDate: '2026-08-30',
        supplementReason: '漏录拖车费',
      },
    } as Awaited<ReturnType<typeof getSource>>);

    render(
      <App>
        <MySupplementSourcePage />
      </App>,
    );

    expect(await screen.findByText('COM-ADJ-001')).toBeInTheDocument();
    expect(screen.getByText('SE20260901001')).toBeInTheDocument();
    expect(screen.getByText('COM-2026-001')).toBeInTheDocument();
    expect(screen.getByText('-88.50 CNY')).toBeInTheDocument();
    expect(screen.getByText('拖车费')).toBeInTheDocument();
    expect(screen.getByText('漏录拖车费')).toBeInTheDocument();
    expect(screen.getByText('待处理')).toBeInTheDocument();

    // 仅供知情：不提供确认、忽略、标记已扣回等任何管理动作。
    expect(screen.queryByText('确认冲减')).not.toBeInTheDocument();
    expect(screen.queryByText('忽略建议')).not.toBeInTheDocument();
    expect(screen.queryByText('标记已扣回')).not.toBeInTheDocument();
  });

  it('404/无权限统一按不存在处理，不泄露他人记录', async () => {
    getSource.mockRejectedValue(
      Object.assign(new Error('Not Found'), { response: { status: 404 } }),
    );
    render(
      <App>
        <MySupplementSourcePage />
      </App>,
    );

    expect(await screen.findByText('记录不存在')).toBeInTheDocument();
    expect(
      screen.getByText('未找到对应的冲减建议，或该记录与您无关。'),
    ).toBeInTheDocument();
  });

  it('路由参数缺失时直接呈现不存在', async () => {
    mockParams.id = undefined;
    render(
      <App>
        <MySupplementSourcePage />
      </App>,
    );

    await waitFor(() => expect(getSource).not.toHaveBeenCalled());
    expect(await screen.findByText('记录不存在')).toBeInTheDocument();
  });
});
