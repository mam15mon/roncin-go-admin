import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  CommissionCreateModal,
  modalState,
  nettingCandidate,
  resetCommissionModalMocks,
  serviceMocks,
  verificationCandidate,
  workspace,
} from './CommissionCreateModal.fixtures';

describe('提成预览 CNY 快照', () => {
  beforeEach(resetCommissionModalMocks);

  it('展示后端返回的 CNY 金额、汇率依据和重新解析提示', async () => {
    renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    if (modalState.dependencyValues.verificationId) {
      await screen.findByText('VR-A｜A单位｜200.00 CNY');
      fireEvent.click(screen.getByRole('button', { name: '选择核销单' }));
    } else {
      fireEvent.click(await screen.findByText('NT-2026-001'));
    }
    fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));

    expect(await screen.findByText('400.00 CNY')).toBeInTheDocument();
    expect(screen.getByText('7.14285714')).toBeInTheDocument();
    expect(screen.getByText('2026-08-31')).toBeInTheDocument();
    expect(screen.getByText('倒数派生')).toBeInTheDocument();
    expect(screen.getByText('预览汇率仅供生成前核对')).toBeInTheDocument();
  });

  it('核销来源创建请求体携带员工与身份而不携带 nettingId 与规则', async () => {
    const onSuccess = vi.fn();
    serviceMocks.createCommission.mockResolvedValue({
      data: { id: 'commission-new', cnyCommissionAmount: '401.00000000' },
    });
    renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={onSuccess}
        />
      </App>,
    );
    if (modalState.dependencyValues.verificationId) {
      await screen.findByText('VR-A｜A单位｜200.00 CNY');
      fireEvent.click(screen.getByRole('button', { name: '选择核销单' }));
    } else {
      fireEvent.click(await screen.findByText('NT-2026-001'));
    }
    fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));
    await screen.findByText('400.00 CNY');

    await act(async () => {
      await modalState.props?.onFinish({
        verificationId: 'verification-1',
        candidateKey: 'employee-1|SALES',
      });
    });

    await waitFor(() => expect(onSuccess).toHaveBeenCalledOnce());
    expect(serviceMocks.createCommission).toHaveBeenCalledWith(
      expect.objectContaining({
        verificationId: 'verification-1',
        employeeId: 'employee-1',
        personnelRole: 'SALES',
        organizationId: 'org-a',
      }),
    );
    expect(serviceMocks.createCommission.mock.calls[0][0]).not.toHaveProperty(
      'nettingId',
    );
    expect(serviceMocks.createCommission.mock.calls[0][0]).not.toHaveProperty(
      'ruleId',
    );
    expect(serviceMocks.createCommission.mock.calls[0][0]).not.toHaveProperty(
      'cnyCommissionAmount',
    );
  });

  it('选择核销来源后从服务端加载「员工 + 身份 + 已解析方案」候选', async () => {
    renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    await screen.findByText('VR-A｜A单位｜200.00 CNY');
    fireEvent.click(screen.getByRole('button', { name: '选择核销单' }));

    await waitFor(() =>
      expect(serviceMocks.listCommissionCandidates).toHaveBeenCalledWith({
        page: 1,
        pageSize: 200,
        organizationId: 'org-a',
        verificationId: 'verification-1',
      }),
    );
    expect(
      await screen.findByText(
        '张三｜业务人员｜销售提成方案（v3）｜预计 60.00 CNY',
      ),
    ).toBeInTheDocument();
    expect(screen.getByTestId('candidate-disabled')).toHaveTextContent('false');
  });

  it('预览请求携带来源、员工、身份与组织且不携带规则', async () => {
    renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );
    if (modalState.dependencyValues.verificationId) {
      await screen.findByText('VR-A｜A单位｜200.00 CNY');
      fireEvent.click(screen.getByRole('button', { name: '选择核销单' }));
    } else {
      fireEvent.click(await screen.findByText('NT-2026-001'));
    }
    fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));

    await waitFor(() =>
      expect(serviceMocks.previewCommission).toHaveBeenCalledWith(
        expect.objectContaining({
          verificationId: 'verification-1',
          employeeId: 'employee-1',
          personnelRole: 'SALES',
          organizationId: 'org-a',
        }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      ),
    );
    expect(serviceMocks.previewCommission.mock.calls[0][0]).not.toHaveProperty(
      'ruleId',
    );
    expect(serviceMocks.previewCommission.mock.calls[0][0]).not.toHaveProperty(
      'nettingId',
    );
  });

  it('对冲提成 Tab 按对冲管理列惯例展示已确认对冲单候选', async () => {
    serviceMocks.listCommissionNettingCandidates.mockResolvedValue({
      data: [nettingCandidate],
    });
    renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

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
      screen.queryByRole('button', { name: '选择核销单' }),
    ).not.toBeInTheDocument();
  });

  it('对冲来源预览与创建请求体携带 nettingId 而不带 verificationId', async () => {
    serviceMocks.listCommissionNettingCandidates.mockResolvedValue({
      data: [nettingCandidate],
    });
    serviceMocks.previewCommission.mockResolvedValue({
      data: {
        employeeName: '张三',
        ruleName: '销售提成方案',
        ruleVersion: '3',
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
    serviceMocks.createCommission.mockResolvedValue({
      data: { id: 'commission-new' },
    });
    renderWithClient(
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
      candidateKey: 'employee-1|SALES',
    };
    fireEvent.click(screen.getByRole('tab', { name: '对冲提成' }));
    if (modalState.dependencyValues.verificationId) {
      await screen.findByText('VR-A｜A单位｜200.00 CNY');
      fireEvent.click(screen.getByRole('button', { name: '选择核销单' }));
    } else {
      fireEvent.click(await screen.findByText('NT-2026-001'));
    }
    fireEvent.click(screen.getByRole('button', { name: '计算并核对预览' }));

    await waitFor(() =>
      expect(serviceMocks.previewCommission).toHaveBeenCalledWith(
        expect.objectContaining({
          nettingId: 'netting-1',
          employeeId: 'employee-1',
          personnelRole: 'SALES',
          organizationId: 'org-a',
        }),
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      ),
    );
    expect(serviceMocks.previewCommission.mock.calls[0][0]).not.toHaveProperty(
      'verificationId',
    );
    expect(await screen.findByText('来源单号')).toBeInTheDocument();
    // 对冲表格与预览来源都会展示对冲单号。
    expect(screen.getAllByText('NT-2026-001').length).toBeGreaterThan(0);

    await act(async () => {
      await modalState.props?.onFinish({
        nettingId: 'netting-1',
        candidateKey: 'employee-1|SALES',
      });
    });

    await waitFor(() =>
      expect(serviceMocks.createCommission).toHaveBeenCalledWith(
        expect.objectContaining({
          nettingId: 'netting-1',
          employeeId: 'employee-1',
          personnelRole: 'SALES',
          organizationId: 'org-a',
        }),
      ),
    );
    expect(serviceMocks.createCommission.mock.calls[0][0]).not.toHaveProperty(
      'verificationId',
    );
  });

  it('核销候选身份未就绪不请求，慢 A 响应不会污染已切换的 B 工作区', async () => {
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
    workspace.organizationId = undefined;
    const { rerender, queryClient } = renderWithClient(
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

    // 工作区身份就绪后自动加载，不再通过表单选择公司。
    workspace.organizationId = 'org-a';
    rerender(
      <QueryClientProvider client={queryClient}>
        <App>
          <CommissionCreateModal
            key="org-a"
            open
            onOpenChange={vi.fn()}
            onSuccess={vi.fn()}
          />
        </App>
      </QueryClientProvider>,
    );
    await waitFor(() =>
      expect(
        serviceMocks.listCommissionVerificationCandidates,
      ).toHaveBeenCalledWith({
        page: 1,
        pageSize: 200,
        organizationId: 'org-a',
      }),
    );

    workspace.organizationId = 'org-b';
    rerender(
      <QueryClientProvider client={queryClient}>
        <App>
          <CommissionCreateModal
            key="org-b"
            open
            onOpenChange={vi.fn()}
            onSuccess={vi.fn()}
          />
        </App>
      </QueryClientProvider>,
    );
    expect(
      await screen.findByText('VR-B｜B单位｜100.00 CNY'),
    ).toBeInTheDocument();

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
    expect(
      screen.queryByText('VR-A｜A单位｜200.00 CNY'),
    ).not.toBeInTheDocument();
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
    renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    expect(
      await screen.findByText('VR-A｜A单位｜200.00 CNY'),
    ).toBeInTheDocument();
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
      await screen.findByText('VR-A-MATCH｜A单位｜200.00 CNY'),
    ).toBeInTheDocument();
  });

  it('对冲候选输入关键字后防抖携带 keyword，切换组织清空关键字并重拉', async () => {
    serviceMocks.listCommissionNettingCandidates.mockResolvedValue({
      data: [nettingCandidate],
    });
    const { rerender, queryClient } = renderWithClient(
      <App>
        <CommissionCreateModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

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

    workspace.organizationId = 'org-b';
    rerender(
      <QueryClientProvider client={queryClient}>
        <App>
          <CommissionCreateModal
            key="org-b"
            open
            onOpenChange={vi.fn()}
            onSuccess={vi.fn()}
          />
        </App>
      </QueryClientProvider>,
    );
    fireEvent.click(screen.getByRole('tab', { name: '对冲提成' }));
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
});
