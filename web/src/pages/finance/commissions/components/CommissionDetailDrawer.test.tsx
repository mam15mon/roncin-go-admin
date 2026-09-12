import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { FinanceCommissionStatus } from '@/enums.generated';
import CommissionDetailDrawer from './CommissionDetailDrawer';

describe('提成详情双口径', () => {
  it('已取消提成保留本位币和 CNY 快照并明确不计入应发', async () => {
    render(
      <App>
        <CommissionDetailDrawer
          open
          onClose={vi.fn()}
          loading={false}
          canManage={false}
          onOpenAdjustment={vi.fn()}
          onTransitionAdjustment={vi.fn()}
          onCancelAdjustment={vi.fn()}
          detail={{
            id: 'commission-1',
            commissionNo: 'COM-001',
            status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED,
            baseCurrency: 'USD',
            commissionDate: '2026-08-31',
            commissionAmount: '56.00000000',
            adjustmentAmount: '1.00000000',
            effectiveCommissionAmount: '57.00000000',
            cnyCommissionAmount: '400.00000000',
            cnyAdjustmentAmount: '7.14285714',
            cnyEffectiveCommissionAmount: '407.14285714',
            cnyExchangeRate: '7.14285714',
            cnyExchangeRateDate: '2026-08-31',
            cnyExchangeRateSource: 'DERIVED',
            lines: [],
            adjustments: [],
          }}
        />
      </App>,
    );

    expect(await screen.findByText('原始提成（CNY）')).toBeInTheDocument();
    expect(screen.getByText('400 CNY')).toBeInTheDocument();
    expect(screen.getByText('快照 57 USD，不计入应发')).toBeInTheDocument();
    expect(
      screen.getByText('快照 407.14285714 CNY，不计入应发'),
    ).toBeInTheDocument();
    expect(screen.getByText('倒数派生')).toBeInTheDocument();
    expect(screen.getAllByText('2026-08-31')).toHaveLength(2);
  });

  it('来源单号按核销或对冲二选一展示，两者都空显示占位符', () => {
    const baseDetail = {
      id: 'commission-2',
      commissionNo: 'COM-002',
      status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT,
      baseCurrency: 'CNY',
      commissionDate: '2026-09-01',
      lines: [],
      adjustments: [],
    };
    const { rerender } = render(
      <App>
        <CommissionDetailDrawer
          open
          onClose={vi.fn()}
          loading={false}
          canManage={false}
          onOpenAdjustment={vi.fn()}
          onTransitionAdjustment={vi.fn()}
          onCancelAdjustment={vi.fn()}
          detail={{ ...baseDetail, verificationNo: 'VR-2026-001' }}
        />
      </App>,
    );
    expect(screen.getByText('来源单号')).toBeInTheDocument();
    expect(screen.getByText('VR-2026-001')).toBeInTheDocument();

    rerender(
      <App>
        <CommissionDetailDrawer
          open
          onClose={vi.fn()}
          loading={false}
          canManage={false}
          onOpenAdjustment={vi.fn()}
          onTransitionAdjustment={vi.fn()}
          onCancelAdjustment={vi.fn()}
          detail={{ ...baseDetail, nettingNo: 'NT-2026-002' }}
        />
      </App>,
    );
    expect(screen.getByText('NT-2026-002')).toBeInTheDocument();

    rerender(
      <App>
        <CommissionDetailDrawer
          open
          onClose={vi.fn()}
          loading={false}
          canManage={false}
          onOpenAdjustment={vi.fn()}
          onTransitionAdjustment={vi.fn()}
          onCancelAdjustment={vi.fn()}
          detail={{ ...baseDetail }}
        />
      </App>,
    );
    const sourceRow = screen.getByText('来源单号').closest('tr');
    expect(sourceRow).not.toBeNull();
    expect(sourceRow?.textContent).toContain('-');
  });

  it('对冲撤销冲减按系统反转来源展示且不暴露取消操作', () => {
    render(
      <App>
        <CommissionDetailDrawer
          open
          onClose={vi.fn()}
          loading={false}
          canManage
          onOpenAdjustment={vi.fn()}
          onTransitionAdjustment={vi.fn()}
          onCancelAdjustment={vi.fn()}
          detail={{
            id: 'commission-3',
            commissionNo: 'COM-003',
            status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
            baseCurrency: 'CNY',
            commissionDate: '2026-09-01',
            lines: [],
            adjustments: [
              {
                id: 'adj-1',
                adjustmentNo: 'ADJ-2026-0001',
                direction: 'DECREASE',
                status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
                amount: '50.00000000',
                reason: '对冲撤销冲减',
                sourceType: 'NETTING_REVERSAL',
                version: '1',
              },
            ],
          }}
        />
      </App>,
    );
    expect(screen.getByText('对冲撤销')).toBeInTheDocument();
    expect(screen.queryByText('手工调整')).not.toBeInTheDocument();
    expect(screen.getByText('待追回')).toBeInTheDocument();
    expect(screen.queryByText('取消')).not.toBeInTheDocument();
  });
});
