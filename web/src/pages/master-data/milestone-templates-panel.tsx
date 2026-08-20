import { CheckOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDigit,
  ProFormList,
  ProFormRadio,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Popconfirm, Space, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  masterDataServiceCreateMilestoneTemplate,
  masterDataServiceListMilestoneTemplates,
  masterDataServicePublishMilestoneTemplate,
  masterDataServiceSetDefaultMilestoneTemplate,
} from '@/services/roncin/masterDataService';

const businessTypeOptions = [
  { label: '海运出口', value: 1 },
  { label: '海运进口', value: 2 },
  { label: '空运出口', value: 3 },
  { label: '空运进口', value: 4 },
  { label: '陆运', value: 5 },
  { label: '铁路', value: 6 },
];

const businessTypeLabels = Object.fromEntries(
  businessTypeOptions.map((item) => [item.value, item.label]),
);

type MilestoneTemplateFormValues = {
  code?: string;
  name?: string;
  businessType?: number;
  tradeTerm?: string;
  version?: number;
  items?: API.MilestoneTemplateItemInput[];
};

type PublishFormValues = {
  isDefault?: boolean;
};

export default function MilestoneTemplatesPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const access = useAccess();
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [publishModalOpen, setPublishModalOpen] = useState(false);
  const [publishingItem, setPublishingItem] = useState<API.MilestoneTemplate>();

  const openCreate = () => {
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const handleSetDefault = async (record: API.MilestoneTemplate) => {
    if (!record.id) return;
    await masterDataServiceSetDefaultMilestoneTemplate(
      { id: record.id },
      { id: record.id },
    );
    message.success('已设为默认版本');
    actionRef.current?.reload();
  };

  const columns: ProColumns<API.MilestoneTemplate>[] = [
    {
      title: '业务类型',
      dataIndex: 'businessType',
      width: 140,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        businessTypeOptions.map((item) => [item.value, { text: item.label }]),
      ),
      render: (_, record) => (
        <Tag>{businessTypeLabels[record.businessType ?? 0] ?? '未知'}</Tag>
      ),
    },
    {
      title: '贸易条款',
      dataIndex: 'tradeTerm',
      width: 120,
      render: (_, record) => record.tradeTerm || '通用',
    },
    { title: '编码', dataIndex: 'code', width: 160, copyable: true },
    { title: '名称', dataIndex: 'name', width: 180 },
    {
      title: '版本',
      dataIndex: 'version',
      width: 90,
      search: false,
      render: (_, record) => `v${record.version ?? 1}`,
    },
    {
      title: '发布状态',
      dataIndex: 'published',
      width: 120,
      valueType: 'select',
      valueEnum: { true: { text: '已发布' }, false: { text: '草稿' } },
      render: (_, record) =>
        record.publishedAt ? (
          <Tag color="processing">已发布</Tag>
        ) : (
          <Tag color="default">草稿</Tag>
        ),
    },
    {
      title: '默认状态',
      dataIndex: 'isDefault',
      width: 100,
      search: false,
      render: (_, record) =>
        record.isDefault ? <Tag color="success">默认</Tag> : <Tag>非默认</Tag>,
    },
    {
      title: '节点数量',
      dataIndex: 'items',
      width: 100,
      search: false,
      render: (_, record) => record.items?.length ?? 0,
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
      width: 150,
      render: (_, record) => {
        if (!access.canManageMasterData) {
          return null;
        }
        const isPublished = Boolean(record.publishedAt);
        const isDefault = Boolean(record.isDefault);

        if (!isPublished) {
          return (
            <Button
              type="link"
              size="small"
              onClick={() => {
                setPublishingItem(record);
                setPublishModalOpen(true);
              }}
            >
              发布
            </Button>
          );
        }

        if (!isDefault) {
          return (
            <Popconfirm
              title="确定设为默认版本？"
              onConfirm={() => handleSetDefault(record)}
            >
              <Button type="link" size="small" icon={<CheckOutlined />}>
                设为默认
              </Button>
            </Popconfirm>
          );
        }

        return null;
      },
    },
  ];

  return (
    <>
      <ProTable<
        API.MilestoneTemplate,
        API.MasterDataServiceListMilestoneTemplatesParams
      >
        headerTitle="里程碑模板"
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        pagination={false}
        request={async (params) => {
          const response = await masterDataServiceListMilestoneTemplates({
            businessType: params.businessType,
            tradeTerm: params.tradeTerm,
            published: params.published,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
          };
        }}
        toolBarRender={() =>
          [
            <Button
              key="refresh"
              icon={<ReloadOutlined />}
              onClick={() => actionRef.current?.reload()}
            >
              刷新
            </Button>,
            access.canManageMasterData ? (
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={openCreate}
              >
                新增里程碑模板
              </Button>
            ) : null,
          ].filter(Boolean) as React.ReactNode[]
        }
      />

      <ModalForm<MilestoneTemplateFormValues>
        title="新增里程碑模板"
        open={modalOpen}
        formRef={formRef}
        initialValues={{
          businessType: 1,
          version: 1,
          items: [
            {
              code: 'ORDER_CREATED',
              label: '订单创建',
              sortOrder: 10,
              enabled: true,
            },
          ],
        }}
        modalProps={{
          destroyOnClose: true,
          width: 900,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          const rawItems = values.items ?? [];
          const items: API.MilestoneTemplateItemInput[] = rawItems.map(
            (item, index) => ({
              code: item.code ?? '',
              label: item.label ?? '',
              description: item.description,
              category: item.category,
              sortOrder: item.sortOrder ?? (index + 1) * 10,
              enabled: item.enabled ?? true,
              dependsOn: item.dependsOn ?? [],
            }),
          );

          await masterDataServiceCreateMilestoneTemplate({
            code: values.code ?? '',
            name: values.name ?? '',
            businessType: values.businessType ?? 1,
            tradeTerm: values.tradeTerm,
            version: values.version ?? 1,
            items,
          });
          message.success('里程碑模板已创建');
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="编码"
          placeholder="如 MS_SE_FOB"
          rules={[{ required: true, message: '请输入模板编码' }]}
        />
        <ProFormText
          name="name"
          label="名称"
          placeholder="如 海运出口 FOB 里程碑流程"
          rules={[{ required: true, message: '请输入模板名称' }]}
        />
        <ProFormSelect
          name="businessType"
          label="业务类型"
          options={businessTypeOptions}
          rules={[{ required: true, message: '请选择业务类型' }]}
        />
        <ProFormText
          name="tradeTerm"
          label="贸易条款"
          placeholder="如 FOB、CIF（可选）"
        />
        <ProFormDigit
          name="version"
          label="版本"
          min={1}
          fieldProps={{ precision: 0 }}
          rules={[{ required: true, message: '请输入版本号' }]}
        />
        <ProFormList
          name="items"
          label="里程碑节点列表"
          creatorButtonProps={{
            creatorButtonText: '添加里程碑节点',
          }}
          min={1}
          rules={[
            {
              validator: async (_, value) => {
                if (!value || value.length === 0) {
                  return Promise.reject(new Error('至少需要一个里程碑节点'));
                }
                const hasEnabled = value.some(
                  (item: { enabled?: boolean }) => item?.enabled ?? true,
                );
                if (!hasEnabled) {
                  return Promise.reject(
                    new Error('里程碑节点中至少需要一个启用的节点'),
                  );
                }
                return Promise.resolve();
              },
            },
          ]}
        >
          {() => (
            <Space direction="horizontal" align="start" size={8} wrap>
              <ProFormText
                name="code"
                label="节点编码"
                width="sm"
                placeholder="如 CUSTOMS_CLEARED"
                rules={[{ required: true, message: '请输入节点编码' }]}
              />
              <ProFormText
                name="label"
                label="节点名称"
                width="sm"
                placeholder="如 报关放行"
                rules={[{ required: true, message: '请输入节点名称' }]}
              />
              <ProFormText
                name="category"
                label="分类"
                width="xs"
                placeholder="如 CUSTOMS"
              />
              <ProFormSelect
                name="dependsOn"
                label="依赖前置编码"
                width="sm"
                placeholder="输入编码后回车"
                fieldProps={{ mode: 'tags', tokenSeparators: [','] }}
              />
              <ProFormDigit
                name="sortOrder"
                label="排序"
                width="xs"
                min={0}
                fieldProps={{ precision: 0 }}
              />
              <ProFormSwitch name="enabled" label="启用" initialValue={true} />
              <ProFormText
                name="description"
                label="说明"
                width="md"
                placeholder="节点说明（可选）"
              />
            </Space>
          )}
        </ProFormList>
      </ModalForm>

      <ModalForm<PublishFormValues>
        title={`发布里程碑模板 - ${publishingItem?.name ?? ''}`}
        open={publishModalOpen}
        modalProps={{
          destroyOnClose: true,
          onCancel: () => setPublishModalOpen(false),
        }}
        onOpenChange={setPublishModalOpen}
        onFinish={async (values) => {
          if (!publishingItem?.id) return false;
          await masterDataServicePublishMilestoneTemplate(
            { id: publishingItem.id },
            { id: publishingItem.id, isDefault: values.isDefault ?? false },
          );
          message.success('里程碑模板发布成功');
          setPublishModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormRadio.Group
          name="isDefault"
          label="发布后设为默认版本"
          initialValue={false}
          options={[
            { label: '否，仅发布为可用版本', value: false },
            { label: '是，发布并设为默认模板', value: true },
          ]}
        />
      </ModalForm>
    </>
  );
}
