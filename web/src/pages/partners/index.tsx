import {
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Space, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  partnerServiceCreatePartner,
  partnerServiceListPartners,
  partnerServiceUpdatePartner,
} from '@/services/roncin/partnerService';

const partnerTypeOptions = [
  { label: '客户', value: 1 },
  { label: '供应商', value: 2 },
  { label: '客户与供应商', value: 3 },
];

const partnerTypeLabels: Record<number, string> = {
  1: '客户',
  2: '供应商',
  3: '客户与供应商',
};

type PartnerFormValues = {
  code?: string;
  name?: string;
  type?: number;
  contactName?: string;
  phone?: string;
  email?: string;
  address?: string;
  enabled?: boolean;
};

export default function Partners() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message } = App.useApp();
  const access = useAccess();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.Partner>();

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (partner: API.Partner) => {
    setEditing(partner);
    setModalOpen(true);
  };

  const columns: ProColumns<API.Partner>[] = [
    {
      title: '编码',
      dataIndex: 'code',
      width: 120,
      fixed: 'left',
      copyable: true,
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 220,
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 140,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        partnerTypeOptions.map((option) => [option.value, { text: option.label }]),
      ),
      render: (_, record) => (
        <Tag>{partnerTypeLabels[record.type ?? 0] ?? '未设置'}</Tag>
      ),
    },
    {
      title: '联系人',
      dataIndex: 'contactName',
      width: 120,
    },
    {
      title: '电话',
      dataIndex: 'phone',
      width: 150,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      width: 220,
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      valueType: 'select',
      valueEnum: {
        true: { text: '启用', status: 'Success' },
        false: { text: '停用', status: 'Default' },
      },
      render: (_, record) =>
        record.enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 100,
      fixed: 'right',
      render: (_, record) =>
        access.canManagePartners ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEdit(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  return (
    <PageContainer
      title="客户与供应商"
      subTitle="维护当前组织的业务往来单位档案"
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => actionRef.current?.reload()}
          >
            刷新
          </Button>
          {access.canManagePartners ? (
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新增往来单位
            </Button>
          ) : null}
        </Space>
      }
    >
      <ProTable<API.Partner>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1200 }}
        request={async (params) => {
          const response = await partnerServiceListPartners({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            type: params.type,
            enabled: params.enabled,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
            total: response.total ?? 0,
          };
        }}
        search={{ labelWidth: 'auto', defaultCollapsed: false }}
        toolBarRender={false}
      />

      <ModalForm<PartnerFormValues>
        title={editing ? '编辑往来单位' : '新增往来单位'}
        open={modalOpen}
        formRef={formRef}
        modalProps={{
          destroyOnClose: true,
          onCancel: () => setModalOpen(false),
        }}
        initialValues={editing}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editing?.id) {
            await partnerServiceUpdatePartner(
              { id: editing.id },
              {
                id: editing.id,
                name: values.name ?? '',
                type: values.type ?? 0,
                contactName: values.contactName,
                phone: values.phone,
                email: values.email,
                address: values.address,
                enabled: values.enabled ?? true,
              },
            );
            message.success('往来单位已更新');
          } else {
            await partnerServiceCreatePartner({
              code: values.code ?? '',
              name: values.name ?? '',
              type: values.type ?? 0,
              contactName: values.contactName,
              phone: values.phone,
              email: values.email,
              address: values.address,
            });
            message.success('往来单位已创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="编码"
          placeholder="请输入唯一编码"
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请输入往来单位编码' }]}
        />
        <ProFormText
          name="name"
          label="名称"
          placeholder="请输入客户或供应商名称"
          rules={[{ required: true, message: '请输入往来单位名称' }]}
        />
        <ProFormSelect
          name="type"
          label="类型"
          options={partnerTypeOptions}
          fieldProps={{
            showSearch: true,
            optionFilterProp: 'label',
          }}
          rules={[{ required: true, message: '请选择往来单位类型' }]}
        />
        <ProFormText name="contactName" label="联系人" />
        <ProFormText name="phone" label="电话" />
        <ProFormText
          name="email"
          label="邮箱"
          rules={[{ type: 'email', message: '请输入正确的邮箱地址' }]}
        />
        <ProFormTextArea name="address" label="地址" fieldProps={{ rows: 3 }} />
        {editing ? (
          <ProFormSwitch name="enabled" label="启用状态" />
        ) : null}
      </ModalForm>
    </PageContainer>
  );
}
