import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App, Form, Input } from 'antd';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  partnerServiceCreatePartner,
  partnerServiceGetPartner,
  partnerServiceListPartnerSettlementRules,
  partnerServiceUpdatePartner,
} from '@/services/roncin/partnerService';
import PartnerDetailPage from './partner-detail';

const state = vi.hoisted(() => ({
  id: 'create',
  pathname: '/partners/customers/create',
  baseCurrency: 'CNY',
  canReadRule: false,
  canCreateRule: false,
  canUpdateRule: false,
  managePartners: true,
}));
const ID = '11111111-1111-4111-8111-111111111111';
vi.mock('react-router', async (load) => ({
  ...(await load<typeof import('react-router')>()),
  useParams: () => ({ id: state.id }),
  useLocation: () => ({ pathname: state.pathname, search: '' }),
  useSearchParams: () => [new URLSearchParams()],
  Link: ({ children }: { children: ReactNode }) => <span>{children}</span>,
}));
vi.mock('@/components/layout/useRouteLeaveGuard', () => ({
  useRouteLeaveGuard: vi.fn(),
}));
vi.mock('@/router/history', () => ({
  history: { push: vi.fn(), replace: vi.fn() },
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        currentOrganization: { baseCurrency: state.baseCurrency },
      },
    },
  }),
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateBusiness: true,
    canOperateOrganization: () => true,
    canCreatePartners: true,
    canUpdatePartners: true,
    canManagePartners: state.managePartners,
    canReadPartnerSettlementRules: state.canReadRule,
    canCreatePartnerSettlementRules: state.canCreateRule,
    canUpdatePartnerSettlementRules: state.canUpdateRule,
  }),
}));
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceGetPartner: vi.fn(),
  partnerServiceCreatePartner: vi.fn(),
  partnerServiceUpdatePartner: vi.fn(),
  partnerServiceListPartnerAssignmentOptions: vi
    .fn()
    .mockResolvedValue({ data: [] }),
  partnerServiceListPartnerSettlementRules: vi.fn(),
}));
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListUsers: vi.fn().mockResolvedValue({ data: [] }),
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('./components/BasicInfoSection', () => ({
  default: (p: {
    newAliasInput: string;
    setNewAliasInput: (v: string) => void;
    onAddAlias: () => void;
  }) => (
    <>
      <Form.Item name="legalName" label="公司抬头" rules={[{ required: true }]}>
        <Input />
      </Form.Item>
      <Input
        aria-label="新增别名"
        value={p.newAliasInput}
        onChange={(e) => p.setNewAliasInput(e.target.value)}
      />
      <button type="button" onClick={p.onAddAlias}>
        添加别名
      </button>
    </>
  ),
}));
vi.mock('./components/ContactsSection', () => ({
  default: (p: { onChange: (v: { name: string }[]) => void }) => (
    <button type="button" onClick={() => p.onChange([{ name: '联系人' }])}>
      添加联系人
    </button>
  ),
}));
vi.mock('./components/AccountsSection', () => ({ default: () => null }));
vi.mock('./components/ContractsSection', () => ({ default: () => null }));
vi.mock('./components/LogsSection', () => ({ default: () => null }));
vi.mock('./components/PartnerAnchorNav', () => ({ default: () => null }));
vi.mock('./components/PresetsSection', () => ({ default: () => null }));
vi.mock('./components/RemarkSection', () => ({ default: () => null }));

function renderPage(edit = false) {
  if (edit) {
    state.id = ID;
    state.pathname = `/partners/customers/${ID}`;
  }
  return renderWithClient(
    <App>
      <PartnerDetailPage />
    </App>,
  );
}
async function save() {
  await waitFor(() =>
    expect(
      screen.getAllByRole('button', { name: /保存客户档案/ })[0],
    ).toBeEnabled(),
  );
  fireEvent.click(screen.getAllByRole('button', { name: /保存客户档案/ })[0]);
}
beforeEach(() => {
  vi.clearAllMocks();
  state.id = 'create';
  state.pathname = '/partners/customers/create';
  state.baseCurrency = 'CNY';
  state.canReadRule = false;
  state.canCreateRule = false;
  state.canUpdateRule = false;
  state.managePartners = true;
  vi.mocked(partnerServiceListPartnerSettlementRules)
    .mockReset()
    .mockResolvedValue({ data: [] });
  vi.mocked(partnerServiceGetPartner)
    .mockReset()
    .mockResolvedValue({
      data: { id: ID, legalName: '原名称', organizationId: 'org' },
    });
  vi.mocked(partnerServiceCreatePartner).mockReset();
  vi.mocked(partnerServiceUpdatePartner).mockReset();
});

describe('主档结算规则独立权限', () => {
  it('新建无规则创建权限时隐藏分节且不构造规则载荷', async () => {
    vi.mocked(partnerServiceCreatePartner).mockResolvedValue({
      data: { id: ID },
    });
    renderPage();
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '普通新客户' },
    });
    expect(screen.queryByText('财务结算规则')).not.toBeInTheDocument();
    await save();
    await waitFor(() => expect(partnerServiceCreatePartner).toHaveBeenCalled());
    expect(
      vi
        .mocked(partnerServiceCreatePartner)
        .mock.calls[0][0].roles?.every(
          (role) => role.settlementRule === undefined,
        ),
    ).toBe(true);
  });

  it('规则只读且读取失败时仍允许保存普通主档，不补建规则', async () => {
    state.canReadRule = true;
    vi.mocked(partnerServiceListPartnerSettlementRules).mockRejectedValue(
      new Error('无可用规则响应'),
    );
    vi.mocked(partnerServiceUpdatePartner).mockResolvedValue({
      data: { id: ID },
    });
    renderPage(true);
    await screen.findByText(
      '结算规则加载失败，可重试查看；普通档案资料仍可保存',
    );
    await save();
    await waitFor(() => expect(partnerServiceUpdatePartner).toHaveBeenCalled());
    expect(
      vi
        .mocked(partnerServiceUpdatePartner)
        .mock.calls[0][1].roles?.every(
          (role) => role.settlementRule === undefined,
        ),
    ).toBe(true);
  });
  it('只有主档管理权限时不请求规则且普通保存不携带规则', async () => {
    vi.mocked(partnerServiceUpdatePartner).mockResolvedValue({
      data: { id: ID },
    });
    renderPage(true);
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('原名称'),
    );
    expect(screen.queryByText('财务结算规则')).not.toBeInTheDocument();
    expect(partnerServiceListPartnerSettlementRules).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('公司抬头'), {
      target: { value: '只改普通名称' },
    });
    await save();
    await waitFor(() => expect(partnerServiceUpdatePartner).toHaveBeenCalled());
    const payload = vi.mocked(partnerServiceUpdatePartner).mock.calls[0][1];
    expect(payload.legalName).toBe('只改普通名称');
    expect(
      payload.roles?.every((role) => role.settlementRule === undefined),
    ).toBe(true);
  });

  it('规则只读时分节禁用，异常额度不阻断普通保存', async () => {
    state.canReadRule = true;
    vi.mocked(partnerServiceListPartnerSettlementRules).mockResolvedValue({
      data: [
        {
          id: 'rule',
          statementMode: 1,
          settlementMethod: 1,
          settlementCurrency: 'USD',
          creditLimitMinor: '10000',
          creditCurrency: 'USD',
        },
      ],
    });
    vi.mocked(partnerServiceUpdatePartner).mockResolvedValue({
      data: { id: ID },
    });
    renderPage(true);
    const credit = await screen.findByLabelText('信用额度(CNY)');
    expect(credit).toBeDisabled();
    expect(screen.getByLabelText('默认账期(天)')).toBeDisabled();
    await save();
    await waitFor(() => expect(partnerServiceUpdatePartner).toHaveBeenCalled());
    expect(
      vi
        .mocked(partnerServiceUpdatePartner)
        .mock.calls[0][1].roles?.every(
          (role) => role.settlementRule === undefined,
        ),
    ).toBe(true);
  });

  it.each([false, true])(
    '已有档案的 Create-only 权限按规则存在性决定写入（已有规则：%s）',
    async (hasRule) => {
      state.canReadRule = true;
      state.canCreateRule = true;
      vi.mocked(partnerServiceListPartnerSettlementRules).mockResolvedValue({
        data: hasRule
          ? [
              {
                id: 'rule',
                statementMode: 1,
                settlementMethod: 1,
                settlementCurrency: 'CNY',
              },
            ]
          : [],
      });
      vi.mocked(partnerServiceUpdatePartner).mockResolvedValue({
        data: { id: ID },
      });
      renderPage(true);
      const credit = await screen.findByLabelText('信用额度(CNY)');
      if (hasRule) expect(credit).toBeDisabled();
      else expect(credit).toBeEnabled();
      await save();
      await waitFor(() =>
        expect(partnerServiceUpdatePartner).toHaveBeenCalled(),
      );
      expect(
        Boolean(
          vi.mocked(partnerServiceUpdatePartner).mock.calls[0][1].roles?.[0]
            ?.settlementRule,
        ),
      ).toBe(!hasRule);
    },
  );
});
