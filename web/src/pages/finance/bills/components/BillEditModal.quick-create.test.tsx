import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  AuthOrganizationKind,
  FinanceBillFeeLockFact,
  PartnerAccountUsage,
} from '@/enums.generated';
import BillEditModal from './BillEditModal';

const api = vi.hoisted(() => ({
  accounts: vi.fn(),
  create: vi.fn(),
  changed: false,
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListBillSettlementAccountUpdateCandidates: api.accounts,
  settlementServiceListBills: vi.fn().mockResolvedValue({ data: [] }),
}));
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartnerAccount: api.create,
  partnerServiceGetPartner: vi.fn().mockResolvedValue({ data: { roles: [] } }),
}));
vi.mock('./use-bill-change-notice', () => ({
  useBillChangeNotice: () => ({ changed: api.changed }),
}));
vi.mock('@/features/master-data/currencies', () => ({
  currencyOptionsQueryOptions: () => ({
    queryKey: ['test-currencies'],
    queryFn: async () => [],
  }),
}));
const user: API.CurrentUser = {
  id: 'user',
  currentOrganization: {
    id: 'company',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'business.partner.account.create', dataScope: 'organization' },
    { key: 'system.finance.bill.update', dataScope: 'organization' },
  ],
};
let form: FormInstance;
function Scene({ id = 'bill', protectedFields = false, open = true } = {}) {
  [form] = Form.useForm();
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <BillEditModal
        open={open}
        editing={{
          id,
          version: '1',
          organizationId: 'company',
          settlementPartyId: 'party',
          settlementPartyName: '结算单位',
          direction: 'PAYABLE',
          currency: 'USD',
          feeLockFacts: protectedFields
            ? [FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_VERIFICATION]
            : [],
        }}
        form={form}
        submitting={false}
        onCancel={vi.fn()}
        onOk={async () => {}}
        onRefresh={async () => false}
      />
    </ConfigProvider>
  );
}
async function openCreate() {
  fireEvent.mouseDown(screen.getByLabelText('结算账户'));
  fireEvent.click(await screen.findByText('新增结算账户'));
  await waitFor(() => expect(screen.getAllByRole('dialog')).toHaveLength(2));
  const scope = within(screen.getAllByRole('dialog')[1]);
  for (const [label, value] of [
    ['账户名称', '新账户'],
    ['账户户名', '账户户名'],
    ['开户银行名称及支行', '测试银行'],
    ['银行开户账号', '6222001'],
  ])
    fireEvent.change(scope.getByLabelText(label), { target: { value } });
  await waitFor(() =>
    expect(scope.getByRole('button', { name: '保存并选用' })).toBeVisible(),
  );
  fireEvent.click(scope.getByRole('button', { name: '保存并选用' }));
  await waitFor(() => expect(api.create).toHaveBeenCalled());
  return scope;
}
beforeEach(() => {
  vi.clearAllMocks();
  api.changed = false;
  api.accounts.mockResolvedValue({ data: [] });
  api.create.mockResolvedValue({ data: { id: 'new-account' } });
});
describe('账单编辑结算账户快建', () => {
  it('按原结算对象、币种和用途创建，自动选中新账户且保留原编辑输入', async () => {
    renderWithApp(<Scene />, { user });
    fireEvent.change(screen.getByLabelText('对账抬头'), {
      target: { value: '保留抬头' },
    });
    await openCreate();
    await waitFor(() =>
      expect(form.getFieldValue('settlementAccountId')).toBe('new-account'),
    );
    expect(form.getFieldValue('statementTitle')).toBe('保留抬头');
    expect(api.create).toHaveBeenCalledWith(
      { partnerId: 'party' },
      expect.objectContaining({
        account: expect.objectContaining({
          currency: 'USD',
          usage: PartnerAccountUsage.PARTNER_ACCOUNT_USAGE_PAYABLE,
        }),
      }),
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(
      screen.getByText('新账户｜测试银行 · 6222001｜USD'),
    ).toBeInTheDocument();
  });
  it('受保护字段及外部变更均不开放创建入口', async () => {
    const view = renderWithApp(<Scene protectedFields />, { user });
    expect(screen.getByLabelText('结算账户')).toBeDisabled();
    view.rerender(<Scene />);
    api.changed = true;
    view.rerender(<Scene />);
    fireEvent.mouseDown(screen.getByLabelText('结算账户'));
    expect(screen.queryByText('新增结算账户')).not.toBeInTheDocument();
    expect(api.create).not.toHaveBeenCalled();
    await act(async () => {});
  });

  it('仅有账户创建权限但无账单编辑权限时不开放创建入口', async () => {
    renderWithApp(<Scene />, {
      user: {
        ...user,
        permissionCapabilities: [
          {
            key: 'business.partner.account.create',
            dataScope: 'organization',
          },
        ],
      },
    });
    fireEvent.mouseDown(screen.getByLabelText('结算账户'));
    await act(async () => {});
    expect(screen.queryByText('新增结算账户')).not.toBeInTheDocument();
    expect(api.create).not.toHaveBeenCalled();
  });
  it('关闭父弹窗再打开同账单不会重开账户新增，也不接收旧结果', async () => {
    let resolve!: (value: API.CreatePartnerAccountResponse) => void;
    api.create.mockImplementation(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    const view = renderWithApp(<Scene />, { user });
    await openCreate();
    view.rerender(<Scene open={false} />);
    view.rerender(<Scene />);
    await waitFor(() => expect(screen.getAllByRole('dialog')).toHaveLength(1));
    await act(async () => {
      resolve({ data: { id: 'closed-account' } });
    });
    expect(form.getFieldValue('settlementAccountId')).toBeUndefined();
    expect(screen.queryByLabelText('账户名称')).not.toBeInTheDocument();
  });
  it('切换账单后旧创建成功不能选入新账单', async () => {
    let resolve!: (value: API.CreatePartnerAccountResponse) => void;
    api.create.mockImplementation(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    const view = renderWithApp(<Scene />, { user });
    await openCreate();
    view.rerender(<Scene id="new-bill" />);
    await act(async () => {
      resolve({ data: { id: 'late-account' } });
    });
    expect(form.getFieldValue('settlementAccountId')).toBeUndefined();
    expect(screen.queryByText(/late-account/)).not.toBeInTheDocument();
  });
});
