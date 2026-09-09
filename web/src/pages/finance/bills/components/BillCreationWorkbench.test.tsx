import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React, { useEffect, useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

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
vi.mock('@umijs/max', () => ({
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

import BillCreationWorkbench from './BillCreationWorkbench';

describe('BillCreationWorkbench 建账候选组织范围', () => {
  beforeEach(() => {
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
      ],
    });
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
    render(
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
          fees: [],
        },
      ],
    });
    render(
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
      expect.objectContaining({ organizationId: 'A' }),
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
    fireEvent.click(screen.getByRole('button', { name: /原子生成 1 张账单/ }));
    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({
          organizationId: 'A',
          groups: [
            expect.objectContaining({ settlementAccountId: 'account-default' }),
          ],
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
    const view = render(
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
      <App>
        <BillCreationWorkbench
          open
          initialFeeIds={['fee-B']}
          initialOrganizationId="B"
          onClose={vi.fn()}
        />
      </App>,
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

  it('两个叶子预览重排后，仍按 groupKey 保留各自默认账户并提交', async () => {
    const groupA = {
      groupKey: 'group-a',
      settlementPartyId: 'partner-a',
      settlementPartyName: '结算单位 A',
      direction: 'RECEIVABLE',
      currency: 'CNY',
      fees: [],
    };
    const groupB = {
      groupKey: 'group-b',
      settlementPartyId: 'partner-b',
      settlementPartyName: '结算单位 B',
      direction: 'PAYABLE',
      currency: 'USD',
      fees: [],
    };
    mocks.preview
      .mockResolvedValueOnce({
        previewToken: 'token-first',
        data: [groupA, groupB],
      })
      .mockResolvedValueOnce({
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
    render(
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
    expect(
      await screen.findByText('账户 B｜测试银行｜USD'),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /刷新快照/ }));
    expect(await screen.findByText('结算单位 B（重排）')).toBeInTheDocument();
    expect(screen.getByText('账户 A｜测试银行｜CNY')).toBeInTheDocument();
    expect(screen.getByText('账户 B｜测试银行｜USD')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /原子生成 2 张账单/ }));
    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({
          previewToken: 'token-second',
          groups: expect.arrayContaining([
            expect.objectContaining({
              groupKey: 'group-a',
              settlementAccountId: 'account-a',
            }),
            expect.objectContaining({
              groupKey: 'group-b',
              settlementAccountId: 'account-b',
            }),
          ]),
        }),
        expect.anything(),
      ),
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
      .mockResolvedValueOnce({
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
    render(
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

    fireEvent.click(screen.getByRole('button', { name: /原子生成 1 张账单/ }));
    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({
          groups: [
            expect.objectContaining({
              groupKey: 'remaining',
              settlementAccountId: 'account-remaining',
            }),
          ],
        }),
        expect.anything(),
      ),
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
    render(
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
      expect(screen.getByText('请选择结算账户')).toBeInTheDocument(),
    );
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
