import { act, render } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  approveRegistration: vi.fn(),
  listOrganizationRoles: vi.fn(),
  listRoles: vi.fn(),
  rejectRegistration: vi.fn(),
}));

const modalState = vi.hoisted(() => ({
  approve: undefined as Record<string, any> | undefined,
  reject: undefined as Record<string, any> | undefined,
}));

const formItems = vi.hoisted(() => new Map<string, Record<string, any>>());

vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (props: Record<string, unknown>) => {
    if (props.title?.toString().startsWith('同意注册')) {
      modalState.approve = props;
    } else {
      modalState.reject = props;
    }
    return <div>{props.children as React.ReactNode}</div>;
  },
  ProFormText: () => null,
  ProFormTextArea: (props: Record<string, unknown>) => {
    formItems.set(String(props.name), props);
    return null;
  },
}));

vi.mock('@/components/ui', () => ({
  ProFormSearchableSelect: (props: Record<string, unknown>) => {
    formItems.set(String(props.name), props);
    return null;
  },
}));

vi.mock('antd', () => ({
  Alert: () => null,
  App: { useApp: () => ({ message: { success: vi.fn() } }) },
  Avatar: ({ children }: { children?: React.ReactNode }) => (
    <span>{children}</span>
  ),
  Space: ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  ),
  Typography: {
    Text: ({ children }: { children?: React.ReactNode }) => (
      <span>{children}</span>
    ),
  },
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceApproveDingTalkRegistration: serviceMocks.approveRegistration,
  adminServiceRejectDingTalkRegistration: serviceMocks.rejectRegistration,
  adminServiceListOrganizationRoles: serviceMocks.listOrganizationRoles,
  adminServiceListRoles: serviceMocks.listRoles,
}));

import { resolveRootOrganizationId } from './constants';
import RegistrationApproveModal from './RegistrationApproveModal';
import RegistrationRejectModal from './RegistrationRejectModal';

const formRef = { current: undefined };

const organizations: API.AdminOrganization[] = [
  { id: 'org-root', name: 'Roncin 总部', code: 'HQ' },
  { id: 'org-cd', name: '成都分公司', code: 'CD', parentId: 'org-root' },
];

describe('resolveRootOrganizationId', () => {
  it('返回组织树根，空列表返回 undefined', () => {
    expect(resolveRootOrganizationId(organizations)).toBe('org-root');
    expect(resolveRootOrganizationId([])).toBeUndefined();
    expect(
      resolveRootOrganizationId([{ id: 'a', name: 'A', parentId: 'missing' }]),
    ).toBe('a');
  });
});

describe('RegistrationApproveModal', () => {
  beforeEach(() => {
    serviceMocks.approveRegistration.mockReset();
    serviceMocks.listOrganizationRoles.mockReset();
    serviceMocks.listRoles.mockReset();
    serviceMocks.listOrganizationRoles.mockResolvedValue({ data: [] });
    serviceMocks.listRoles.mockResolvedValue({ data: [] });
    modalState.approve = undefined;
    formItems.clear();
  });

  it('自选目标组织的注册按该组织加载角色', async () => {
    await act(async () => {
      render(
        <RegistrationApproveModal
          registration={{
            userId: 'user-9',
            displayName: '钉钉新员工',
            requestedOrganizationId: 'org-cd',
            requestedOrganizationName: '成都分公司',
          }}
          open
          onOpenChange={() => {}}
          formRef={formRef}
          organizations={organizations}
          currentOrganizationId="org-root"
          canReadRoles
          onReload={() => {}}
        />,
      );
    });

    expect(serviceMocks.listOrganizationRoles).toHaveBeenCalledWith({
      organizationId: 'org-cd',
    });
    expect(serviceMocks.listRoles).not.toHaveBeenCalled();
  });

  it('未自选目标组织（总部兜底）按组织树根加载当前组织角色', async () => {
    await act(async () => {
      render(
        <RegistrationApproveModal
          registration={{
            userId: 'user-8',
            displayName: '存量注册',
          }}
          open
          onOpenChange={() => {}}
          formRef={formRef}
          organizations={organizations}
          currentOrganizationId="org-root"
          canReadRoles
          onReload={() => {}}
        />,
      );
    });

    // 路由组织为组织树根（总部），且等于当前组织时走组织内 ListRoles。
    expect(serviceMocks.listRoles).toHaveBeenCalledWith();
    expect(serviceMocks.listOrganizationRoles).not.toHaveBeenCalled();
  });

  it('同意提交携带注册用户 ID 与初始角色', async () => {
    serviceMocks.approveRegistration.mockResolvedValue({});
    await act(async () => {
      render(
        <RegistrationApproveModal
          registration={{
            userId: 'user-9',
            displayName: '钉钉新员工',
            requestedOrganizationId: 'org-cd',
            requestedOrganizationName: '成都分公司',
          }}
          open
          onOpenChange={() => {}}
          formRef={formRef}
          organizations={organizations}
          currentOrganizationId="org-root"
          canReadRoles
          onReload={() => {}}
        />,
      );
    });

    const roleItem = formItems.get('roleIds');
    expect(roleItem).toBeTruthy();
    expect(
      (roleItem?.rules as Array<Record<string, unknown>> | undefined)?.some(
        (rule) => rule.required,
      ),
    ).toBe(true);

    const modalProps = modalState.approve;
    expect(modalProps).toBeTruthy();
    let finishResult: unknown;
    await act(async () => {
      finishResult = await modalProps?.onFinish?.({
        roleIds: ['role-1', 'role-2'],
      });
    });
    expect(finishResult).toBe(true);
    expect(serviceMocks.approveRegistration).toHaveBeenCalledWith(
      { id: 'user-9' },
      { id: 'user-9', roleIds: ['role-1', 'role-2'] },
    );
  });
});

describe('RegistrationRejectModal', () => {
  beforeEach(() => {
    serviceMocks.rejectRegistration.mockReset();
    modalState.reject = undefined;
    formItems.clear();
  });

  it('拒绝原因必填且提交原因去空格', async () => {
    serviceMocks.rejectRegistration.mockResolvedValue({});
    await act(async () => {
      render(
        <RegistrationRejectModal
          registration={{ userId: 'user-9', displayName: '钉钉新员工' }}
          open
          onOpenChange={() => {}}
          formRef={formRef}
          onReload={() => {}}
        />,
      );
    });

    const reasonItem = formItems.get('reason');
    expect(reasonItem).toBeTruthy();
    expect(
      (reasonItem?.rules as Array<Record<string, unknown>> | undefined)?.some(
        (rule) => rule.required,
      ),
    ).toBe(true);

    const modalProps = modalState.reject;
    expect(modalProps).toBeTruthy();
    let finishResult: unknown;
    await act(async () => {
      finishResult = await modalProps?.onFinish?.({
        reason: '  非本企业在职人员  ',
      });
    });
    expect(finishResult).toBe(true);
    expect(serviceMocks.rejectRegistration).toHaveBeenCalledWith(
      { id: 'user-9' },
      { id: 'user-9', reason: '非本企业在职人员' },
    );
  });
});
