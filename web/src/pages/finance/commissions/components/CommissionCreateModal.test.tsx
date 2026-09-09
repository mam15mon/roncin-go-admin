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
  listFinanceOrganizationOptions: vi.fn(),
  listCommissionVerificationCandidates: vi.fn(),
  listCommissionRuleCandidates: vi.fn(),
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
  ProFormSearchableSelect: (props: Record<string, any>) => {
    const [options, setOptions] = React.useState<any[]>([]);
    return (
      <div>
        {props.label === '所属公司' && (
          <>
            <button
              type="button"
              onClick={() => props.fieldProps?.onChange?.('org-a')}
            >
              选择公司 A
            </button>
            <button
              type="button"
              onClick={() => props.fieldProps?.onChange?.('org-b')}
            >
              选择公司 B
            </button>
          </>
        )}
        {props.request && (
          <button
            type="button"
            onClick={async () => setOptions(await props.request())}
          >
            加载{props.label}
          </button>
        )}
        {options.map((option) => (
          <span key={option.value}>{option.label}</span>
        ))}
      </div>
    );
  },
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCreateCommission: serviceMocks.createCommission,
  settlementServiceListCommissionCandidates: vi.fn(),
  settlementServiceListCommissionRuleCandidates:
    serviceMocks.listCommissionRuleCandidates,
  settlementServiceListCommissionRules: vi.fn(),
  settlementServiceListFinanceOrganizationOptions:
    serviceMocks.listFinanceOrganizationOptions,
  settlementServiceListCommissionVerificationCandidates:
    serviceMocks.listCommissionVerificationCandidates,
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
    serviceMocks.listFinanceOrganizationOptions.mockReset();
    serviceMocks.listCommissionVerificationCandidates.mockReset();
    serviceMocks.listCommissionRuleCandidates.mockReset();
    serviceMocks.listFinanceOrganizationOptions.mockResolvedValue({ data: [] });
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

  it('未选公司不请求核销候选，慢 A 响应不会污染已切换的 B 公司', async () => {
    let resolveA: (value: unknown) => void = () => undefined;
    const slowA = new Promise((resolve) => {
      resolveA = resolve;
    });
    serviceMocks.listCommissionVerificationCandidates.mockImplementation(
      ({ organizationId }: { organizationId: string }) =>
        organizationId === 'org-a'
          ? slowA
          : Promise.resolve({
              data: [
                {
                  id: 'verification-b',
                  verificationNo: 'VR-B',
                  settlementPartyName: 'B单位',
                  amount: '100',
                  currency: 'CNY',
                },
              ],
            }),
    );
    render(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    expect(
      serviceMocks.listCommissionVerificationCandidates,
    ).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '选择公司 A' }));
    fireEvent.click(screen.getByRole('button', { name: '加载有效应收核销' }));
    await waitFor(() =>
      expect(
        serviceMocks.listCommissionVerificationCandidates,
      ).toHaveBeenCalledWith({
        page: 1,
        pageSize: 200,
        organizationId: 'org-a',
      }),
    );

    fireEvent.click(screen.getByRole('button', { name: '选择公司 B' }));
    fireEvent.click(screen.getByRole('button', { name: '加载有效应收核销' }));
    expect(await screen.findByText('VR-B｜B单位｜100 CNY')).toBeInTheDocument();

    await act(async () => {
      resolveA({
        data: [
          {
            id: 'verification-a',
            verificationNo: 'VR-A',
            settlementPartyName: 'A单位',
            amount: '200',
            currency: 'CNY',
          },
        ],
      });
      await slowA;
    });
    expect(screen.queryByText('VR-A｜A单位｜200 CNY')).not.toBeInTheDocument();
  });

  it('考核规则候选携带当前公司，并使用提成管理专用接口', async () => {
    serviceMocks.listCommissionRuleCandidates.mockResolvedValue({
      data: [
        {
          id: 'rule-b',
          name: 'B公司销售提成',
          personnelRole: 'SALES',
          calculationBasis: 'REALIZED_PROFIT',
          ratePercent: '2.5',
        },
      ],
    });
    render(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: '选择公司 B' }));
    fireEvent.click(screen.getByRole('button', { name: '加载考核规则' }));

    await waitFor(() =>
      expect(serviceMocks.listCommissionRuleCandidates).toHaveBeenCalledWith({
        page: 1,
        pageSize: 200,
        organizationId: 'org-b',
      }),
    );
  });
});
