import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { cleanup } from '@testing-library/react';
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
  ProFormDependency: () => null,
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
  adminServiceCreateUserMemberships: state.create,
  adminServiceUpdateUserMembership: state.update,
  adminServiceListOrganizationRoles: vi.fn(),
}));
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  state.switches = [];
});
function mount(membershipEditing?: API.AdminUserMembership) {
  renderWithClient(
    <UserMembershipModal
      open
      onOpenChange={vi.fn()}
      userId="user"
      membershipEditing={membershipEditing}
      memberships={[]}
      organizations={[]}
      onSaved={vi.fn()}
    />,
  );
  return state.props.onFinish as (
    values: Record<string, unknown>,
  ) => Promise<boolean>;
}
describe('组织成员关系保存', () => {
  it('新增成员保存组织及选填角色', async () => {
    const finish = mount();
    expect(state.switches).not.toContain('primary');
    await finish({ organizationIds: ['company'] });
    expect(state.create).toHaveBeenCalledWith(
      { userId: 'user' },
      {
        userId: 'user',
        items: [{ organizationId: 'company', roleIds: [] }],
      },
    );
  });
  it('编辑成员保存指定关系的角色与状态', async () => {
    const finish = mount({ id: 'membership' });
    await finish({ enabled: true });
    expect(state.update).toHaveBeenCalledWith(
      { userId: 'user', id: 'membership' },
      { userId: 'user', id: 'membership', enabled: true, roleIds: [] },
    );
  });
});
