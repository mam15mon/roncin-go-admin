import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  createCommission: vi.fn(),
  previewCommission: vi.fn(),
}));

const modalState = vi.hoisted(() => ({
  props: undefined as Record<string, any> | undefined,
}));

vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: Record<string, any>) => {
    modalState.props = props;
    return <div>{props.children}</div>;
  },
  ProFormDependency: ({ children }: Record<string, any>) => (
    <>
      {children({
        verificationId: 'verification-1',
        ruleId: 'rule-1',
        employeeId: 'employee-1',
      })}
    </>
  ),
  ProFormTextArea: () => null,
}));

vi.mock('@/components/ui', () => ({
  ProFormSearchableSelect: () => null,
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCreateCommission: serviceMocks.createCommission,
  settlementServiceListCommissionCandidates: vi.fn(),
  settlementServiceListCommissionRules: vi.fn(),
  settlementServiceListVerifications: vi.fn(),
  settlementServicePreviewCommission: serviceMocks.previewCommission,
}));

vi.mock('./CommissionLineTable', () => ({
  previewColumns: [],
  renderExpandedFees: () => null,
}));

import CommissionCreateModal from './CommissionCreateModal';

describe('提成预览 CNY 快照', () => {
  beforeEach(() => {
    modalState.props = undefined;
    serviceMocks.createCommission.mockReset();
    serviceMocks.previewCommission.mockReset();
    serviceMocks.previewCommission.mockResolvedValue({
      data: {
        employeeName: '张三',
        ruleName: '销售提成',
        ruleVersion: '1',
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: '2.5000',
        baseCurrency: 'USD',
        commissionAmount: '56.00000000',
        cnyCommissionAmount: '400.00000000',
        cnyExchangeRate: '7.14285714',
        cnyExchangeRateDate: '2026-08-31',
        cnyExchangeRateSource: 'DERIVED',
        lines: [],
      },
    });
  });

  it('展示后端返回的 CNY 金额、汇率依据和重新解析提示', async () => {
    render(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));

    expect(await screen.findByText('400 CNY')).toBeInTheDocument();
    expect(screen.getByText('7.14285714')).toBeInTheDocument();
    expect(screen.getByText('2026-08-31')).toBeInTheDocument();
    expect(screen.getByText('倒数派生')).toBeInTheDocument();
    expect(screen.getByText('预览汇率仅供生成前核对')).toBeInTheDocument();
  });

  it('创建成功后触发列表刷新，不把预览结果作为创建参数', async () => {
    const onSuccess = vi.fn();
    serviceMocks.createCommission.mockResolvedValue({
      data: { cnyCommissionAmount: '401.00000000' },
    });
    render(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={onSuccess}
        />
      </App>,
    );
    fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));
    await screen.findByText('400 CNY');

    await act(async () => {
      await modalState.props?.onFinish({
        verificationId: 'verification-1',
        ruleId: 'rule-1',
        employeeId: 'employee-1',
      });
    });

    await waitFor(() => expect(onSuccess).toHaveBeenCalledOnce());
    expect(serviceMocks.createCommission).toHaveBeenCalledWith(
      expect.objectContaining({
        verificationId: 'verification-1',
        ruleId: 'rule-1',
        employeeId: 'employee-1',
      }),
    );
    expect(serviceMocks.createCommission.mock.calls[0][0]).not.toHaveProperty(
      'cnyCommissionAmount',
    );
  });
});
