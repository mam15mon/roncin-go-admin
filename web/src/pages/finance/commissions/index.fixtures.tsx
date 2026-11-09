import { cleanup } from '@testing-library/react';
import React from 'react';
import { vi } from 'vitest';

type PageTableProps = {
  columns: {
    valueType?: string;
    render: (
      value: undefined,
      record: Partial<API.FinanceCommission>,
    ) => React.ReactNode;
  }[];
  request: (params: { current: number; pageSize: number }) => Promise<unknown>;
};
type PageSearchProps = {
  items?: { name: string }[];
  extraRight?: React.ReactNode;
  onSearch: (values: Record<string, unknown>) => void;
  onReset: () => void;
};
type DetailProps = {
  detail?: API.FinanceCommission;
  onTransitionAdjustment: (
    record: API.FinanceCommissionAdjustment,
    target: 'CONFIRMED',
  ) => void;
  onCancelAdjustment: (record: API.FinanceCommissionAdjustment) => void;
};

const workspace = vi.hoisted(() => ({
  organizationId: 'org-a' as string | undefined,
  userId: 'user-1' as string | undefined,
  canOperate: true,
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
const serviceMocks = vi.hoisted(() => ({
  exportCommissions: vi.fn(),
  listCommissions: vi.fn(),
  getCommission: vi.fn(),
  confirmAdjustment: vi.fn(),
  cancelAdjustment: vi.fn(),
}));

const componentProps = vi.hoisted(() => ({
  searchFilter: undefined as PageSearchProps | undefined,
  proTable: undefined as PageTableProps | undefined,
}));

const accessState = vi.hoisted(() => ({
  canOperateOrganization: (id?: string) =>
    workspace.canOperate && Boolean(id) && id === workspace.organizationId,
  canReadFinanceCommissions: true,
  canConfigureFinanceCommissions: false,
  canExportFinanceCommissions: false,
  canManageFinanceCommissions: false,
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState,
}));

vi.mock('@ant-design/pro-components', () => ({
  PageContainer: ({
    children,
    extra,
  }: {
    children?: React.ReactNode;
    extra?: React.ReactNode;
  }) => (
    <>
      <div data-testid="page-extra">{extra}</div>
      {children}
    </>
  ),
  ProTable: (props: PageTableProps) => {
    componentProps.proTable = props;
    return null;
  },
}));

vi.mock('@/components/ui', async () => ({
  GuardedModalForm: (await import('@/components/ui/GuardedModalForm'))
    .GuardedModalForm,
  ResponsiveProTable: (await import('@ant-design/pro-components')).ProTable,
  SearchFilterTemplate: (props: PageSearchProps) => {
    componentProps.searchFilter = props;
    return <>{props.extraRight}</>;
  },
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCancelCommission: vi.fn(),
  settlementServiceCancelCommissionAdjustment: serviceMocks.cancelAdjustment,
  settlementServiceConfirmCommission: vi.fn(),
  settlementServiceConfirmCommissionAdjustment: serviceMocks.confirmAdjustment,
  settlementServiceExportCommissions: serviceMocks.exportCommissions,
  settlementServiceGetCommission: serviceMocks.getCommission,
  settlementServiceListCommissions: serviceMocks.listCommissions,
  settlementServiceMarkCommissionAdjustmentPaid: vi.fn(),
  settlementServiceMarkCommissionPaid: vi.fn(),
}));

vi.mock('./components/CommissionAdjustmentModal', () => ({
  default: () => null,
}));
vi.mock('./components/CommissionCreateModal', () => ({ default: () => null }));
// 仅替换明细排版；动作和确认框均由真实页面、makeVersionActions 与 App 构造。
vi.mock('./components/CommissionDetailDrawer', () => ({
  default: (props: DetailProps) => {
    const adjustment = props.detail?.adjustments?.[0];
    if (!adjustment) return null;
    return (
      <>
        <button
          type="button"
          onClick={() => props.onTransitionAdjustment(adjustment, 'CONFIRMED')}
        >
          确认内嵌调整
        </button>
        <button
          type="button"
          onClick={() => props.onCancelAdjustment(adjustment)}
        >
          取消内嵌调整
        </button>
      </>
    );
  },
}));
vi.mock('./components/CommissionRulesDrawer', () => ({ default: () => null }));

const panelState = vi.hoisted(() => ({
  renderCount: 0,
}));

vi.mock('./components/PendingDecreasePanel', () => ({
  default: () => {
    panelState.renderCount += 1;
    return <div data-testid="pending-decrease-panel">待处理冲减视图</div>;
  },
}));

import ActualFinanceCommissionsPage from './index';

export const FinanceCommissionsPage = ActualFinanceCommissionsPage;
export { accessState, componentProps, panelState, serviceMocks, workspace };
export function resetCommissionPageMocks() {
  cleanup();
  workspace.organizationId = 'org-a';
  workspace.userId = 'user-1';
  workspace.canOperate = true;
  accessState.canReadFinanceCommissions = true;
  accessState.canManageFinanceCommissions = false;
  accessState.canExportFinanceCommissions = false;
  accessState.canConfigureFinanceCommissions = false;
  componentProps.searchFilter = undefined;
  componentProps.proTable = undefined;
  panelState.renderCount = 0;
  vi.restoreAllMocks();
  serviceMocks.exportCommissions.mockReset();
  serviceMocks.listCommissions.mockReset();
  serviceMocks.getCommission.mockReset();
  serviceMocks.confirmAdjustment.mockReset();
  serviceMocks.cancelAdjustment.mockReset();
}
