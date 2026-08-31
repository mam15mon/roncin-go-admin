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
});
