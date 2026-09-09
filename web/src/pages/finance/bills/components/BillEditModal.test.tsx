import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Form } from 'antd';
import React, { useEffect } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { BillFormValues } from './billConstants';

const mocks = vi.hoisted(() => ({ accounts: vi.fn() }));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListBillSettlementAccountUpdateCandidates: mocks.accounts,
}));

import BillEditModal from './BillEditModal';

function EditHarness({ editing }: { editing?: API.FinanceBill }) {
  const [form] = Form.useForm<BillFormValues>();
  const settlementAccountId = Form.useWatch('settlementAccountId', form);
  useEffect(() => {
    form.setFieldsValue({
      statementTitle: editing?.statementTitle || '',
      settlementAccountId: editing?.settlementAccountId || '',
    });
  }, [editing, form]);
  return (
    <>
      <BillEditModal
        open
        editing={editing}
        form={form}
        submitting={false}
        onCancel={vi.fn()}
        onOk={async () => {}}
      />
      <output data-testid="settlement-account-id">{settlementAccountId}</output>
    </>
  );
}

describe('BillEditModal 结算账户候选', () => {
  beforeEach(() => {
    mocks.accounts.mockReset();
  });

  it('按草稿账单 ID 加载 update 候选，切换账单后不消费旧响应', async () => {
    let resolveA: ((value: unknown) => void) | undefined;
    mocks.accounts.mockImplementation(({ billId }) =>
      billId === 'bill-a'
        ? new Promise((resolve) => {
            resolveA = resolve;
          })
        : Promise.resolve({
            data: [
              {
                id: 'account-b',
                name: '账户 B',
                bankName: '银行 B',
                currency: 'CNY',
              },
            ],
          }),
    );
    const view = render(
      <EditHarness
        editing={{
          id: 'bill-a',
          billNo: 'BILL-A',
          settlementAccountId: 'account-a',
        }}
      />,
    );
    await waitFor(() =>
      expect(mocks.accounts).toHaveBeenCalledWith({ billId: 'bill-a' }),
    );
    view.rerender(
      <EditHarness
        editing={{
          id: 'bill-b',
          billNo: 'BILL-B',
          settlementAccountId: 'account-b',
        }}
      />,
    );
    await waitFor(() =>
      expect(mocks.accounts).toHaveBeenCalledWith({ billId: 'bill-b' }),
    );
    resolveA?.({
      data: [
        {
          id: 'account-a',
          name: '账户 A',
          bankName: '银行 A',
          currency: 'CNY',
        },
      ],
    });
    await waitFor(() =>
      expect(screen.getByText('编辑账单 BILL-B')).toBeInTheDocument(),
    );
    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(
      (await screen.findAllByText('账户 B｜银行 B｜CNY')).length,
    ).toBeGreaterThan(1);
    expect(screen.queryByText('账户 A｜银行 A｜CNY')).not.toBeInTheDocument();
  });

  it('当前账户不再属于最新候选时，清空并选用最新默认账户', async () => {
    mocks.accounts.mockResolvedValue({
      data: [
        {
          id: 'replacement-account',
          name: '替代账户',
          bankName: '替代银行',
          currency: 'CNY',
          isDefault: true,
        },
      ],
    });
    render(
      <EditHarness
        editing={{
          id: 'bill-a',
          billNo: 'BILL-A',
          settlementAccountId: 'removed-account',
        }}
      />,
    );
    await waitFor(() =>
      expect(mocks.accounts).toHaveBeenCalledWith({ billId: 'bill-a' }),
    );
    await waitFor(() =>
      expect(screen.getByTestId('settlement-account-id')).toHaveTextContent(
        'replacement-account',
      ),
    );
  });
});
