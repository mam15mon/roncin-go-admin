import { EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import { ModalForm, ProFormSelect, ProFormSwitch, ProFormText, ProTable } from '@ant-design/pro-components';
import { App, Button, Space, Tag } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import {
  adminServiceCreateRole,
  adminServiceListPermissions,
  adminServiceListRoles,
  adminServiceUpdateRole,
} from '@/services/roncin/adminService';

const dataScopeOptions = [
  { label: '全部组织', value: 1 },
  { label: '当前组织', value: 2 },
  { label: '组织树', value: 3 },
  { label: '仅本人', value: 4 },
];

type RoleFormValues = { code?: string; name?: string; dataScope?: number; permissionKeys?: string[]; enabled?: boolean };

export default function RolesPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminRole>();
  const [permissions, setPermissions] = useState<API.AdminPermission[]>([]);

  useEffect(() => {
    adminServiceListPermissions().then((response) => setPermissions(response.data ?? []));
  }, []);

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (role: API.AdminRole) => {
    setEditing(role);
    setModalOpen(true);
  };

  const columns: ProColumns<API.AdminRole>[] = [
    { title: '编码', dataIndex: 'code', width: 180, copyable: true },
    { title: '名称', dataIndex: 'name', width: 180 },
    { title: '数据范围', dataIndex: 'dataScope', width: 140, valueEnum: Object.fromEntries(dataScopeOptions.map((item) => [item.value, { text: item.label }])), render: (_, record) => <Tag>{dataScopeOptions.find((item) => item.value === record.dataScope)?.label ?? '未设置'}</Tag> },
    { title: '权限数', dataIndex: 'permissionKeys', width: 100, search: false, render: (_, record) => record.permissionKeys?.length ?? 0 },
    { title: '状态', dataIndex: 'enabled', width: 100, render: (_, record) => record.enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag> },
    { title: '更新时间', dataIndex: 'updatedAt', valueType: 'dateTime', width: 180, search: false },
    { title: '操作', valueType: 'option', width: 100, render: (_, record) => <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>编辑</Button> },
  ];

  return (
    <>
      <ProTable<API.AdminRole>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={false}
        pagination={false}
        request={async () => {
          const response = await adminServiceListRoles();
          return { data: response.data ?? [], success: response.success ?? true };
        }}
        toolBarRender={() => [
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>刷新</Button>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增角色</Button>,
        ]}
      />
      <ModalForm<RoleFormValues>
        title={editing ? '编辑角色' : '新增角色'}
        open={modalOpen}
        formRef={formRef}
        initialValues={editing}
        modalProps={{ destroyOnClose: true, width: 640, onCancel: () => setModalOpen(false) }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editing?.id) {
            await adminServiceUpdateRole({ id: editing.id }, { id: editing.id, name: values.name ?? '', dataScope: values.dataScope ?? 0, enabled: values.enabled ?? true, permissionKeys: values.permissionKeys ?? [] });
            message.success('角色已更新');
          } else {
            await adminServiceCreateRole({ code: values.code ?? '', name: values.name ?? '', dataScope: values.dataScope ?? 0, permissionKeys: values.permissionKeys ?? [] });
            message.success('角色已创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="code" label="编码" disabled={Boolean(editing)} rules={[{ required: true, message: '请输入角色编码' }]} />
        <ProFormText name="name" label="名称" rules={[{ required: true, message: '请输入角色名称' }]} />
        <ProFormSelect name="dataScope" label="数据范围" options={dataScopeOptions} rules={[{ required: true, message: '请选择数据范围' }]} />
        <ProFormSelect
          name="permissionKeys"
          label="权限"
          mode="multiple"
          options={permissions.map((permission) => ({ label: `${permission.name}（${permission.key}）`, value: permission.key }))}
          fieldProps={{ optionRender: (option) => <Space direction="vertical" size={0}><span>{option.label}</span><small>{permissions.find((item) => item.key === option.value)?.description}</small></Space> }}
        />
        {editing ? <ProFormSwitch name="enabled" label="启用状态" /> : null}
      </ModalForm>
    </>
  );
}
