import { describe, expect, it, vi } from 'vitest';
import { buildUserColumns } from './userColumns';

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
});
