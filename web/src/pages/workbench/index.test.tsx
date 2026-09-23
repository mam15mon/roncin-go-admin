import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { WorkbenchCommissionStatus } from '@/enums.generated';

const serviceMocks = vi.hoisted(() => ({
  getOverview: vi.fn(),
  listMyCommissions: vi.fn(),
  listMyReceivables: vi.fn(),
  listMyRecentOrders: vi.fn(),
  submitApplication: vi.fn(),
  listApplicationCandidates: vi.fn(),
  listMyApplications: vi.fn(),
  getMyApplication: vi.fn(),
}));

const umiState = vi.hoisted(() => ({
  user: {
    id: 'user-1',
    displayName: '张三',
    username: 'zhangsan',
    permissions: [] as string[],
    roleScopes: [] as unknown[],
    currentOrganization: { id: 'org-1', name: '上海公司', code: 'SH01' },
  } as Record<string, unknown>,
  access: {
    canReadSEOrders: true,
    canOperateBusiness: true,
    canReadFinanceFees: false,
    canReadFinanceBills: false,
    canReadFinanceVerifications: false,
    canReadFinanceCommissions: false,
    canReadPartners: false,
    canReadMasterData: false,
  } as Record<string, boolean>,
  historyPush: vi.fn(),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({ initialState: { currentUser: umiState.user } }),
}));

vi.mock('@/app/access', () => ({
  useAccess: () => umiState.access,
}));

vi.mock('@/router/history', () => ({
  history: { push: umiState.historyPush },
}));

vi.mock('@ant-design/pro-components', () => ({
  PageContainer: ({
    title,
    extra,
    children,
  }: {
    title?: React.ReactNode;
    extra?: React.ReactNode;
    children?: React.ReactNode;
  }) => (
    <div>
      <div data-testid="page-title">{title}</div>
      <div data-testid="page-extra">{extra}</div>
      {children}
    </div>
  ),
  ProCard: ({
    title,
    extra,
    children,
    ...rest
  }: {
    title?: React.ReactNode;
    extra?: React.ReactNode;
    children?: React.ReactNode;
  } & Record<string, unknown>) => (
    <div data-testid={rest['data-testid'] as string | undefined}>
      {title ? <div>{title}</div> : null}
      {extra}
      {children}
    </div>
  ),
}));

vi.mock('@/services/roncin/workbenchService', () => ({
  workbenchServiceGetWorkbenchOverview: (...args: unknown[]) =>
    serviceMocks.getOverview(...args),
  workbenchServiceListMyCommissions: (...args: unknown[]) =>
    serviceMocks.listMyCommissions(...args),
  workbenchServiceListMyReceivables: (...args: unknown[]) =>
    serviceMocks.listMyReceivables(...args),
  workbenchServiceListMyRecentOrders: (...args: unknown[]) =>
    serviceMocks.listMyRecentOrders(...args),
  workbenchServiceSubmitMyCommissionApplication: (...args: unknown[]) =>
    serviceMocks.submitApplication(...args),
  workbenchServiceListMyApplicationCandidates: (...args: unknown[]) =>
    serviceMocks.listApplicationCandidates(...args),
  workbenchServiceListMyCommissionApplications: (...args: unknown[]) =>
    serviceMocks.listMyApplications(...args),
  workbenchServiceGetMyCommissionApplication: (...args: unknown[]) =>
    serviceMocks.getMyApplication(...args),
}));

import WorkbenchPage from './index';

function deferred<T>(): { promise: Promise<T>; resolve: (value: T) => void } {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

function renderUi() {
  return (
    <App>
      <WorkbenchPage />
    </App>
  );
}

function renderPage() {
  return renderWithClient(renderUi());
}

function overviewResponse(data: API.GetWorkbenchOverviewData) {
  return { success: true, data };
}

beforeEach(() => {
  umiState.user.currentOrganization = {
    id: 'org-1',
    name: '上海公司',
    code: 'SH01',
  };
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  umiState.access.canReadFinanceCommissions = false;
});

describe('工作台提成门禁', () => {
  it('门禁为 false：加载中与加载后从首次渲染起都无提成 DOM，仍显示近期订单与待办', async () => {
    const pending = deferred<API.GetWorkbenchOverviewResponse>();
    serviceMocks.getOverview.mockReturnValue(pending.promise);

    renderPage();

    // 加载中：只有骨架，无提成标题与入口。
    expect(screen.getByTestId('workbench-loading')).toBeInTheDocument();
    expect(screen.queryByText('我的提成')).not.toBeInTheDocument();
    expect(screen.queryByText('在途回款')).not.toBeInTheDocument();

    pending.resolve(
      overviewResponse({
        hasCommissionEligibility: false,
        recentOrders: [
          {
            orderId: 'o-1',
            orderNo: 'SE2026090001',
            customerName: '客户甲',
            flowStatus: 'BOOKED',
            terminationStatus: 'ACTIVE',
            orderDate: '2026-09-01',
          },
        ],
        todos: { unbilledFeeCount: 2, openAbnormalCount: 0 },
      }) as never,
    );

    await waitFor(() =>
      expect(screen.getByText('SE2026090001')).toBeInTheDocument(),
    );
    expect(screen.getByText('我的作业待办')).toBeInTheDocument();
    expect(screen.queryByText('我的提成')).not.toBeInTheDocument();
    expect(serviceMocks.getOverview).toHaveBeenCalledTimes(1);
  });

  it('门禁为 undefined（false 被省略）：无提成 DOM，职能人员只看欢迎/组织/账号边界/快捷入口', async () => {
    const pending = deferred<API.GetWorkbenchOverviewResponse>();
    serviceMocks.getOverview.mockReturnValue(pending.promise);

    renderPage();

    expect(screen.queryByText('我的提成')).not.toBeInTheDocument();

    pending.resolve(
      overviewResponse({
        hasCommissionEligibility: undefined as unknown as boolean,
      }) as never,
    );

    await waitFor(() =>
      expect(screen.getByText('账号与数据边界')).toBeInTheDocument(),
    );
    expect(screen.getByText('当前组织：上海公司')).toBeInTheDocument();
    expect(screen.getByText('快捷入口')).toBeInTheDocument();
    expect(screen.queryByText('我的提成')).not.toBeInTheDocument();
    expect(screen.queryByText('我负责的近期订单')).not.toBeInTheDocument();
    expect(screen.queryByText('我的作业待办')).not.toBeInTheDocument();
    expect(screen.queryByText('财务与审批待办')).not.toBeInTheDocument();
  });

  it('门禁为 true 且无数据：展示业务空态与未来生效提示，不把未生效解释为零', async () => {
    serviceMocks.getOverview.mockResolvedValue(
      overviewResponse({
        hasCommissionEligibility: true,
        nextEffectiveDate: '2026-10-01',
        baseCurrency: 'CNY',
      }),
    );

    renderPage();

    expect(await screen.findByText('我的提成')).toBeInTheDocument();
    expect(
      screen.getByText(/您的提成方案自 2026-10-01 起生效/),
    ).toBeInTheDocument();
    expect(
      screen.getByText('当前组织暂无可计提、待发或已发的提成记录'),
    ).toBeInTheDocument();
    // 无法估算时明确说明原因，不显示假精确金额。
    expect(screen.getByText('暂无法估算')).toBeInTheDocument();
    // 冲减摘要行：无任何冲减记录时不渲染。
    expect(screen.queryByText('冲减')).not.toBeInTheDocument();
  });
});

describe('工作台提成分桶与冲减语义', () => {
  const summaryData: API.GetWorkbenchOverviewData = {
    hasCommissionEligibility: true,
    baseCurrency: 'CNY',
    commissionSummary: {
      baseCurrency: 'CNY',
      draftCount: 1,
      draftAmount: '500.00',
      confirmedCount: 2,
      confirmedAmount: '1200.50',
      paidCount: 3,
      paidAmount: '9000.00',
      paidAmountThisYear: '3200.00',
      paidAmountThisMonth: '800.00',
      decreaseDraftCount: 1,
      decreaseDraftAmount: '100.00',
      decreaseConfirmedCount: 1,
      decreaseConfirmedAmount: '50.00',
      decreasePaidCount: 1,
      decreasePaidAmount: '30.00',
    },
  };

  it('三桶流程条与冲减单行摘要按工作台语义展示，本年/本月已发双指标同屏', async () => {
    serviceMocks.getOverview.mockResolvedValue(overviewResponse(summaryData));

    renderPage();

    expect(await screen.findByText('待财务确认')).toBeInTheDocument();
    expect(screen.getByText('已确认待发')).toBeInTheDocument();
    expect(screen.getByText('已发放')).toBeInTheDocument();
    expect(screen.getByText('1,200.50')).toBeInTheDocument();
    // 冲减降级为单行摘要：待处理带笔数，已确认/已扣回只列金额。
    expect(screen.getByText(/待处理 100\.00 · 1 笔/)).toBeInTheDocument();
    expect(screen.getByText(/已确认 50\.00/)).toBeInTheDocument();
    expect(screen.getByText(/已扣回 30\.00/)).toBeInTheDocument();

    // hero 双指标同屏：本年已发主数字与本月已发副指标同时可见，无需切换。
    expect(screen.getByText('本年已发')).toBeInTheDocument();
    expect(screen.getByText('3,200.00')).toBeInTheDocument();
    expect(screen.getByText('本月已发')).toBeInTheDocument();
    expect(screen.getByText('800.00')).toBeInTheDocument();
  });

  it('点击查看提成明细打开下钻抽屉：服务端分页、状态过滤与翻页', async () => {
    serviceMocks.getOverview.mockResolvedValue(overviewResponse(summaryData));
    const firstPage = deferred<API.ListMyCommissionsResponse>();
    serviceMocks.listMyCommissions.mockReturnValueOnce(firstPage.promise);

    renderPage();

    fireEvent.click(await screen.findByRole('button', { name: /提成明细/ }));
    expect(serviceMocks.listMyCommissions).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
    });

    firstPage.resolve({
      success: true,
      total: '45',
      page: 1,
      pageSize: 20,
      data: [
        {
          id: 'c-1',
          commissionNo: 'FC2026090001',
          status:
            WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_CONFIRMED,
          personnelRole: 'SALES',
          ruleName: '销售方案A',
          calculationBasis: 'REALIZED_PROFIT',
          baseCurrency: 'CNY',
          commissionAmount: '1200.50',
          commissionDate: '2026-09-01',
          verificationNo: 'VR-001',
          adjustments: [
            {
              id: 'adj-1',
              adjustmentNo: 'ADJ-001',
              direction: 'DECREASE',
              status:
                WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_DRAFT,
              amount: '100.00',
              reason: '锁后补录',
            },
          ],
        },
      ],
    } as never);

    expect(await screen.findByText('FC2026090001')).toBeInTheDocument();
    expect(screen.getByText('共 45 条')).toBeInTheDocument();
    // 抽屉行状态 Tag 与主卡桶文案同源：卡片 1 处 + 行内 Tag 1 处。
    expect(screen.getAllByText('已确认待发')).toHaveLength(2);

    // 状态过滤：切换状态后回到第 1 页并携带 status。
    const filtered = deferred<API.ListMyCommissionsResponse>();
    serviceMocks.listMyCommissions.mockReturnValueOnce(filtered.promise);
    fireEvent.mouseDown(screen.getAllByRole('combobox')[0]);
    const draftOption = await waitFor(() => {
      const option = document.querySelector('[title="待财务确认"]');
      expect(option).toBeTruthy();
      return option as Element;
    });
    fireEvent.click(draftOption);
    expect(serviceMocks.listMyCommissions).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 20,
      status: WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_DRAFT,
    });
    filtered.resolve({ success: true, total: '45', data: [] } as never);
    await waitFor(() => expect(filtered.promise).resolves.toBeTruthy());

    // 翻页：点击第 2 页请求携带 page=2，且保留状态过滤。
    const page2 = deferred<API.ListMyCommissionsResponse>();
    serviceMocks.listMyCommissions.mockReturnValueOnce(page2.promise);
    const pager = screen.getByTitle('2');
    fireEvent.click(pager);
    expect(serviceMocks.listMyCommissions).toHaveBeenLastCalledWith({
      page: 2,
      pageSize: 20,
      status: WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_DRAFT,
    });
    page2.resolve({ success: true, total: '45', data: [] } as never);
    await waitFor(() =>
      expect(screen.getByTitle('2')).toHaveClass('ant-pagination-item-active'),
    );
  });

  it('在途回款抽屉：逐票原币余额与逾期状态，分页请求', async () => {
    serviceMocks.getOverview.mockResolvedValue(overviewResponse(summaryData));
    serviceMocks.listMyReceivables.mockResolvedValue({
      success: true,
      total: '2',
      data: [
        {
          billId: 'b-1',
          billNo: 'BILL-001',
          settlementPartyName: '客户甲',
          currency: 'USD',
          totalAmount: '10000.00',
          unverifiedAmount: '4000.00',
          dueDate: '2026-08-01',
          overdueDays: 12,
        },
        {
          billId: 'b-2',
          billNo: 'BILL-002',
          settlementPartyName: '客户乙',
          currency: 'EUR',
          totalAmount: '5000.00',
          unverifiedAmount: '500.00',
          dueDate: '2026-12-01',
          overdueDays: 0,
        },
      ],
    });

    renderPage();

    fireEvent.click(await screen.findByRole('button', { name: /在途回款/ }));
    expect(serviceMocks.listMyReceivables).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
    });

    expect(await screen.findByText('BILL-001')).toBeInTheDocument();
    // 原币余额按币种并列展示，不合并。
    expect(screen.getByText('4,000.00 USD')).toBeInTheDocument();
    expect(screen.getByText('500.00 EUR')).toBeInTheDocument();
    expect(screen.getByText('逾期 12 天')).toBeInTheDocument();
    expect(screen.getByText('未逾期')).toBeInTheDocument();
  });
});

describe('工作台财务卡与组织切换', () => {
  it('补录审批待办按实时资格出现，读取权限控制冲减与待发摘要', async () => {
    serviceMocks.getOverview.mockResolvedValue(
      overviewResponse({
        hasCommissionEligibility: false,
        finance: {
          canReadCommission: undefined as unknown as boolean,
          canManageCommission: undefined as unknown as boolean,
          pendingSupplementApprovalCount: 1,
          pendingSupplementApprovals: [
            {
              requestId: 'req-1',
              orderId: 'order-1',
              orderNo: 'SE2026090099',
              feeName: '滞箱费',
              currency: 'CNY',
              amount: '900.00',
              reason: '锁后补录',
              requestedByName: '李四',
              requestedAt: '2026-09-10 10:00:00',
            },
          ],
        },
      }),
    );

    renderPage();

    expect(await screen.findByText('财务与审批待办')).toBeInTheDocument();
    expect(screen.getByText('1 条待审批')).toBeInTheDocument();
    expect(screen.getByText('SE2026090099')).toBeInTheDocument();
    expect(screen.queryByText('待处理冲减建议')).not.toBeInTheDocument();
    expect(
      screen.queryByText('已确认待发提成（组织范围）'),
    ).not.toBeInTheDocument();

    // 前往处理跳转既有费用录入页，不发明新审批动作。
    fireEvent.click(screen.getByRole('button', { name: '前往处理' }));
    expect(umiState.historyPush).toHaveBeenCalledWith(
      '/orders/sea-export/order-1/fees',
    );
  });

  it('持有提成读取权限时展示冲减与已确认待发摘要入口', async () => {
    serviceMocks.getOverview.mockResolvedValue(
      overviewResponse({
        hasCommissionEligibility: false,
        finance: {
          canReadCommission: true,
          canManageCommission: true,
          confirmedCommissionCount: 7,
          pendingDecreaseCount: 3,
          pendingSupplementApprovalCount: 0,
        },
      }),
    );

    renderPage();

    expect(await screen.findByText('财务与审批待办')).toBeInTheDocument();
    expect(screen.getByText('3 条（尚未扣回）')).toBeInTheDocument();
    expect(screen.getByText('7 笔')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '去处理冲减' }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '去处理冲减' }));
    expect(umiState.historyPush).toHaveBeenCalledWith('/finance/commissions');
  });

  it('组织切换迟到响应不覆盖当前页面', async () => {
    const org1 = deferred<API.GetWorkbenchOverviewResponse>();
    serviceMocks.getOverview.mockReturnValueOnce(org1.promise);

    const utils = renderPage();
    expect(serviceMocks.getOverview).toHaveBeenCalledTimes(1);

    // 切换组织：旧请求作废，新请求发出。
    const org2 = deferred<API.GetWorkbenchOverviewResponse>();
    serviceMocks.getOverview.mockReturnValueOnce(org2.promise);
    umiState.user.currentOrganization = {
      id: 'org-2',
      name: '成都公司',
      code: 'CD01',
    };
    utils.rerender(
      <QueryClientProvider client={utils.queryClient}>
        {renderUi()}
      </QueryClientProvider>,
    );
    expect(serviceMocks.getOverview).toHaveBeenCalledTimes(2);

    // 旧组织响应迟到：携带提成与月度申请数据，但必须被丢弃。
    org1.resolve(
      overviewResponse({
        hasCommissionEligibility: true,
        baseCurrency: 'USD',
        commissionSummary: {
          paidAmountThisYear: '99999.00',
          paidCount: 9,
          paidAmount: '99999.00',
        },
        applicationSummary: {
          baseCurrency: 'USD',
          applyGroups: [
            {
              commissionMonth: '2026-07',
              commissionCount: 9,
              commissionAmount: '88888.00',
            },
          ],
          accumulatingCount: 1,
          accumulatingAmount: '10.00',
          pendingReviewCount: 0,
          approvedCount: 0,
        },
      }) as never,
    );

    // 新组织响应：门禁为假。
    org2.resolve(
      overviewResponse({
        hasCommissionEligibility: false,
        recentOrders: [
          {
            orderId: 'o-2',
            orderNo: 'SE-CD-001',
            customerName: '成都客户',
            flowStatus: 'DRAFT',
            terminationStatus: 'ACTIVE',
          },
        ],
      }) as never,
    );

    await waitFor(() =>
      expect(screen.getByText('SE-CD-001')).toBeInTheDocument(),
    );
    expect(screen.getByText('当前组织：成都公司')).toBeInTheDocument();
    expect(screen.queryByText('我的提成')).not.toBeInTheDocument();
    expect(screen.queryByText('99999.00')).not.toBeInTheDocument();
    expect(screen.queryByText('月度申请')).not.toBeInTheDocument();
    expect(screen.queryByText('88888.00')).not.toBeInTheDocument();
  });

  it('Overview 携带月度申请摘要时展示可申请分组与本月累计', async () => {
    serviceMocks.getOverview.mockResolvedValue(
      overviewResponse({
        hasCommissionEligibility: true,
        baseCurrency: 'CNY',
        commissionSummary: {
          draftCount: 1,
          draftAmount: '500.00',
        },
        applicationSummary: {
          baseCurrency: 'CNY',
          applyGroups: [
            {
              commissionMonth: '2026-07',
              commissionCount: 3,
              commissionAmount: '500.00',
            },
          ],
          accumulatingCount: 2,
          accumulatingAmount: '300.00',
          pendingReviewCount: 1,
          approvedCount: 1,
        },
      }),
    );

    renderPage();

    expect(await screen.findByText('月度申请')).toBeInTheDocument();
    expect(screen.getByText('2026-07')).toBeInTheDocument();
    expect(screen.getByText('合计 3 笔 · CNY')).toBeInTheDocument();
    expect(screen.getByText('本月累计中')).toBeInTheDocument();
    expect(screen.getByText('审批中 1 张')).toBeInTheDocument();
    expect(screen.getByText('已批准 1 张')).toBeInTheDocument();
    // 累计中不提供申请能力：入口只服务可申请分组。
    expect(screen.queryByRole('button', { name: /去\s*申\s*请/ })).toBeTruthy();
    expect(screen.getByTestId('apply-submit-button')).not.toHaveAttribute(
      'disabled',
    );
  });
});
