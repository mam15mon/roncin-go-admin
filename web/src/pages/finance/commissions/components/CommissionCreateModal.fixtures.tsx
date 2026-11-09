import { Form } from 'antd';
import React from 'react';
import { vi } from 'vitest';
import type { CreateValues } from '../types';

type Option = { value: string; label?: React.ReactNode };
type ModalProps = {
  children?: React.ReactNode;
  onFinish: (
    values: CreateValues & { organizationId?: string },
  ) => Promise<boolean>;
  onValuesChange?: (
    values: Partial<CreateValues>,
    allValues: CreateValues,
  ) => void;
  submitter: { submitButtonProps: { disabled?: boolean } };
};
type SelectProps = {
  name?: string;
  label?: React.ReactNode;
  disabled?: boolean;
  options?: Option[];
  request?: () => Promise<Option[]>;
  fieldProps?: {
    onChange?: (value?: string) => void;
    onSearch?: (value: string) => void;
    loading?: boolean;
  };
};

const workspace = vi.hoisted(() => ({
  organizationId: 'org-a' as string | undefined,
  userId: 'user-1' as string | undefined,
  canOperate: true,
  canManage: true,
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: workspace.userId,
        currentOrganization: { id: workspace.organizationId },
      },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canReadFinanceCommissions: true,
    canConfigureFinanceCommissions: true,
    canManageFinanceCommissions: workspace.canManage,
    canOperateOrganization: (id?: string) =>
      workspace.canOperate && Boolean(id) && id === workspace.organizationId,
  }),
}));
const serviceMocks = vi.hoisted(() => ({
  createCommission: vi.fn(),
  previewCommission: vi.fn(),
  listCommissionVerificationCandidates: vi.fn(),
  listCommissionNettingCandidates: vi.fn(),
  listCommissionCandidates: vi.fn(),
}));

const modalState = vi.hoisted(() => ({
  props: undefined as ModalProps | undefined,
  dependencyValues: {
    verificationId: 'verification-1',
    candidateKey: 'employee-1|SALES',
  } as CreateValues,
}));

vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: ModalProps) => {
    modalState.props = props;
    // 组件内渲染了真实 Form.Item（对冲 Tab 的隐藏 nettingId 字段），
    // 测试替身必须像真实 ModalForm 一样提供 FormContext，否则 Field
    // 挂载会触发 "Can not find FormContext" 警告。
    return (
      <Form onFinish={props.onFinish} onValuesChange={props.onValuesChange}>
        {props.children}
      </Form>
    );
  },
  ProFormDependency: ({
    children,
  }: {
    children: (values: CreateValues) => React.ReactNode;
  }) => <>{children(modalState.dependencyValues)}</>,
  ProFormTextArea: () => null,
}));

vi.mock('@/components/ui', async () => ({
  GuardedModalForm: (await import('@/components/ui/GuardedModalForm'))
    .GuardedModalForm,
  ResponsiveTable: (await import('antd')).Table,
  MODAL_SIZE: { SM: 520, MD: 680, LG: 960, XL: 1200 },
  DRAWER_SIZE: { SM: 600, MD: 860, LG: 1080, XL: 1200 },
  ProFormSearchableSelect: (props: SelectProps) => {
    const [options, setOptions] = React.useState<Option[]>([]);
    return (
      <div>
        {props.label === '有效应收核销' && (
          <button
            type="button"
            disabled={props.disabled}
            onClick={() => props.fieldProps?.onChange?.('verification-1')}
          >
            选择核销单
          </button>
        )}
        {props.request && (
          <button
            type="button"
            onClick={async () => setOptions((await props.request?.()) ?? [])}
          >
            加载{props.label}
          </button>
        )}
        {(props.options ?? options).map((option: Option) => (
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
        {props.name === 'candidateKey' && (
          <span data-testid="candidate-disabled">{String(props.disabled)}</span>
        )}
        {props.fieldProps?.loading ? <span>候选加载中</span> : null}
      </div>
    );
  },
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCreateCommission: serviceMocks.createCommission,
  settlementServiceListCommissionCandidates:
    serviceMocks.listCommissionCandidates,
  settlementServiceListCommissionNettingCandidates:
    serviceMocks.listCommissionNettingCandidates,
  settlementServiceListCommissionRules: vi.fn(),
  settlementServiceListCommissionVerificationCandidates:
    serviceMocks.listCommissionVerificationCandidates,
  settlementServicePreviewCommission: serviceMocks.previewCommission,
}));

vi.mock('./CommissionLineTable', () => ({
  previewColumns: [],
  renderExpandedFees: () => null,
}));

import ActualCommissionCreateModal from './CommissionCreateModal';

const nettingCandidate = {
  id: 'netting-1',
  organizationId: 'org-a',
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

const commissionCandidate = {
  employeeId: 'employee-1',
  employeeName: '张三',
  personnelRole: 'SALES',
  customerCount: 1,
  orderCount: 2,
  feeCount: 3,
  baseCurrency: 'CNY',
  commissionAmount: '60.00000000',
  ruleId: 'rule-1',
  ruleName: '销售提成方案',
  calculationBasis: 'REALIZED_PROFIT',
  ruleVersion: '3',
};

export const CommissionCreateModal = ActualCommissionCreateModal;
export {
  modalState,
  nettingCandidate,
  serviceMocks,
  verificationCandidate,
  workspace,
};
export function resetCommissionModalMocks() {
  workspace.organizationId = 'org-a';
  workspace.userId = 'user-1';
  workspace.canOperate = true;
  workspace.canManage = true;
  modalState.props = undefined;
  modalState.dependencyValues = {
    verificationId: 'verification-1',
    candidateKey: 'employee-1|SALES',
  };
  for (const mock of Object.values(serviceMocks)) {
    mock.mockReset();
  }
  serviceMocks.listCommissionVerificationCandidates.mockResolvedValue({
    data: [
      {
        ...verificationCandidate,
        id: 'verification-1',
        organizationId: 'org-a',
      },
    ],
  });
  serviceMocks.listCommissionNettingCandidates.mockResolvedValue({
    data: [],
  });
  serviceMocks.listCommissionCandidates.mockResolvedValue({
    data: [commissionCandidate],
  });
  serviceMocks.previewCommission.mockResolvedValue({
    data: {
      employeeName: '张三',
      ruleName: '销售提成方案',
      ruleVersion: '3',
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
}
