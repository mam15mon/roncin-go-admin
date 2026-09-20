import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React, { useEffect, useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { BillGroupingMode } from '@/enums.generated';

const mocks = vi.hoisted(() => ({
  organizations: vi.fn(),
  candidates: vi.fn(),
  preview: vi.fn(),
  create: vi.fn(),
  accounts: vi.fn(),
}));
vi.mock('@ant-design/pro-components', () => ({
  ProTable: (props: any) => {
    const [data, setData] = useState<any[]>([]);
    useEffect(() => {
      if (props.request)
        void props
          .request({ current: 1, pageSize: 20 })
          .then((r: any) => setData(r.data || []));
    }, []);
    return (
      <div>
        {data.map((row) => (
          <button
            key={row.id}
            type="button"
            onClick={() => props.rowSelection?.onChange([row.id], [row])}
          >
            {row.feeName}
          </button>
        ))}
      </div>
    );
  },
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ hasAction: () => true }),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFinanceOrganizationOptions: mocks.organizations,
  settlementServiceListBillCreationCandidates: mocks.candidates,
  settlementServicePreviewBillBatch: mocks.preview,
  settlementServiceCreateBillBatch: mocks.create,
  settlementServiceConfirmBillBatch: vi.fn(),
  settlementServiceListBillSettlementAccountCandidates: mocks.accounts,
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([
    { label: 'CNY - 人民币', value: 'CNY' },
    { label: 'USD - 美元', value: 'USD' },
  ]),
}));

import BillCreationWorkbench from './BillCreationWorkbench';

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/** 防抖窗口断言仍用真实定时器；包进 act 让窗口期落地的预览/账户回调在 act 内更新状态。 */
const sleepInAct = async (ms: number) => {
  await act(async () => {
    await sleep(ms);
  });
};

function resetWorkbenchMocks() {
  mocks.organizations.mockReset();
  mocks.candidates.mockReset();
  mocks.preview.mockReset();
  mocks.create.mockReset();
  mocks.accounts.mockReset();
  mocks.create.mockResolvedValue({ data: { id: 'batch-1', bills: [] } });
  mocks.organizations.mockResolvedValue({
    data: [
      { id: 'A', name: '公司A' },
      { id: 'B', name: '公司B' },
    ],
  });
  mocks.candidates.mockImplementation(({ organizationId }) =>
    Promise.resolve({
      data: [
        {
          id: `fee-${organizationId}`,
          feeName: `费用${organizationId}`,
          status: 2,
        },
      ],
    }),
  );
  mocks.accounts.mockResolvedValue({
    data: [
      {
        id: 'account-default',
        name: '默认结算账户',
        bankName: '测试银行',
        accountNo: '6222',
        currency: 'CNY',
        isDefault: true,
      },
      {
        id: 'account-second',
        name: '第二账户',
        bankName: '测试银行',
        accountNo: '6223',
        currency: 'CNY',
        isDefault: false,
      },
    ],
  });
}

const singlePreviewGroup = {
  groupKey: 'g1',
  settlementPartyId: 'partner-a',
  settlementPartyName: '结算单位甲',
  direction: 'RECEIVABLE',
  currency: 'CNY',
  baseCurrency: 'CNY',
  totalAmount: '100.00',
  baseCurrencyAmount: '100.00',
  estimatedInvoiceCurrency: 'CNY',
  estimatedInvoiceRate: '1',
  estimatedInvoiceAmount: '100.00',
  configurationComplete: true,
  fees: [],
};

function pickBillDate(isoDate: string) {
  const input = screen.getByLabelText('账单日期');
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: isoDate } });
  fireEvent.keyDown(input, { key: 'Enter', code: 'Enter', keyCode: 13 });
}

describe('BillCreationWorkbench 建账候选组织范围', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
  });
  it('未选组织不请求，A 慢响应切 B 后不回填且清空选择', async () => {
    let resolveA: ((value: unknown) => void) | undefined;
    mocks.candidates.mockImplementation(({ organizationId }) =>
      organizationId === 'A'
        ? new Promise((resolve) => {
            resolveA = resolve;
          })
        : Promise.resolve({
            data: [{ id: 'fee-B', feeName: '费用B', status: 2 }],
          }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench open onClose={vi.fn()} />
      </App>,
    );
    await waitFor(() =>
      expect(mocks.organizations).toHaveBeenCalledWith({ purpose: 11 }),
    );
    expect(mocks.candidates).not.toHaveBeenCalled();
    fireEvent.mouseDown(screen.getByLabelText('所属公司'));
    fireEvent.click(await screen.findByText('公司A'));
    await waitFor(() =>
      expect(mocks.candidates).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'A' }),
      ),
    );
    fireEvent.mouseDown(screen.getByLabelText('所属公司'));
    fireEvent.click(await screen.findByText('公司B'));
    await waitFor(() => {
      expect(
        screen.queryByRole('button', { name: '费用A' }),
      ).not.toBeInTheDocument();
      expect(screen.getByRole('button', { name: '费用B' })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: '下一步' }));
    expect(mocks.preview).not.toHaveBeenCalled();
    resolveA?.({ data: [{ id: 'fee-A', feeName: '费用A', status: 2 }] });
    await waitFor(() => {
      expect(
        screen.queryByRole('button', { name: '费用A' }),
      ).not.toBeInTheDocument();
      expect(screen.getByRole('button', { name: '费用B' })).toBeInTheDocument();
    });
  });
  it('预览后的对账抬头默认结算单位名称，且不请求 Partner 服务', async () => {
    mocks.preview.mockResolvedValue({
      previewToken: 'token',
      data: [
        {
          groupKey: 'g',
          settlementPartyId: 'partner-a',
          settlementPartyName: '结算单位甲',
          direction: 'RECEIVABLE',
          currency: 'CNY',
          configurationComplete: true,
          fees: [],
        },
      ],
    });
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(await screen.findByDisplayValue('结算单位甲')).toBeInTheDocument();
    expect(mocks.preview).toHaveBeenCalledWith(
      expect.objectContaining({
        organizationId: 'A',
        groupingPolicy: {
          mode: BillGroupingMode.BILL_GROUPING_MODE_NORMAL,
          splitByOrder: true,
          splitByTaxRate: false,
        },
      }),
      expect.anything(),
    );
    await waitFor(() =>
      expect(mocks.accounts).toHaveBeenCalledWith({
        organizationId: 'A',
        settlementPartyId: 'partner-a',
        direction: 'RECEIVABLE',
        currency: 'CNY',
      }),
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    fireEvent.click(screen.getByRole('button', { name: /原子生成 1 张账单/ }));
    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({
          organizationId: 'A',
          groups: [
            expect.objectContaining({
              settlementAccountId: 'account-default',
            }),
          ],
          previewToken: 'token',
        }),
        expect.anything(),
      ),
    );
  });

  it('固定来源从 A 切到 B 后丢弃迟到的 A 预览', async () => {
    let resolveA: ((value: unknown) => void) | undefined;
    mocks.preview.mockImplementation(({ organizationId }) =>
      organizationId === 'A'
        ? new Promise((resolve) => {
            resolveA = resolve;
          })
        : Promise.resolve({
            previewToken: 'token-b',
            data: [
              {
                groupKey: 'group-b',
                settlementPartyName: '结算单位B',
                fees: [],
              },
            ],
          }),
    );
    const view = renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'A' }),
        expect.anything(),
      ),
    );

    view.rerender(
      <QueryClientProvider client={view.queryClient}>
        <App>
          <BillCreationWorkbench
            open
            initialFeeIds={['fee-B']}
            initialOrganizationId="B"
            onClose={vi.fn()}
          />
        </App>
      </QueryClientProvider>,
    );
    expect(await screen.findByDisplayValue('结算单位B')).toBeInTheDocument();

    resolveA?.({
      previewToken: 'token-a',
      data: [
        {
          groupKey: 'group-a',
          settlementPartyName: '结算单位A',
          fees: [],
        },
      ],
    });
    await waitFor(() => {
      expect(screen.queryByDisplayValue('结算单位A')).not.toBeInTheDocument();
      expect(screen.getByDisplayValue('结算单位B')).toBeInTheDocument();
    });
  });

  it('两个叶子预览重排后，仍按 groupKey 保留各自默认账户', async () => {
    const groupA = {
      groupKey: 'group-a',
      settlementPartyId: 'partner-a',
      settlementPartyName: '结算单位 A',
      direction: 'RECEIVABLE',
      currency: 'CNY',
      configurationComplete: true,
      fees: [],
    };
    const groupB = {
      groupKey: 'group-b',
      settlementPartyId: 'partner-b',
      settlementPartyName: '结算单位 B',
      direction: 'RECEIVABLE',
      currency: 'USD',
      configurationComplete: true,
      fees: [],
    };
    mocks.preview
      .mockResolvedValueOnce({
        previewToken: 'token-first',
        data: [groupA, groupB],
      })
      .mockResolvedValue({
        previewToken: 'token-second',
        data: [
          { ...groupB, settlementPartyName: '结算单位 B（重排）' },
          { ...groupA, settlementPartyName: '结算单位 A（重排）' },
        ],
      });
    mocks.accounts.mockImplementation(({ settlementPartyId }) =>
      Promise.resolve({
        data: [
          {
            id: settlementPartyId === 'partner-a' ? 'account-a' : 'account-b',
            name: settlementPartyId === 'partner-a' ? '账户 A' : '账户 B',
            bankName: '测试银行',
            currency: settlementPartyId === 'partner-a' ? 'CNY' : 'USD',
            isDefault: true,
          },
        ],
      }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(await screen.findByDisplayValue('结算单位 A')).toBeInTheDocument();
    await waitFor(() => expect(mocks.accounts).toHaveBeenCalledTimes(2));
    expect(
      await screen.findByText('账户 A｜测试银行｜CNY'),
    ).toBeInTheDocument();
    fireEvent.click(await screen.findByText(/结算单位 B.*USD/));
    expect(await screen.findByDisplayValue(/结算单位 B/)).toBeInTheDocument();
    expect(
      await screen.findByText('账户 B｜测试银行｜USD'),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /刷新快照/ }));
    expect(
      (await screen.findAllByText(/结算单位 B.*重排/)).length,
    ).toBeGreaterThan(0);
    expect(screen.getByText('账户 B｜测试银行｜USD')).toBeInTheDocument();

    const createBtn = await screen.findByRole('button', {
      name: /原子生成 2 张账单/,
    });
    await waitFor(() => expect(createBtn).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(createBtn);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({
        previewToken: 'token-second',
        groups: expect.arrayContaining([
          expect.objectContaining({
            groupKey: 'group-b',
            statementTitle: '结算单位 B',
            settlementAccountId: 'account-b',
          }),
          expect.objectContaining({
            groupKey: 'group-a',
            statementTitle: '结算单位 A',
            settlementAccountId: 'account-a',
          }),
        ]),
      }),
      expect.anything(),
    );
  });

  it('已移除叶子的迟到候选不会污染仍存在叶子的账户草稿', async () => {
    let resolveRemoved: ((value: unknown) => void) | undefined;
    const remainingGroup = {
      groupKey: 'remaining',
      settlementPartyId: 'partner-b',
      settlementPartyName: '保留单位',
      direction: 'PAYABLE',
      currency: 'USD',
      fees: [],
    };
    mocks.preview
      .mockResolvedValueOnce({
        previewToken: 'token-first',
        data: [
          {
            groupKey: 'removed',
            settlementPartyId: 'partner-a',
            settlementPartyName: '移除单位',
            direction: 'RECEIVABLE',
            currency: 'CNY',
            fees: [],
          },
          remainingGroup,
        ],
      })
      .mockResolvedValue({
        previewToken: 'token-second',
        data: [remainingGroup],
      });
    mocks.accounts.mockImplementation(({ settlementPartyId }) =>
      settlementPartyId === 'partner-a'
        ? new Promise((resolve) => {
            resolveRemoved = resolve;
          })
        : Promise.resolve({
            data: [
              {
                id: 'account-remaining',
                name: '保留账户',
                bankName: '测试银行',
                currency: 'USD',
                isDefault: true,
              },
            ],
          }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(await screen.findByDisplayValue('移除单位')).toBeInTheDocument();
    fireEvent.click(screen.getByText(/保留单位 · - USD/));
    expect(await screen.findByDisplayValue('保留单位')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /刷新快照/ }));
    await waitFor(() =>
      expect(screen.queryByDisplayValue('移除单位')).not.toBeInTheDocument(),
    );
    resolveRemoved?.({
      data: [
        {
          id: 'account-removed',
          name: '旧账户',
          bankName: '旧银行',
          currency: 'CNY',
          isDefault: true,
        },
      ],
    });
    await waitFor(() =>
      expect(screen.queryByText('旧账户｜旧银行｜CNY')).not.toBeInTheDocument(),
    );
  });

  it('任一叶子没有可选结算账户时阻断整批创建', async () => {
    mocks.preview.mockResolvedValue({
      previewToken: 'token',
      data: [
        {
          groupKey: 'without-account',
          settlementPartyId: 'partner-a',
          settlementPartyName: '未配置账户单位',
          direction: 'RECEIVABLE',
          currency: 'CNY',
          fees: [],
        },
      ],
    });
    mocks.accounts.mockResolvedValue({ data: [] });
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(
      await screen.findByDisplayValue('未配置账户单位'),
    ).toBeInTheDocument();
    await waitFor(() => expect(mocks.accounts).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByRole('button', { name: /原子生成 1 张账单/ }));
    await waitFor(() =>
      expect(
        screen.getByText('请先补齐首个标记叶子的日期和结算账户'),
      ).toBeInTheDocument(),
    );
    expect(mocks.create).not.toHaveBeenCalled();
  });
});

describe('BillCreationWorkbench 预览触发边界', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
    mocks.preview.mockResolvedValue({
      previewToken: 'token',
      data: [singlePreviewGroup],
    });
  });

  it('修改对账抬头或备注不触发预览，修改日期、账户、预计开票配置触发预览', async () => {
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    expect(await screen.findByDisplayValue('结算单位甲')).toBeInTheDocument();
    // 初始预览 + 默认账户回填触发的一次防抖预览。
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await sleepInAct(700);
    expect(mocks.preview).toHaveBeenCalledTimes(2);

    // 非计算字段：对账抬头、备注，不应触发预览重请求。
    fireEvent.change(screen.getByDisplayValue('结算单位甲'), {
      target: { value: '新对账抬头' },
    });
    fireEvent.change(screen.getByPlaceholderText('选填'), {
      target: { value: '备注内容' },
    });
    await sleepInAct(700);
    expect(mocks.preview).toHaveBeenCalledTimes(2);

    // 结算账户变化触发预览。
    fireEvent.mouseDown(screen.getByLabelText('结算账户'));
    fireEvent.click(await screen.findByText('第二账户｜测试银行 · 6223｜CNY'));
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(3));

    // 账单日期变化触发预览。
    pickBillDate('2026-09-05');
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(4));

    // 预计开票币种变化触发预览。
    fireEvent.mouseDown(screen.getByLabelText('预计开票币种'));
    fireEvent.click(await screen.findByText('USD - 美元'));
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(5));

    // 预计开票汇率变化触发预览。
    fireEvent.change(
      screen.getByPlaceholderText('服务端按预计口径计算，可调整'),
      { target: { value: '6.5' } },
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(6));
    await sleepInAct(500);
    expect(mocks.preview).toHaveBeenCalledTimes(6);
  });
});

describe('BillCreationWorkbench 预览竞态与规模', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
  });

  it('账单日期快速从 D1 改到 D2，迟到的 D1 预览不得覆盖 D2 结果', async () => {
    const pending: Array<(value: unknown) => void> = [];
    mocks.preview.mockImplementation(
      () =>
        new Promise((resolve) => {
          pending.push(resolve);
        }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(1));
    await act(async () => {
      pending.shift()?.({
        previewToken: 'token-init',
        data: [singlePreviewGroup],
      });
    });
    expect(await screen.findByDisplayValue('结算单位甲')).toBeInTheDocument();
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await act(async () => {
      pending.shift()?.({
        previewToken: 'token-init',
        data: [singlePreviewGroup],
      });
    });
    await sleepInAct(700);
    const baseline = mocks.preview.mock.calls.length;

    // 请求 1：日期改为 D1，保持挂起。
    pickBillDate('2026-09-05');
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledTimes(baseline + 1),
    );
    // 请求 2：立即改为 D2，同样挂起。
    pickBillDate('2026-09-18');
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledTimes(baseline + 2),
    );
    expect(pending).toHaveLength(2);

    // 请求 2（D2）先完成并生效；resolve 的后续状态更新包进 act。
    await act(async () => {
      pending[1]?.({
        previewToken: 'token-d2',
        data: [
          {
            ...singlePreviewGroup,
            baseCurrencyAmount: '200.00',
            estimatedInvoiceAmount: '200.00',
          },
        ],
      });
    });
    expect(await screen.findByText('200.00 CNY')).toBeInTheDocument();

    // 请求 1（D1）后完成，不得覆盖金额、完成状态或创建令牌。
    await act(async () => {
      pending[0]?.({
        previewToken: 'token-d1',
        data: [
          {
            ...singlePreviewGroup,
            baseCurrencyAmount: '999.00',
            estimatedInvoiceAmount: '999.00',
          },
        ],
      });
    });
    await sleepInAct(300);
    expect(screen.queryByText('999.00 CNY')).not.toBeInTheDocument();
    expect(screen.getByText('200.00 CNY')).toBeInTheDocument();

    const createBtn = await screen.findByRole('button', {
      name: /原子生成 1 张账单/,
    });
    await waitFor(() => expect(createBtn).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(createBtn);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({ previewToken: 'token-d2' }),
      expect.anything(),
    );
  });

  it('预计开票币种与汇率的迟到预览同样不覆盖新结果', async () => {
    const pending: Array<(value: unknown) => void> = [];
    mocks.preview.mockImplementation(
      () =>
        new Promise((resolve) => {
          pending.push(resolve);
        }),
    );
    renderWithClient(
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-A']}
          initialOrganizationId="A"
          onClose={vi.fn()}
        />
      </App>,
    );
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(1));
    await act(async () => {
      pending.shift()?.({
        previewToken: 'token-init',
        data: [singlePreviewGroup],
      });
    });
    expect(await screen.findByDisplayValue('结算单位甲')).toBeInTheDocument();
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await act(async () => {
      pending.shift()?.({
        previewToken: 'token-init',
        data: [singlePreviewGroup],
      });
    });
    await sleepInAct(700);
    const baseline = mocks.preview.mock.calls.length;

    // 请求 1：预计开票币种改为 USD，保持挂起。
    fireEvent.mouseDown(screen.getByLabelText('预计开票币种'));
    fireEvent.click(await screen.findByText('USD - 美元'));
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledTimes(baseline + 1),
    );
    // 请求 2：继续填写预计开票汇率，同样挂起。
    fireEvent.change(
      screen.getByPlaceholderText('服务端按预计口径计算，可调整'),
      { target: { value: '6.5' } },
    );
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledTimes(baseline + 2),
    );
    expect(pending).toHaveLength(2);

    // 请求 2 先完成并生效；resolve 的后续状态更新包进 act。
    await act(async () => {
      pending[1]?.({
        previewToken: 'token-rate',
        data: [
          {
            ...singlePreviewGroup,
            baseCurrencyAmount: '300.00',
            estimatedInvoiceAmount: '300.00',
          },
        ],
      });
    });
    expect(await screen.findByText('300.00 CNY')).toBeInTheDocument();

    // 请求 1 后完成，不得覆盖。
    await act(async () => {
      pending[0]?.({
        previewToken: 'token-currency',
        data: [
          {
            ...singlePreviewGroup,
            baseCurrencyAmount: '111.00',
            estimatedInvoiceAmount: '111.00',
          },
        ],
      });
    });
    await sleepInAct(300);
    expect(screen.queryByText('111.00 CNY')).not.toBeInTheDocument();
    expect(screen.getByText('300.00 CNY')).toBeInTheDocument();

    const createBtn = await screen.findByRole('button', {
      name: /原子生成 1 张账单/,
    });
    await waitFor(() => expect(createBtn).not.toHaveClass('ant-btn-loading'));
    fireEvent.click(createBtn);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({ previewToken: 'token-rate' }),
      expect.anything(),
    );
  });
});
