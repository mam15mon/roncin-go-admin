import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App, Form, Input } from 'antd';
import type { ReactNode } from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import {
  partnerServiceCreatePartner,
  partnerServiceGetPartner,
  partnerServiceUpdatePartner,
} from '@/services/roncin/partnerService';
import PartnerDetailPage from './partner-detail';

const state = vi.hoisted(() => ({
  id: 'create',
  pathname: '/partners/customers/create',
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
      currentUser: { currentOrganization: { baseCurrency: 'CNY' } },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateBusiness: true,
    canOperateOrganization: () => true,
    canCreatePartners: true,
    canUpdatePartners: true,
    canManagePartners: false,
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
vi.mock('./components/SettlementSection', () => ({ default: () => null }));
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
  vi.mocked(partnerServiceGetPartner)
    .mockReset()
    .mockResolvedValue({
      data: { id: ID, legalName: '原名称', organizationId: 'org' },
    });
  vi.mocked(partnerServiceCreatePartner).mockReset();
  vi.mocked(partnerServiceUpdatePartner).mockReset();
});
it.each([undefined, false, true])(
  '客商 enabled=%s 编辑保存不改变启停',
  async (enabled) => {
    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: ID,
        legalName: '状态回归',
        organizationId: 'org',
        ...(enabled === undefined ? {} : { enabled }),
      },
    });
    vi.mocked(partnerServiceUpdatePartner).mockResolvedValue({
      data: { id: ID },
    });
    renderPage(true);
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('状态回归'),
    );
    fireEvent.change(screen.getByLabelText('公司抬头'), {
      target: { value: '只修改名称' },
    });
    await save();
    await waitFor(() =>
      expect(partnerServiceUpdatePartner).toHaveBeenCalledWith(
        { id: ID },
        expect.objectContaining({
          legalName: '只修改名称',
          enabled: enabled === true,
        }),
      ),
    );
  },
);
