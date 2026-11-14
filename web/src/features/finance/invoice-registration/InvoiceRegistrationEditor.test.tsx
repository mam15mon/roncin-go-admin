import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import dayjs from 'dayjs';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import InvoiceRegistrationEditor from './InvoiceRegistrationEditor';
import type { InvoiceRegistrationDraft } from './types';

const mocks = vi.hoisted(() => ({ profiles: vi.fn() }));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListInvoiceProfilesForParty: mocks.profiles,
}));
const user: API.CurrentUser = {
  id: 'user',
  currentOrganization: {
    id: 'company',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'system.finance.invoice.create', dataScope: 'organization' },
  ],
};
let form: FormInstance<{ records: InvoiceRegistrationDraft[] }>;
function Scene({ allowAdd = true }: { allowAdd?: boolean }) {
  [form] = Form.useForm();
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          records: [
            {
              invoiceProfileId: 'profile',
              invoiceType: 'NORMAL',
              taxInvoiceNo: 'TAX-1',
              invoiceDate: dayjs('2026-11-14'),
              totalAmount: '10',
              allocations: [
                {
                  billId: 'bill',
                  amount: '10',
                  fees: [{ orderFeeId: 'fee', amount: '10', taxAmount: '0' }],
                },
              ],
            },
          ],
        }}
      >
        <InvoiceRegistrationEditor
          name={['records']}
          organizationId="company"
          settlementPartyId="party"
          allowAdd={allowAdd}
          bills={[
            {
              id: 'bill',
              version: '1',
              billNo: 'BI-1',
              currency: 'CNY',
              fees: [
                {
                  orderFeeId: 'fee',
                  feeCode: 'FEE',
                  feeName: '海运费',
                  invoiceableAmount: '100',
                },
              ],
            },
          ]}
        />
      </Form>
    </ConfigProvider>
  );
}
beforeEach(() => {
  mocks.profiles.mockReset();
  mocks.profiles.mockResolvedValue({
    success: true,
    data: {
      organizationId: 'company',
      settlementPartyId: 'party',
      data: [
        {
          id: 'profile',
          invoiceTitle: '抬头',
          taxpayerIdentificationNo: 'TAX',
        },
      ],
    },
  });
});
describe('共享发票编辑器', () => {
  it('只需登记权限即可加载抬头；来源税率未设显示缺省，添加与移除不丢其他值', async () => {
    renderWithApp(<Scene />, { user });
    await waitFor(() =>
      expect(mocks.profiles).toHaveBeenCalledWith(
        { settlementPartyId: 'party' },
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      ),
    );
    fireEvent.mouseDown(screen.getByLabelText('来源费用及可登记余额'));
    expect(
      screen.getAllByText(/FEE 海运费 · 100.00 · 来源税率 -/)[0],
    ).toBeVisible();
    fireEvent.click(screen.getByText('继续添加发票'));
    expect(form.getFieldValue('records')).toHaveLength(2);
    expect(form.getFieldValue(['records', 0, 'taxInvoiceNo'])).toBe('TAX-1');
    await act(async () => {});
  });
  it('含税额正零拒绝，手填税额0允许，金额降低触发税额复核', async () => {
    renderWithApp(<Scene />, { user });
    await waitFor(() => expect(mocks.profiles).toHaveBeenCalled());
    await act(async () => {
      await expect(form.validateFields()).resolves.toBeTruthy();
    });
    fireEvent.change(screen.getByLabelText('发票含税总额'), {
      target: { value: '0' },
    });
    await act(async () => {
      await expect(form.validateFields()).rejects.toBeTruthy();
    });
    expect(
      await screen.findByText('请输入大于0的金额，最多8位小数'),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('发票含税总额'), {
      target: { value: '10' },
    });
    fireEvent.change(screen.getByLabelText('本次登记税额（财务填写）'), {
      target: { value: '5' },
    });
    await act(async () => {
      await expect(form.validateFields()).resolves.toBeTruthy();
    });
    fireEvent.change(screen.getByLabelText('本次含税金额'), {
      target: { value: '1' },
    });
    expect(
      await screen.findByText('税额须在0与本次含税金额之间，最多8位小数'),
    ).toBeInTheDocument();
  });
  it('无新增票授权只显示原登记编辑，不提供新增行', async () => {
    renderWithApp(<Scene allowAdd={false} />, { user });
    expect(screen.queryByText('继续添加发票')).not.toBeInTheDocument();
    expect(screen.getByLabelText('已有发票号码')).toHaveValue('TAX-1');
    await waitFor(() => expect(mocks.profiles).toHaveBeenCalled());
  });
});
