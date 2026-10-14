import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import React, { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DingTalkAssignmentModal from './DingTalkAssignmentModal';

const state = vi.hoisted(() => ({
  batch: vi.fn(),
  roles: vi.fn(),
  reload: vi.fn(),
  remaining: vi.fn(),
}));
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceBatchAuthorizeDingTalkUsers: state.batch,
  adminServiceListOrganizationRoles: state.roles,
}));
const users = [
  { id: 'person-1', displayName: '张三', pendingAssignment: true },
  { id: 'person-2', displayName: '李四', pendingAssignment: true },
];
const organizations: API.AdminOrganization[] = [
  { id: 'system', kind: 1, enabled: true, name: '系统管理' },
  { id: 'company', kind: 2, enabled: true, name: '天津公司' },
  {
    id: 'department',
    parentId: 'company',
    kind: 3,
    enabled: true,
    name: '操作部',
  },
  { id: 'other', kind: 2, enabled: true, name: '上海公司' },
  { id: 'disabled', kind: 2, enabled: false, name: '停用公司' },
  {
    id: 'disabled-child',
    parentId: 'disabled',
    kind: 3,
    enabled: true,
    name: '停用公司部门',
  },
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
async function select(label: string, option: string, excludedOption?: string) {
  const input = screen.getByRole('combobox', { name: label });
  fireEvent.mouseDown(input);
  await waitFor(() => expect(input).toHaveAttribute('aria-expanded', 'true'));
  const popup = await waitFor(() => {
    const list = document.getElementById(
      input.getAttribute('aria-controls') ?? '',
    );
    const dropdown = list?.closest('.ant-select-dropdown');
    if (!dropdown) throw new Error('缺少候选浮层');
    return dropdown as HTMLElement;
  });
  const choice = await within(popup as HTMLElement).findByText(option, {
    selector: '.ant-select-item-option-content',
  });
  if (excludedOption) {
    expect(
      within(popup as HTMLElement).queryByText(excludedOption),
    ).not.toBeInTheDocument();
  }
  fireEvent.click(choice);
  fireEvent.mouseDown(document.body);
  fireEvent.blur(input);
}
async function submit(name = '分配并开通') {
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name }));
  });
}
beforeEach(() => {
  vi.clearAllMocks();
  state.roles.mockImplementation(async ({ organizationId }) => ({
    data: [
      {
        id: `role-${organizationId}`,
        name: organizationId === 'other' ? '上海角色' : '天津角色',
        code: organizationId,
        enabled: true,
      },
    ],
  }));
  state.batch.mockResolvedValue({
    success: true,
    data: users.map((user) => ({ userId: user.id, success: true })),
  });
});
describe('钉钉分配开通真实表单', () => {
  it('多组织独立角色，部分组织空角色合法，部分成功后仅重试失败人员并保留输入', async () => {
    state.batch
      .mockResolvedValueOnce({
        success: true,
        data: [
          { userId: 'person-1', success: true },
          {
            userId: 'person-2',
            reason: 'DINGTALK_ASSIGNMENT_CONFLICT',
            message: '已被其他管理员更新',
          },
        ],
      })
      .mockResolvedValueOnce({
        success: true,
        data: [{ userId: 'person-2', success: true }],
      });
    renderWithApp(<Harness />);
    await select('所属组织', '天津公司 / 操作部');
    await select('所属组织', '上海公司');
    await select('天津公司 / 操作部 · 角色（选填）', '天津角色 (department)');
    await submit();
    expect(
      await screen.findByText(/李四：已被其他管理员更新/),
    ).toBeInTheDocument();
    expect(state.remaining).toHaveBeenCalledWith([users[1]]);
    const items = [
      { organizationId: 'department', roleIds: ['role-department'] },
      { organizationId: 'other', roleIds: [] },
    ];
    expect(state.batch).toHaveBeenNthCalledWith(
      1,
      { userIds: ['person-1', 'person-2'], items },
      { skipErrorHandler: true },
    );
    await submit('重试失败人员');
    await waitFor(() =>
      expect(state.batch).toHaveBeenLastCalledWith(
        { userIds: ['person-2'], items },
        { skipErrorHandler: true },
      ),
    );
    await waitFor(() => expect(state.reload).toHaveBeenCalledTimes(2));
  });
  it('所有组织角色为空时提示且不请求开通', async () => {
    renderWithApp(<Harness />);
    await select('所属组织', '天津公司');
    await select('所属组织', '上海公司');
    await submit();
    expect(
      await screen.findByText('请至少为一个组织分配角色'),
    ).toBeInTheDocument();
    expect(state.batch).not.toHaveBeenCalled();
    await select('上海公司 · 角色（选填）', '上海角色 (other)');
    await submit();
    await waitFor(() => expect(state.batch).toHaveBeenCalledOnce());
  });
  it('删除组织清空本行角色，重新选回保持其他组织输入', async () => {
    renderWithApp(<Harness />);
    await select('所属组织', '天津公司');
    await select('所属组织', '上海公司');
    await select('天津公司 · 角色（选填）', '天津角色 (company)');
    await select('上海公司 · 角色（选填）', '上海角色 (other)');
    const tag = screen
      .getByText('天津公司', { selector: '.ant-select-selection-item-content' })
      .closest('.ant-select-selection-item');
    const removeButton = tag?.querySelector(
      '.ant-select-selection-item-remove',
    );
    if (!removeButton) throw new Error('缺少组织移除按钮');
    fireEvent.click(removeButton);
    await waitFor(() =>
      expect(
        screen.queryByRole('combobox', { name: '天津公司 · 角色（选填）' }),
      ).not.toBeInTheDocument(),
    );
    await select('所属组织', '天津公司');
    await submit();
    await waitFor(() =>
      expect(state.batch).toHaveBeenCalledWith(
        {
          userIds: ['person-1', 'person-2'],
          items: [
            { organizationId: 'other', roleIds: ['role-other'] },
            { organizationId: 'company', roleIds: [] },
          ],
        },
        { skipErrorHandler: true },
      ),
    );
  });
  it('候选只含启用业务组织，两个延迟角色查询返回顺序不串用', async () => {
    let resolveCompany!: (value: any) => void;
    let resolveOther!: (value: any) => void;
    state.roles.mockImplementation(
      ({ organizationId }) =>
        new Promise((resolve) => {
          if (organizationId === 'company') resolveCompany = resolve;
          else resolveOther = resolve;
        }),
    );
    renderWithApp(<Harness />);
    fireEvent.mouseDown(screen.getByRole('combobox', { name: '所属组织' }));
    expect(
      screen.queryByText('系统管理', {
        selector: '.ant-select-item-option-content',
      }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('停用公司', {
        selector: '.ant-select-item-option-content',
      }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('停用公司 / 停用公司部门', {
        selector: '.ant-select-item-option-content',
      }),
    ).not.toBeInTheDocument();
    fireEvent.click(
      await screen.findByText('天津公司', {
        selector: '.ant-select-item-option-content',
      }),
    );
    fireEvent.mouseDown(document.body);
    fireEvent.blur(screen.getByRole('combobox', { name: '所属组织' }));
    await select('所属组织', '上海公司');
    await waitFor(() => expect(state.roles).toHaveBeenCalledTimes(2));
    await act(async () =>
      resolveOther({
        data: [{ id: 'sh', name: '同名操作员', code: 'SH', enabled: true }],
      }),
    );
    await act(async () =>
      resolveCompany({
        data: [{ id: 'tj', name: '同名操作员', code: 'TJ', enabled: true }],
      }),
    );
    await select(
      '天津公司 · 角色（选填）',
      '同名操作员 (TJ)',
      '同名操作员 (SH)',
    );
    await select(
      '上海公司 · 角色（选填）',
      '同名操作员 (SH)',
      '同名操作员 (TJ)',
    );
    await submit();
    await waitFor(() =>
      expect(state.batch).toHaveBeenCalledWith(
        {
          userIds: ['person-1', 'person-2'],
          items: [
            { organizationId: 'company', roleIds: ['tj'] },
            { organizationId: 'other', roleIds: ['sh'] },
          ],
        },
        { skipErrorHandler: true },
      ),
    );
  });
  it.each([
    ['连接中断', undefined],
    [
      '缺少人员结果',
      { success: true, data: [{ userId: 'person-1', success: true }] },
    ],
    [
      '结果缺少成功标记',
      { success: true, data: [{ userId: 'person-1' }, { userId: 'person-2' }] },
    ],
  ])('%s时刷新核实且禁止盲重试', async (_, response) => {
    if (response) state.batch.mockResolvedValue(response);
    else state.batch.mockRejectedValue(new Error('连接中断'));
    renderWithApp(<Harness />);
    await select('所属组织', '天津公司');
    await select('天津公司 · 角色（选填）', '天津角色 (company)');
    await submit();
    expect(await screen.findByText('请核实开通结果')).toBeInTheDocument();
    expect(state.remaining).not.toHaveBeenCalled();
    expect(state.reload).toHaveBeenCalledOnce();
    expect(screen.getByRole('button', { name: '分配并开通' })).toBeDisabled();
    expect(screen.queryByText('已开通 2 人')).not.toBeInTheDocument();
  });
});
