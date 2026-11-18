import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import SettlementRulesPanel from './SettlementRulesPanel';

const state = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
  list: vi.fn(),
}));
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartnerSettlementRules: state.list,
  partnerServiceCreatePartnerSettlementRule: state.create,
  partnerServiceUpdatePartnerSettlementRule: state.update,
}));
const customerType = PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER;
const partner = {
  id: 'partner',
  roles: [{ type: customerType, enabled: true }],
};
function mount(
  options: {
    currency?: string;
    read?: boolean;
    create?: boolean;
    update?: boolean;
    target?: API.Partner;
  } = {},
) {
  return renderWithApp(
    <SettlementRulesPanel
      partner={options.target ?? partner}
      canRead={options.read ?? true}
      canCreate={options.create ?? true}
      canUpdate={options.update ?? true}
    />,
    {
      user: {
        id: 'user',
        currentOrganization: {
          id: 'company',
          baseCurrency: options.currency ?? 'CNY',
        },
      },
    },
  );
}
async function submit() {
  fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
}
beforeEach(() => {
  vi.clearAllMocks();
  state.list.mockReset().mockResolvedValue({ data: [] });
  state.create.mockResolvedValue({ data: { id: 'rule' } });
  state.update.mockResolvedValue({ data: { id: 'rule' } });
});

it('新建默认为未约定，选择 N 天后提交本币额度和天数', async () => {
  mount({ currency: 'USD' });
  fireEvent.click(await screen.findByRole('button', { name: '设置收款约定' }));
  expect(await screen.findByRole('radio', { name: '未约定' })).toBeChecked();
  fireEvent.click(screen.getByRole('radio', { name: '建账后 N 天' }));
  fireEvent.change(await screen.findByLabelText('天数'), {
    target: { value: '45' },
  });
  fireEvent.change(screen.getByLabelText('信用额度（USD）'), {
    target: { value: '1000.01' },
  });
  await submit();
  await waitFor(() =>
    expect(state.create).toHaveBeenCalledWith(
      { partnerId: 'partner', roleType: customerType },
      {
        partnerId: 'partner',
        roleType: customerType,
        rule: {
          paymentTermsDays: 45,
          creditLimitMinor: '100001',
          creditCurrency: 'USD',
        },
      },
    ),
  );
});

it('已有约定回填现结，从供应商多角色档案进入仍只读写客户角色', async () => {
  state.list.mockResolvedValue({ data: [{ id: 'rule', paymentTermsDays: 0 }] });
  mount({
    target: {
      ...partner,
      roles: [
        { type: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER },
        ...partner.roles,
      ],
    },
  });
  await screen.findByText('现结');
  expect(state.list).toHaveBeenCalledWith({
    partnerId: 'partner',
    roleType: customerType,
  });
  fireEvent.click(screen.getByRole('button', { name: '编辑收款约定' }));
  expect(await screen.findByRole('radio', { name: '现结' })).toBeChecked();
  fireEvent.click(screen.getByRole('radio', { name: '未约定' }));
  await submit();
  await waitFor(() =>
    expect(state.update).toHaveBeenCalledWith(
      { partnerId: 'partner', roleType: customerType, id: 'rule' },
      { partnerId: 'partner', roleType: customerType, id: 'rule', rule: {} },
    ),
  );
});

it('异常原币额度要求明确重录，不能只保存其他字段改币种', async () => {
  state.list.mockResolvedValue({
    data: [{ id: 'rule', creditLimitMinor: '100000', creditCurrency: 'USD' }],
  });
  mount();
  fireEvent.click(await screen.findByRole('button', { name: '编辑收款约定' }));
  await screen.findByText('原信用额度为 1,000.00 USD，请重新填写 CNY 额度');
  expect(screen.getByLabelText('信用额度（CNY）')).toHaveValue('');
  await submit();
  await screen.findByText('请明确重新填写本币额度，填 0 表示不设额度');
  expect(state.update).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText('信用额度（CNY）'), {
    target: { value: '2000' },
  });
  await submit();
  await waitFor(() =>
    expect(state.update).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        rule: { creditLimitMinor: '200000', creditCurrency: 'CNY' },
      }),
    ),
  );
});

it('没有读权限或客户角色不发送查询，也不开放写入', () => {
  const first = mount({ read: false });
  expect(state.list).not.toHaveBeenCalled();
  expect(screen.queryByRole('button')).not.toBeInTheDocument();
  first.unmount();
  mount({
    target: {
      id: 'supplier',
      roles: [{ type: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER }],
    },
  });
  expect(state.list).not.toHaveBeenCalled();
  expect(screen.queryByRole('button')).not.toBeInTheDocument();
});

it('创建和修改按约定存在性分别授权，不能新增第二条', async () => {
  const first = mount({ create: false, update: true });
  await screen.findAllByText('未约定', {
    selector: '.ant-descriptions-item-content',
  });
  expect(
    screen.queryByRole('button', { name: '设置收款约定' }),
  ).not.toBeInTheDocument();
  first.unmount();
  state.list.mockResolvedValue({ data: [{ id: 'rule' }] });
  mount({ create: true, update: false });
  await screen.findAllByText('未约定', {
    selector: '.ant-descriptions-item-content',
  });
  expect(
    screen.queryByRole('button', { name: '编辑收款约定' }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole('button', { name: '设置收款约定' }),
  ).not.toBeInTheDocument();
});

it('查询失败与不完整响应不当空，重试成功才开放创建', async () => {
  state.list
    .mockRejectedValueOnce(new Error('读取失败'))
    .mockResolvedValueOnce(undefined)
    .mockResolvedValue({ success: true });
  mount();
  await screen.findByText('收款约定加载失败，请重试');
  expect(
    screen.queryByRole('button', { name: '设置收款约定' }),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '重新加载收款约定' }));
  await waitFor(() => expect(state.list).toHaveBeenCalledTimes(2));
  await screen.findByText('收款约定加载失败，请重试');
  expect(
    screen.queryByRole('button', { name: '设置收款约定' }),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '重新加载收款约定' }));
  await screen.findByRole('button', { name: '设置收款约定' });
});

it('后台读取失败和重试保留弹窗输入，失败期间不能提交', async () => {
  state.list.mockResolvedValue({
    data: [{ id: 'rule', paymentTermsDays: 30 }],
  });
  const { queryClient } = mount();
  fireEvent.click(await screen.findByRole('button', { name: '编辑收款约定' }));
  fireEvent.change(await screen.findByLabelText('天数'), {
    target: { value: '45' },
  });
  state.list.mockRejectedValueOnce(new Error('后台失败'));
  await act(async () => {
    await queryClient.refetchQueries({ queryKey: ['partner-detail'] });
  });
  const dialog = within(screen.getByRole('dialog'));
  await dialog.findByText('收款约定加载失败，请重试');
  expect(dialog.getByLabelText('天数')).toHaveValue('45');
  expect(dialog.getByRole('button', { name: /确\s*定/ })).toBeDisabled();
  await submit();
  expect(state.update).not.toHaveBeenCalled();
  fireEvent.click(dialog.getByRole('button', { name: '重新加载收款约定' }));
  await screen.findByText('建账后 30 天');
  expect(screen.getByLabelText('天数')).toHaveValue('45');
  await submit();
  await waitFor(() =>
    expect(state.update).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ rule: { paymentTermsDays: 45 } }),
    ),
  );
});
