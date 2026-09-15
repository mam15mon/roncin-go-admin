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
  listCommissionNettingCandidates: vi.fn(),
  listCommissionRuleCandidates: vi.fn(),
  listCommissionEmployees: vi.fn(),
}));

const modalState = vi.hoisted(() => ({
  props: undefined as Record<string, any> | undefined,
  dependencyValues: {
    verificationId: 'verification-1',
    ruleId: 'rule-1',
    employeeId: 'employee-1',
  } as Record<string, any>,
}));

vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: Record<string, any>) => {
    modalState.props = props;
    return <div>{props.children}</div>;
  },
  ProFormDependency: ({ children }: Record<string, any>) => (
    <>{children(modalState.dependencyValues)}</>
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
        {(props.options ?? options).map((option: any) => (
          <span key={option.value}>{option.label}</span>
        ))}
        {props.fieldProps?.onSearch && (
          <input
            aria-label={`搜索${props.label}`}
            onChange={(event) =>
              props.fieldProps?.onSearch?.(event.target.value)
            }
          />
        )}
        {props.fieldProps?.loading ? <span>候选加载中</span> : null}
      </div>
    );
  },
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCreateCommission: serviceMocks.createCommission,
  settlementServiceListCommissionCandidates: vi.fn(),
  settlementServiceListCommissionNettingCandidates:
    serviceMocks.listCommissionNettingCandidates,
  settlementServiceListCommissionEmployees:
    serviceMocks.listCommissionEmployees,
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

const nettingCandidate = {
  id: 'netting-1',
  nettingNo: 'NT-2026-001',
  settlementPartyName: '对冲单位A',
  currency: 'CNY',
  amount: '100.00',
  confirmedAt: '2026-09-01 10:00:00',
};

const verificationCandidate = {
  id: 'verification-a',
  verificationNo: 'VR-A',
  settlementPartyName: 'A单位',
  amount: '200',
  currency: 'CNY',
};

describe('提成预览 CNY 快照', () => {
  beforeEach(() => {
    modalState.props = undefined;
    modalState.dependencyValues = {
      verificationId: 'verification-1',
      ruleId: 'rule-1',
      employeeId: 'employee-1',
    };
    for (const mock of Object.values(serviceMocks)) {
      mock.mockReset();
    }
    serviceMocks.listFinanceOrganizationOptions.mockResolvedValue({ data: [] });
    serviceMocks.listCommissionVerificationCandidates.mockResolvedValue({
      data: [],
    });
    serviceMocks.listCommissionNettingCandidates.mockResolvedValue({
      data: [],
    });
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

  it('核销来源创建请求体携带 verificationId 而不带 nettingId', async () => {
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
      'nettingId',
    );
    expect(serviceMocks.createCommission.mock.calls[0][0]).not.toHaveProperty(
      'cnyCommissionAmount',
    );
  });

  it('对冲提成 Tab 按对冲管理列惯例展示已确认对冲单候选', async () => {
    serviceMocks.listCommissionNettingCandidates.mockResolvedValue({
      data: [nettingCandidate],
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

    fireEvent.click(screen.getByRole('button', { name: '选择公司 A' }));
    fireEvent.click(screen.getByRole('tab', { name: '对冲提成' }));

    await waitFor(() =>
      expect(serviceMocks.listCommissionNettingCandidates).toHaveBeenCalledWith(
        {
          page: 1,
          pageSize: 200,
          organizationId: 'org-a',
        },
      ),
    );
    expect(await screen.findByText('NT-2026-001')).toBeInTheDocument();
    expect(screen.getByText('对冲单位A')).toBeInTheDocument();
    expect(screen.getByText('100.00 CNY')).toBeInTheDocument();
    expect(screen.getByText('2026-09-01 10:00:00')).toBeInTheDocument();
    // 默认核销 Tab 的下拉入口在对冲 Tab 下不再渲染。
    expect(
      screen.queryByRole('button', { name: '加载有效应收核销' }),
    ).not.toBeInTheDocument();
  });

  it('对冲来源预览与创建请求体携带 nettingId 而不带 verificationId', async () => {
    serviceMocks.listCommissionNettingCandidates.mockResolvedValue({
      data: [nettingCandidate],
    });
    serviceMocks.previewCommission.mockResolvedValue({
      data: {
        employeeName: '张三',
        ruleName: '销售提成',
        ruleVersion: '1',
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: '2.5000',
        baseCurrency: 'CNY',
        commissionAmount: '56.00000000',
        cnyCommissionAmount: '56.00000000',
        cnyExchangeRate: '1',
        cnyExchangeRateDate: '2026-09-01',
        cnyExchangeRateSource: 'BASE_CURRENCY',
        nettingId: 'netting-1',
        nettingNo: 'NT-2026-001',
        lines: [],
      },
    });
    serviceMocks.createCommission.mockResolvedValue({ data: {} });
    render(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    modalState.dependencyValues = {
      nettingId: 'netting-1',
      ruleId: 'rule-1',
      employeeId: 'employee-1',
    };
    fireEvent.click(screen.getByRole('tab', { name: '对冲提成' }));
    fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));

    await waitFor(() =>
      expect(serviceMocks.previewCommission).toHaveBeenCalledWith(
        expect.objectContaining({
          nettingId: 'netting-1',
          employeeId: 'employee-1',
          ruleId: 'rule-1',
        }),
      ),
    );
    expect(serviceMocks.previewCommission.mock.calls[0][0]).not.toHaveProperty(
      'verificationId',
    );
    expect(await screen.findByText('来源单号')).toBeInTheDocument();
    expect(screen.getByText('NT-2026-001')).toBeInTheDocument();

    await act(async () => {
      await modalState.props?.onFinish({
        nettingId: 'netting-1',
        ruleId: 'rule-1',
        employeeId: 'employee-1',
      });
    });

    await waitFor(() =>
      expect(serviceMocks.createCommission).toHaveBeenCalledWith(
        expect.objectContaining({
          nettingId: 'netting-1',
          ruleId: 'rule-1',
          employeeId: 'employee-1',
        }),
      ),
    );
    expect(serviceMocks.createCommission.mock.calls[0][0]).not.toHaveProperty(
      'verificationId',
    );
  });

  it('核销候选未选公司不请求，慢 A 响应不会污染已切换的 B 公司', async () => {
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

    // 选择公司后自动拉取首批候选，无需手动触发。
    fireEvent.click(screen.getByRole('button', { name: '选择公司 A' }));
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

  it('核销候选输入关键字后防抖携带 keyword 重新拉取', async () => {
    serviceMocks.listCommissionVerificationCandidates.mockImplementation(
      ({ keyword }: { keyword?: string }) =>
        Promise.resolve({
          data: keyword
            ? [{ ...verificationCandidate, verificationNo: 'VR-A-MATCH' }]
            : [verificationCandidate],
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

    fireEvent.click(screen.getByRole('button', { name: '选择公司 A' }));
    expect(await screen.findByText('VR-A｜A单位｜200 CNY')).toBeInTheDocument();
    expect(
      serviceMocks.listCommissionVerificationCandidates,
    ).toHaveBeenCalledWith({
      page: 1,
      pageSize: 200,
      organizationId: 'org-a',
    });

    fireEvent.change(screen.getByLabelText('搜索有效应收核销'), {
      target: { value: 'A单位' },
    });
    // 防抖期内不发起搜索请求。
    expect(
      serviceMocks.listCommissionVerificationCandidates,
    ).toHaveBeenCalledTimes(1);

    await waitFor(() =>
      expect(
        serviceMocks.listCommissionVerificationCandidates,
      ).toHaveBeenLastCalledWith({
        page: 1,
        pageSize: 200,
        organizationId: 'org-a',
        keyword: 'A单位',
      }),
    );
    expect(
      await screen.findByText('VR-A-MATCH｜A单位｜200 CNY'),
    ).toBeInTheDocument();
  });

  it('对冲候选输入关键字后防抖携带 keyword，切换组织清空关键字并重拉', async () => {
    serviceMocks.listCommissionNettingCandidates.mockResolvedValue({
      data: [nettingCandidate],
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

    fireEvent.click(screen.getByRole('button', { name: '选择公司 A' }));
    fireEvent.click(screen.getByRole('tab', { name: '对冲提成' }));
    expect(await screen.findByText('NT-2026-001')).toBeInTheDocument();
    expect(serviceMocks.listCommissionNettingCandidates).toHaveBeenCalledWith({
      page: 1,
      pageSize: 200,
      organizationId: 'org-a',
    });

    fireEvent.change(screen.getByPlaceholderText('搜索对冲单号 / 结算单位'), {
      target: { value: '对冲' },
    });
    // 防抖期内不发起搜索请求。
    expect(serviceMocks.listCommissionNettingCandidates).toHaveBeenCalledTimes(
      1,
    );

    await waitFor(() =>
      expect(
        serviceMocks.listCommissionNettingCandidates,
      ).toHaveBeenLastCalledWith({
        page: 1,
        pageSize: 200,
        organizationId: 'org-a',
        keyword: '对冲',
      }),
    );

    fireEvent.click(screen.getByRole('button', { name: '选择公司 B' }));
    expect(screen.getByPlaceholderText('搜索对冲单号 / 结算单位')).toHaveValue(
      '',
    );
    await waitFor(() =>
      expect(
        serviceMocks.listCommissionNettingCandidates,
      ).toHaveBeenLastCalledWith({
        page: 1,
        pageSize: 200,
        organizationId: 'org-b',
      }),
    );
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
