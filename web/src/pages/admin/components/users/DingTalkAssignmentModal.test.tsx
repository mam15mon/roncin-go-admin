import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, screen, waitFor } from '@testing-library/react';
import React, { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const state = vi.hoisted(() => ({
  finish: undefined as
    | undefined
    | ((values: {
        organizationId: string;
        roleIds: string[];
      }) => Promise<boolean>),
  fields: new Map<string, any>(),
  batch: vi.fn(),
  roles: vi.fn(),
  reload: vi.fn(),
  remaining: vi.fn(),
}));
vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: any) => {
    state.finish = props.onFinish;
    return <div>{props.children}</div>;
  },
}));
vi.mock('@/components/ui', () => ({
  ProFormSearchableSelect: (props: any) => {
    state.fields.set(props.name, props);
    return null;
  },
}));
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceBatchAuthorizeDingTalkUsers: state.batch,
  adminServiceListOrganizationRoles: state.roles,
}));
vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  return {
    ...actual,
    App: {
      useApp: () => ({ message: { success: vi.fn(), warning: vi.fn() } }),
    },
  };
});

import DingTalkAssignmentModal from './DingTalkAssignmentModal';

const users = [
  { id: 'person-1', displayName: '张三', pendingAssignment: true },
  { id: 'person-2', displayName: '李四', pendingAssignment: true },
];
const organizations = [
  { id: 'system', kind: 1, enabled: true, name: '系统管理' },
  { id: 'company', kind: 2, enabled: true, name: '天津公司' },
  {
    id: 'department',
    parentId: 'company',
    kind: 3,
    enabled: true,
    name: '操作部',
  },
  { id: 'disabled', kind: 2, enabled: false, name: '停用公司' },
];
function Harness() {
  const [remaining, setRemaining] = useState<API.AdminUser[]>(users);
  return (
    <DingTalkAssignmentModal
      users={remaining}
      organizations={organizations}
      actionRef={{ current: { reload: state.reload } as any }}
      onClose={vi.fn()}
      onRemaining={(next) => {
        state.remaining(next);
        setRemaining(next);
      }}
    />
  );
}
const values = { organizationId: 'department', roleIds: ['role-1'] };
beforeEach(() => {
  state.fields.clear();
  state.batch.mockReset();
  state.roles.mockReset();
  state.roles.mockResolvedValue({
    data: [{ id: 'role-1', name: '操作员', enabled: true }],
  });
  state.remaining.mockReset();
  state.reload.mockReset();
});
describe('钉钉分配开通', () => {
  it('部分成功保留失败人员，重试只提交失败人员', async () => {
    state.batch
      .mockResolvedValueOnce({
        data: [
          { userId: 'person-1', success: true },
          { userId: 'person-2', success: false, message: '已被其他管理员更新' },
        ],
      })
      .mockResolvedValueOnce({ data: [{ userId: 'person-2', success: true }] });
    renderWithClient(<Harness />);
    await act(async () => {
      expect(await state.finish?.(values)).toBe(false);
    });
    expect(
      await screen.findByText(/李四：已被其他管理员更新/),
    ).toBeInTheDocument();
    expect(state.remaining).toHaveBeenCalledWith([users[1]]);
    await act(async () => {
      expect(await state.finish?.(values)).toBe(true);
    });
    expect(state.batch).toHaveBeenLastCalledWith(
      { ...values, userIds: ['person-2'] },
      { skipErrorHandler: true },
    );
    expect(state.reload).toHaveBeenCalledTimes(2);
  });
  it('组织仅包含启用公司及其部门，按目标组织读取公司角色', async () => {
    renderWithClient(<Harness />);
    expect(
      state.fields.get('organizationId').options.map((item: any) => item.value),
    ).toEqual(['company', 'department']);
    await act(async () =>
      state.fields.get('organizationId').fieldProps.onChange('department'),
    );
    await waitFor(() =>
      expect(state.roles).toHaveBeenCalledWith({
        organizationId: 'department',
      }),
    );
    await waitFor(() =>
      expect(state.fields.get('roleIds').options).toEqual([
        expect.objectContaining({ value: 'role-1' }),
      ]),
    );
  });
  it('响应中断不报告成功，刷新人员状态供管理员核实', async () => {
    state.batch.mockRejectedValue(new Error('连接中断'));
    renderWithClient(<Harness />);
    await act(async () => {
      expect(await state.finish?.(values)).toBe(false);
    });
    expect(screen.getByText('请核实开通结果')).toBeInTheDocument();
    expect(
      screen.getByText('开通结果未能确认，请刷新人员状态后处理'),
    ).toBeInTheDocument();
    expect(state.remaining).not.toHaveBeenCalled();
    expect(state.reload).toHaveBeenCalledOnce();
  });
});
