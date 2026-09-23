import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const listOrganizations = vi.hoisted(() => vi.fn());

type ChartNode = { title: string; children?: ChartNode[] };

function flattenTitles(nodes: ChartNode[]): string[] {
  return nodes.flatMap((node) => [
    node.title,
    ...flattenTitles(node.children ?? []),
  ]);
}

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    isSystemWorkspace: true,
    canCreateOrganizations: true,
    canUpdateOrganizations: true,
  }),
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListOrganizations: listOrganizations,
}));

vi.mock('./components/org/OrgChartCanvas', () => ({
  default: ({
    treeData,
    selectedOrg,
  }: {
    treeData: ChartNode[];
    selectedOrg: API.AdminOrganization | null;
  }) => (
    <div data-testid="org-chart">
      {flattenTitles(treeData).join(',')}
      <span data-testid="chart-selection">{selectedOrg?.name ?? '未选择'}</span>
    </div>
  ),
}));

vi.mock('./components/org/OrgDetailCard', () => ({
  default: ({ selectedOrg }: { selectedOrg: API.AdminOrganization | null }) => (
    <div data-testid="org-detail">{selectedOrg?.name ?? '未选择'}</div>
  ),
}));

vi.mock('./components/org/OrgCreateModal', () => ({ default: () => null }));
vi.mock('./components/org/OrgEditModal', () => ({ default: () => null }));

import OrganizationsPanel from './organizations';

const organizations: API.AdminOrganization[] = [
  { id: 'system', name: '系统管理', code: 'SYS', kind: 1, enabled: true },
  {
    id: 'company',
    name: '上海公司',
    code: 'SH',
    kind: 2,
    enabled: true,
  },
  {
    id: 'department',
    name: '操作部',
    code: 'OPS',
    kind: 3,
    parentId: 'company',
    enabled: true,
  },
  {
    id: 'team',
    name: '一组',
    code: 'TEAM',
    kind: 4,
    parentId: 'department',
    enabled: true,
  },
];

describe('OrganizationsPanel', () => {
  beforeEach(() => {
    listOrganizations.mockReset();
    listOrganizations.mockResolvedValue({ data: organizations });
  });

  it('拓扑图和树表均只展示经营组织，并默认选中公司', async () => {
    renderWithApp(<OrganizationsPanel />);

    const chart = await screen.findByTestId('org-chart');
    await waitFor(() => {
      expect(within(chart).getByTestId('chart-selection')).toHaveTextContent(
        '上海公司',
      );
    });
    expect(chart).toHaveTextContent('上海公司,操作部,一组');
    expect(chart).not.toHaveTextContent('系统管理');
    expect(screen.getByText('共 3 个组织节点')).toBeInTheDocument();

    fireEvent.click(screen.getByText('树表列表'));
    expect(screen.getByText('3 个节点')).toBeInTheDocument();
    const tree = screen.getByRole('tree');
    expect(within(tree).getByText('上海公司')).toBeInTheDocument();
    expect(within(tree).getByText('操作部')).toBeInTheDocument();
    expect(within(tree).getByText('一组')).toBeInTheDocument();
    expect(within(tree).queryByText('系统管理')).not.toBeInTheDocument();
    expect(screen.getByTestId('org-detail')).toHaveTextContent('上海公司');

    fireEvent.change(screen.getByPlaceholderText('搜索组织名称或编码...'), {
      target: { value: 'SYS' },
    });
    expect(screen.getByText('未找到匹配的组织机构')).toBeInTheDocument();
    expect(screen.getByTestId('org-detail')).toHaveTextContent('上海公司');

    fireEvent.change(screen.getByPlaceholderText('搜索组织名称或编码...'), {
      target: { value: '系统管理' },
    });
    expect(screen.getByText('未找到匹配的组织机构')).toBeInTheDocument();

    listOrganizations.mockResolvedValue({ data: organizations.slice(0, 1) });
    fireEvent.click(screen.getByText('刷新数据'));
    await waitFor(() => {
      expect(screen.getByText('共 0 个组织节点')).toBeInTheDocument();
      expect(screen.getByTestId('org-detail')).toHaveTextContent('未选择');
    });
  });

  it('仅返回系统管理时显示空状态且无选中详情', async () => {
    listOrganizations.mockResolvedValue({ data: organizations.slice(0, 1) });
    renderWithApp(<OrganizationsPanel />);

    expect(await screen.findByText('共 0 个组织节点')).toBeInTheDocument();
    expect(screen.getByTestId('chart-selection')).toHaveTextContent('未选择');

    fireEvent.click(screen.getByText('树表列表'));
    expect(screen.getByText('0 个节点')).toBeInTheDocument();
    expect(screen.getByText('暂无组织数据')).toBeInTheDocument();
    expect(screen.getByTestId('org-detail')).toHaveTextContent('未选择');
  });

  it('不展示未知类型的组织节点', async () => {
    listOrganizations.mockResolvedValue({
      data: [...organizations, { id: 'unknown', name: '未知类型', kind: 0 }],
    });
    renderWithApp(<OrganizationsPanel />);

    const chart = await screen.findByTestId('org-chart');
    await waitFor(() => {
      expect(chart).toHaveTextContent('上海公司,操作部,一组');
    });
    expect(chart).not.toHaveTextContent('未知类型');
    expect(screen.getByText('共 3 个组织节点')).toBeInTheDocument();
  });
});
