import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import dayjs from 'dayjs';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import InvoiceRegistrationEditor from './InvoiceRegistrationEditor';

const api = vi.hoisted(() => ({
  profiles: vi.fn(),
  create: vi.fn(),
  submit: vi.fn(),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListInvoiceProfilesForParty: api.profiles,
}));
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartnerInvoiceProfile: api.create,
}));
const user: API.CurrentUser = {
  id: 'user',
  currentOrganization: {
    id: 'company',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'system.finance.invoice.create', dataScope: 'organization' },
    { key: 'business.partner.update', dataScope: 'organization' },
  ],
};
let form: FormInstance;
function Scene({
  party = 'party',
  original = false,
  manual = false,
  disabled = false,
} = {}) {
  [form] = Form.useForm();
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form
        form={form}
        disabled={disabled}
        onFinish={api.submit}
        initialValues={{
          records: [
            {
              sourceInvoiceId: original ? 'invoice' : undefined,
              invoiceType: manual ? 'PAPER_NORMAL' : 'DIGITAL_NORMAL',
              invoiceTypeSource: manual ? 'manual' : undefined,
              taxInvoiceNo: 'INV-123',
              invoiceDate: dayjs('2026-11-17'),
              note: '保留备注',
              totalAmount: '100',
              allocations: [],
            },
          ],
        }}
      >
        <InvoiceRegistrationEditor
          name={['records']}
          organizationId="company"
          settlementPartyId={party}
          bills={[]}
        />
      </Form>
    </ConfigProvider>
  );
}
async function openCreate() {
  fireEvent.mouseDown(screen.getByLabelText('开票抬头'));
  fireEvent.click(await screen.findByText('新增开票抬头'));
  const dialog = await screen.findByRole('dialog');
  await waitFor(() => expect(dialog).toBeVisible());
  const scope = within(dialog);
  fireEvent.change(scope.getByLabelText('发票抬头'), {
    target: { value: '测试抬头' },
  });
  fireEvent.change(scope.getByLabelText('纳税人识别号'), {
    target: { value: 'TAX123' },
  });
  return scope;
}
async function save(scope: ReturnType<typeof within>) {
  fireEvent.click(scope.getByRole('button', { name: '保存并选用' }));
  await waitFor(() => expect(api.create).toHaveBeenCalled());
}
beforeEach(() => {
  vi.clearAllMocks();
  api.profiles.mockImplementation(async ({ settlementPartyId }) => ({
    success: true,
    data: { organizationId: 'company', settlementPartyId, data: [] },
  }));
  api.create.mockResolvedValue({
    data: {
      id: 'new-profile',
      invoiceTitle: '测试抬头',
      taxpayerIdentificationNo: 'TAX123',
      defaultInvoiceType: 'DIGITAL_SPECIAL',
    },
  });
});
describe('登记抬头下拉快速创建', () => {
  it('创建后选中并带入默认票种，保留发票号码与备注，不触发登记提交', async () => {
    renderWithApp(<Scene />, { user });
    await save(await openCreate());
    await waitFor(() =>
      expect(form.getFieldValue(['records', 0, 'invoiceProfileId'])).toBe(
        'new-profile',
      ),
    );
    expect(form.getFieldValue(['records', 0, 'invoiceType'])).toBe(
      'DIGITAL_SPECIAL',
    );
    expect(form.getFieldValue(['records', 0, 'taxInvoiceNo'])).toBe('INV-123');
    expect(form.getFieldValue(['records', 0, 'note'])).toBe('保留备注');
    expect(api.create).toHaveBeenCalledWith(
      { partnerId: 'party' },
      expect.objectContaining({
        partnerId: 'party',
        invoiceTitle: '测试抬头',
        taxpayerIdentificationNo: 'TAX123',
      }),
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(api.submit).not.toHaveBeenCalled();
    expect(screen.getByText('测试抬头 · TAX123')).toBeInTheDocument();
  });
  it('创建失败保留输入可重试，成功后保留手动票种', async () => {
    api.create.mockResolvedValueOnce(undefined);
    renderWithApp(<Scene manual />, { user });
    const scope = await openCreate();
    await save(scope);
    expect(scope.getByLabelText('发票抬头')).toHaveValue('测试抬头');
    expect(
      form.getFieldValue(['records', 0, 'invoiceProfileId']),
    ).toBeUndefined();
    fireEvent.click(scope.getByRole('button', { name: '保存并选用' }));
    await waitFor(() =>
      expect(form.getFieldValue(['records', 0, 'invoiceProfileId'])).toBe(
        'new-profile',
      ),
    );
    expect(form.getFieldValue(['records', 0, 'invoiceType'])).toBe(
      'PAPER_NORMAL',
    );
  });
  it('缺少客商编辑权限隐藏入口，原票修订和父表单冻结保持只读', async () => {
    const view = renderWithApp(<Scene />, {
      user: {
        ...user,
        permissionCapabilities: [
          { key: 'system.finance.invoice.create', dataScope: 'organization' },
        ],
      },
    });
    fireEvent.mouseDown(screen.getByLabelText('开票抬头'));
    await waitFor(() => expect(api.profiles).toHaveBeenCalled());
    expect(screen.queryByText('新增开票抬头')).not.toBeInTheDocument();
    view.unmount();
    const originalView = renderWithApp(<Scene original />, { user });
    expect(screen.getByLabelText('开票抬头')).toBeDisabled();
    originalView.unmount();
    renderWithApp(<Scene disabled />, { user });
    expect(screen.getByLabelText('开票抬头')).toBeDisabled();
    expect(api.create).not.toHaveBeenCalled();
    await act(async () => {});
  });
  it('创建中的记录被移除后，迟到结果不回填移到该位置的另一张发票', async () => {
    let resolve!: (value: API.CreatePartnerInvoiceProfileResponse) => void;
    api.create.mockImplementation(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    renderWithApp(<Scene />, { user });
    fireEvent.click(screen.getByText('继续添加发票'));
    fireEvent.mouseDown(screen.getAllByLabelText('开票抬头')[0]);
    fireEvent.click(await screen.findByText('新增开票抬头'));
    const scope = within(await screen.findByRole('dialog'));
    fireEvent.change(scope.getByLabelText('发票抬头'), {
      target: { value: '将被移除' },
    });
    fireEvent.change(scope.getByLabelText('纳税人识别号'), {
      target: { value: 'TAX' },
    });
    await waitFor(() =>
      expect(scope.getByRole('button', { name: '保存并选用' })).toBeVisible(),
    );
    await save(scope);
    fireEvent.click(screen.getAllByText('移除发票记录')[0]);
    await act(async () => {
      resolve({ data: { id: 'removed-profile' } });
    });
    expect(form.getFieldValue('records')).toHaveLength(1);
    expect(
      form.getFieldValue(['records', 0, 'invoiceProfileId']),
    ).toBeUndefined();
  });
  it('默认专票要求税务开户资料，超过抬头长度上限不发送创建请求', async () => {
    renderWithApp(<Scene />, { user });
    const scope = await openCreate();
    fireEvent.mouseDown(scope.getByLabelText('默认发票类型'));
    fireEvent.click(await screen.findByText('数电专票'));
    await act(async () => {
      fireEvent.click(scope.getByRole('button', { name: '保存并选用' }));
    });
    expect(
      await scope.findByText('专票必须填写税务登记地址'),
    ).toBeInTheDocument();
    expect(api.create).not.toHaveBeenCalled();
    for (const label of [
      '税务登记地址',
      '税务登记电话',
      '开户银行',
      '银行账号',
    ])
      fireEvent.change(scope.getByLabelText(label), {
        target: { value: '测试资料' },
      });
    fireEvent.change(scope.getByLabelText('发票抬头'), {
      target: { value: '中'.repeat(201) },
    });
    await act(async () => {
      fireEvent.click(scope.getByRole('button', { name: '保存并选用' }));
    });
    expect(
      await scope.findByText('发票抬头不能超过200个字符'),
    ).toBeInTheDocument();
    expect(api.create).not.toHaveBeenCalled();
  });
  it('切换结算单位后拒绝旧创建响应', async () => {
    let resolve!: (value: API.CreatePartnerInvoiceProfileResponse) => void;
    api.create.mockImplementation(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    const view = renderWithApp(<Scene />, { user });
    await save(await openCreate());
    view.rerender(<Scene party="another-party" />);
    await act(async () => {
      resolve({ data: { id: 'late-profile' } });
    });
    expect(
      form.getFieldValue(['records', 0, 'invoiceProfileId']),
    ).toBeUndefined();
    expect(screen.queryByText(/late-profile/)).not.toBeInTheDocument();
  });
});
