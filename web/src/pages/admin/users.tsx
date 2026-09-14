import { PlusOutlined, ReloadOutlined, UserOutlined } from '@ant-design/icons';
import type { ActionType, ProFormInstance } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess, useModel } from '@umijs/max';
import { App, Button, Card, Space, Tabs } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { SearchFilterTemplate } from '@/components/ui';
import {
  adminServiceListOrganizations,
  adminServiceListRoles,
  adminServiceListUsers,
  adminServiceTerminateUser,
} from '@/services/roncin/adminService';
import { toTableRequest, unwrapList } from '@/utils/api';
import ResetPasswordModal from './components/users/ResetPasswordModal';
import UserFormModal from './components/users/UserFormModal';
import { buildUserColumns } from './components/users/userColumns';

// 用户列表按「在职 / 离职」拆分为两个页签，共享关键字搜索，页签切换回到第一页。
type UserListTab = 'active' | 'departed';

export default function UsersPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const { initialState } = useModel('@@initialState');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminUser>();
  const [resetting, setResetting] = useState<API.AdminUser>();
  const [roles, setRoles] = useState<API.AdminRole[]>([]);
  const [organizations, setOrganizations] = useState<API.AdminOrganization[]>(
    [],
  );
  const [searchParams, setSearchParams] = useState<{ keyword?: string }>({});
  const [listTab, setListTab] = useState<UserListTab>('active');

  // ProTable 的 params 变化只按当前页码重新请求，切页签时显式回到第一页，
  // 避免大页码切到数据较少的页签时停留在超出范围的空页。
  const switchListTab = (tab: string) => {
    setListTab(tab as UserListTab);
    actionRef.current?.setPageInfo?.({ current: 1 });
  };

  useEffect(() => {
    // 角色与组织数据源按权限分流：普通组织管理员只加载当前组织的 ListRoles，
    // 全组织列表仅限具备全局组织读取权限的管理员，避免进入用户页即触发预期外 403。
    if (access.canReadRoles) {
      adminServiceListRoles().then((response) =>
        setRoles(unwrapList(response)),
      );
    }
    if (access.canReadOrganizations) {
      adminServiceListOrganizations().then((response) =>
        setOrganizations(unwrapList(response)),
      );
    }
  }, [access.canReadRoles, access.canReadOrganizations]);

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (user: API.AdminUser) => {
    setEditing(user);
    setModalOpen(true);
  };

  const handleTerminate = async (record: API.AdminUser) => {
    if (!record.id) return;
    await adminServiceTerminateUser({ id: record.id }, { id: record.id });
    message.success('离职办理完成，账号和历史记录已保留');
    actionRef.current?.reload();
  };

  const columns = buildUserColumns({
    roles,
    showActions: listTab === 'active',
    canUpdateUsers: access.canUpdateUsers,
    canResetUserPasswords: access.canResetUserPasswords,
    canTerminateUsers: access.canTerminateUsers,
    canReadAllUserMemberships: access.canReadAllUserMemberships,
    currentUserId: initialState?.currentUser?.id,
    onEdit: openEdit,
    onResetPassword: setResetting,
    onTerminate: handleTerminate,
  });

  return (
    <>
      <SearchFilterTemplate
        layout="bar"
        keywordPlaceholder="搜索用户名、姓名、拼音或邮箱..."
        onSearch={(values) => {
          setSearchParams(values);
          actionRef.current?.reload();
        }}
        onReset={() => {
          setSearchParams({});
          actionRef.current?.reload();
        }}
        extraRight={
          <Space size={8}>
            <Button
              key="refresh"
              icon={<ReloadOutlined />}
              onClick={() => actionRef.current?.reload()}
            >
              刷新
            </Button>
            {access.canCreateUsers && (
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={openCreate}
              >
                新增用户
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
          backgroundColor: '#ffffff',
          marginBottom: 12,
        }}
        styles={{ body: { padding: '0 16px' } }}
      >
        <Tabs
          activeKey={listTab}
          onChange={switchListTab}
          items={[
            { key: 'active', label: '在职用户' },
            { key: 'departed', label: '离职用户' },
          ]}
          tabBarStyle={{ marginBottom: 0 }}
        />
      </Card>
      <ProTable<API.AdminUser>
        headerTitle={
          <Space size={8}>
            <UserOutlined style={{ color: '#1677ff' }} />
            <span>成员账号列表</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: true,
          showQuickJumper: true,
        }}
        params={{ enabled: listTab === 'active' }}
        request={async (params) => {
          const response = await adminServiceListUsers({
            page: params.current,
            pageSize: params.pageSize,
            keyword: searchParams.keyword,
            enabled: params.enabled,
          });
          return toTableRequest(response);
        }}
        search={false}
        toolBarRender={false}
      />

      <UserFormModal
        open={modalOpen}
        onOpenChange={setModalOpen}
        editing={editing}
        formRef={formRef}
        roles={roles}
        organizations={organizations}
        canReadAllUserMemberships={access.canReadAllUserMemberships}
        canManageUserMemberships={access.canManageUserMemberships}
        canAuthorizeWeComUsers={access.canAuthorizeWeComUsers}
        canAuthorizeDingTalkUsers={access.canAuthorizeDingTalkUsers}
        currentUserId={initialState?.currentUser?.id}
        defaultOrganizationId={
          initialState?.currentUser?.currentOrganization?.id
        }
        onReload={() => {
          // 新建用户固定为在职账号；在离职页签发起创建后落回在职页签第一页，
          // 避免新用户在当前页签不可见造成「创建失败」的误解。
          if (!editing) {
            switchListTab('active');
          }
          actionRef.current?.reload();
        }}
      />

      <ResetPasswordModal
        user={resetting}
        onClose={() => setResetting(undefined)}
        onReload={() => actionRef.current?.reload()}
      />
    </>
  );
}
