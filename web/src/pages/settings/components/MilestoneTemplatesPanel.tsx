import { CheckOutlined, FlagOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
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
import { App, Button, Card, Popconfirm, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import {
  masterDataServiceCreateMilestoneTemplate,
  masterDataServiceListMilestoneTemplates,
  masterDataServicePublishMilestoneTemplate,
  masterDataServiceSetDefaultMilestoneTemplate,
} from '@/services/roncin/masterDataService';

const { Text } = Typography;

const businessTypeOptions = [
  { label: '海运出口', value: 1, color: 'blue' },
  { label: '海运进口', value: 2, color: 'cyan' },
  { label: '空运出口', value: 3, color: 'geekblue' },
  { label: '空运进口', value: 4, color: 'purple' },
  { label: '陆运', value: 5, color: 'green' },
  { label: '铁路', value: 6, color: 'volcano' },
];

const businessTypeMap = new Map(
  businessTypeOptions.map((item) => [item.value, item]),
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

export function MilestoneTemplatesPanel() {
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
    message.success('已成功设为该业务类型的默认里程碑模板');
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
      render: (_, record) => {
        const item = businessTypeMap.get(record.businessType ?? 0);
        return item ? (
          <Tag color={item.color} bordered={false}>
            {item.label}
          </Tag>
        ) : (
          <Tag>未知</Tag>
        );
      },
    },
    {
      title: '贸易条款',
      dataIndex: 'tradeTerm',
      width: 120,
      render: (_, record) =>
        record.tradeTerm ? (
          <Tag color="orange" bordered={false}>
            {record.tradeTerm}
          </Tag>
        ) : (
          <Tag bordered={false}>通用条款</Tag>
        ),
    },
    {
      title: '模板编码',
      dataIndex: 'code',
      width: 160,
      copyable: true,
      render: (code) => (
        <Text style={{ fontFamily: 'monospace', fontWeight: 500 }}>
          {code}
        </Text>
      ),
    },
    {
      title: '模板名称',
      dataIndex: 'name',
      width: 200,
      render: (name) => <Text strong>{name}</Text>,
    },
    {
      title: '版本号',
      dataIndex: 'version',
      width: 90,
      search: false,
      render: (_, record) => (
        <Tag color="geekblue" bordered={false}>
          v{record.version ?? 1}
        </Tag>
      ),
    },
    {
      title: '发布状态',
      dataIndex: 'published',
      width: 110,
      valueType: 'select',
      valueEnum: { true: { text: '已发布' }, false: { text: '草稿' } },
      render: (_, record) =>
        record.publishedAt ? (
          <Tag color="success">已发布</Tag>
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
        record.isDefault ? (
          <Tag color="blue">默认模板</Tag>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '节点数量',
      dataIndex: 'items',
      width: 110,
      search: false,
      render: (_, record) => `${record.items?.length ?? 0} 个里程碑`,
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
      fixed: 'right',
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
              发布模板
            </Button>
          );
        }

        if (!isDefault) {
          return (
            <Popconfirm
              title="确定设为该业务类型的默认里程碑模板？"
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
    <Card
      bordered={false}
      style={{
        borderRadius: 8,
        border: '1px solid #f0f0f0',
        backgroundColor: '#ffffff',
      }}
      styles={{ body: { padding: '12px 16px' } }}
    >
      <ProTable<
        API.MilestoneTemplate,
        API.MasterDataServiceListMilestoneTemplatesParams
      >
        headerTitle={
          <Space size={8}>
            <FlagOutlined style={{ color: '#1677ff' }} />
            <span>订单履约里程碑模板列表</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        pagination={false}
        expandable={{
          expandedRowRender: (record) => (
            <div style={{ padding: '8px 16px', backgroundColor: '#f8fafc', borderRadius: 6 }}>
              <Text strong style={{ fontSize: 12, color: '#475569', display: 'block', marginBottom: 8 }}>
                包含的履约里程碑流水节点：
              </Text>
              <Space wrap size={[6, 6]}>
                {(record.items ?? []).map((item, idx) => (
                  <Tag
                    key={item.code}
                    color="blue"
                    bordered={false}
                    style={{ fontSize: 12, padding: '2px 8px' }}
                  >
                    {idx + 1}. {item.label} ({item.code})
                    {item.category ? ` · [${item.category}]` : ''}
                  </Tag>
                ))}
              </Space>
            </div>
          ),
        }}
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
        title="新增履约里程碑模板"
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
          width: 920,
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
          message.success('里程碑模板已成功创建');
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="模板编码"
          placeholder="例如：MS_SE_FOB"
          rules={[{ required: true, message: '请输入模板编码' }]}
        />
        <ProFormText
          name="name"
          label="模板名称"
          placeholder="例如：海运出口 FOB 履约里程碑流程"
          rules={[{ required: true, message: '请输入模板名称' }]}
        />
        <ProFormSelect
          name="businessType"
          label="适用业务类型"
          options={businessTypeOptions}
          rules={[{ required: true, message: '请选择业务类型' }]}
        />
        <ProFormText
          name="tradeTerm"
          label="贸易条款限制"
          placeholder="如 FOB、CIF（留空则适用于通用贸易条款）"
        />
        <ProFormDigit
          name="version"
          label="版本号"
          min={1}
          fieldProps={{ precision: 0 }}
          rules={[{ required: true, message: '请输入版本号' }]}
        />
        <ProFormList
          name="items"
          label="里程碑节点流水列表（按时序排列）"
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
            <Space direction="horizontal" align="start" size={10} wrap>
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
                label="业务分类"
                width="xs"
                placeholder="如 CUSTOMS"
              />
              <ProFormSelect
                name="dependsOn"
                label="前置依赖编码"
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
                label="节点说明"
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
          width: 520,
          onCancel: () => setPublishModalOpen(false),
        }}
        onOpenChange={setPublishModalOpen}
        onFinish={async (values) => {
          if (!publishingItem?.id) return false;
          await masterDataServicePublishMilestoneTemplate(
            { id: publishingItem.id },
            { id: publishingItem.id, isDefault: values.isDefault ?? false },
          );
          message.success('里程碑模板已成功发布');
          setPublishModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormRadio.Group
          name="isDefault"
          label="发布后是否设为默认版本"
          initialValue={false}
          options={[
            { label: '否，仅发布为可用版本', value: false },
            { label: '是，发布并设为默认里程碑模板', value: true },
          ]}
        />
      </ModalForm>
    </Card>
  );
}

export default MilestoneTemplatesPanel;
