import type { ProColumns } from '@ant-design/pro-components';
import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App, Table } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const accessState = vi.hoisted(() => ({
  value: {
    isSystemWorkspace: true,
    canReadUserMemberships: true,
  } as Record<string, boolean>,
}));

vi.mock('@/app/access', () => ({ useAccess: () => accessState.value }));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({ initialState: { currentUser: { id: 'user-1' } } }),
}));
vi.mock('@/components/ui', () => ({ SearchFilterTemplate: () => null }));
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListUsers: vi.fn().mockResolvedValue({ data: [], total: 0 }),
  adminServiceListRoles: vi.fn().mockResolvedValue({ data: [] }),
  adminServiceListOrganizations: vi.fn().mockResolvedValue({ data: [] }),
  adminServiceTerminateUser: vi.fn(),
}));
vi.mock('./components/users/UserFormModal', () => ({ default: () => null }));
vi.mock('./components/users/ResetPasswordModal', () => ({
  default: () => null,
}));
vi.mock('./components/users/DingTalkAssignmentModal', () => ({
  default: () => null,
}));
vi.mock('./components/users/DingTalkDirectorySyncPanel', () => ({
  default: () => null,
}));
// 保留真实列配置、列偏好和列设置入口，只用轻量表格替代 ProTable 的请求引擎。
vi.mock('@ant-design/pro-components', () => ({
  ProTable: ({
    columns,
    toolBarRender,
  }: {
    columns: ProColumns<API.AdminUser>[];
    toolBarRender: () => React.ReactNode[];
  }) => (
    <>
      {toolBarRender()}
      <Table
        columns={columns
          .filter((column) => !column.hideInTable)
          .map((column) => ({
            key: String(column.key ?? column.dataIndex),
            title:
              typeof column.title === 'function' ? undefined : column.title,
          }))}
        dataSource={[]}
        pagination={false}
      />
    </>
  ),
}));

import UsersPanel from './users';

function renderPanel() {
  return renderWithClient(
    <App>
      <UsersPanel />
    </App>,
  );
}
function columnTitles() {
  return screen
    .getAllByRole('columnheader')
    .map((column) => column.textContent);
}
async function changeTab(name: string) {
  await act(async () => {
    fireEvent.click(screen.getByRole('tab', { name }));
  });
}
async function openSettings() {
  fireEvent.click(screen.getByRole('button', { name: '列设置' }));
  await screen.findByPlaceholderText('搜索列名');
}

beforeEach(() => {
  accessState.value = {
    isSystemWorkspace: true,
    canReadUserMemberships: true,
  };
  window.localStorage.clear();
});
afterEach(() => window.localStorage.clear());

describe('人员页签所属组织列与已保存列设置', () => {
  it('待分配页签删除组织列及其设置项，其他页签保留组织和钉钉部门', async () => {
    window.localStorage.setItem(
      'roncin:column-settings:v1:admin:users:anonymous:default',
      JSON.stringify({
        order: ['organizations', 'dingtalkDepartments'],
        hidden: [],
      }),
    );
    renderPanel();
    expect(columnTitles()).not.toContain('所属组织');
    expect(columnTitles()).toContain('钉钉部门');
    await openSettings();
    expect(
      screen.queryByRole('checkbox', { name: '所属组织' }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '列设置' }));

    await changeTab('已开通');
    expect(columnTitles()).toContain('所属组织');
    expect(columnTitles()).toContain('钉钉部门');
    await changeTab('离职 / 停用');
    expect(columnTitles()).toContain('所属组织');
    await changeTab('待分配');
    expect(columnTitles()).not.toContain('所属组织');
  });

  it('保存隐藏偏好后切换待分配再返回，组织列保持用户选择的隐藏状态', async () => {
    renderPanel();
    await changeTab('已开通');
    await openSettings();
    fireEvent.click(screen.getByRole('checkbox', { name: '所属组织' }));
    expect(columnTitles()).not.toContain('所属组织');
    fireEvent.click(screen.getByRole('button', { name: '列设置' }));
    await changeTab('待分配');
    expect(columnTitles()).not.toContain('所属组织');
    await changeTab('已开通');
    expect(columnTitles()).not.toContain('所属组织');
    await openSettings();
    fireEvent.click(screen.getByRole('checkbox', { name: '所属组织' }));
    await waitFor(() => expect(columnTitles()).toContain('所属组织'));
  });

  it('缺少成员读取权限时所有页签均不展示组织列，列偏好不能扩大权限', async () => {
    accessState.value.canReadUserMemberships = false;
    window.localStorage.setItem(
      'roncin:column-settings:v1:admin:users:anonymous:default',
      JSON.stringify({ order: ['organizations'], hidden: [] }),
    );
    renderPanel();
    expect(columnTitles()).not.toContain('所属组织');
    await changeTab('已开通');
    expect(columnTitles()).not.toContain('所属组织');
    await changeTab('离职 / 停用');
    expect(columnTitles()).not.toContain('所属组织');
    expect(columnTitles()).toContain('钉钉部门');
  });

  it.each([true, false])(
    '公司页签沿用成员读取权限（%s）且保留钉钉部门',
    async (canReadUserMemberships) => {
      accessState.value = { isSystemWorkspace: false, canReadUserMemberships };
      renderPanel();
      expect(
        screen.queryByRole('tab', { name: '待分配' }),
      ).not.toBeInTheDocument();
      expect(columnTitles().includes('所属组织')).toBe(canReadUserMemberships);
      expect(columnTitles()).toContain('钉钉部门');
      await changeTab('离职 / 停用');
      expect(columnTitles().includes('所属组织')).toBe(canReadUserMemberships);
      expect(columnTitles()).toContain('钉钉部门');
    },
  );
});
