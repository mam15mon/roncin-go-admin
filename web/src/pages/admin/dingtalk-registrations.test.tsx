import { act, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const accessState = vi.hoisted(() => ({
  value: {
    canReadOrganizations: false,
    canManageDingTalkInvitations: true,
    canReadRoles: true,
  },
}));

const serviceMocks = vi.hoisted(() => ({
  listRegistrations: vi.fn(),
  listOrganizations: vi.fn(),
  listTransferOrganizations: vi.fn(),
}));

const proTableState = vi.hoisted(() => ({
  props: undefined as Record<string, any> | undefined,
}));

const modalState = vi.hoisted(() => ({
  approve: undefined as Record<string, any> | undefined,
  reject: undefined as Record<string, any> | undefined,
  transfer: undefined as Record<string, any> | undefined,
}));

vi.mock('@umijs/max', () => ({
  useAccess: () => accessState.value,
  useModel: () => ({
    initialState: {
      currentUser: { currentOrganization: { id: 'org-1', name: '总部' } },
    },
  }),
}));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: (props: Record<string, unknown>) => {
    proTableState.props = props;
    return <div />;
  },
}));

vi.mock('antd', () => ({
  Avatar: ({ children }: { children?: React.ReactNode }) => (
    <span>{children}</span>
  ),
  Button: ({
    children,
    onClick,
  }: {
    children?: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  Space: ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  ),
  Tag: ({ children }: { children?: React.ReactNode }) => (
    <span>{children}</span>
  ),
  Typography: {
    Text: ({ children }: { children?: React.ReactNode }) => (
      <span>{children}</span>
    ),
  },
}));

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceListDingTalkRegistrations: serviceMocks.listRegistrations,
  adminServiceListOrganizations: serviceMocks.listOrganizations,
  adminServiceListTransferOrganizations: serviceMocks.listTransferOrganizations,
}));

vi.mock('./components/dingtalk/RegistrationApproveModal', () => ({
  default: (props: Record<string, unknown>) => {
    modalState.approve = props;
    return null;
  },
}));

vi.mock('./components/dingtalk/RegistrationRejectModal', () => ({
  default: (props: Record<string, unknown>) => {
    modalState.reject = props;
    return null;
  },
}));

vi.mock('./components/dingtalk/RegistrationTransferModal', () => ({
  default: (props: Record<string, unknown>) => {
    modalState.transfer = props;
    return null;
  },
}));

import DingTalkRegistrationsPanel from './dingtalk-registrations';

describe('DingTalkRegistrationsPanel', () => {
  beforeEach(() => {
    serviceMocks.listRegistrations.mockReset();
    serviceMocks.listOrganizations.mockReset();
    serviceMocks.listTransferOrganizations.mockReset();
    serviceMocks.listRegistrations.mockResolvedValue({ data: [], total: 0 });
    serviceMocks.listTransferOrganizations.mockResolvedValue({ data: [] });
    proTableState.props = undefined;
    modalState.approve = undefined;
    modalState.reject = undefined;
    modalState.transfer = undefined;
  });

  it('队列请求只带分页参数', async () => {
    await act(async () => {
      render(<DingTalkRegistrationsPanel />);
    });

    await act(async () => {
      await proTableState.props?.request?.({ current: 1, pageSize: 20 });
    });
    expect(serviceMocks.listRegistrations).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
    });
    // 待审批队列固定为当前组织范围，不依赖全组织列表。
    expect(serviceMocks.listOrganizations).not.toHaveBeenCalled();
  });

  it('未自选目标组织展示总部兜底，自选组织展示组织名', async () => {
    await act(async () => {
      render(<DingTalkRegistrationsPanel />);
    });

    const orgColumn = (proTableState.props?.columns ?? []).find(
      (column: Record<string, unknown>) =>
        column.dataIndex === 'requestedOrganizationName',
    );
    expect(orgColumn).toBeTruthy();

    const { container, unmount } = render(
      <>
        {orgColumn.render(null, {
          userId: 'u-1',
          requestedOrganizationId: undefined,
        })}
        {orgColumn.render(null, {
          userId: 'u-2',
          requestedOrganizationId: 'org-2',
          requestedOrganizationName: '成都分公司',
        })}
      </>,
    );
    expect(container.textContent).toContain('总部兜底');
    expect(container.textContent).toContain('成都分公司');
    unmount();
  });

  it('同意/拒绝操作把对应注册传给审批模态', async () => {
    let renderResult: ReturnType<typeof render> | undefined;
    await act(async () => {
      renderResult = render(<DingTalkRegistrationsPanel />);
    });

    const optionColumn = (proTableState.props?.columns ?? []).find(
      (column: Record<string, unknown>) => column.valueType === 'option',
    );
    const registration: API.DingTalkRegistration = {
      userId: 'user-9',
      displayName: '钉钉新员工',
      requestedOrganizationId: 'org-1',
      requestedOrganizationName: '总部',
    };

    const { unmount } = render(
      optionColumn.render(null, registration) as React.ReactElement,
    );

    await act(async () => {
      fireEvent.click(screen.getByText('同意'));
    });
    expect(modalState.approve?.open).toBe(true);
    expect(modalState.approve?.registration?.userId).toBe('user-9');

    await act(async () => {
      fireEvent.click(screen.getByText('转派'));
    });
    expect(modalState.transfer?.open).toBe(true);
    expect(modalState.transfer?.registration?.userId).toBe('user-9');

    await act(async () => {
      fireEvent.click(screen.getByText('拒绝'));
    });
    expect(modalState.reject?.open).toBe(true);
    expect(modalState.reject?.registration?.userId).toBe('user-9');

    // 模态回调关闭时清空注册记录。
    await act(async () => {
      modalState.approve?.onOpenChange?.(false);
    });
    expect(modalState.approve?.open).toBe(false);
    unmount();
    renderResult?.unmount();
  });
});
