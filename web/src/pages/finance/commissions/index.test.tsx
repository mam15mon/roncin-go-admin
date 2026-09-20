import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FinanceCommissionStatus } from '@/enums.generated';

const serviceMocks = vi.hoisted(() => ({
  exportCommissions: vi.fn(),
  listCommissions: vi.fn(),
  listFinanceOrganizationOptions: vi.fn().mockResolvedValue({ data: [] }),
}));

const componentProps = vi.hoisted(() => ({
  searchFilter: undefined as Record<string, any> | undefined,
  proTable: undefined as Record<string, any> | undefined,
}));

const accessState = vi.hoisted(() => ({
  canOperateOrganization: () => true,
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
  ProTable: (props: Record<string, any>) => {
    componentProps.proTable = props;
    return null;
  },
}));

vi.mock('@/components/ui', () => ({
  SearchFilterTemplate: (props: Record<string, any>) => {
    componentProps.searchFilter = props;
    return <>{props.extraRight}</>;
  },
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCancelCommission: vi.fn(),
  settlementServiceCancelCommissionAdjustment: vi.fn(),
  settlementServiceConfirmCommission: vi.fn(),
  settlementServiceConfirmCommissionAdjustment: vi.fn(),
  settlementServiceExportCommissions: serviceMocks.exportCommissions,
  settlementServiceGetCommission: vi.fn(),
  settlementServiceListCommissions: serviceMocks.listCommissions,
  settlementServiceListFinanceOrganizationOptions:
    serviceMocks.listFinanceOrganizationOptions,
  settlementServiceMarkCommissionAdjustmentPaid: vi.fn(),
  settlementServiceMarkCommissionPaid: vi.fn(),
}));

vi.mock('@/utils/versionActions', () => ({
  makeVersionActions: () => ({ confirm: vi.fn() }),
}));

vi.mock('./components/CommissionAdjustmentModal', () => ({
  default: () => null,
}));
vi.mock('./components/CommissionCreateModal', () => ({ default: () => null }));
vi.mock('./components/CommissionDetailDrawer', () => ({ default: () => null }));
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

import FinanceCommissionsPage from './index';

describe('提成导出按钮', () => {
  afterEach(() => {
    cleanup();
    accessState.canExportFinanceCommissions = false;
    accessState.canConfigureFinanceCommissions = false;
    componentProps.searchFilter = undefined;
    componentProps.proTable = undefined;
    panelState.renderCount = 0;
    vi.restoreAllMocks();
    serviceMocks.exportCommissions.mockReset();
    serviceMocks.listCommissions.mockReset();
    serviceMocks.listFinanceOrganizationOptions.mockReset();
    serviceMocks.listFinanceOrganizationOptions.mockResolvedValue({ data: [] });
  });

  it('总部配置权限保留考核规则维护入口，不授予提成办理能力', async () => {
    accessState.canConfigureFinanceCommissions = true;
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    await act(async () => {});
    expect(
      screen.getByRole('button', { name: /考核规则/ }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /生成提成/ }),
    ).not.toBeInTheDocument();
  });

  it('有导出权限时显示按钮', async () => {
    accessState.canExportFinanceCommissions = true;
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    // 等待挂载触发的公司候选请求在 act 内收敛。
    await act(async () => {});
    expect(
      screen.getByRole('button', { name: /导出提成/ }),
    ).toBeInTheDocument();
  });

  it('无导出权限时隐藏按钮', async () => {
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    // 等待挂载触发的公司候选请求在 act 内收敛。
    await act(async () => {});
    expect(
      screen.queryByRole('button', { name: /导出提成/ }),
    ).not.toBeInTheDocument();
  });

  it('列表和导出使用同一份规范化筛选', async () => {
    accessState.canExportFinanceCommissions = true;
    serviceMocks.listCommissions.mockResolvedValue({ data: [], total: 0 });
    serviceMocks.exportCommissions.mockResolvedValue({ data: [] });
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    // 等待挂载触发的公司候选请求在 act 内收敛。
    await act(async () => {});

    const searchValues = {
      keyword: '  COM-001  ',
      status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
      commissionMonth: [dayjs('2026-07-01'), dayjs('2026-08-01')],
    };
    act(() => componentProps.searchFilter?.onSearch(searchValues));
    await componentProps.proTable?.request({ current: 2, pageSize: 20 });
    fireEvent.click(screen.getByRole('button', { name: /导出提成/ }));

    const expectedFilters = {
      keyword: 'COM-001',
      status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
      commissionDateFrom: '2026-07-01',
      commissionDateTo: '2026-08-31',
    };
    expect(serviceMocks.listCommissions).toHaveBeenCalledWith({
      page: 2,
      pageSize: 20,
      ...expectedFilters,
    });
    await waitFor(() =>
      expect(serviceMocks.exportCommissions).toHaveBeenCalledWith(
        expectedFilters,
      ),
    );
  });

  it('空结果不创建下载对象', async () => {
    accessState.canExportFinanceCommissions = true;
    serviceMocks.exportCommissions.mockResolvedValue({ data: [] });
    const createObjectURL = vi.spyOn(URL, 'createObjectURL');
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: /导出提成/ }));

    await waitFor(() =>
      expect(serviceMocks.exportCommissions).toHaveBeenCalled(),
    );
    expect(createObjectURL).not.toHaveBeenCalled();
  });

  it('成功下载后移除链接并回收 Blob URL', async () => {
    accessState.canExportFinanceCommissions = true;
    serviceMocks.exportCommissions.mockResolvedValue({
      data: [{ commissionNo: 'COM-001' }],
    });
    const createObjectURL = vi
      .spyOn(URL, 'createObjectURL')
      .mockReturnValue('blob:commission-export');
    const revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL');
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined);
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: /导出提成/ }));

    await waitFor(() => expect(click).toHaveBeenCalledOnce());
    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:commission-export');
    expect(document.querySelector('a[download]')).toBeNull();
  });

  it('列表来源单号按核销或对冲二选一展示，两者都空显示占位符', async () => {
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    // 等待挂载触发的公司候选请求在 act 内收敛。
    await act(async () => {});

    const columns: Record<string, any>[] =
      componentProps.proTable?.columns ?? [];
    const sourceColumn = columns.find((column) => column.title === '来源单号');
    expect(sourceColumn).toBeTruthy();
    if (!sourceColumn) throw new Error('来源单号列不存在');
    expect(
      sourceColumn.render(undefined, { verificationNo: 'VR-2026-001' }),
    ).toBe('VR-2026-001');
    expect(sourceColumn.render(undefined, { nettingNo: 'NT-2026-001' })).toBe(
      'NT-2026-001',
    );
    expect(sourceColumn.render(undefined, {})).toBe('-');
  });

  it('切换到待处理冲减视图时渲染冲减面板，可切回台账', async () => {
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    // 等待挂载触发的公司候选请求在 act 内收敛。
    await act(async () => {});
    expect(
      screen.queryByTestId('pending-decrease-panel'),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('待处理冲减'));

    expect(screen.getByTestId('pending-decrease-panel')).toBeInTheDocument();

    fireEvent.click(screen.getByText('提成台账'));
    expect(
      screen.queryByTestId('pending-decrease-panel'),
    ).not.toBeInTheDocument();
  });
});
