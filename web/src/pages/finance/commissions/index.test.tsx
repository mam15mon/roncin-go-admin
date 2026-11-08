import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  act,
  cleanup,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, ConfigProvider } from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { FinanceCommissionStatus } from '@/enums.generated';

const serviceMocks = vi.hoisted(() => ({
  exportCommissions: vi.fn(),
  listCommissions: vi.fn(),
  getCommission: vi.fn(),
  confirmAdjustment: vi.fn(),
  cancelAdjustment: vi.fn(),
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

vi.mock('@/components/ui', async () => ({
  GuardedModalForm: (await import('@/components/ui/GuardedModalForm'))
    .GuardedModalForm,
  ResponsiveProTable: (await import('@ant-design/pro-components')).ProTable,
  SearchFilterTemplate: (props: Record<string, any>) => {
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
  settlementServiceListFinanceOrganizationOptions:
    serviceMocks.listFinanceOrganizationOptions,
  settlementServiceMarkCommissionAdjustmentPaid: vi.fn(),
  settlementServiceMarkCommissionPaid: vi.fn(),
}));

vi.mock('./components/CommissionAdjustmentModal', () => ({
  default: () => null,
}));
vi.mock('./components/CommissionCreateModal', () => ({ default: () => null }));
// 仅替换明细排版；动作和确认框均由真实页面、makeVersionActions 与 App 构造。
vi.mock('./components/CommissionDetailDrawer', () => ({
  default: (props: Record<string, any>) => {
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

import FinanceCommissionsPage from './index';

describe('提成台账动作', () => {
  afterEach(() => {
    cleanup();
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
    serviceMocks.listFinanceOrganizationOptions.mockReset();
    serviceMocks.listFinanceOrganizationOptions.mockResolvedValue({ data: [] });
  });

  it('系统管理配置权限保留考核规则维护入口，不授予提成办理能力', async () => {
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

    fireEvent.click(screen.getByText('待执行调整'));

    expect(screen.getByTestId('pending-decrease-panel')).toBeInTheDocument();

    fireEvent.click(screen.getByText('提成台账'));
    expect(
      screen.queryByTestId('pending-decrease-panel'),
    ).not.toBeInTheDocument();
  });

  const openEmbeddedAdjustment = async () => {
    accessState.canManageFinanceCommissions = true;
    const adjustment = {
      id: 'adjustment-beijing',
      adjustmentNo: 'ADJ001',
      organizationId: 'company-beijing',
      organizationName: '融迅（北京）供应链管理有限公司',
      direction: 'INCREASE',
      amount: '1.00000000',
      status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT,
      sourceType: 'MANUAL',
      version: '7',
    };
    serviceMocks.listFinanceOrganizationOptions.mockResolvedValue({
      data: [{ id: 'company-tianjin', name: '融迅天津当前工作台' }],
    });
    serviceMocks.getCommission.mockResolvedValue({
      data: {
        id: 'commission-beijing',
        organizationId: adjustment.organizationId,
        organizationName: adjustment.organizationName,
        adjustments: [adjustment],
      },
    });
    renderWithClient(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <FinanceCommissionsPage />
        </App>
      </ConfigProvider>,
    );
    await act(async () => {});
    const actionColumn = componentProps.proTable?.columns.find(
      (column: Record<string, any>) => column.valueType === 'option',
    );
    if (!actionColumn) throw new Error('提成操作列不存在');
    renderWithClient(
      <div>{actionColumn.render(undefined, { id: 'commission-beijing' })}</div>,
    );
    fireEvent.click(screen.getByText('明细'));
    await screen.findByRole('button', { name: '确认内嵌调整' });
    return adjustment;
  };

  it('确认内嵌调整显示记录真实公司，提交保留该记录版本', async () => {
    const adjustment = await openEmbeddedAdjustment();
    serviceMocks.confirmAdjustment.mockResolvedValue({
      data: { id: adjustment.id },
    });

    fireEvent.click(screen.getByRole('button', { name: '确认内嵌调整' }));

    const dialog = await screen.findByRole('dialog');
    expect(dialog).toHaveTextContent('确认调整 ADJ001？');
    expect(dialog).toHaveTextContent(
      `所属公司：${adjustment.organizationName}`,
    );
    expect(dialog).not.toHaveTextContent('所属公司未标识');
    expect(dialog).not.toHaveTextContent('融迅天津当前工作台');
    fireEvent.click(within(dialog).getByRole('button', { name: 'OK' }));
    await waitFor(() =>
      expect(serviceMocks.confirmAdjustment).toHaveBeenCalledWith(
        { id: adjustment.id },
        { id: adjustment.id, expectedVersion: '7' },
      ),
    );
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });

  it('取消内嵌调整显示记录真实公司，保留版本与必填取消原因', async () => {
    const adjustment = await openEmbeddedAdjustment();
    serviceMocks.cancelAdjustment.mockResolvedValue({
      data: { id: adjustment.id },
    });

    fireEvent.click(screen.getByRole('button', { name: '取消内嵌调整' }));

    const dialog = await screen.findByRole('dialog');
    expect(dialog).toHaveTextContent(
      `取消 ${adjustment.organizationName} 的调整 ADJ001？`,
    );
    expect(dialog).not.toHaveTextContent('所属公司未标识');
    expect(dialog).not.toHaveTextContent('融迅天津当前工作台');
    const submit = within(dialog).getByRole('button', { name: /确.*认/ });
    fireEvent.click(submit);
    expect(serviceMocks.cancelAdjustment).not.toHaveBeenCalled();
    fireEvent.change(
      within(dialog).getByPlaceholderText('请输入取消原因（必填）'),
      {
        target: { value: '  公司归属核对后取消  ' },
      },
    );
    fireEvent.click(submit);
    await waitFor(() =>
      expect(serviceMocks.cancelAdjustment).toHaveBeenCalledWith(
        { id: adjustment.id },
        {
          id: adjustment.id,
          expectedVersion: '7',
          reason: '公司归属核对后取消',
        },
      ),
    );
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });
});
