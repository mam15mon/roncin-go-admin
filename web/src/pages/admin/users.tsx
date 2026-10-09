import { PlusOutlined, ReloadOutlined, UserOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { App, Button, Card, Space, Tabs } from 'antd';
import React, { useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { SearchFilterTemplate } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { AdminUserStatus } from '@/enums.generated';
import {
  adminServiceListOrganizations,
  adminServiceListRoles,
  adminServiceListUsers,
  adminServiceTerminateUser,
} from '@/services/roncin/adminService';
import { ensureListResponse, toTableRequest, unwrapList } from '@/utils/api';
import DingTalkAssignmentModal from './components/users/DingTalkAssignmentModal';
import DingTalkDirectorySyncPanel from './components/users/DingTalkDirectorySyncPanel';
import ResetPasswordModal from './components/users/ResetPasswordModal';
import UserFormModal from './components/users/UserFormModal';
import { buildUserColumns } from './components/users/userColumns';

type UserListTab = 'pending' | 'active' | 'departed';
type UserSearchParams = { keyword?: string };

export default function UsersPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useInitialState();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminUser>();
  const [resetting, setResetting] = useState<API.AdminUser>();
  const [selected, setSelected] = useState<API.AdminUser[]>([]);
  const [assigning, setAssigning] = useState<API.AdminUser[]>([]);
  const [searchParams, setSearchParams] = useState<UserSearchParams>({});
  const [listTab, setListTab] = useState<UserListTab>(
    access.isSystemWorkspace ? 'pending' : 'active',
  );
  const rolesQuery = useQuery({
    queryKey: ['admin-users', 'roles'],
    enabled: Boolean(access.canReadRoles),
    queryFn: async () =>
      unwrapList(
        ensureListResponse(await adminServiceListRoles(), '读取角色失败'),
      ),
  });
  const organizationsQuery = useQuery({
    queryKey: ['admin-users', 'organizations'],
    enabled: Boolean(access.canReadOrganizations),
    queryFn: async () =>
      unwrapList(
        ensureListResponse(
          await adminServiceListOrganizations(),
          '读取组织失败',
        ),
      ),
  });
  const roles = rolesQuery.data ?? [];
  const organizations = organizationsQuery.data ?? [];
  const switchListTab = (tab: string) => {
    setListTab(tab as UserListTab);
    setSelected([]);
    actionRef.current?.setPageInfo?.({ current: 1 });
  };
  const columns = buildUserColumns({
    roles,
    organizations,
    showActions: true,
    canUpdateUsers: access.canUpdateUsers || access.canManageUserMemberships,
    canAuthorizeDingTalkUsers: access.canAuthorizeDingTalkUsers,
    canResetUserPasswords: access.canResetUserPasswords,
    canTerminateUsers: access.canTerminateUsers,
    canReadUserMemberships: access.canReadUserMemberships,
    currentUserId: initialState?.currentUser?.id,
    onEdit: (user) => {
      setEditing(user);
      setModalOpen(true);
    },
    onAssign: (user) => setAssigning([user]),
    onResetPassword: setResetting,
    onTerminate: async (record) => {
      if (!record.id) return;
      const response = await adminServiceTerminateUser(
        { id: record.id },
        { id: record.id },
      );
      if (!response) return;
      message.success('离职办理完成，账号和历史记录已保留');
      actionRef.current?.reload();
    },
  });
  const columnSettings = useColumnSettings<ProColumns<API.AdminUser>>({
    tableKey: 'admin:users',
    columns,
  });
  return (
    <>
      {access.canSyncDingTalkDirectory && (
        <DingTalkDirectorySyncPanel actionRef={actionRef} />
      )}
      <SearchFilterTemplate<UserSearchParams>
        layout="bar"
        keywordPlaceholder="搜索姓名、用户名、拼音或邮箱..."
        onSearch={(values) => {
          setSelected([]);
          setSearchParams(values);
          actionRef.current?.setPageInfo?.({ current: 1 });
        }}
        onReset={() => {
          setSelected([]);
          setSearchParams({});
          actionRef.current?.setPageInfo?.({ current: 1 });
        }}
        extraRight={
          <Space size={8}>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => actionRef.current?.reload()}
            >
              刷新
            </Button>
            {access.canAuthorizeDingTalkUsers && listTab === 'pending' && (
              <Button
                type="primary"
                disabled={!selected.length || organizationsQuery.isFetching}
                onClick={() => setAssigning(selected)}
              >
                批量分配 ({selected.length})
              </Button>
            )}
            {access.canCreateUsers && (
              <Button
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditing(undefined);
                  formRef.current?.resetFields();
                  setModalOpen(true);
                }}
              >
                新增本地用户
              </Button>
            )}
          </Space>
        }
      />
      <Card
        variant="borderless"
        style={{
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          marginBottom: 'var(--roncin-page-section-gap)',
        }}
        styles={{ body: { padding: '0 16px' } }}
      >
        <Tabs
          activeKey={listTab}
          onChange={switchListTab}
          tabBarStyle={{ marginBottom: 0 }}
          items={[
            ...(access.isSystemWorkspace
              ? [{ key: 'pending', label: '待分配' }]
              : []),
            { key: 'active', label: '已开通' },
            { key: 'departed', label: '离职 / 停用' },
          ]}
        />
      </Card>
      <ProTable<API.AdminUser>
        headerTitle={
          <Space size={8}>
            <UserOutlined style={{ color: '#1677ff' }} />
            <span>人员名单</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columnSettings.columns}
        bordered
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: true,
          showQuickJumper: true,
        }}
        params={{ listTab, keyword: searchParams.keyword }}
        request={async (params) => {
          const response = await adminServiceListUsers({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            ...(params.listTab === 'pending'
              ? { status: AdminUserStatus.ADMIN_USER_STATUS_PENDING_ASSIGNMENT }
              : { enabled: params.listTab === 'active' }),
          });
          if (!response) throw new Error('读取人员名单失败');
          return toTableRequest(response);
        }}
        rowSelection={
          access.canAuthorizeDingTalkUsers && listTab === 'pending'
            ? {
                selectedRowKeys: selected.flatMap((user) =>
                  user.id ? [user.id] : [],
                ),
                onChange: (_, rows) => setSelected(rows),
                preserveSelectedRowKeys: false,
                getCheckboxProps: (record) => ({
                  disabled:
                    !record.pendingAssignment || !record.dingtalkLastSyncedAt,
                }),
              }
            : undefined
        }
        onDataSourceChange={(rows) => {
          const visible = new Set(rows.map((row) => row.id));
          setSelected((previous) =>
            previous.filter((row) => visible.has(row.id)),
          );
        }}
        onChange={() => setSelected([])}
        search={false}
        options={{ reload: true, density: true, setting: false }}
        toolBarRender={() => [columnSettings.entry]}
      />
      <UserFormModal
        open={modalOpen}
        onOpenChange={setModalOpen}
        editing={editing}
        formRef={formRef}
        roles={roles}
        organizations={organizations}
        canReadUserMemberships={access.canReadUserMemberships}
        canManageUserMemberships={access.canManageUserMemberships}
        canUpdateUserProfile={access.canUpdateUsers}
        canAuthorizeWeComUsers={access.canAuthorizeWeComUsers}
        currentUserId={initialState?.currentUser?.id}
        defaultOrganizationId={
          initialState?.currentUser?.currentOrganization?.id
        }
        onReload={() => {
          if (!editing) switchListTab('active');
          actionRef.current?.reload();
        }}
      />
      <ResetPasswordModal
        user={resetting}
        onClose={() => setResetting(undefined)}
        onReload={() => actionRef.current?.reload()}
      />
      {assigning.length > 0 && (
        <DingTalkAssignmentModal
          users={assigning}
          organizations={organizations}
          actionRef={actionRef}
          onClose={() => setAssigning([])}
          onRemaining={(remaining) => {
            setAssigning(remaining);
            setSelected(remaining);
          }}
        />
      )}
    </>
  );
}
