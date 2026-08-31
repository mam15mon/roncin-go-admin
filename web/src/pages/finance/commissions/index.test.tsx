import {
  act,
  cleanup,
  fireEvent,
  render,
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
}));

const componentProps = vi.hoisted(() => ({
  searchFilter: undefined as Record<string, any> | undefined,
  proTable: undefined as Record<string, any> | undefined,
}));

const accessState = vi.hoisted(() => ({
  canExportFinanceCommissions: false,
  canManageFinanceCommissions: false,
}));

vi.mock('@umijs/max', () => ({
  useAccess: () => accessState,
}));

vi.mock('@ant-design/pro-components', () => ({
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

import FinanceCommissionsPage from './index';

describe('提成导出按钮', () => {
  afterEach(() => {
    cleanup();
    accessState.canExportFinanceCommissions = false;
    componentProps.searchFilter = undefined;
    componentProps.proTable = undefined;
    vi.restoreAllMocks();
    serviceMocks.exportCommissions.mockReset();
    serviceMocks.listCommissions.mockReset();
  });

  it('有导出权限时显示按钮', () => {
    accessState.canExportFinanceCommissions = true;
    render(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    expect(
      screen.getByRole('button', { name: /导出提成/ }),
    ).toBeInTheDocument();
  });

  it('无导出权限时隐藏按钮', () => {
    render(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    expect(
      screen.queryByRole('button', { name: /导出提成/ }),
    ).not.toBeInTheDocument();
  });

  it('列表和导出使用同一份规范化筛选', async () => {
    accessState.canExportFinanceCommissions = true;
    serviceMocks.listCommissions.mockResolvedValue({ data: [], total: 0 });
    serviceMocks.exportCommissions.mockResolvedValue({ data: [] });
    render(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );

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
    render(
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
    render(
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
});
