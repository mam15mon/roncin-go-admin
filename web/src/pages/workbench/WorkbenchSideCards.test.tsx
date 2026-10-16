import { renderWithApp } from '@root/tests/renderWithApp';
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { AccountBoundaryCard } from './WorkbenchSideCards';

describe('工作台账号与角色摘要', () => {
  it('保留角色名称与当前组织，不显示已退役的数据范围', () => {
    renderWithApp(
      <AccountBoundaryCard
        user={{
          username: 'operator',
          currentOrganization: { name: '上海公司', code: 'SH01' },
          roleScopes: [
            { roleCode: 'operator', roleName: '操作员' },
            { roleCode: 'reviewer' },
          ],
          permissions: ['system.platform.access'],
        }}
      />,
    );

    expect(screen.getByText('上海公司')).toBeInTheDocument();
    expect(screen.getByText('已分配角色')).toBeInTheDocument();
    expect(screen.getByText('操作员')).toBeInTheDocument();
    expect(screen.getByText('reviewer')).toBeInTheDocument();
    expect(screen.getByText('1 项')).toBeInTheDocument();
    expect(
      screen.queryByText(/数据授权范围|数据范围:/),
    ).not.toBeInTheDocument();
  });

  it('无角色时显示明确空态', () => {
    renderWithApp(<AccountBoundaryCard user={{ roleScopes: [] }} />);
    expect(screen.getByText('暂无已分配角色')).toBeInTheDocument();
  });
});
