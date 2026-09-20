import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { history } from '@/router/history';
import {
  partnerServiceGetPartner,
  partnerServiceListPartnerAuditLogs,
  partnerServiceListPartnerContracts,
  partnerServiceListPartnerSettlementRules,
  partnerServiceListPartnerShippingPresets,
} from '@/services/roncin/partnerService';
import PartnerDetailPage from './partner-detail';

const VALID_UUID_1 = '11111111-1111-4111-8111-111111111111';
const VALID_UUID_CASUAL = '22222222-2222-4222-8222-222222222222';

const routeState = vi.hoisted(() => ({
  params: { id: 'create' } as { id?: string },
  pathname: '/partners/customers/create',
  search: '?legalName=%E4%B8%8A%E6%B5%B7%20%E5%AE%89%E5%8F%AF',
}));

const accessState = vi.hoisted(() => ({
  canOperateOrganization: vi.fn(() => true),
  canOperateBusiness: true,
  canManagePartners: false,
  canReadPartners: true,
  canCreatePartners: true,
  canUpdatePartners: true,
  canReadPartnerAccounts: true,
  canCreatePartnerAccounts: true,
  canUpdatePartnerAccounts: true,
  canReadPartnerContracts: true,
  canCreatePartnerContracts: true,
  canUpdatePartnerContracts: true,
  canReadPartnerSettlementRules: true,
  canCreatePartnerSettlementRules: true,
  canUpdatePartnerSettlementRules: true,
  canReadPartnerShippingPresets: true,
  canCreatePartnerShippingPresets: true,
  canUpdatePartnerShippingPresets: true,
  canReadPartnerAudit: true,
}));

vi.mock('@/router/history', () => ({
  history: { push: vi.fn(), replace: vi.fn() },
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState,
}));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
    useLocation: () => ({
      pathname: routeState.pathname,
      search: routeState.search,
    }),
    useParams: () => routeState.params,
    useSearchParams: () => [new URLSearchParams(routeState.search)],
  };
});

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListOrganizations: vi.fn().mockResolvedValue({ data: [] }),
  adminServiceListUsers: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceCreatePartner: vi.fn(),
  partnerServiceGetPartner: vi.fn(),
  partnerServiceListPartnerAccounts: vi.fn().mockResolvedValue({ data: [] }),
  partnerServiceListPartnerAssignmentOptions: vi
    .fn()
    .mockResolvedValue({ data: [] }),
  partnerServiceListPartnerSettlementRules: vi.fn(),
  partnerServiceListPartnerContracts: vi.fn().mockResolvedValue({ data: [] }),
  partnerServiceListPartnerShippingPresets: vi
    .fn()
    .mockResolvedValue({ data: [] }),
  partnerServiceListPartnerAuditLogs: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0 }),
  partnerServiceUpdatePartner: vi.fn(),
}));

vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));

describe('PartnerDetailPage', () => {
  beforeEach(() => {
    accessState.canOperateOrganization.mockReturnValue(true);
    vi.clearAllMocks();
    routeState.params = { id: 'create' };
    routeState.pathname = '/partners/customers/create';
    routeState.search = '?legalName=%E4%B8%8A%E6%B5%B7%20%E5%AE%89%E5%8F%AF';

    // 默认具备完整权限
    Object.assign(accessState, {
      canManagePartners: false,
      canReadPartners: true,
      canCreatePartners: true,
      canUpdatePartners: true,
      canReadPartnerAccounts: true,
      canCreatePartnerAccounts: true,
      canUpdatePartnerAccounts: true,
      canReadPartnerContracts: true,
      canCreatePartnerContracts: true,
      canUpdatePartnerContracts: true,
      canReadPartnerSettlementRules: true,
      canCreatePartnerSettlementRules: true,
      canUpdatePartnerSettlementRules: true,
      canReadPartnerShippingPresets: true,
      canCreatePartnerShippingPresets: true,
      canUpdatePartnerShippingPresets: true,
      canReadPartnerAudit: true,
    });

    vi.mocked(partnerServiceGetPartner).mockReset();
    vi.mocked(partnerServiceListPartnerSettlementRules).mockReset();
    vi.mocked(partnerServiceListPartnerContracts).mockClear();
    vi.mocked(partnerServiceListPartnerShippingPresets).mockClear();
    vi.mocked(partnerServiceListPartnerAuditLogs).mockClear();
  });

  it('创建模式按当前查询参数预填，并在同组件地址变化时重建默认值', async () => {
    const { rerender, queryClient } = renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    expect(await screen.findByLabelText('公司抬头')).toHaveValue('上海 安可');

    // rerender 需自带同一 QueryClientProvider，保持同一组件树与查询缓存
    const rerenderWithClient = () =>
      rerender(
        <QueryClientProvider client={queryClient}>
          <App>
            <PartnerDetailPage />
          </App>
        </QueryClientProvider>,
      );

    routeState.search = '?legalName=%E6%96%B0%26%E5%85%AC%E5%8F%B8';
    rerenderWithClient();
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('新&公司'),
    );

    routeState.search = '';
    rerenderWithClient();
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue(''),
    );
    expect(partnerServiceGetPartner).not.toHaveBeenCalled();
  });

  it('创建态绝不发起档案详情、结算规则、合同、单证预设、操作日志请求', async () => {
    routeState.params = { id: 'create' };
    routeState.pathname = '/partners/customers/create';
    routeState.search = '';

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    expect(await screen.findByLabelText('公司抬头')).toBeInTheDocument();

    // 断言 5 个查询接口均未被调用
    expect(partnerServiceGetPartner).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerSettlementRules).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerContracts).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerShippingPresets).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerAuditLogs).not.toHaveBeenCalled();
  });

  it('动态路由中遇到非法 UUID（如 /new 或拼写错误）时，阻断所有请求并重定向回列表', async () => {
    routeState.params = { id: 'new' };
    routeState.pathname = '/partners/customers/new';
    routeState.search = '';

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    await waitFor(() => {
      expect(history.replace).toHaveBeenCalledWith('/partners/customers');
    });

    expect(partnerServiceGetPartner).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerSettlementRules).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerContracts).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerShippingPresets).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerAuditLogs).not.toHaveBeenCalled();
  });

  it('编辑模式忽略 legalName 查询参数并使用服务端档案名称', async () => {
    routeState.params = { id: VALID_UUID_1 };
    routeState.pathname = `/partners/customers/${VALID_UUID_1}`;
    routeState.search = '?legalName=%E4%B8%8D%E5%BA%94%E8%A6%86%E7%9B%96';
    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: VALID_UUID_1,
        legalName: '服务端伙伴名称',
        enabled: true,
      } as never,
    });
    vi.mocked(partnerServiceListPartnerSettlementRules).mockResolvedValue({
      data: [],
    } as never);

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('服务端伙伴名称'),
    );
    expect(screen.getByLabelText('公司抬头')).not.toHaveValue('不应覆盖');
    expect(partnerServiceGetPartner).toHaveBeenCalledWith({ id: VALID_UUID_1 });
  });

  it('编辑散客档案时页头展示散客标签且单次合作开关开启', async () => {
    routeState.params = { id: VALID_UUID_CASUAL };
    routeState.pathname = `/partners/customers/${VALID_UUID_CASUAL}`;
    routeState.search = '';
    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: VALID_UUID_CASUAL,
        code: 'P00000001',
        legalName: '单次合作散客公司',
        enabled: true,
        isCasual: true,
      } as never,
    });
    vi.mocked(partnerServiceListPartnerSettlementRules).mockResolvedValue({
      data: [],
    } as never);

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('单次合作散客公司'),
    );
    const headerTags = document.querySelector('.roncin-page-header-tags');
    expect(headerTags).toHaveTextContent('散客');
    expect(
      screen.getByRole('switch', { name: '单次合作 (散客)' }),
    ).toBeChecked();
  });

  it('低权限账号打开档案详情时只请求基本档案，不请求无权限的结算规则、合同、单证预设及操作日志', async () => {
    // 降级为仅有档案读权限，没有结算规则、合同、预设和审计日志读权限
    Object.assign(accessState, {
      canReadPartners: true,
      canReadPartnerSettlementRules: false,
      canReadPartnerContracts: false,
      canReadPartnerShippingPresets: false,
      canReadPartnerAudit: false,
      canReadPartnerAccounts: false,
    });

    routeState.params = { id: VALID_UUID_1 };
    routeState.pathname = `/partners/customers/${VALID_UUID_1}`;
    routeState.search = '';

    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: VALID_UUID_1,
        legalName: '低权限档案查看测试',
        enabled: true,
      } as never,
    });

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue(
        '低权限档案查看测试',
      ),
    );

    // 档案被调用，但子权限接口均未被调用
    expect(partnerServiceGetPartner).toHaveBeenCalledWith({ id: VALID_UUID_1 });
    expect(partnerServiceListPartnerSettlementRules).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerContracts).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerShippingPresets).not.toHaveBeenCalled();
    expect(partnerServiceListPartnerAuditLogs).not.toHaveBeenCalled();

    // 页面不应展示无权限的区块
    expect(screen.queryByText('合同管理')).not.toBeInTheDocument();
    expect(screen.queryByText('常用信息')).not.toBeInTheDocument();
    expect(screen.queryByText('操作记录')).not.toBeInTheDocument();
  });

  it('创建模式下无创建权限时不展示保存按钮', async () => {
    accessState.canCreatePartners = false;
    routeState.params = { id: 'create' };
    routeState.pathname = '/partners/customers/create';

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    expect(await screen.findByLabelText('公司抬头')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /保存客户档案/ }),
    ).not.toBeInTheDocument();
  });

  it('编辑模式下无更新权限时不展示保存按钮', async () => {
    accessState.canUpdatePartners = false;
    routeState.params = { id: VALID_UUID_1 };
    routeState.pathname = `/partners/customers/${VALID_UUID_1}`;
    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: VALID_UUID_1,
        legalName: '测试权限公司',
        enabled: true,
      } as never,
    });

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('测试权限公司'),
    );
    expect(
      screen.queryByRole('button', { name: /保存客户档案/ }),
    ).not.toBeInTheDocument();
  });

  it('编辑模式下具备更新权限时展示保存按钮', async () => {
    accessState.canUpdatePartners = true;
    routeState.params = { id: VALID_UUID_1 };
    routeState.pathname = `/partners/customers/${VALID_UUID_1}`;
    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: VALID_UUID_1,
        legalName: '测试权限公司',
        enabled: true,
      } as never,
    });

    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('测试权限公司'),
    );
    const saveButtons = screen.getAllByRole('button', { name: /保存客户档案/ });
    expect(saveButtons.length).toBeGreaterThanOrEqual(1);
  });
  it('跨公司档案即使具备更新与管理权限也只读，仍展示档案内容', async () => {
    accessState.canOperateOrganization.mockReturnValue(false);
    accessState.canManagePartners = true;
    routeState.params = { id: VALID_UUID_1 };
    routeState.pathname = `/partners/customers/${VALID_UUID_1}`;
    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: {
        id: VALID_UUID_1,
        organizationId: 'other-company',
        legalName: '其他公司客户',
        enabled: true,
      } as never,
    });
    renderWithClient(
      <App>
        <PartnerDetailPage />
      </App>,
    );
    await waitFor(() =>
      expect(screen.getByLabelText('公司抬头')).toHaveValue('其他公司客户'),
    );
    expect(screen.getByLabelText('公司抬头')).toBeDisabled();
    expect(
      screen.queryByRole('button', { name: /保存客户档案/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /新增账户|新增合同/ }),
    ).not.toBeInTheDocument();
    expect(accessState.canOperateOrganization).toHaveBeenCalledWith(
      'other-company',
    );
  });
});
