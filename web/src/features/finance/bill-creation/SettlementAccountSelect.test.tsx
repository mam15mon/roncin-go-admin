import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind, PartnerAccountUsage } from '@/enums.generated';
import SettlementAccountSelect from './SettlementAccountSelect';

const mocks = vi.hoisted(() => ({ candidates: vi.fn(), create: vi.fn() }));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListBillSettlementAccountCandidates: mocks.candidates,
}));
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartnerAccount: mocks.create,
}));

const group: API.BillBatchPreviewGroup = {
  groupKey: 'g1',
  settlementPartyId: 'partner-a',
  settlementPartyName: '结算单位甲',
  direction: 'RECEIVABLE',
  currency: 'CNY',
};
const user: API.CurrentUser = {
  id: 'user',
  currentOrganization: {
    id: 'A',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'business.partner.account.create', dataScope: 'organization' },
  ],
};
let form: FormInstance;
const changed = vi.fn();
function Scene({
  organizationId = 'A',
  session = 'one',
  second = false,
  disabled = false,
  direction = 'RECEIVABLE',
}) {
  [form] = Form.useForm();
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form
        form={form}
        disabled={disabled}
        initialValues={{
          groups: {
            g1: {
              note: '保留备注',
              paymentTermsDays: 30,
              estimatedInvoiceRate: '1.25',
            },
            g2: { settlementAccountId: 'second-account', note: '另一组备注' },
          },
        }}
      >
        <SettlementAccountSelect
          group={{ ...group, direction }}
          organizationId={organizationId}
          sessionIdentity={session}
          onConfigurationChange={changed}
        />
        {second && (
          <SettlementAccountSelect
            group={{ ...group, groupKey: 'g2', settlementPartyId: 'partner-b' }}
            organizationId={organizationId}
            sessionIdentity={session}
            onConfigurationChange={changed}
          />
        )}
      </Form>
    </ConfigProvider>
  );
}
async function openQuickCreate() {
  fireEvent.mouseDown(screen.getAllByLabelText('结算账户')[0]);
  fireEvent.click(await screen.findByText('新增结算账户'));
  const dialog = await screen.findByRole('dialog');
  await waitFor(() => expect(dialog).toBeVisible());
  return within(dialog);
}
async function fillAndSave(dialog: ReturnType<typeof within>) {
  for (const [label, value] of [
    ['账户名称', '新账户'],
    ['账户户名', '测试户名'],
    ['开户银行名称及支行', '测试银行'],
    ['银行开户账号', '6222001'],
  ]) {
    fireEvent.change(dialog.getByLabelText(label), { target: { value } });
  }
  await act(async () => {
    fireEvent.click(dialog.getByRole('button', { name: '保存并选用' }));
  });
}

beforeEach(() => {
  mocks.candidates.mockReset().mockResolvedValue({ data: [] });
  mocks.create
    .mockReset()
    .mockResolvedValue({ data: { id: 'created-account' } });
  changed.mockReset();
});

describe('建账结算账户快建', () => {
  it('锁定来源并只重载、回填当前叶子，其他配置完整保留', async () => {
    renderWithApp(<Scene second />, { user });
    await waitFor(() => expect(mocks.candidates).toHaveBeenCalledTimes(2));
    const dialog = await openQuickCreate();
    expect(dialog.getByLabelText('结算对象')).toHaveValue('结算单位甲');
    expect(dialog.getByLabelText('结算对象')).toBeDisabled();
    expect(dialog.getByLabelText('账户币种')).toHaveValue('CNY');
    expect(dialog.getByLabelText('账户币种')).toBeDisabled();
    expect(dialog.getByLabelText('允许用途')).toHaveValue('应收');
    expect(dialog.getByLabelText('允许用途')).toBeDisabled();
    fireEvent.click(dialog.getByLabelText('设为应收默认账户'));
    await fillAndSave(dialog);
    await waitFor(() =>
      expect(form.getFieldValue(['groups', 'g1', 'settlementAccountId'])).toBe(
        'created-account',
      ),
    );
    expect(mocks.create).toHaveBeenCalledWith(
      { partnerId: 'partner-a' },
      {
        partnerId: 'partner-a',
        account: {
          name: '新账户',
          accountHolder: '测试户名',
          bankName: '测试银行',
          accountNo: '6222001',
          swiftCode: undefined,
          currency: 'CNY',
          usage: PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_RECEIVABLE,
          isDefaultReceivable: true,
          isDefaultPayable: false,
          enabled: true,
        },
      },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(form.getFieldValue(['groups', 'g1', 'note'])).toBe('保留备注');
    expect(form.getFieldValue(['groups', 'g1', 'paymentTermsDays'])).toBe(30);
    expect(form.getFieldValue(['groups', 'g1', 'estimatedInvoiceRate'])).toBe(
      '1.25',
    );
    expect(form.getFieldValue(['groups', 'g2', 'settlementAccountId'])).toBe(
      'second-account',
    );
    expect(form.getFieldValue(['groups', 'g2', 'note'])).toBe('另一组备注');
    await waitFor(() => expect(mocks.candidates).toHaveBeenCalledTimes(3));
    expect(
      mocks.candidates.mock.calls.filter(
        ([params]) => params.settlementPartyId === 'partner-b',
      ),
    ).toHaveLength(1);
    expect(changed).toHaveBeenCalledTimes(1);
    expect(
      screen.getAllByText('新账户｜测试银行 · 6222001｜CNY')[0],
    ).toBeInTheDocument();
  });

  it('取消不创建；失败保留弹窗输入并允许重试，重开后不恢复上次输入', async () => {
    renderWithApp(<Scene />, { user });
    let dialog = await openQuickCreate();
    fireEvent.click(dialog.getByRole('button', { name: /取\s*消/ }));
    expect(mocks.create).not.toHaveBeenCalled();
    dialog = await openQuickCreate();
    mocks.create.mockResolvedValueOnce(undefined);
    await fillAndSave(dialog);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(dialog.getByLabelText('账户名称')).toHaveValue('新账户');
    expect(
      form.getFieldValue(['groups', 'g1', 'settlementAccountId']),
    ).toBeUndefined();
    await act(async () => {
      fireEvent.click(dialog.getByRole('button', { name: '保存并选用' }));
    });
    await waitFor(() =>
      expect(form.getFieldValue(['groups', 'g1', 'settlementAccountId'])).toBe(
        'created-account',
      ),
    );
    dialog = await openQuickCreate();
    expect(dialog.getByLabelText('账户名称')).toHaveValue('');
  });

  it('创建成功缺少业务 ID 显式报错且不回填', async () => {
    mocks.create.mockResolvedValue({ data: { name: '没有身份的账户' } });
    renderWithApp(<Scene />, { user });
    const dialog = await openQuickCreate();
    await fillAndSave(dialog);
    expect(
      await screen.findByText('创建结果缺少账户 ID，请核对后重试'),
    ).toBeInTheDocument();
    expect(dialog.getByLabelText('账户名称')).toHaveValue('新账户');
    expect(changed).not.toHaveBeenCalled();
  });

  it('无快建权限仍可选择候选，隐藏快建入口', async () => {
    renderWithApp(<Scene />, { user: { ...user, permissionCapabilities: [] } });
    fireEvent.mouseDown(screen.getByLabelText('结算账户'));
    await waitFor(() => expect(mocks.candidates).toHaveBeenCalledTimes(1));
    expect(screen.queryByText('新增结算账户')).not.toBeInTheDocument();
  });

  it('来源公司与当前公司不符时不请求候选、不允许快建', async () => {
    renderWithApp(<Scene organizationId="B" />, { user });
    expect(screen.getByLabelText('结算账户')).toBeDisabled();
    expect(mocks.candidates).not.toHaveBeenCalled();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it('应付组固定应付用途，默认开关只写应付默认值', async () => {
    renderWithApp(<Scene direction="PAYABLE" />, { user });
    const dialog = await openQuickCreate();
    expect(dialog.getByLabelText('允许用途')).toHaveValue('应付');
    fireEvent.click(dialog.getByLabelText('设为应付默认账户'));
    await fillAndSave(dialog);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create.mock.calls[0][1].account).toMatchObject({
      usage: PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_PAYABLE,
      isDefaultPayable: true,
      isDefaultReceivable: false,
    });
  });

  it('叶子身份改变后，旧候选响应不能回填当前组', async () => {
    let resolveOld!: (
      value: API.ListBillSettlementAccountCandidatesResponse,
    ) => void;
    mocks.candidates.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveOld = resolve;
        }),
    );
    const view = renderWithApp(<Scene />, { user });
    await waitFor(() => expect(mocks.candidates).toHaveBeenCalledTimes(1));
    view.rerender(<Scene session="two" />);
    await waitFor(() => expect(mocks.candidates).toHaveBeenCalledTimes(2));
    await act(async () => {
      resolveOld({
        data: [
          {
            id: 'old-default',
            name: '旧候选',
            isDefault: true,
            currency: 'CNY',
          },
        ],
      });
    });
    expect(
      form.getFieldValue(['groups', 'g1', 'settlementAccountId']),
    ).toBeUndefined();
    expect(changed).not.toHaveBeenCalled();
  });

  it('父表单冻结时继承禁用且关闭快建入口', async () => {
    const view = renderWithApp(<Scene />, { user });
    await openQuickCreate();
    view.rerender(<Scene disabled />);
    expect(screen.getByLabelText('结算账户')).toBeDisabled();
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it('新会话不接收旧会话创建结果或旧候选响应', async () => {
    let resolveCreate!: (value: API.CreatePartnerAccountResponse) => void;
    mocks.create.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve;
        }),
    );
    const view = renderWithApp(<Scene />, { user });
    const dialog = await openQuickCreate();
    await fillAndSave(dialog);
    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    view.rerender(<Scene session="two" />);
    await act(async () => {
      resolveCreate({ data: { id: 'late-account' } });
    });
    expect(
      form.getFieldValue(['groups', 'g1', 'settlementAccountId']),
    ).toBeUndefined();
    expect(changed).not.toHaveBeenCalled();
    expect(screen.queryByText(/late-account/)).not.toBeInTheDocument();
  });
});
