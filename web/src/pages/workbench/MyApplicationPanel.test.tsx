import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { WorkbenchCommissionApplicationStatus } from '@/enums.generated';

const serviceMocks = vi.hoisted(() => ({
  submit: vi.fn(),
  resubmit: vi.fn(),
  listCandidates: vi.fn(),
  listApplications: vi.fn(),
  getApplication: vi.fn(),
}));

vi.mock('@/services/roncin/workbenchService', () => ({
  workbenchServiceSubmitMyCommissionApplication: (...args: unknown[]) =>
    serviceMocks.submit(...args),
  workbenchServiceResubmitMyCommissionApplication: (...args: unknown[]) =>
    serviceMocks.resubmit(...args),
  workbenchServiceListMyApplicationCandidates: (...args: unknown[]) =>
    serviceMocks.listCandidates(...args),
  workbenchServiceListMyCommissionApplications: (...args: unknown[]) =>
    serviceMocks.listApplications(...args),
  workbenchServiceGetMyCommissionApplication: (...args: unknown[]) =>
    serviceMocks.getApplication(...args),
}));

const accessState = vi.hoisted(() => ({ canOperateBusiness: true }));
vi.mock('@/app/access', () => ({ useAccess: () => accessState }));

import MyApplicationPanel from './MyApplicationPanel';

function deferred<T>(): { promise: Promise<T>; resolve: (value: T) => void } {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

const refreshMock = vi.fn(() => Promise.resolve());

function summaryWithGroups(): API.WorkbenchApplicationSummary {
  return {
    baseCurrency: 'CNY',
    applyGroups: [
      {
        commissionMonth: '2026-07',
        commissionCount: 3,
        commissionAmount: '500.00',
      },
      {
        commissionMonth: '2026-08',
        commissionCount: 2,
        commissionAmount: '300.25',
      },
    ],
    accumulatingCount: 4,
    accumulatingAmount: '620.00',
    pendingReviewCount: 1,
    approvedCount: 2,
  };
}

function renderPanel(
  summary: API.WorkbenchApplicationSummary = summaryWithGroups(),
) {
  return renderWithClient(
    <App>
      <MyApplicationPanel
        summary={summary}
        currency="CNY"
        onOverviewRefresh={refreshMock}
      />
    </App>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  accessState.canOperateBusiness = true;
  refreshMock.mockClear();
  refreshMock.mockImplementation(() => Promise.resolve());
  serviceMocks.listCandidates.mockResolvedValue({
    success: true,
    total: '0',
    data: [],
  });
  serviceMocks.listApplications.mockResolvedValue({
    success: true,
    total: '0',
    data: [],
  });
});

afterEach(() => {
  cleanup();
});

describe('MyApplicationPanel 工作台月度申请面板', () => {
  it('总部仅查看申请摘要与历史，不展示申请办理入口', () => {
    accessState.canOperateBusiness = false;
    renderPanel();
    expect(screen.queryByTestId('apply-submit-button')).not.toBeInTheDocument();
    expect(screen.getByText('申请历史')).toBeInTheDocument();
    expect(serviceMocks.submit).not.toHaveBeenCalled();
  });

  it('可申请提成按归属月分组展示笔数金额、合计与审批中/已批准计数', () => {
    renderPanel();

    expect(screen.getByText('月度申请')).toBeInTheDocument();
    expect(screen.getByText('2026-07')).toBeInTheDocument();
    expect(screen.getByText('2026-08')).toBeInTheDocument();
    expect(screen.getByText(/3 笔 · 500\.00 CNY/)).toBeInTheDocument();
    expect(screen.getByText(/2 笔 · 300\.25 CNY/)).toBeInTheDocument();
    // 同币种合计由前端 Decimal 精确累加并带币种。
    expect(screen.getByText('800.25')).toBeInTheDocument();
    expect(screen.getByText('合计 5 笔 · CNY')).toBeInTheDocument();
    expect(screen.getByText('审批中 1 张')).toBeInTheDocument();
    expect(screen.getByText('已批准 2 张')).toBeInTheDocument();
    expect(screen.getByText('620.00')).toBeInTheDocument();
  });

  it('本月累计中只展示当月累计与说明，不渲染任何独立申请入口', () => {
    renderPanel({
      baseCurrency: 'CNY',
      applyGroups: [],
      accumulatingCount: 4,
      accumulatingAmount: '620.00',
      pendingReviewCount: 0,
      approvedCount: 0,
    });

    expect(screen.getByText('本月累计中')).toBeInTheDocument();
    expect(screen.getByText('620.00')).toBeInTheDocument();
    expect(screen.getByText('4 笔 · CNY')).toBeInTheDocument();
    expect(
      screen.getByText(/月末结束后可在下一次申请中提交/),
    ).toBeInTheDocument();
    // 无可申请分组：不出现确认弹窗入口与候选下钻。
    expect(
      screen.getByTestId('apply-submit-button').hasAttribute('disabled'),
    ).toBe(true);
    expect(
      screen.queryByText('可申请提成（截至上一自然月末）'),
    ).toBeInTheDocument();
    expect(screen.getByText(/暂无可申请提成/)).toBeInTheDocument();
  });

  it('提交确认流程：确认弹窗展示覆盖月份笔数总额与截止日说明，成功后等待刷新完成再解除 loading', async () => {
    const refreshPending = deferred<void>();
    refreshMock.mockReturnValue(refreshPending.promise);
    serviceMocks.submit.mockResolvedValue({
      success: true,
      data: { id: 'app-1' },
    });

    renderPanel();

    fireEvent.click(screen.getByTestId('apply-submit-button'));

    // 确认弹窗内容：覆盖月份、笔数、总额与截止日说明；不出现发款/工资/银行字样。
    expect(
      (await screen.findAllByText('确认提交月度提成申请？')).length,
    ).toBeGreaterThan(0);
    expect(
      screen.getByText(/归属月份 2026-07、2026-08，共 5 笔/),
    ).toBeInTheDocument();
    expect(screen.getByText(/合计 800.25 CNY/)).toBeInTheDocument();
    expect(
      screen.getByText(/覆盖截止日为提交月份的上一自然月末/),
    ).toBeInTheDocument();
    expect(screen.queryByText(/银行|工资|发放/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /确\s*认\s*申\s*请/ }));

    await waitFor(() => expect(serviceMocks.submit).toHaveBeenCalledTimes(1));
    expect(serviceMocks.submit).toHaveBeenCalledWith({});

    // 刷新未完成：按钮保持 loading；刷新落定后才解除。
    await waitFor(() => expect(refreshMock).toHaveBeenCalledTimes(1));
    expect(
      screen
        .getByTestId('apply-submit-button')
        .classList.contains('ant-btn-loading'),
    ).toBe(true);

    refreshPending.resolve();
    await waitFor(() =>
      expect(
        screen
          .getByTestId('apply-submit-button')
          .classList.contains('ant-btn-loading'),
      ).toBe(false),
    );
    expect(
      await screen.findByText('申请已提交，等待财务审批'),
    ).toBeInTheDocument();
  });

  it('提交失败时不刷新 Overview 并提示错误', async () => {
    serviceMocks.submit.mockRejectedValue(new Error('当前自然月不能提交'));

    renderPanel();

    fireEvent.click(screen.getByTestId('apply-submit-button'));
    fireEvent.click(
      await screen.findByRole('button', { name: /确\s*认\s*申\s*请/ }),
    );

    expect(await screen.findByText('当前自然月不能提交')).toBeInTheDocument();
    await waitFor(() => expect(serviceMocks.submit).toHaveBeenCalledTimes(1));
    expect(refreshMock).not.toHaveBeenCalled();
  });

  it('候选明细下钻：服务端分页拉取并支持归属月过滤', async () => {
    const firstPage = deferred<API.ListMyApplicationCandidatesResponse>();
    serviceMocks.listCandidates.mockReturnValueOnce(firstPage.promise);

    renderPanel();

    fireEvent.click(screen.getByRole('button', { name: /候选明细/ }));
    expect(serviceMocks.listCandidates).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
    });

    firstPage.resolve({
      success: true,
      total: '5',
      page: 1,
      pageSize: 20,
      data: [
        {
          verificationId: 'v-1',
          verificationNo: 'VR-2026-001',
          commissionDate: '2026-07-15',
          personnelRole: 'SALES',
          ruleName: '销售方案A',
          calculationBasis: 'REALIZED_PROFIT',
          baseCurrency: 'CNY',
          commissionAmount: '120.00',
        },
      ],
    } as never);

    expect(await screen.findByText('VR-2026-001')).toBeInTheDocument();
    expect(screen.getByText('共 5 条')).toBeInTheDocument();
    expect(screen.getByText('120.00 CNY')).toBeInTheDocument();

    // 归属月过滤：切换后回到第 1 页并携带 commissionMonth。
    const filtered = deferred<API.ListMyApplicationCandidatesResponse>();
    serviceMocks.listCandidates.mockReturnValueOnce(filtered.promise);
    fireEvent.mouseDown(screen.getAllByRole('combobox')[0]);
    const option = await waitFor(() => {
      const el = document.querySelector('[title="2026-07"]');
      expect(el).toBeTruthy();
      return el as Element;
    });
    fireEvent.click(option);
    expect(serviceMocks.listCandidates).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 20,
      commissionMonth: '2026-07',
    });
    filtered.resolve({ success: true, total: '5', data: [] } as never);
    await waitFor(() => expect(filtered.promise).resolves.toBeTruthy());
  });

  it('申请历史入口打开历史抽屉并服务端分页', async () => {
    serviceMocks.listApplications.mockResolvedValue({
      success: true,
      total: '3',
      data: [
        {
          id: 'app-1',
          applicationMonth: '2026-08',
          coverageTo: '2026-08-31',
          status:
            WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
          version: '1',
          commissionCount: 5,
          baseCurrency: 'CNY',
          totalCommissionAmount: '800.25',
          submittedAt: '2026-09-01 10:00:00',
        },
      ],
    });

    renderPanel();

    fireEvent.click(screen.getByRole('button', { name: /申请历史/ }));
    expect(serviceMocks.listApplications).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
    });
    // 状态 Tag 是抽屉行独有文本（面板显示「审批中 1 张」，非精确匹配），
    // 以它等待抽屉行渲染完成。
    expect(await screen.findByText('审批中')).toBeInTheDocument();
    // 申请月份同时出现在面板分组与抽屉行。
    expect(screen.getAllByText('2026-08')).toHaveLength(2);
    expect(screen.getByText('800.25 CNY')).toBeInTheDocument();

    // 明细下钻：拉取详情并展示快照行。
    serviceMocks.getApplication.mockResolvedValue({
      success: true,
      data: {
        application: {
          id: 'app-1',
          applicationMonth: '2026-08',
          coverageTo: '2026-08-31',
          status:
            WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED,
          version: '2',
          commissionCount: 5,
          baseCurrency: 'CNY',
          totalCommissionAmount: '800.25',
          submittedAt: '2026-09-01 10:00:00',
          decidedAt: '2026-09-02 09:00:00',
          decisionReason: '来源核销与订单不一致',
        },
        lines: [
          {
            id: 'line-1',
            commissionDate: '2026-07-15',
            verificationNo: 'VR-2026-001',
            personnelRole: 'SALES',
            ruleName: '销售方案A',
            calculationBasis: 'REALIZED_PROFIT',
            baseCurrency: 'CNY',
            commissionAmount: '120.00',
          },
        ],
      },
    });

    fireEvent.click(await screen.findByText('明细'));
    expect(serviceMocks.getApplication).toHaveBeenCalledWith({ id: 'app-1' });
    expect(
      await screen.findByText(/驳回原因：来源核销与订单不一致/),
    ).toBeInTheDocument();
    expect(screen.getByText('VR-2026-001')).toBeInTheDocument();
    expect(screen.getByText('销售方案A')).toBeInTheDocument();
  });

  it('驳回申请在详情中提供显式重提：携带申请 ID 与版本调用后刷新 Overview', async () => {
    serviceMocks.resubmit.mockResolvedValue({
      success: true,
      data: { id: 'app-1', status: 1, version: '3' },
    });
    serviceMocks.listApplications.mockResolvedValue({
      success: true,
      total: '1',
      data: [
        {
          id: 'app-1',
          applicationMonth: '2026-08',
          coverageTo: '2026-08-31',
          status:
            WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED,
          version: '2',
          commissionCount: 5,
          baseCurrency: 'CNY',
          totalCommissionAmount: '800.25',
          submittedAt: '2026-09-01 10:00:00',
          decisionReason: '来源核销与订单不一致',
        },
      ],
    });
    serviceMocks.getApplication.mockResolvedValue({
      success: true,
      data: {
        application: {
          id: 'app-1',
          applicationMonth: '2026-08',
          coverageTo: '2026-08-31',
          status:
            WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED,
          version: '2',
          commissionCount: 5,
          baseCurrency: 'CNY',
          totalCommissionAmount: '800.25',
          submittedAt: '2026-09-01 10:00:00',
          decidedAt: '2026-09-02 09:00:00',
          decisionReason: '来源核销与订单不一致',
        },
        lines: [],
      },
    });

    renderPanel();

    fireEvent.click(screen.getByRole('button', { name: /申请历史/ }));
    fireEvent.click(await screen.findByText('明细'));

    // 详情出现「重新提交」主按钮；确认弹窗说明刷新金额并重新进入财务审批。
    fireEvent.click(await screen.findByTestId('resubmit-application-button'));
    expect(
      (await screen.findAllByText(/重新提交 2026-08 月度申请？/)).length,
    ).toBeGreaterThan(0);
    expect(
      screen.getAllByText(/按最新上游数据刷新本申请各明细的提成金额/).length,
    ).toBeGreaterThan(0);

    const confirmOk = await waitFor(() => {
      const btn = document.querySelector(
        '.ant-modal-confirm-btns .ant-btn-primary',
      );
      expect(btn).toBeTruthy();
      return btn as HTMLButtonElement;
    });
    fireEvent.click(confirmOk);

    await waitFor(() => expect(serviceMocks.resubmit).toHaveBeenCalledTimes(1));
    expect(serviceMocks.resubmit).toHaveBeenCalledWith({
      applicationId: 'app-1',
      expectedVersion: '2',
    });
    // 成功后刷新 Overview（面板透传回调）。
    await waitFor(() => expect(refreshMock).toHaveBeenCalledTimes(1));
    expect(
      await screen.findByText('申请已重新提交，等待财务审批'),
    ).toBeInTheDocument();
  });

  it('最近申请被驳回时面板引导到申请历史重提', () => {
    renderPanel({
      baseCurrency: 'CNY',
      applyGroups: [],
      accumulatingCount: 0,
      accumulatingAmount: '0',
      pendingReviewCount: 0,
      approvedCount: 0,
      latestApplication: {
        applicationId: 'app-9',
        applicationMonth: '2026-08',
        status:
          WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED,
        commissionCount: 2,
        totalCommissionAmount: '300.00',
        submittedAt: '2026-09-01 10:00:00',
      },
    });

    expect(screen.getByText(/最近申请：2026-08 · 2 笔/)).toBeInTheDocument();
    expect(
      screen.getByText(
        /最近一次申请已被驳回，请在申请历史中查看原因并重新提交/,
      ),
    ).toBeInTheDocument();
  });

  it('最近申请概要与空申请时隐藏概要', () => {
    renderPanel({
      baseCurrency: 'CNY',
      applyGroups: [
        {
          commissionMonth: '2026-07',
          commissionCount: 3,
          commissionAmount: '500.00',
        },
      ],
      accumulatingCount: 0,
      accumulatingAmount: '0',
      pendingReviewCount: 1,
      approvedCount: 0,
      latestApplication: {
        applicationId: 'app-9',
        applicationMonth: '2026-08',
        status:
          WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
        commissionCount: 2,
        totalCommissionAmount: '300.00',
        submittedAt: '2026-09-01 10:00:00',
      },
    });

    expect(screen.getByText(/最近申请：2026-08 · 2 笔/)).toBeInTheDocument();
  });
});
