import { cleanup, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { buildUserColumns } from './userColumns';

type RoleColumnRender = (
  dom: unknown,
  entity: API.AdminUser,
) => ReactNode | undefined;

function getUserRoleColumnRender(
  deps: Parameters<typeof buildUserColumns>[0],
): ((entity: API.AdminUser) => ReactNode | undefined) | undefined {
  const column = buildUserColumns(deps).find(
    (item) => item.title === '已分配角色',
  );
  const columnRender = column?.render as RoleColumnRender | undefined;
  return columnRender ? (entity) => columnRender(null, entity) : undefined;
}

const baseDeps = {
  roles: [],
  canUpdateUsers: true,
  canResetUserPasswords: true,
  canTerminateUsers: true,
  canReadAllUserMemberships: true,
  currentUserId: 'user-1',
  onEdit: vi.fn(),
  onResetPassword: vi.fn(),
  onTerminate: vi.fn(),
};

afterEach(() => {
  cleanup();
});

describe('buildUserColumns 按页签隐藏操作列', () => {
  it('默认保留操作列与展示列', () => {
    const columns = buildUserColumns(baseDeps);
    expect(columns.find((column) => column.title === '操作')).toBeTruthy();
    expect(columns.some((column) => column.title === '已分配角色')).toBe(true);
    expect(columns.some((column) => column.title === '所属组织')).toBe(true);
  });

  it('showActions=false 隐藏操作列，展示列保留', () => {
    const columns = buildUserColumns({ ...baseDeps, showActions: false });
    expect(columns.find((column) => column.title === '操作')).toBeUndefined();
    expect(columns.some((column) => column.title === '已分配角色')).toBe(true);
    expect(columns.some((column) => column.title === '所属组织')).toBe(true);
    expect(columns.some((column) => column.title === '更新时间')).toBe(true);
  });

  it('已分配角色优先展示后端返回的角色名', () => {
    const renderColumn = getUserRoleColumnRender(baseDeps);

    render(
      <>
        {renderColumn?.({
          id: 'user-2',
          roleCodes: ['role_te22ck559e', 'operator'],
          roleNames: ['财务', '调度操作员'],
        } as API.AdminUser)}
      </>,
    );

    expect(screen.getByText('财务')).toBeInTheDocument();
    expect(screen.getByText('调度操作员')).toBeInTheDocument();
    expect(screen.queryByText('role_te22ck559e')).not.toBeInTheDocument();
  });

  it('后端未返回角色名时回退角色字典，再回退角色码', () => {
    const renderColumn = getUserRoleColumnRender({
      ...baseDeps,
      roles: [{ code: 'operator', name: '调度操作员' } as API.AdminRole],
    });

    render(
      <>
        {renderColumn?.({
          id: 'user-3',
          roleCodes: ['operator', 'role_unknown'],
        } as API.AdminUser)}
      </>,
    );

    expect(screen.getByText('调度操作员')).toBeInTheDocument();
    expect(screen.getByText('role_unknown')).toBeInTheDocument();
  });

  it('所属组织为部门时展示所属公司层级路径', () => {
    const orgs = [
      { id: 'company-1', name: '融迅（北京）供应链管理有限公司', kind: 2 },
      { id: 'dept-1', name: '北京财务', kind: 3, parentId: 'company-1' },
    ] as API.AdminOrganization[];

    const columns = buildUserColumns({
      ...baseDeps,
      organizations: orgs,
    });

    const orgColumn = columns.find((col) => col.title === '所属组织');
    expect(orgColumn).toBeTruthy();

    const columnRender = orgColumn?.render as (
      dom: unknown,
      entity: API.AdminUser,
    ) => ReactNode;

    render(
      <div>
        {columnRender(null, {
          id: 'user-ces',
          organizations: [
            { organizationId: 'dept-1', organizationName: '北京财务', primary: true },
          ],
        } as API.AdminUser)}
      </div>,
    );

    expect(
      screen.getByText('融迅（北京）供应链管理有限公司 / 北京财务'),
    ).toBeInTheDocument();
  });
});
