import { EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { App, Button, Space, Tag } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import {
  adminServiceCreateUser,
  adminServiceListRoles,
  adminServiceListUsers,
  adminServiceUpdateUser,
} from '@/services/roncin/adminService';

type UserFormValues = {
  username?: string;
  displayName?: string;
  password?: string;
  email?: string;
  enabled?: boolean;
  roleIds?: string[];
};

export default function UsersPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminUser>();
  const [roles, setRoles] = useState<API.AdminRole[]>([]);

  useEffect(() => {
    adminServiceListRoles().then((response) => setRoles(response.data ?? []));
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

  const columns: ProColumns<API.AdminUser>[] = [
    { title: '用户名', dataIndex: 'username', width: 180, copyable: true },
    { title: '显示名称', dataIndex: 'displayName', width: 160 },
    { title: '邮箱', dataIndex: 'email', width: 220, ellipsis: true },
    {
      title: '角色',
      dataIndex: 'roleCodes',
      width: 240,
      search: false,
      render: (_, record) => <Space wrap>{(record.roleCodes ?? []).map((code) => <Tag key={code}>{code}</Tag>)}</Space>,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      valueEnum: { true: { text: '启用' }, false: { text: '停用' } },
      render: (_, record) => record.enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
    },
    { title: '更新时间', dataIndex: 'updatedAt', valueType: 'dateTime', width: 180, search: false },
    {
      title: '操作',
      valueType: 'option',
      width: 100,
      render: (_, record) => <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>编辑</Button>,
    },
  ];

  return (
    <>
      <ProTable<API.AdminUser>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        request={async (params) => {
          const response = await adminServiceListUsers({ page: params.current, pageSize: params.pageSize, keyword: params.keyword });
          return { data: response.data ?? [], success: response.success ?? true, total: response.total ?? 0 };
        }}
        toolBarRender={() => [
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>刷新</Button>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增用户</Button>,
        ]}
      />
      <ModalForm<UserFormValues>
        title={editing ? '编辑用户' : '新增用户'}
        open={modalOpen}
        formRef={formRef}
        initialValues={editing}
        modalProps={{ destroyOnClose: true, onCancel: () => setModalOpen(false) }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editing?.id) {
            await adminServiceUpdateUser({ id: editing.id }, { id: editing.id, displayName: values.displayName ?? '', email: values.email, enabled: values.enabled ?? true, roleIds: values.roleIds ?? [] });
            message.success('用户已更新');
          } else {
            await adminServiceCreateUser({ username: values.username ?? '', displayName: values.displayName ?? '', password: values.password ?? '', email: values.email, roleIds: values.roleIds ?? [] });
            message.success('用户已创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="username" label="用户名" disabled={Boolean(editing)} rules={[{ required: true, message: '请输入用户名' }]} />
        <ProFormText name="displayName" label="显示名称" rules={[{ required: true, message: '请输入显示名称' }]} />
        {!editing ? <ProFormText name="password" label="初始密码" fieldProps={{ type: 'password' }} rules={[{ required: true, min: 12, message: '初始密码至少 12 位' }]} /> : null}
        <ProFormText name="email" label="邮箱" rules={[{ type: 'email', message: '请输入正确的邮箱地址' }]} />
        <ProFormSelect name="roleIds" label="角色" mode="multiple" options={roles.map((role) => ({ label: `${role.name}（${role.code}）`, value: role.id }))} />
        {editing ? <ProFormSwitch name="enabled" label="启用状态" /> : null}
      </ModalForm>
    </>
  );
}
