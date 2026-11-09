import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it } from 'vitest';
import { BillGroupingMode } from '@/enums.generated';
import {
  BillCreationWorkbench,
  mocks,
  pickBillDate,
  resetWorkbenchMocks,
  singlePreviewGroup,
  sleepInAct,
} from './BillCreationWorkbench.fixtures';

describe('BillCreationWorkbench 建账候选组织范围', () => {
  beforeEach(() => {
    resetWorkbenchMocks();
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

  it('预览种子的预计开票汇率预填时收敛服务端尾零（1.00000000 → 1）', async () => {
    mocks.preview.mockResolvedValue({
      previewToken: 'token',
      data: [
        {
          ...singlePreviewGroup,
          estimatedInvoiceRate: '1.00000000',
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

    // 输入框预填值为去尾零后的变长小数，而非服务端 decimal 原串
    expect(await screen.findByDisplayValue('1')).toBeInTheDocument();
    // 等默认账户回填的防抖预览在用例内触发并收敛，避免溢出到后续用例
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(2));
    await sleepInAct(700);
    expect(mocks.preview).toHaveBeenCalledTimes(2);
    expect(screen.queryByDisplayValue('1.00000000')).not.toBeInTheDocument();
  });
});
