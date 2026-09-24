import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DingTalkInvitationStatus } from '@/enums.generated';

const accessState = vi.hoisted(() => ({
  value: {
    canReadOrganizations: false,
    canManageDingTalkInvitations: true,
    canReadRoles: true,
  },
}));

const initialStateState = vi.hoisted(() => ({
  model: {
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '天津分公司', code: 'TJ' },
      },
    },
  },
}));

const serviceMocks = vi.hoisted(() => ({
  getInvitation: vi.fn(),
  listInvitations: vi.fn(),
  listOrganizations: vi.fn(),
  revokeInvitation: vi.fn(),
}));

const proTableState = vi.hoisted(() => ({
  props: undefined as Record<string, any> | undefined,
}));

vi.mock('@/app/access', () => ({
  useAccess: () => accessState.value,
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => initialStateState.model,
}));

vi.mock('@ant-design/pro-components', () => ({
  ProTable: (props: Record<string, unknown>) => {
    proTableState.props = props;
    return <div />;
  },
}));

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  return {
    ...actual,
    App: { useApp: () => ({ message: { success: vi.fn() } }) },
    Button: ({ children }: { children?: React.ReactNode }) => (
      <button type="button">{children}</button>
    ),
    Popconfirm: ({
      children,
      onConfirm,
    }: {
      children?: React.ReactNode;
      onConfirm?: () => void;
    }) => <span onClick={() => onConfirm?.()}>{children}</span>,
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
  };
});

vi.mock('@/services/roncin/adminService', () => ({
  adminServiceGetDingTalkInvitation: serviceMocks.getInvitation,
  adminServiceListDingTalkInvitations: serviceMocks.listInvitations,
  adminServiceListOrganizations: serviceMocks.listOrganizations,
  adminServiceRevokeDingTalkInvitation: serviceMocks.revokeInvitation,
}));

vi.mock('./components/dingtalk/InvitationFormModal', () => ({
  default: () => null,
}));

vi.mock('./components/dingtalk/InvitationQrModal', () => ({
  default: () => null,
}));

import DingTalkInvitationsPanel, {
  normalizeInvitationStatusFilter,
} from './dingtalk-invitations';

describe('normalizeInvitationStatusFilter', () => {
  it('空值不发送筛选参数，数字字符串转枚举值', () => {
    expect(normalizeInvitationStatusFilter(undefined)).toBeUndefined();
    expect(normalizeInvitationStatusFilter('')).toBeUndefined();
    expect(normalizeInvitationStatusFilter(null)).toBeUndefined();
    expect(normalizeInvitationStatusFilter('1')).toBe(1);
    expect(normalizeInvitationStatusFilter(4)).toBe(4);
  });
});

describe('DingTalkInvitationsPanel', () => {
  beforeEach(() => {
    serviceMocks.listInvitations.mockReset();
    serviceMocks.listOrganizations.mockReset();
    serviceMocks.revokeInvitation.mockReset();
    serviceMocks.listInvitations.mockResolvedValue({ data: [], total: 0 });
    proTableState.props = undefined;
  });

  it('普通组织管理员不请求全组织列表，也不提供组织筛选列', async () => {
    accessState.value = {
      canReadOrganizations: false,
      canManageDingTalkInvitations: true,
      canReadRoles: true,
    };
    await act(async () => {
      render(<DingTalkInvitationsPanel />);
    });

    expect(serviceMocks.listOrganizations).not.toHaveBeenCalled();
    const orgFilterColumns = (proTableState.props?.columns ?? []).filter(
      (column: Record<string, unknown>) =>
        column.dataIndex === 'organizationId',
    );
    expect(orgFilterColumns).toHaveLength(0);
  });

  it('全局管理员请求全组织列表并提供组织筛选列', async () => {
    accessState.value = {
      canReadOrganizations: true,
      canManageDingTalkInvitations: true,
      canReadRoles: true,
    };
    serviceMocks.listOrganizations.mockResolvedValue({ data: [] });

    await act(async () => {
      render(<DingTalkInvitationsPanel />);
    });

    await waitFor(() => {
      expect(serviceMocks.listOrganizations).toHaveBeenCalledTimes(1);
    });
    const orgFilterColumns = (proTableState.props?.columns ?? []).filter(
      (column: Record<string, unknown>) =>
        column.dataIndex === 'organizationId',
    );
    expect(orgFilterColumns).toHaveLength(1);
    expect(orgFilterColumns[0].hideInTable).toBe(true);
  });

  it('列表请求参数映射分页、状态与组织筛选', async () => {
    accessState.value = {
      canReadOrganizations: false,
      canManageDingTalkInvitations: true,
      canReadRoles: true,
    };
    await act(async () => {
      render(<DingTalkInvitationsPanel />);
    });

    await act(async () => {
      await proTableState.props?.request?.({
        current: 2,
        pageSize: 50,
        status: '1',
        organizationId: 'org-9',
      });
    });
    expect(serviceMocks.listInvitations).toHaveBeenCalledWith({
      page: 2,
      pageSize: 50,
      status: DingTalkInvitationStatus.DING_TALK_INVITATION_STATUS_PENDING,
      organizationId: 'org-9',
    });

    await act(async () => {
      await proTableState.props?.request?.({
        current: 1,
        pageSize: 20,
        status: '',
        organizationId: '',
      });
    });
    expect(serviceMocks.listInvitations).toHaveBeenLastCalledWith({
      page: 1,
      pageSize: 20,
      status: undefined,
      organizationId: undefined,
    });
  });

  it('撤销操作只对待使用邀请开放并携带邀请 ID', async () => {
    accessState.value = {
      canReadOrganizations: false,
      canManageDingTalkInvitations: true,
      canReadRoles: true,
    };
    serviceMocks.revokeInvitation.mockResolvedValue({});

    await act(async () => {
      render(<DingTalkInvitationsPanel />);
    });

    const optionColumn = (proTableState.props?.columns ?? []).find(
      (column: Record<string, unknown>) => column.valueType === 'option',
    );
    expect(optionColumn).toBeTruthy();

    // 非待使用状态（已消费）不渲染撤销入口。
    expect(
      optionColumn.render(null, {
        id: 'inv-2',
        status: DingTalkInvitationStatus.DING_TALK_INVITATION_STATUS_CONSUMED,
      }),
    ).toBeNull();

    // 待使用邀请可撤销，撤销调用携带邀请 ID。
    serviceMocks.revokeInvitation.mockClear();
    const { unmount } = render(
      optionColumn.render(null, {
        id: 'inv-1',
        status: DingTalkInvitationStatus.DING_TALK_INVITATION_STATUS_PENDING,
      }),
    );
    expect(screen.getByText('二维码')).toBeTruthy();
    fireEvent.click(screen.getByText('撤销'));
    await waitFor(() => {
      expect(serviceMocks.revokeInvitation).toHaveBeenCalledWith({
        id: 'inv-1',
      });
    });
    unmount();
  });

  it('类型列区分通用入职码与定向邀请', async () => {
    await act(async () => {
      render(<DingTalkInvitationsPanel />);
    });

    const kindColumn = (proTableState.props?.columns ?? []).find(
      (column: Record<string, unknown>) => column.dataIndex === 'kind',
    );
    expect(kindColumn).toBeTruthy();

    const { container, unmount } = render(
      <>
        {kindColumn.render(null, {
          kind: 2, // GENERIC
        })}
        {kindColumn.render(null, {
          kind: 1, // TARGETED
        })}
      </>,
    );
    expect(container.textContent).toContain('通用入职码');
    expect(container.textContent).toContain('定向邀请');
    unmount();
  });
});
