import { EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { App, Button, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  adminServiceCreateOrganization,
  adminServiceListOrganizations,
  adminServiceUpdateOrganization,
} from '@/services/roncin/adminService';

type OrganizationFormValues = {
  code?: string;
  name?: string;
  enabled?: boolean;
};

export default function OrganizationsPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.AdminOrganization>();

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (organization: API.AdminOrganization) => {
    setEditing(organization);
    setModalOpen(true);
  };

  const columns: ProColumns<API.AdminOrganization>[] = [
    { title: '编码', dataIndex: 'code', width: 180, copyable: true },
    { title: '名称', dataIndex: 'name', width: 240, ellipsis: true },
    {
      title: '上级组织',
      dataIndex: 'parentId',
      width: 260,
      render: (_, record) => record.parentId ?? '根组织',
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      render: (_, record) =>
        record.enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 100,
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EditOutlined />}
          onClick={() => openEdit(record)}
        >
          编辑
        </Button>
      ),
    },
  ];

  return (
    <>
      <ProTable<API.AdminOrganization>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={false}
        pagination={false}
        request={async () => {
          const response = await adminServiceListOrganizations();
          return { data: response.data ?? [], success: response.success ?? true };
        }}
        toolBarRender={() => [
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
            刷新
          </Button>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新增组织
          </Button>,
        ]}
      />
      <ModalForm<OrganizationFormValues>
        title={editing ? '编辑组织' : '新增组织'}
        open={modalOpen}
        formRef={formRef}
        initialValues={editing}
        modalProps={{ destroyOnClose: true, onCancel: () => setModalOpen(false) }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editing?.id) {
            await adminServiceUpdateOrganization(
              { id: editing.id },
              { id: editing.id, name: values.name ?? '', enabled: values.enabled ?? true },
            );
            message.success('组织已更新');
          } else {
            await adminServiceCreateOrganization({ code: values.code ?? '', name: values.name ?? '' });
            message.success('组织已创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="编码"
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请输入组织编码' }]}
        />
        <ProFormText
          name="name"
          label="名称"
          rules={[{ required: true, message: '请输入组织名称' }]}
        />
        {editing ? <ProFormSwitch name="enabled" label="启用状态" /> : null}
      </ModalForm>
    </>
  );
}
