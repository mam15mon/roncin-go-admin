import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FinanceCommissionApplicationStatus } from '@/enums.generated';
import {
  settlementServiceApproveCommissionApplication,
  settlementServiceGetCommissionApplication,
  settlementServiceListCommissionApplications,
  settlementServiceListCommissionEmployees,
  settlementServiceRejectCommissionApplication,
} from '@/services/roncin/settlementService';
import CommissionApplicationsPanel from './CommissionApplicationsPanel';

const accessState = vi.hoisted(() => ({
  canOperateOrganization: () => true,
  canConfigureFinanceCommissions: false,
  canManageFinanceCommissions: true,
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState,
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListCommissionApplications: vi.fn(),
  settlementServiceGetCommissionApplication: vi.fn(),
  settlementServiceApproveCommissionApplication: vi.fn(),
  settlementServiceRejectCommissionApplication: vi.fn(),
  settlementServiceListCommissionEmployees: vi.fn(),
}));

const listApplications = vi.mocked(settlementServiceListCommissionApplications);
const getApplication = vi.mocked(settlementServiceGetCommissionApplication);
const approveApplication = vi.mocked(
  settlementServiceApproveCommissionApplication,
);
const rejectApplication = vi.mocked(
  settlementServiceRejectCommissionApplication,
);
const listEmployees = vi.mocked(settlementServiceListCommissionEmployees);

function applicationRow(
  overrides: Partial<API.FinanceCommissionApplication> = {},
): API.FinanceCommissionApplication {
  return {
    id: 'app-1',
    applicationMonth: '2026-08',
    coverageTo: '2026-08-31',
    status:
      FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
    version: '3',
    commissionCount: 12,
    baseCurrency: 'CNY',
    totalCommissionAmount: '3580.50',
    totalCnyCommissionAmount: '3580.50',
    employeeId: 'emp-1',
    employeeName: '张三',
    submittedAt: '2026-09-01T10:00:00+08:00',
    ...overrides,
  };
}

function renderPanel() {
  return render(
    <App>
      <CommissionApplicationsPanel />
    </App>,
  );
}

beforeEach(() => {
  accessState.canManageFinanceCommissions = true;
  vi.clearAllMocks();
  listApplications.mockResolvedValue({
    data: [],
    total: '0',
    success: true,
  } as Awaited<ReturnType<typeof listApplications>>);
  listEmployees.mockResolvedValue({
    data: [
      { id: 'emp-1', displayName: '张三' },
      { id: 'emp-2', displayName: '李四' },
    ],
    success: true,
  } as Awaited<ReturnType<typeof listEmployees>>);
});

describe('CommissionApplicationsPanel 月度申请批次面板', () => {
  it('默认聚焦待审批队列并加载员工过滤选项', async () => {
    renderPanel();

    await waitFor(() => expect(listApplications).toHaveBeenCalledTimes(1));
    expect(listApplications).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status:
        FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
      applicationMonth: undefined,
      employeeId: undefined,
    });
    await waitFor(() => expect(listEmployees).toHaveBeenCalledTimes(1));
    expect(listEmployees).toHaveBeenCalledWith({ page: 1, pageSize: 200 });
  });

  it('真实 MonthPicker 选择提交月份后按 YYYY-MM 字符串发起查询', async () => {
    renderPanel();
    await waitFor(() => expect(listApplications).toHaveBeenCalledTimes(1));

    // 打开真实月份选择面板并选中 2026-08（antd 6 的 rc-picker 只在 click 时打开）。
    const monthInput = screen.getByPlaceholderText('选择提交月份');
    fireEvent.click(monthInput);
    const monthCell = await waitFor(() => {
      const cell = document.querySelector(
        '.ant-picker-month-panel .ant-picker-cell[title="2026-08"]',
      );
      expect(cell).toBeTruthy();
      return cell as HTMLElement;
    });
    fireEvent.mouseDown(monthCell);
    fireEvent.click(monthCell);

    // 确认面板把选中的月份回填到输入框，再点击「查询」提交。
    await waitFor(() =>
      expect((monthInput as HTMLInputElement).value).toContain('2026-08'),
    );
    // htmlType="submit" 的激活行为只在点击按钮本身时触发，必须按角色取按钮而非内层文本。
    fireEvent.click(screen.getByRole('button', { name: /查询/ }));

    await waitFor(() => expect(listApplications).toHaveBeenCalledTimes(2));
    // 服务端 application_month 只接受 YYYY-MM：Dayjs 表单值必须在提交前格式化。
    expect(listApplications).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 20,
      status:
        FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
      applicationMonth: '2026-08',
      employeeId: undefined,
    });

    // 重置：清空已提交条件并回到默认待审批队列。
    fireEvent.click(screen.getByText('重置'));
    await waitFor(() => expect(listApplications).toHaveBeenCalledTimes(3));
    expect(listApplications).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 20,
      status:
        FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
      applicationMonth: undefined,
      employeeId: undefined,
    });
  });

  it('批次列表展示员工、申请月份、覆盖截止日、笔数、本位币总额与状态', async () => {
    listApplications.mockResolvedValue({
      data: [
        applicationRow(),
        applicationRow({
          id: 'app-2',
          employeeName: '李四',
          status:
            FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_APPROVED,
        }),
      ],
      total: '2',
      success: true,
    } as Awaited<ReturnType<typeof listApplications>>);
    renderPanel();

    expect(await screen.findByText('张三')).toBeInTheDocument();
    expect(screen.getByText('李四')).toBeInTheDocument();
    // 两行同月同额：申请月份、覆盖截止日与金额各出现两次。
    expect(screen.getAllByText('2026-08')).toHaveLength(2);
    expect(screen.getAllByText('2026-08-31')).toHaveLength(2);
    expect(screen.getAllByText('3580.5 CNY')).toHaveLength(2);
    expect(screen.getByText('待审批')).toBeInTheDocument();
    expect(screen.getByText('已批准')).toBeInTheDocument();
  });

  it('整单批准携带 expectedVersion 且确认内容只承诺确认计算结果', async () => {
    approveApplication.mockResolvedValue({
      success: true,
    } as Awaited<ReturnType<typeof approveApplication>>);
    listApplications.mockResolvedValue({
      data: [applicationRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listApplications>>);
    renderPanel();

    fireEvent.click(await screen.findByText('整单批准'));
    expect(screen.getByText(/整单共 12 笔/)).toBeInTheDocument();
    expect(screen.getByText(/不支持部分批准/)).toBeInTheDocument();
    fireEvent.click(
      await screen.findByRole('button', { name: /整\s*单\s*批\s*准/ }),
    );

    await waitFor(() => expect(approveApplication).toHaveBeenCalledTimes(1));
    expect(approveApplication.mock.calls[0][0]).toEqual({ id: 'app-1' });
    expect(approveApplication.mock.calls[0][1]).toEqual({
      id: 'app-1',
      expectedVersion: '3',
    });
    await waitFor(() => expect(listApplications).toHaveBeenCalledTimes(2));
  });

  it('版本冲突时提示「已被处理，请刷新」并刷新列表', async () => {
    approveApplication.mockRejectedValue(
      Object.assign(new Error('已被更新，请刷新后重试'), {
        data: { code: 409, message: '已被更新，请刷新后重试' },
      }),
    );
    listApplications.mockResolvedValue({
      data: [applicationRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listApplications>>);
    renderPanel();

    fireEvent.click(await screen.findByText('整单批准'));
    fireEvent.click(
      await screen.findByRole('button', { name: /整\s*单\s*批\s*准/ }),
    );

    expect(
      await screen.findByText('该申请已被处理，请刷新后查看最新状态'),
    ).toBeInTheDocument();
    await waitFor(() => expect(listApplications).toHaveBeenCalledTimes(2));
  });

  it('整单驳回原因必填：空原因不调用接口', async () => {
    rejectApplication.mockResolvedValue({
      success: true,
    } as Awaited<ReturnType<typeof rejectApplication>>);
    listApplications.mockResolvedValue({
      data: [applicationRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listApplications>>);
    renderPanel();

    fireEvent.click(await screen.findByText('整单驳回'));
    const reasonInput = await screen.findByPlaceholderText(
      '请输入驳回原因（必填）',
    );
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));
    expect(await screen.findByText('请输入驳回原因')).toBeInTheDocument();
    expect(rejectApplication).not.toHaveBeenCalled();

    fireEvent.change(reasonInput, {
      target: { value: '来源核销与订单不一致' },
    });
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));

    await waitFor(() => expect(rejectApplication).toHaveBeenCalledTimes(1));
    expect(rejectApplication.mock.calls[0][1]).toEqual({
      id: 'app-1',
      expectedVersion: '3',
      reason: '来源核销与订单不一致',
    });
    await waitFor(() => expect(listApplications).toHaveBeenCalledTimes(2));
  });

  it('无 commission.manage 权限时隐藏整单批准/驳回动作', async () => {
    accessState.canManageFinanceCommissions = false;
    listApplications.mockResolvedValue({
      data: [applicationRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listApplications>>);
    renderPanel();

    await waitFor(() => expect(listApplications).toHaveBeenCalled());
    expect(screen.getByText('明细')).toBeInTheDocument();
    expect(screen.queryByText('整单批准')).not.toBeInTheDocument();
    expect(screen.queryByText('整单驳回')).not.toBeInTheDocument();
  });

  it('明细下钻展示来源核销/对冲、人员身份、方案快照与金额', async () => {
    listApplications.mockResolvedValue({
      data: [applicationRow()],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listApplications>>);
    getApplication.mockResolvedValue({
      success: true,
      data: {
        application: applicationRow(),
        lines: [
          {
            id: 'line-1',
            commissionId: 'c-1',
            commissionNo: 'FC2026080001',
            commissionDate: '2026-07-15',
            verificationNo: 'VR-2026-001',
            personnelRole: 'SALES',
            ruleName: '销售方案A',
            ruleVersion: '2',
            calculationBasis: 'REALIZED_PROFIT',
            baseCurrency: 'CNY',
            commissionAmount: '120.00',
            cnyCommissionAmount: '120.00',
          },
          {
            id: 'line-2',
            commissionId: 'c-2',
            commissionNo: 'FC2026080002',
            commissionDate: '2026-08-02',
            nettingNo: 'NT-2026-009',
            personnelRole: 'OPERATOR',
            ruleName: '操作方案B',
            ruleVersion: '1',
            calculationBasis: 'REALIZED_REVENUE',
            baseCurrency: 'USD',
            commissionAmount: '80.00',
            cnyCommissionAmount: '570.00',
          },
        ],
      },
    } as Awaited<ReturnType<typeof getApplication>>);
    renderPanel();

    fireEvent.click(await screen.findByText('明细'));
    expect(getApplication).toHaveBeenCalledWith({ id: 'app-1' });

    expect(await screen.findByText('FC2026080001')).toBeInTheDocument();
    expect(screen.getByText('VR-2026-001')).toBeInTheDocument();
    expect(screen.getByText('NT-2026-009')).toBeInTheDocument();
    expect(screen.getByText('业务人员')).toBeInTheDocument();
    expect(screen.getByText('操作人员')).toBeInTheDocument();
    expect(screen.getByText('销售方案A')).toBeInTheDocument();
    expect(screen.getByText('v2')).toBeInTheDocument();
    // 原币与 CNY 金额带币种并列展示，不裸相加；首行原币与 CNY 同值。
    expect(screen.getAllByText('120 CNY')).toHaveLength(2);
    expect(screen.getByText('80 USD')).toBeInTheDocument();
    expect(screen.getByText('570 CNY')).toBeInTheDocument();
    // 页面不提供任何部分批准、剔除明细或拆分入口。
    expect(screen.queryByText(/部分批准/)).not.toBeInTheDocument();
  });

  it('待审批之外的批次在详情抽屉中不渲染决策按钮', async () => {
    listApplications.mockResolvedValue({
      data: [
        applicationRow({
          status:
            FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_APPROVED,
        }),
      ],
      total: '1',
      success: true,
    } as Awaited<ReturnType<typeof listApplications>>);
    getApplication.mockResolvedValue({
      success: true,
      data: {
        application: applicationRow({
          status:
            FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_APPROVED,
        }),
        lines: [],
      },
    } as Awaited<ReturnType<typeof getApplication>>);
    renderPanel();

    fireEvent.click(await screen.findByText('明细'));
    expect(getApplication).toHaveBeenCalledWith({ id: 'app-1' });
    // 列表行与详情头都展示已批准：允许重复出现。
    expect((await screen.findAllByText('已批准')).length).toBeGreaterThan(0);
    expect(screen.queryByText('整单批准')).not.toBeInTheDocument();
    expect(screen.queryByText('整单驳回')).not.toBeInTheDocument();
  });
});
