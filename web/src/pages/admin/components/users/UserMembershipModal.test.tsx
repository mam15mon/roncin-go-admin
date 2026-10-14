import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import UserMembershipModal from './UserMembershipModal';

const state = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
  roles: vi.fn(),
  onSaved: vi.fn(),
  onOpenChange: vi.fn(),
}));
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceCreateUserMemberships: state.create,
  adminServiceUpdateUserMembership: state.update,
  adminServiceListOrganizationRoles: state.roles,
}));
const organizations: API.AdminOrganization[] = [
  { id: 'company', kind: 2, enabled: true, name: '天津公司', code: 'TJ' },
  { id: 'other', kind: 2, enabled: true, name: '上海公司', code: 'SH' },
];
function mount(membershipEditing?: API.AdminUserMembership) {
  renderWithApp(
    <UserMembershipModal
      open
      onOpenChange={state.onOpenChange}
      userId="user"
      membershipEditing={membershipEditing}
      memberships={[]}
      organizations={organizations}
      onSaved={state.onSaved}
    />,
  );
}
async function select(label: string, option: string) {
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
  fireEvent.click(choice);
  fireEvent.mouseDown(document.body);
  fireEvent.blur(input);
}
async function submit() {
  await act(async () =>
    fireEvent.click(screen.getByRole('button', { name: /确\s*定/ })),
  );
}
beforeEach(() => {
  vi.clearAllMocks();
  state.roles.mockResolvedValue({
    data: [{ id: 'role', name: '操作员', code: 'OP', enabled: true }],
  });
  state.create.mockResolvedValue({
    success: true,
    data: [{ id: 'membership-1' }, { id: 'membership-2' }],
  });
  state.update.mockResolvedValue({ success: true, data: { id: 'membership' } });
});
describe('组织成员关系真实表单', () => {
  it('已开通用户多个组织角色全空仍可保存', async () => {
    mount();
    await select('所属组织', '天津公司 (TJ)');
    await select('所属组织', '上海公司 (SH)');
    await submit();
    await waitFor(() =>
      expect(state.create).toHaveBeenCalledWith(
        { userId: 'user' },
        {
          userId: 'user',
          items: [
            { organizationId: 'company', roleIds: [] },
            { organizationId: 'other', roleIds: [] },
          ],
        },
      ),
    );
    await waitFor(() => expect(state.onSaved).toHaveBeenCalledOnce());
    expect(state.onOpenChange).toHaveBeenCalledWith(false);
  });
  it('保存失败保留组织角色，去掉再选回清空本行并保留其余角色', async () => {
    state.create.mockResolvedValueOnce(undefined);
    mount();
    await select('所属组织', '天津公司 (TJ)');
    await select('所属组织', '上海公司 (SH)');
    await select('天津公司 · 角色（选填）', '操作员 (OP)');
    await select('上海公司 · 角色（选填）', '操作员 (OP)');
    await submit();
    await waitFor(() => expect(state.create).toHaveBeenCalledOnce());
    expect(state.onSaved).not.toHaveBeenCalled();
    expect(state.onOpenChange).not.toHaveBeenCalledWith(false);
    const tag = screen
      .getByText('天津公司 (TJ)', {
        selector: '.ant-select-selection-item-content',
      })
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
    await select('所属组织', '天津公司 (TJ)');
    await submit();
    await waitFor(() =>
      expect(state.create).toHaveBeenLastCalledWith(
        { userId: 'user' },
        {
          userId: 'user',
          items: [
            { organizationId: 'other', roleIds: ['role'] },
            { organizationId: 'company', roleIds: [] },
          ],
        },
      ),
    );
    await waitFor(() => expect(state.onSaved).toHaveBeenCalledOnce());
  });
  it('编辑失败不关闭，重试保存指定关系的角色和状态', async () => {
    state.update.mockResolvedValueOnce(undefined);
    mount({
      id: 'membership',
      organizationId: 'company',
      roleIds: [],
      enabled: true,
    });
    expect(screen.getByRole('combobox', { name: '所属组织' })).toBeDisabled();
    await select('组织角色', '操作员 (OP)');
    fireEvent.click(screen.getByRole('switch'));
    await submit();
    await waitFor(() => expect(state.update).toHaveBeenCalledOnce());
    expect(state.onOpenChange).not.toHaveBeenCalledWith(false);
    await submit();
    await waitFor(() =>
      expect(state.update).toHaveBeenLastCalledWith(
        { userId: 'user', id: 'membership' },
        { userId: 'user', id: 'membership', enabled: false, roleIds: ['role'] },
      ),
    );
    await waitFor(() => expect(state.onSaved).toHaveBeenCalledOnce());
  });
});
