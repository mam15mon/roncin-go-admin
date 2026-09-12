import { act, render } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DingTalkInvitationKind } from '@/enums.generated';

const serviceMocks = vi.hoisted(() => ({
  createInvitation: vi.fn(),
  listOrganizationRoles: vi.fn(),
  listRoles: vi.fn(),
}));

const modalState = vi.hoisted(() => ({
  props: undefined as Record<string, any> | undefined,
}));

const formItems = vi.hoisted(() => new Map<string, Record<string, any>>());

vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: Record<string, unknown>) => {
    modalState.props = props;
    return <div>{props.children as React.ReactNode}</div>;
  },
  ProFormText: (props: Record<string, unknown>) => {
    formItems.set(String(props.name), props);
    return null;
  },
  ProFormRadio: {
    Group: (props: Record<string, unknown>) => {
      formItems.set(String(props.name), props);
      return null;
    },
  },
}));

vi.mock('@/components/ui', () => ({
  ProFormSearchableSelect: (props: Record<string, unknown>) => {
    formItems.set(String(props.name), props);
    return null;
  },
}));

vi.mock('antd', () => ({
  App: { useApp: () => ({ message: { success: vi.fn() } }) },
  Typography: {
    Text: ({ children }: { children?: React.ReactNode }) => (
      <span>{children}</span>
    ),
  },
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceCreateDingTalkInvitation: serviceMocks.createInvitation,
  adminServiceListOrganizationRoles: serviceMocks.listOrganizationRoles,
  adminServiceListRoles: serviceMocks.listRoles,
}));

import InvitationFormModal from './InvitationFormModal';

const formRef = { current: undefined };

const organizations: API.AdminOrganization[] = [
  { id: 'org-1', name: '天津分公司', code: 'TJ' },
  { id: 'org-2', name: '成都分公司', code: 'CD' },
];

describe('InvitationFormModal', () => {
  beforeEach(() => {
    serviceMocks.createInvitation.mockReset();
    serviceMocks.listRoles.mockReset();
    serviceMocks.listOrganizationRoles.mockReset();
    serviceMocks.listRoles.mockResolvedValue({ data: [] });
    serviceMocks.listOrganizationRoles.mockResolvedValue({ data: [] });
    modalState.props = undefined;
    formItems.clear();
  });

  it('打开时按默认目标组织（当前组织）加载初始角色', async () => {
    await act(async () => {
      render(
        <InvitationFormModal
          open
          onOpenChange={() => {}}
          formRef={formRef}
          organizations={organizations}
          currentOrganizationId="org-1"
          canReadRoles
          onReload={() => {}}
        />,
      );
    });

    expect(serviceMocks.listRoles).toHaveBeenCalledWith();
    expect(serviceMocks.listOrganizationRoles).not.toHaveBeenCalled();
  });

  it('切换目标组织时清空已选角色并加载新组织角色', async () => {
    await act(async () => {
      render(
        <InvitationFormModal
          open
          onOpenChange={() => {}}
          formRef={formRef}
          organizations={organizations}
          currentOrganizationId="org-1"
          canReadRoles
          onReload={() => {}}
        />,
      );
    });

    const organizationItem = formItems.get('organizationId');
    expect(organizationItem).toBeTruthy();
    await act(async () => {
      organizationItem?.fieldProps.onChange('org-2');
    });

    expect(serviceMocks.listOrganizationRoles).toHaveBeenCalledWith({
      organizationId: 'org-2',
    });
  });

  it('提交参数规范化：定向邀请手机号与备注姓名去空格，空备注不发送', async () => {
    serviceMocks.createInvitation.mockResolvedValue({
      data: { id: 'inv-1' },
      invitationUrl: '/login?invite=tok-1',
    });
    const onSuccess = vi.fn();
    await act(async () => {
      render(
        <InvitationFormModal
          open
          onOpenChange={() => {}}
          formRef={formRef}
          organizations={organizations}
          currentOrganizationId="org-1"
          canReadRoles
          onReload={() => {}}
          onSuccess={onSuccess}
        />,
      );
    });

    const modalProps = modalState.props;
    expect(modalProps).toBeTruthy();
    let finishResult: unknown;
    await act(async () => {
      finishResult = await modalProps?.onFinish?.({
        kind: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED,
        mobile: ' 13800138000 ',
        organizationId: 'org-2',
        roleId: 'role-9',
        displayName: '  张三  ',
        expiresInHours: 168,
      });
    });
    expect(finishResult).toBe(true);
    expect(serviceMocks.createInvitation).toHaveBeenCalledWith({
      kind: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_TARGETED,
      mobile: '13800138000',
      organizationId: 'org-2',
      roleId: 'role-9',
      displayName: '张三',
      expiresInHours: 168,
    });
    expect(onSuccess).toHaveBeenCalledWith(
      { id: 'inv-1' },
      '/login?invite=tok-1',
    );
  });

  it('通用入职码模式：手机号不发送，角色为可选', async () => {
    serviceMocks.createInvitation.mockResolvedValue({
      data: { id: 'inv-2' },
      invitationUrl: '/login?invite=generic-tok',
    });
    await act(async () => {
      render(
        <InvitationFormModal
          open
          onOpenChange={() => {}}
          formRef={formRef}
          organizations={organizations}
          currentOrganizationId="org-1"
          canReadRoles
          onReload={() => {}}
        />,
      );
    });

    const modalProps = modalState.props;
    expect(modalProps).toBeTruthy();
    let finishResult: unknown;
    await act(async () => {
      finishResult = await modalProps?.onFinish?.({
        kind: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_GENERIC,
        organizationId: 'org-1',
        expiresInHours: 72,
      });
    });
    expect(finishResult).toBe(true);
    expect(serviceMocks.createInvitation).toHaveBeenCalledWith({
      kind: DingTalkInvitationKind.DING_TALK_INVITATION_KIND_GENERIC,
      mobile: undefined,
      organizationId: 'org-1',
      roleId: undefined,
      displayName: undefined,
      expiresInHours: 72,
    });
  });

  it('定向模式下手机号与目标组织、初始角色为必填校验', async () => {
    await act(async () => {
      render(
        <InvitationFormModal
          open
          onOpenChange={() => {}}
          formRef={formRef}
          organizations={organizations}
          currentOrganizationId="org-1"
          canReadRoles
          onReload={() => {}}
        />,
      );
    });

    for (const name of ['mobile', 'organizationId', 'roleId']) {
      const item = formItems.get(name);
      expect(item).toBeTruthy();
      expect(
        (item?.rules as Array<Record<string, unknown>> | undefined)?.some(
          (rule) => rule.required,
        ),
      ).toBe(true);
    }
    expect(formItems.get('displayName')).toBeTruthy();
  });
});
