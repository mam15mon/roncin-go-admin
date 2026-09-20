import { act, cleanup, render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { FinanceNettingStatus } from '@/enums.generated';

const serviceMocks = vi.hoisted(() => ({
  cancelNetting: vi.fn(),
  confirmNetting: vi.fn(),
  listNettings: vi.fn(),
  listFinanceOrganizationOptions: vi.fn().mockResolvedValue({ data: [] }),
  reverseNetting: vi.fn(),
}));

const templateProps: { columns?: any[] } = {};

const accessState = vi.hoisted(() => ({
  canOperateOrganization: () => true,
  canConfigureFinanceCommissions: false,
  canConfirmFinanceNettings: true,
  canReadFinanceNettings: true,
  canReverseFinanceNettings: true,
}));

const appMocks = vi.hoisted(() => ({
  message: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
}));

const reasonFlow = vi.hoisted(() => ({
  submit: undefined as ((reason: string) => Promise<void> | void) | undefined,
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState,
}));

vi.mock('@/components/ui', async () => {
  const React = await import('react');
  return {
    FinanceLedgerTemplate: (props: Record<string, any>) => {
      templateProps.columns = props.columns;
      React.useEffect(() => {
        void props.request?.({ current: 1, pageSize: 20 });
      }, []);
      return null;
    },
  };
});

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCancelNetting: serviceMocks.cancelNetting,
  settlementServiceConfirmNetting: serviceMocks.confirmNetting,
  settlementServiceListFinanceOrganizationOptions:
    serviceMocks.listFinanceOrganizationOptions,
  settlementServiceListNettings: serviceMocks.listNettings,
  settlementServiceReverseNetting: serviceMocks.reverseNetting,
}));

vi.mock('@/utils/confirmWithReason', () => ({
  confirmWithReason: (
    _app: unknown,
    _title: string,
    onSubmit: (reason: string) => Promise<void> | void,
  ) => {
    reasonFlow.submit = onSubmit;
  },
}));

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  const App = Object.assign(actual.App, {
    useApp: () => ({
      message: appMocks.message,
      modal: { confirm: vi.fn() },
    }),
  });
  return { ...actual, App };
});

import FinanceNettingsPage from './index';

function operationColumn() {
  const column = templateProps.columns?.find(
    (item) => item.valueType === 'option',
  );
  if (!column) throw new Error('未找到操作列');
  return column;
}

function renderOperationCell(record: API.FinanceNetting) {
  const nodes = operationColumn().render(null, record) as React.ReactElement[];
  const visible = nodes.filter(Boolean);
  return render(<div>{visible}</div>);
}

const confirmedNetting: API.FinanceNetting = {
  id: 'netting-1',
  nettingNo: 'NT-20260910-0001',
  status: FinanceNettingStatus.FINANCE_NETTING_STATUS_CONFIRMED,
  organizationId: 'org-1',
  organizationName: '验收公司',
  settlementPartyId: 'party-1',
  settlementPartyName: '验收同行',
  currency: 'CNY',
  amount: '70.00000000',
  baseCurrency: 'CNY',
  baseCurrencyAmount: '70.00000000',
  payableBaseAmount: '70.00000000',
  exchangeGainLoss: '0.00000000',
  version: '3',
  allocations: [],
};

const draftNetting: API.FinanceNetting = {
  ...confirmedNetting,
  id: 'netting-2',
  nettingNo: 'NT-20260910-0002',
  status: FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT,
  version: '1',
};

describe('对冲结算单管理页', () => {
  beforeEach(() => {
    serviceMocks.listNettings.mockResolvedValue({
      data: [],
      total: '0',
      summary: { amountsByBaseCurrency: [], confirmedCount: '0' },
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    reasonFlow.submit = undefined;
  });

  it('渲染列表列并请求已确认抵销汇总', async () => {
    render(<FinanceNettingsPage />);
    await waitFor(() => expect(serviceMocks.listNettings).toHaveBeenCalled());
    const titles = templateProps.columns?.map((column) => column.title);
    expect(titles).toContain('关键词');
    expect(titles).toContain('对冲单号');
    expect(titles).toContain('抵销金额');
    expect(titles).toContain('应收本币抵销额');
    expect(titles).toContain('应付本币抵销额');
    expect(titles).toContain('对冲汇差');
  });

  it('已确认对冲提供反转入口，反转请求携带 expectedVersion 与原因', async () => {
    const page = render(<FinanceNettingsPage />);
    await waitFor(() => expect(serviceMocks.listNettings).toHaveBeenCalled());

    const cell = renderOperationCell(confirmedNetting);
    expect(screen.getByText('反转')).toBeInTheDocument();
    act(() => {
      screen.getByText('反转').click();
    });
    serviceMocks.reverseNetting.mockResolvedValue({
      data: confirmedNetting,
    });
    const submit = reasonFlow.submit;
    expect(submit).toBeTypeOf('function');
    await act(async () => {
      await submit?.('回退对冲');
    });
    expect(serviceMocks.reverseNetting).toHaveBeenCalledWith(
      { id: 'netting-1' },
      { id: 'netting-1', expectedVersion: '3', reason: '回退对冲' },
    );
    cell.unmount();
    page.unmount();
  });

  it('草稿对冲提供确认入口且确认请求携带 expectedVersion', async () => {
    const page = render(<FinanceNettingsPage />);
    await waitFor(() => expect(serviceMocks.listNettings).toHaveBeenCalled());

    const cell = renderOperationCell(draftNetting);
    expect(screen.getByText('确认')).toBeInTheDocument();
    expect(screen.getByText('取消')).toBeInTheDocument();
    expect(screen.queryByText('反转')).not.toBeInTheDocument();
    serviceMocks.confirmNetting.mockResolvedValue({ data: draftNetting });
    act(() => {
      screen.getByText('确认').click();
    });
    await waitFor(() =>
      expect(serviceMocks.confirmNetting).toHaveBeenCalledWith(
        { id: 'netting-2' },
        { id: 'netting-2', expectedVersion: '1' },
      ),
    );
    cell.unmount();
    page.unmount();
  });

  it('无确认权限时不渲染确认与反转入口', async () => {
    accessState.canConfirmFinanceNettings = false;
    accessState.canReverseFinanceNettings = false;
    const page = render(<FinanceNettingsPage />);
    await waitFor(() => expect(serviceMocks.listNettings).toHaveBeenCalled());

    const cell = renderOperationCell(draftNetting);
    expect(screen.queryByText('确认')).not.toBeInTheDocument();
    expect(screen.queryByText('取消')).not.toBeInTheDocument();
    expect(screen.getByText('详情')).toBeInTheDocument();
    cell.unmount();
    page.unmount();
    accessState.canConfirmFinanceNettings = true;
    accessState.canReverseFinanceNettings = true;
  });
});
