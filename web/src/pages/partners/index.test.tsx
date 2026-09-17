import { fireEvent, render, screen } from '@testing-library/react';
import { history } from '@umijs/max';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  partnerServiceListPartners,
  partnerServiceSetPartnerRoleBlacklist,
} from '@/services/roncin/partnerService';
import Partners from './index';

const routeState = vi.hoisted(() => ({
  pathname: '/partners/customers',
}));

vi.mock('@umijs/max', () => ({
  history: { push: vi.fn() },
  useLocation: () => ({ pathname: routeState.pathname }),
  useAccess: () => ({
    canManagePartners: true,
  }),
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartners: vi.fn(),
  partnerServiceExportPartners: vi.fn(),
  partnerServiceImportPartners: vi.fn(),
  partnerServiceSetPartnerRoleBlacklist: vi.fn(),
}));

describe('Partners 列表页', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.pathname = '/partners/customers';
  });

  it('表格渲染合作类型列与快捷筛选，并正确展示散客与正式标签', async () => {
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [
        {
          id: 'p-1',
          code: 'CUST-CASUAL',
          legalName: '单次合作散客公司',
          isCasual: true,
          enabled: true,
          roles: [{ type: 1, enabled: true }],
        },
        {
          id: 'p-2',
          code: 'CUST-REGULAR',
          legalName: '长期正式合作公司',
          isCasual: false,
          enabled: true,
          roles: [{ type: 1, enabled: true }],
        },
      ],
      total: 2,
    } as never);

    render(
      <App>
        <Partners />
      </App>,
    );

    expect(await screen.findByText('单次合作散客公司')).toBeInTheDocument();
    expect(screen.getByText('长期正式合作公司')).toBeInTheDocument();

    // 合作类型表头
    expect(
      screen.getByRole('columnheader', { name: '合作类型' }),
    ).toBeInTheDocument();

    // 行内散客与正式 Tag
    expect(screen.getByText('散客')).toBeInTheDocument();
    expect(screen.getByText('正式')).toBeInTheDocument();

    // 快捷筛选包含合作类型
    const placeholders = screen.getAllByText('合作类型');
    expect(placeholders.length).toBeGreaterThanOrEqual(1);
  });

  it('国外代理列表页隐藏合作类型列与散客筛选', async () => {
    routeState.pathname = '/partners/foreign-agents';
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [
        {
          id: 'fa-1',
          code: 'AGT-001',
          legalName: 'Apex Global Logistics Ltd.',
          enabled: true,
          roles: [{ type: 3, enabled: true }],
        },
      ],
      total: 1,
    } as never);

    render(
      <App>
        <Partners />
      </App>,
    );

    expect(
      await screen.findByText('Apex Global Logistics Ltd.'),
    ).toBeInTheDocument();

    // 国外代理不应有合作类型表头与快捷筛选
    expect(
      screen.queryByRole('columnheader', { name: '合作类型' }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('合作类型')).not.toBeInTheDocument();
  });

  it('列表页展示导出 Excel 与导入 Excel 按钮，并在操作列提供编辑与更多操作菜单', async () => {
    routeState.pathname = '/partners/customers';
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [
        {
          id: 'p-1',
          code: 'CUST-001',
          legalName: '测试转换角色公司',
          enabled: true,
          roles: [{ type: 1, enabled: true }],
        },
      ],
      total: 1,
    } as never);

    render(
      <App>
        <Partners />
      </App>,
    );

    expect(await screen.findByText('测试转换角色公司')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /导出 Excel/ }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /导入 Excel/ }),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /编辑/ })).toBeInTheDocument();

    const moreBtn = screen.getByRole('button', { name: /更多/ });
    expect(moreBtn).toBeInTheDocument();
    fireEvent.click(moreBtn);

    expect(await screen.findByText('转角色')).toBeInTheDocument();
    expect(screen.getByText('账户/合同')).toBeInTheDocument();
  });

  it('客户、供应商、国外代理列表点击新建分别跳转到对应的 /create 地址', async () => {
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [],
      total: 0,
    } as never);

    // 1. 客户列表页 -> /partners/customers/create
    routeState.pathname = '/partners/customers';
    const { unmount: unmountCust } = render(
      <App>
        <Partners />
      </App>,
    );
    const createCustBtn = await screen.findByRole('button', {
      name: /新增客户/,
    });
    fireEvent.click(createCustBtn);
    expect(history.push).toHaveBeenCalledWith('/partners/customers/create');
    unmountCust();

    // 2. 供应商列表页 -> /partners/suppliers/create
    routeState.pathname = '/partners/suppliers';
    const { unmount: unmountSupp } = render(
      <App>
        <Partners />
      </App>,
    );
    const createSuppBtn = await screen.findByRole('button', {
      name: /新增供应商/,
    });
    fireEvent.click(createSuppBtn);
    expect(history.push).toHaveBeenCalledWith('/partners/suppliers/create');
    unmountSupp();

    // 3. 国外代理列表页 -> /partners/foreign-agents/create
    routeState.pathname = '/partners/foreign-agents';
    const { unmount: unmountAgent } = render(
      <App>
        <Partners />
      </App>,
    );
    const createAgentBtn = await screen.findByRole('button', {
      name: /新增国外代理/,
    });
    fireEvent.click(createAgentBtn);
    expect(history.push).toHaveBeenCalledWith(
      '/partners/foreign-agents/create',
    );
    unmountAgent();
  });

  it('黑名单弹窗回退到档案已有角色并提交该角色当前状态', async () => {
    vi.mocked(partnerServiceListPartners).mockResolvedValue({
      data: [
        {
          id: 'p-blacklist',
          code: 'PARTNER-001',
          legalName: '多角色合作单位',
          enabled: true,
          roles: [{ type: 2, enabled: true, blacklisted: true }],
        },
      ],
      total: 1,
    } as never);
    vi.mocked(partnerServiceSetPartnerRoleBlacklist).mockResolvedValue({
      success: true,
    });

    render(
      <App>
        <Partners />
      </App>,
    );

    expect(await screen.findByText('多角色合作单位')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /更多/ }));
    fireEvent.click(await screen.findByText('黑名单'));

    expect(
      await screen.findByText('角色黑名单管理 - 多角色合作单位'),
    ).toBeInTheDocument();
    expect(screen.getByRole('switch')).toBeChecked();

    fireEvent.change(screen.getByLabelText('变更原因与说明'), {
      target: { value: '  延续供应商限制  ' },
    });
    fireEvent.click(screen.getByRole('button', { name: '确 定' }));

    await vi.waitFor(() => {
      expect(partnerServiceSetPartnerRoleBlacklist).toHaveBeenCalledWith(
        { id: 'p-blacklist' },
        {
          id: 'p-blacklist',
          roleType: 2,
          blacklisted: true,
          reason: '延续供应商限制',
        },
      );
    });
  });
});
