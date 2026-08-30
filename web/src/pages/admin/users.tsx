import { PlusOutlined, ReloadOutlined, UserOutlined } from '@ant-design/icons';
import type { ActionType, ProFormInstance } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess, useModel } from '@umijs/max';
import { App, Button, Space } from 'antd';
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

  useEffect(() => {
    adminServiceListRoles().then((response) => setRoles(unwrapList(response)));
    adminServiceListOrganizations().then((response) =>
      setOrganizations(unwrapList(response)),
    );
  }, []);

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
    canUpdateUsers: access.canUpdateUsers,
    canResetUserPasswords: access.canResetUserPasswords,
    canTerminateUsers: access.canTerminateUsers,
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
        request={async (params) => {
          const response = await adminServiceListUsers({
            page: params.current,
            pageSize: params.pageSize,
            keyword: searchParams.keyword,
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
        currentUserId={initialState?.currentUser?.id}
        defaultOrganizationId={
          initialState?.currentUser?.currentOrganization?.id
        }
        onReload={() => actionRef.current?.reload()}
      />

      <ResetPasswordModal
        user={resetting}
        onClose={() => setResetting(undefined)}
        onReload={() => actionRef.current?.reload()}
      />
    </>
  );
}
