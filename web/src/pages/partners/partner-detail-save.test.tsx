import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App, Form, Input } from 'antd';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { isTabDirty } from '@/components/layout/tabCloseGuard';
import { history } from '@/router/history';
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
describe('客商保存与失败恢复', () => {
  it.each([false, true])(
    '旧保存成功不得影响新身份的输入和dirty（是否往返原档案：%s）',
    async (returnToOriginal) => {
      const otherId = '22222222-2222-4222-8222-222222222222';
      let resolveSave:
        | ((response: API.UpdatePartnerResponse) => void)
        | undefined;
      vi.mocked(partnerServiceGetPartner).mockImplementation(
        async ({ id }) => ({
          data: { id, legalName: id === ID ? '原档案' : '另一个档案' },
        }),
      );
      vi.mocked(partnerServiceUpdatePartner).mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveSave = resolve;
          }),
      );
      const { rerender, queryClient } = renderPage(true);
      await waitFor(() =>
        expect(screen.getByLabelText('公司抬头')).toHaveValue('原档案'),
      );
      fireEvent.change(screen.getByLabelText('公司抬头'), {
        target: { value: '旧请求提交值' },
      });
      await save();
      await waitFor(() => expect(resolveSave).toBeDefined());
      const navigateTo = async (id: string, name: string) => {
        state.id = id;
        state.pathname = `/partners/customers/${id}`;
        rerender(
          <QueryClientProvider client={queryClient}>
            <App>
              <PartnerDetailPage />
            </App>
          </QueryClientProvider>,
        );
        await waitFor(() =>
          expect(screen.getByLabelText('公司抬头')).toHaveValue(name),
        );
      };
      await navigateTo(otherId, '另一个档案');
      if (returnToOriginal) await navigateTo(ID, '原档案');
      fireEvent.change(screen.getByLabelText('公司抬头'), {
        target: { value: '当前身份未保存值' },
      });
      await act(async () => {
        resolveSave?.({ data: { id: ID } });
      });
      expect(screen.getByLabelText('公司抬头')).toHaveValue('当前身份未保存值');
      expect(isTabDirty('/partners/customers')).toBe(true);
      expect(history.push).not.toHaveBeenCalled();
      expect(screen.queryByText('客户档案已成功更新')).not.toBeInTheDocument();
    },
  );

  it('创建返回undefined及缺ID不导航、不报成功、保留输入和dirty，真实成功后清dirty', async () => {
    vi.mocked(partnerServiceCreatePartner)
      .mockResolvedValueOnce(undefined as never)
      .mockResolvedValueOnce({ data: {} })
      .mockResolvedValueOnce({ success: false, data: { id: ID } })
      .mockResolvedValueOnce({ data: { id: ID } });
    renderPage();
    fireEvent.change(await screen.findByLabelText('公司抬头'), {
      target: { value: '待保存客户' },
    });
    for (let n = 1; n <= 3; n++) {
      await save();
      await waitFor(() =>
        expect(partnerServiceCreatePartner).toHaveBeenCalledTimes(n),
      );
      if (n > 1) await screen.findAllByText('保存失败，请重试');
      await waitFor(() =>
        expect(screen.queryByText('保存中...')).not.toBeInTheDocument(),
      );
      expect(history.push).not.toHaveBeenCalled();
      expect(screen.queryByText('客户档案已成功创建')).not.toBeInTheDocument();
      expect(screen.getByLabelText('公司抬头')).toHaveValue('待保存客户');
      expect(isTabDirty('/partners/customers')).toBe(true);
    }
    await save();
    await waitFor(() =>
      expect(history.push).toHaveBeenCalledWith(`/partners/customers/${ID}`),
    );
    expect(isTabDirty('/partners/customers')).toBe(false);
  });

  it('更新reject与undefined不导航、不报成功，重试成功正常返回列表', async () => {
    vi.mocked(partnerServiceUpdatePartner)
      .mockRejectedValueOnce(new Error('模拟更新失败'))
      .mockResolvedValueOnce(undefined as never)
      .mockResolvedValueOnce({ data: { id: ID } });
    renderPage(true);
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('原名称'),
    );
    fireEvent.change(screen.getByLabelText('公司抬头'), {
      target: { value: '修改后的名称' },
    });
    for (let n = 1; n <= 2; n++) {
      await save();
      await waitFor(() =>
        expect(partnerServiceUpdatePartner).toHaveBeenCalledTimes(n),
      );
      if (n === 1) await screen.findAllByText('模拟更新失败');
      await waitFor(() =>
        expect(screen.queryByText('保存中...')).not.toBeInTheDocument(),
      );
      expect(history.push).not.toHaveBeenCalled();
      expect(isTabDirty('/partners/customers')).toBe(true);
      expect(screen.getByLabelText('公司抬头')).toHaveValue('修改后的名称');
    }
    await save();
    await waitFor(() =>
      expect(history.push).toHaveBeenCalledWith('/partners/customers'),
    );
    expect(isTabDirty('/partners/customers')).toBe(false);
  });

  it('首次GET失败显示恢复页，重试成功才显示档案', async () => {
    vi.mocked(partnerServiceGetPartner)
      .mockResolvedValueOnce(undefined as never)
      .mockResolvedValueOnce({ data: { id: ID, legalName: '恢复后的名称' } });
    renderPage(true);
    await screen.findByText('加载档案详情失败');
    expect(screen.queryByLabelText('公司抬头')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /保存客户档案/ }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('恢复后的名称'),
    );
  });

  it('后台重查成功或失败均不覆盖已有未保存输入', async () => {
    const { queryClient } = renderPage(true);
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('原名称'),
    );
    fireEvent.change(screen.getByLabelText('公司抬头'), {
      target: { value: '未保存新值' },
    });
    vi.mocked(partnerServiceGetPartner)
      .mockResolvedValueOnce({ data: { id: ID, legalName: '新的服务器值' } })
      .mockRejectedValueOnce(new Error('重查失败'));
    await act(async () => {
      await queryClient.refetchQueries({
        queryKey: ['partner-detail', 'profile'],
      });
    });
    expect(screen.getByLabelText('公司抬头')).toHaveValue('未保存新值');
    await act(async () => {
      await queryClient.refetchQueries({
        queryKey: ['partner-detail', 'profile'],
      });
    });
    expect(screen.getByLabelText('公司抬头')).toHaveValue('未保存新值');
  });

  it('联系人和未提交别名输入均计入dirty', async () => {
    const { unmount } = renderPage();
    await screen.findByLabelText('公司抬头');
    expect(isTabDirty('/partners/customers')).toBe(false);
    fireEvent.click(screen.getByText('添加联系人'));
    expect(isTabDirty('/partners/customers')).toBe(true);
    unmount();
    renderPage();
    fireEvent.change(await screen.findByLabelText('新增别名'), {
      target: { value: '别名' },
    });
    expect(isTabDirty('/partners/customers')).toBe(true);
  });
});
