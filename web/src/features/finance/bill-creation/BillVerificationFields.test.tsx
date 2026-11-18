import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { Button, ConfigProvider, Form, Input } from 'antd';
import dayjs from 'dayjs';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import BillVerificationFields from './BillVerificationFields';
import {
  billGroupOperationPayload,
  type WorkbenchFormValue,
} from './billWorkbenchHelpers';

const mocks = vi.hoisted(() => ({
  candidates: vi.fn(),
  canUse: true,
  canCashflow: true,
  canInvoice: true,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateFinanceInvoices: mocks.canInvoice,
    canCreateFinanceVerifications: mocks.canUse,
    canCreateFinanceCashflows: mocks.canCashflow,
    canUpdateFinanceCashflows: mocks.canCashflow,
    canOperateOrganization: (id: string) => id === 'company',
  }),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListVerificationCashflowCandidates: mocks.candidates,
}));

const group: API.BillBatchPreviewGroup = {
  groupKey: 'group',
  direction: 'RECEIVABLE',
  settlementPartyId: 'party',
  currency: 'CNY',
  totalAmount: '100',
  fees: [{ id: 'fee', totalAmount: '100' }],
};
const cashflow = {
  id: 'cash-1',
  organizationId: 'company',
  flowNo: 'SK-001',
  unverifiedAmount: '60',
};

function InvoiceDraftActions() {
  const form = Form.useFormInstance();
  return (
    <Button
      onClick={() => form.setFieldValue(['groups', 'group', 'invoices'], [])}
    >
      移除全部发票
    </Button>
  );
}
function InvoiceDraftField() {
  return null;
}

function renderFields(onFinish = vi.fn(), onEdit = vi.fn()) {
  return renderWithClient(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form<WorkbenchFormValue>
        layout="vertical"
        initialValues={{
          groups: {
            group: {
              statementTitle: '保留的账单抬头',
              invoices: [
                {
                  invoiceProfileId: 'profile',
                  invoiceType: 'DIGITAL_NORMAL',
                  taxInvoiceNo: 'TAX',
                  invoiceDate: dayjs(),
                  totalAmount: '20',
                  allocations: [
                    {
                      billId: 'group',
                      amount: '20',
                      fees: [
                        { orderFeeId: 'fee', amount: '20', taxAmount: '0' },
                      ],
                    },
                  ],
                },
              ],
              verifications: [],
            },
          },
        }}
        onFinish={onFinish}
        onValuesChange={onEdit}
      >
        <Form.Item
          name={['groups', 'group', 'statementTitle']}
          label="账单抬头"
        >
          <Input />
        </Form.Item>
        <Form.Item name={['groups', 'group', 'invoices']} hidden>
          <InvoiceDraftField />
        </Form.Item>
        <InvoiceDraftActions />
        <BillVerificationFields group={group} organizationId="company" />
        <Button htmlType="submit">保存本组</Button>
      </Form>
    </ConfigProvider>,
  );
}

function verificationSection() {
  return screen.getByRole('button', { name: '同次核销（可选）' });
}

async function selectExistingCashflow() {
  fireEvent.click(verificationSection());
  fireEvent.click(screen.getByRole('button', { name: /添加同次核销/ }));
  fireEvent.click(screen.getByRole('button', { name: /添加既有资金分配/ }));
  const select = screen.getByLabelText('既有资金流水');
  fireEvent.mouseDown(select);
  fireEvent.click(await screen.findByText('SK-001 · 余额 60.00 CNY'));
  return select;
}

describe('建账附带多笔核销输入', () => {
  beforeEach(() => {
    mocks.canUse = true;
    mocks.canCashflow = true;
    mocks.canInvoice = true;
    mocks.candidates.mockReset().mockResolvedValue({ data: [cashflow] });
  });

  it('真实表单新增两笔核销，既有资金与差额随同本组提交', async () => {
    const onFinish = vi.fn();
    const onEdit = vi.fn();
    renderFields(onFinish, onEdit);
    expect(verificationSection()).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByRole('button', { name: /添加同次核销/ })).toBeNull();
    await selectExistingCashflow();
    fireEvent.change(screen.getByLabelText('本次金额（CNY）'), {
      target: { value: '60' },
    });
    fireEvent.click(screen.getByRole('button', { name: /添加同次核销/ }));
    expect(screen.getByText('第2笔核销')).toBeVisible();
    fireEvent.click(screen.getAllByLabelText('补录差额收付款')[1]);
    fireEvent.change(screen.getByLabelText('差额金额（全额用于本次核销）'), {
      target: { value: '40' },
    });
    fireEvent.change(screen.getByLabelText('我方账户'), {
      target: { value: '公司收款账户' },
    });
    fireEvent.click(verificationSection());
    fireEvent.click(screen.getByRole('button', { name: '保存本组' }));
    await waitFor(() => expect(onFinish).toHaveBeenCalledTimes(1));
    const values = onFinish.mock.calls[0][0] as WorkbenchFormValue;
    expect(values.groups.group.statementTitle).toBe('保留的账单抬头');
    const payload = billGroupOperationPayload(group, values.groups.group);
    expect(payload.verifications).toHaveLength(2);
    expect(payload.verifications?.[0].allocations).toEqual([
      { cashflowId: 'cash-1', amount: '60.00000000' },
    ]);
    expect(payload.verifications?.[1]).toEqual(
      expect.objectContaining({
        allocations: [{ cashflowId: '', amount: '40.00000000' }],
        shortfallCashflow: expect.objectContaining({
          amount: '40.00000000',
          ourAccount: '公司收款账户',
          paymentMethod: 'BANK_TRANSFER',
        }),
      }),
    );
    onEdit.mockClear();
    fireEvent.click(verificationSection());
    fireEvent.click(screen.getAllByRole('button', { name: /移除核销/ })[1]);
    expect(screen.queryByText('第2笔核销')).not.toBeInTheDocument();
    expect(onEdit).toHaveBeenCalledTimes(1);
    expect(screen.getByLabelText('本次金额（CNY）')).toHaveValue('60');
    await act(async () => {});
  });

  it('搜索空结果保留已选资金和金额，查询范围固定当前公司与本组', async () => {
    renderFields();
    const select = await selectExistingCashflow();
    fireEvent.change(screen.getByLabelText('本次金额（CNY）'), {
      target: { value: '35' },
    });
    mocks.candidates.mockResolvedValue({ data: [] });
    fireEvent.change(select, { target: { value: '不存在的流水' } });
    await waitFor(() =>
      expect(mocks.candidates).toHaveBeenLastCalledWith(
        expect.objectContaining({
          organizationId: 'company',
          settlementPartyId: 'party',
          direction: 'RECEIVABLE',
          currency: 'CNY',
          keyword: '不存在的流水',
        }),
      ),
    );
    await waitFor(() =>
      expect(
        screen.getAllByText('SK-001 · 余额 60.00 CNY').length,
      ).toBeGreaterThan(0),
    );
    expect(screen.getByLabelText('本次金额（CNY）')).toHaveValue('35');
    await act(async () => {});
  });

  it('移除发票保留核销草稿并拒绝提交，无发票权限不提供新核销', async () => {
    const finish = vi.fn();
    const view = renderFields(finish);
    await selectExistingCashflow();
    fireEvent.change(screen.getByLabelText('本次金额（CNY）'), {
      target: { value: '35' },
    });
    fireEvent.click(screen.getByRole('button', { name: '移除全部发票' }));
    expect(screen.getByLabelText('本次金额（CNY）')).toHaveValue('35');
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /添加同次核销/ }),
      ).toBeDisabled(),
    );
    fireEvent.click(verificationSection());
    fireEvent.click(screen.getByRole('button', { name: '保存本组' }));
    await waitFor(() =>
      expect(
        screen.getByText('账单尚未开票，请先登记发票后核销'),
      ).toBeInTheDocument(),
    );
    expect(finish).not.toHaveBeenCalled();
    fireEvent.click(verificationSection());
    expect(
      await screen.findByText('账单尚未开票，请先登记发票后核销'),
    ).toBeVisible();
    expect(finish).not.toHaveBeenCalled();
    view.unmount();
    mocks.canInvoice = false;
    mocks.candidates.mockClear();
    renderFields(finish);
    fireEvent.click(verificationSection());
    expect(screen.getByRole('button', { name: /添加同次核销/ })).toBeDisabled();
    expect(mocks.candidates).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '保存本组' }));
    await waitFor(() => expect(finish).toHaveBeenCalledTimes(1));
  });

  it('无核销权限不加载候选，只有核销权限时不提供差额登记', async () => {
    mocks.canUse = false;
    const view = renderFields();
    expect(screen.queryByText('添加同次核销')).not.toBeInTheDocument();
    expect(mocks.candidates).not.toHaveBeenCalled();
    view.unmount();
    mocks.canUse = true;
    mocks.canCashflow = false;
    renderFields();
    fireEvent.click(verificationSection());
    fireEvent.click(screen.getByRole('button', { name: /添加同次核销/ }));
    expect(screen.queryByLabelText('补录差额收付款')).not.toBeInTheDocument();
    await waitFor(() => expect(mocks.candidates).toHaveBeenCalledTimes(1));
    await act(async () => {});
  });
});
