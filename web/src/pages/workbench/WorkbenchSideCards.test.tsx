import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AccountBoundaryCard, TodosCard } from './WorkbenchSideCards';

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

describe('工作台待办摘要入口', () => {
  const user: API.CurrentUser = {
    currentOrganization: { kind: 2 },
    permissionCapabilities: [
      'business.order.se.read',
      'business.order.se.fee.read',
      'business.order.se.abnormal_case.read',
    ].map((key) => ({ key, dataScope: 'organization' })),
  };

  it('分别打开费用和异常明细，笔数与件数保持独立', () => {
    const onOpen = vi.fn();
    renderWithApp(
      <TodosCard
        todos={{ unbilledFeeCount: 2, openAbnormalCount: 3 }}
        onOpen={onOpen}
      />,
      { user },
    );
    expect(screen.getByText('2 笔')).toBeInTheDocument();
    expect(screen.getByText('3 件')).toBeInTheDocument();
    const buttons = screen.getAllByRole('button', { name: '查看明细' });
    fireEvent.click(buttons[0]);
    expect(onOpen).toHaveBeenLastCalledWith('fees');
    fireEvent.click(buttons[1]);
    expect(onOpen).toHaveBeenLastCalledWith('abnormal');
  });

  it('只有审核权限无法看到费用计数或入口，无异常读取权限也隐藏异常', () => {
    renderWithApp(
      <TodosCard
        todos={{ unbilledFeeCount: 99, openAbnormalCount: 88 }}
        onOpen={vi.fn()}
      />,
      {
        user: {
          ...user,
          permissionCapabilities: [
            'business.order.se.read',
            'business.order.se.lock',
          ].map((key) => ({ key, dataScope: 'organization' })),
        },
      },
    );
    expect(screen.queryByTestId('todos-card')).not.toBeInTheDocument();
    expect(screen.queryByText('99 笔')).not.toBeInTheDocument();
    expect(screen.queryByText('88 件')).not.toBeInTheDocument();
  });

  it('两类待办均为空时隐藏卡片', () => {
    renderWithApp(<TodosCard todos={{}} onOpen={vi.fn()} />, { user });
    expect(screen.queryByTestId('todos-card')).not.toBeInTheDocument();
  });
});
