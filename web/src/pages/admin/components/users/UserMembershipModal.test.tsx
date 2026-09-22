import { cleanup, render } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import UserMembershipModal from './UserMembershipModal';

const state = vi.hoisted(() => ({
  system: false,
  props: {} as Record<string, unknown>,
  switches: [] as string[],
  create: vi.fn(),
  update: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ isSystemWorkspace: state.system }),
}));
vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: Record<string, unknown>) => {
    state.props = props;
    return <>{props.children as React.ReactNode}</>;
  },
  ProFormSwitch: ({ name }: { name: string }) => {
    state.switches.push(name);
    return null;
  },
}));
vi.mock('@/components/ui', () => ({ ProFormSearchableSelect: () => null }));
vi.mock('antd', () => ({
  Alert: () => null,
  App: { useApp: () => ({ message: { success: vi.fn() } }) },
}));
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceCreateUserMembership: state.create,
  adminServiceUpdateUserMembership: state.update,
  adminServiceListOrganizationRoles: vi.fn(),
}));
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  state.switches = [];
});
function mount(membershipEditing?: API.AdminUserMembership) {
  render(
    <UserMembershipModal
      open
      onOpenChange={vi.fn()}
      userId="user"
      membershipEditing={membershipEditing}
      memberships={[]}
      organizations={[]}
      membershipRoles={[]}
      onMembershipRolesChange={vi.fn()}
      onSaved={vi.fn()}
    />,
  );
  return state.props.onFinish as (
    values: Record<string, unknown>,
  ) => Promise<boolean>;
}
describe('公司成员默认工作台保护', () => {
  it('公司新增成员不设置全局主要组织，即使提交值要求开启', async () => {
    const finish = mount();
    expect(state.switches).not.toContain('primary');
    await finish({ organizationId: 'company', primary: true });
    expect(state.create).toHaveBeenCalledWith(
      { userId: 'user' },
      {
        userId: 'user',
        organizationId: 'company',
        primary: false,
        roleIds: [],
      },
    );
  });
  it.each([true, false])('公司更新成员保留原主要组织值 %s', async (primary) => {
    const finish = mount({ id: 'membership', primary });
    await finish({ primary: !primary, enabled: true });
    expect(state.update).toHaveBeenCalledWith(
      { userId: 'user', id: 'membership' },
      { userId: 'user', id: 'membership', primary, enabled: true, roleIds: [] },
    );
  });
});
