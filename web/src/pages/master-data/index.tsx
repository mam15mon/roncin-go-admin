import {
  CheckOutlined,
  DatabaseOutlined,
  EditOutlined,
  FieldTimeOutlined,
  FlagOutlined,
  NodeIndexOutlined,
  NumberOutlined,
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
  ProFormDependency,
  ProFormDigit,
  ProFormList,
  ProFormRadio,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Popconfirm, Space, Tag, Typography } from 'antd';
import React, { useMemo, useRef, useState } from 'react';
import {
  masterDataServiceCreateItem,
  masterDataServiceCreateNumberRule,
  masterDataServiceCreateStatusTemplate,
  masterDataServiceListItems,
  masterDataServiceListNumberRules,
  masterDataServiceListStatusTemplates,
  masterDataServicePublishStatusTemplate,
  masterDataServiceSetDefaultStatusTemplate,
  masterDataServiceUpdateItem,
  masterDataServiceUpdateNumberRule,
} from '@/services/roncin/masterDataService';
import MilestoneTemplatesPanel from './milestone-templates-panel';

const { Text } = Typography;

const kindOptions = [
  { label: '币种', value: 1, color: 'gold' },
  { label: '国家', value: 2, color: 'blue' },
  { label: '地区', value: 3, color: 'cyan' },
  { label: '港口', value: 4, color: 'geekblue' },
  { label: '机场', value: 5, color: 'purple' },
  { label: '承运人', value: 6, color: 'orange' },
  { label: '箱型', value: 7, color: 'magenta' },
  { label: '服务类型', value: 8, color: 'green' },
  { label: '货物类别', value: 9, color: 'volcano' },
];

const kindMap = new Map(kindOptions.map((item) => [item.value, item]));

const documentTypeOptions = [
  { label: '订单', value: 1, color: 'blue' },
  { label: '订舱', value: 2, color: 'cyan' },
  { label: 'HBL (分单)', value: 3, color: 'purple' },
  { label: 'MBL (主单)', value: 4, color: 'geekblue' },
  { label: '账单', value: 5, color: 'orange' },
  { label: '对账单', value: 6, color: 'gold' },
  { label: '付款', value: 7, color: 'green' },
  { label: '发票', value: 8, color: 'volcano' },
];

const documentTypeMap = new Map(
  documentTypeOptions.map((item) => [item.value, item]),
);

const dateFormatOptions = [
  { label: 'yyyyMMdd (年月日)', value: 1 },
  { label: 'yyyyMM (年月)', value: 2 },
  { label: 'yyyy (年)', value: 3 },
  { label: '无日期', value: 4 },
];

const dateFormatLabels = Object.fromEntries(
  dateFormatOptions.map((item) => [item.value, item.label]),
);

const resetPolicyOptions = [
  { label: '每日重置', value: 1 },
  { label: '每月重置', value: 2 },
  { label: '每年重置', value: 3 },
  { label: '永不重置', value: 4 },
];

const resetPolicyLabels = Object.fromEntries(
  resetPolicyOptions.map((item) => [item.value, item.label]),
);

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

type MasterDataFormValues = {
  kind?: number;
  code?: string;
  name?: string;
  nameEn?: string;
  parentCode?: string;
  transportMode?: string;
  teuFactor?: string;
  source?: string;
  sortOrder?: number;
  enabled?: boolean;
};

type NumberRuleFormValues = {
  documentType?: number;
  prefix?: string;
  dateFormat?: number;
  sequenceLength?: number;
  resetPolicy?: number;
  enabled?: boolean;
};

type StatusTemplateFormValues = {
  code?: string;
  name?: string;
  businessType?: number;
  version?: number;
  items?: API.StatusTemplateItemInput[];
};

type PublishFormValues = {
  isDefault?: boolean;
};

function MasterDataCatalogPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const access = useAccess();
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.MasterDataItem>();

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (item: API.MasterDataItem) => {
    setEditing(item);
    setModalOpen(true);
  };

  const columns: ProColumns<API.MasterDataItem>[] = [
    {
      title: '类型',
      dataIndex: 'kind',
      width: 130,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        kindOptions.map((item) => [item.value, { text: item.label }]),
      ),
      render: (_, record) => {
        const item = kindMap.get(record.kind ?? 0);
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
      title: '编码',
      dataIndex: 'code',
      width: 150,
      copyable: true,
      render: (_, record) => (
        <Text style={{ fontFamily: 'monospace', fontWeight: 500 }}>
          {record.code}
        </Text>
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 200,
      ellipsis: true,
      render: (_, record) => <Text strong>{record.name}</Text>,
    },
    {
      title: '英文名称',
      dataIndex: 'nameEn',
      width: 200,
      ellipsis: true,
      search: false,
      render: (_, record) => record.nameEn || <Text type="secondary">-</Text>,
    },
    {
      title: '上级编码',
      dataIndex: 'parentCode',
      width: 130,
      search: false,
      render: (_, record) =>
        record.parentCode ? (
          <Tag bordered={false} style={{ fontFamily: 'monospace' }}>
            {record.parentCode}
          </Tag>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '运输方式',
      dataIndex: 'transportMode',
      width: 110,
      search: false,
      render: (_, record) =>
        record.transportMode ? (
          <Tag color="blue" bordered={false}>
            {record.transportMode}
          </Tag>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: 'TEU 系数',
      dataIndex: 'teuFactor',
      width: 100,
      search: false,
      render: (_, record) => record.teuFactor || <Text type="secondary">-</Text>,
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 110,
      search: false,
      render: (_, record) => (
        <Tag bordered={false} style={{ fontSize: 11 }}>
          {record.source || 'manual'}
        </Tag>
      ),
    },
    {
      title: '排序',
      dataIndex: 'sortOrder',
      width: 80,
      search: false,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 90,
      valueType: 'select',
      valueEnum: { true: { text: '启用' }, false: { text: '停用' } },
      render: (_, record) =>
        record.enabled ? (
          <Tag color="success">启用</Tag>
        ) : (
          <Tag color="default">停用</Tag>
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 90,
      fixed: 'right',
      render: (_, record) =>
        access.canManageMasterData ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            style={{ padding: 0 }}
            onClick={() => openEdit(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  return (
    <>
      <ProTable<API.MasterDataItem>
        headerTitle={
          <Space size={8}>
            <DatabaseOutlined style={{ color: '#1677ff' }} />
            <span>主数据目录列表</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        scroll={{ x: 1300 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        request={async (params) => {
          const response = await masterDataServiceListItems({
            page: params.current,
            pageSize: params.pageSize,
            kind: params.kind,
            keyword: params.keyword,
            enabled: params.enabled,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
            total: response.total ?? 0,
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
                新增主数据
              </Button>
            ) : null,
          ].filter(Boolean) as React.ReactNode[]
        }
      />
      <ModalForm<MasterDataFormValues>
        title={editing ? `编辑主数据：${editing.name} (${editing.code})` : '新增主数据'}
        open={modalOpen}
        formRef={formRef}
        initialValues={{ source: 'manual', sortOrder: 100, ...editing }}
        modalProps={{
          destroyOnClose: true,
          width: 600,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editing?.id) {
            await masterDataServiceUpdateItem(
              { id: editing.id },
              {
                id: editing.id,
                kind: values.kind ?? 0,
                name: values.name ?? '',
                nameEn: values.nameEn,
                parentCode: values.parentCode,
                transportMode: values.transportMode,
                teuFactor: values.teuFactor,
                source: values.source,
                sortOrder: values.sortOrder,
                enabled: values.enabled ?? true,
              },
            );
            message.success('主数据已成功更新');
          } else {
            await masterDataServiceCreateItem({
              kind: values.kind ?? 0,
              code: values.code ?? '',
              name: values.name ?? '',
              nameEn: values.nameEn,
              parentCode: values.parentCode,
              transportMode: values.transportMode,
              teuFactor: values.teuFactor,
              source: values.source,
              sortOrder: values.sortOrder,
            });
            message.success('主数据已成功创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          name="kind"
          label="主数据类型"
          options={kindOptions}
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请选择主数据类型' }]}
        />
        <ProFormText
          name="code"
          label="字典编码"
          disabled={Boolean(editing)}
          placeholder="例如：CN_SHA 或 USD"
          rules={[{ required: true, message: '请输入编码' }]}
        />
        <ProFormText
          name="name"
          label="中文名称"
          placeholder="例如：上海港 或 美元"
          rules={[{ required: true, message: '请输入名称' }]}
        />
        <ProFormText name="nameEn" label="英文名称" placeholder="例如：Port of Shanghai" />
        <ProFormDependency name={['kind']}>
          {({ kind }) =>
            kind === 3 || kind === 4 || kind === 5 ? (
              <ProFormText name="parentCode" label="所属上级国家/地区编码" placeholder="例如：CN" />
            ) : null
          }
        </ProFormDependency>
        <ProFormDependency name={['kind']}>
          {({ kind }) =>
            kind === 6 ? (
              <ProFormSelect
                name="transportMode"
                label="运输方式"
                options={[
                  { label: '海运 (SEA)', value: 'SEA' },
                  { label: '空运 (AIR)', value: 'AIR' },
                  { label: '陆运 (LAND)', value: 'LAND' },
                  { label: '铁路 (RAIL)', value: 'RAIL' },
                ]}
              />
            ) : null
          }
        </ProFormDependency>
        <ProFormDependency name={['kind']}>
          {({ kind }) =>
            kind === 7 ? (
              <ProFormText
                name="teuFactor"
                label="TEU 折算系数"
                placeholder="例如：1.0 或 2.0"
                rules={[
                  { pattern: /^\d+(\.\d+)?$/, message: '请输入大于 0 的数字' },
                ]}
              />
            ) : null
          }
        </ProFormDependency>
        <ProFormText
          name="source"
          label="数据来源标识"
          placeholder="例如：manual / unlocode / cbrc"
          rules={[{ required: true, message: '请输入来源' }]}
        />
        <ProFormDigit
          name="sortOrder"
          label="显示排序"
          min={0}
          fieldProps={{ precision: 0 }}
        />
        {editing ? <ProFormSwitch name="enabled" label="启用状态" /> : null}
      </ModalForm>
    </>
  );
}

function NumberRulesPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const access = useAccess();
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.NumberRule>();

  const openCreate = () => {
    setEditing(undefined);
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const openEdit = (rule: API.NumberRule) => {
    setEditing(rule);
    setModalOpen(true);
  };

  const columns: ProColumns<API.NumberRule>[] = [
    {
      title: '单据类型',
      dataIndex: 'documentType',
      width: 160,
      render: (_, record) => {
        const item = documentTypeMap.get(record.documentType ?? 0);
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
      title: '前缀',
      dataIndex: 'prefix',
      width: 120,
      render: (prefix) => (
        <Text style={{ fontFamily: 'monospace', fontWeight: 600 }}>
          {prefix}
        </Text>
      ),
    },
    {
      title: '日期格式',
      dataIndex: 'dateFormat',
      width: 160,
      render: (_, record) => (
        <Tag bordered={false} style={{ fontFamily: 'monospace' }}>
          {dateFormatLabels[record.dateFormat ?? 0] ?? '未知'}
        </Tag>
      ),
    },
    {
      title: '流水号长度',
      dataIndex: 'sequenceLength',
      width: 120,
      render: (len) => `${len ?? 4} 位`,
    },
    {
      title: '重置周期',
      dataIndex: 'resetPolicy',
      width: 130,
      render: (_, record) => (
        <Tag color="cyan" bordered={false}>
          <FieldTimeOutlined style={{ marginRight: 4 }} />
          {resetPolicyLabels[record.resetPolicy ?? 0] ?? '未知'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      render: (_, record) =>
        record.enabled ? (
          <Tag color="success">启用</Tag>
        ) : (
          <Tag color="default">停用</Tag>
        ),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      valueType: 'dateTime',
      width: 180,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 90,
      fixed: 'right',
      render: (_, record) =>
        access.canManageMasterData ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            style={{ padding: 0 }}
            onClick={() => openEdit(record)}
          >
            编辑
          </Button>
        ) : null,
    },
  ];

  return (
    <>
      <ProTable<API.NumberRule>
        headerTitle={
          <Space size={8}>
            <NumberOutlined style={{ color: '#1677ff' }} />
            <span>单据自动编号生成规则</span>
          </Space>
        }
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        bordered
        search={false}
        pagination={false}
        request={async () => {
          const response = await masterDataServiceListNumberRules();
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
                新增编号规则
              </Button>
            ) : null,
          ].filter(Boolean) as React.ReactNode[]
        }
      />
      <ModalForm<NumberRuleFormValues>
        title={editing ? '编辑编号生成规则' : '新增编号生成规则'}
        open={modalOpen}
        formRef={formRef}
        initialValues={{
          dateFormat: 1,
          sequenceLength: 4,
          resetPolicy: 1,
          enabled: true,
          ...editing,
        }}
        modalProps={{
          destroyOnClose: true,
          width: 560,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editing?.id) {
            await masterDataServiceUpdateNumberRule(
              { id: editing.id },
              {
                id: editing.id,
                prefix: values.prefix ?? '',
                dateFormat: values.dateFormat ?? 1,
                sequenceLength: values.sequenceLength ?? 4,
                resetPolicy: values.resetPolicy ?? 1,
                enabled: values.enabled ?? true,
              },
            );
            message.success('编号规则已成功更新');
          } else {
            await masterDataServiceCreateNumberRule({
              documentType: values.documentType ?? 1,
              prefix: values.prefix ?? '',
              dateFormat: values.dateFormat ?? 1,
              sequenceLength: values.sequenceLength ?? 4,
              resetPolicy: values.resetPolicy ?? 1,
            });
            message.success('编号规则已成功创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          name="documentType"
          label="单据类型"
          options={documentTypeOptions}
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请选择单据类型' }]}
        />
        <ProFormText
          name="prefix"
          label="编号前缀"
          placeholder="例如：ORD、BKG、HBL"
          rules={[{ required: true, message: '请输入编号前缀' }]}
        />
        <ProFormSelect
          name="dateFormat"
          label="日期嵌入格式"
          options={dateFormatOptions}
          rules={[{ required: true, message: '请选择日期格式' }]}
        />
        <ProFormDigit
          name="sequenceLength"
          label="自增流水号位数"
          min={1}
          max={12}
          fieldProps={{ precision: 0 }}
          rules={[{ required: true, message: '请输入序号长度(1-12)' }]}
        />
        <ProFormSelect
          name="resetPolicy"
          label="流水计数重置周期"
          options={resetPolicyOptions}
          rules={[{ required: true, message: '请选择重置周期' }]}
        />
        {editing ? <ProFormSwitch name="enabled" label="启用状态" /> : null}
      </ModalForm>
    </>
  );
}

function StatusTemplatesPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const access = useAccess();
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [publishModalOpen, setPublishModalOpen] = useState(false);
  const [publishingItem, setPublishingItem] = useState<API.StatusTemplate>();

  const openCreate = () => {
    formRef.current?.resetFields();
    setModalOpen(true);
  };

  const handleSetDefault = async (record: API.StatusTemplate) => {
    if (!record.id) return;
    await masterDataServiceSetDefaultStatusTemplate(
      { id: record.id },
      { id: record.id },
    );
    message.success('已设为默认版本');
    actionRef.current?.reload();
  };

  const columns: ProColumns<API.StatusTemplate>[] = [
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
      title: '状态条目数',
      dataIndex: 'items',
      width: 110,
      search: false,
      render: (_, record) => `${record.items?.length ?? 0} 个节点`,
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
              title="确定设为该业务类型的默认模板？"
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
        API.StatusTemplate,
        API.MasterDataServiceListStatusTemplatesParams
      >
        headerTitle={
          <Space size={8}>
            <NodeIndexOutlined style={{ color: '#1677ff' }} />
            <span>订单业务状态流转模板</span>
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
                包含的状态节点流水：
              </Text>
              <Space wrap size={[6, 6]}>
                {(record.items ?? []).map((item, idx) => (
                  <Tag
                    key={item.code}
                    color={item.colorToken || 'blue'}
                    bordered={false}
                    style={{ fontSize: 12, padding: '2px 8px' }}
                  >
                    {idx + 1}. {item.label} ({item.code})
                  </Tag>
                ))}
              </Space>
            </div>
          ),
        }}
        request={async (params) => {
          const response = await masterDataServiceListStatusTemplates({
            businessType: params.businessType,
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
                新增状态模板
              </Button>
            ) : null,
          ].filter(Boolean) as React.ReactNode[]
        }
      />

      <ModalForm<StatusTemplateFormValues>
        title="新增订单状态流转模板"
        open={modalOpen}
        formRef={formRef}
        initialValues={{
          businessType: 1,
          version: 1,
          items: [
            {
              code: 'DRAFT',
              label: '草稿',
              sortOrder: 10,
              enabled: true,
              colorToken: 'default',
              system: true,
            },
          ],
        }}
        modalProps={{
          destroyOnClose: true,
          width: 860,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          const rawItems = values.items ?? [];
          const items: API.StatusTemplateItemInput[] = rawItems.map(
            (item, index) => ({
              code: item.code ?? '',
              label: item.label ?? '',
              sortOrder: item.sortOrder ?? (index + 1) * 10,
              enabled: item.enabled ?? true,
              colorToken: item.colorToken,
              system: item.system ?? false,
            }),
          );

          await masterDataServiceCreateStatusTemplate({
            code: values.code ?? '',
            name: values.name ?? '',
            businessType: values.businessType ?? 1,
            version: values.version ?? 1,
            items,
          });
          message.success('状态模板已成功创建');
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="模板编码"
          placeholder="例如：SE_DEFAULT"
          rules={[{ required: true, message: '请输入模板编码' }]}
        />
        <ProFormText
          name="name"
          label="模板名称"
          placeholder="例如：海运出口标准状态流转流程"
          rules={[{ required: true, message: '请输入模板名称' }]}
        />
        <ProFormSelect
          name="businessType"
          label="适用业务类型"
          options={businessTypeOptions}
          rules={[{ required: true, message: '请选择业务类型' }]}
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
          label="状态节点列表（按执行顺序列出）"
          creatorButtonProps={{
            creatorButtonText: '添加状态节点',
          }}
          min={1}
          rules={[
            {
              validator: async (_, value) => {
                if (!value || value.length === 0) {
                  return Promise.reject(new Error('至少需要一个状态条目'));
                }
                const hasDraft = value.some(
                  (item: API.StatusTemplateItemInput) =>
                    item &&
                    item.code?.trim().toUpperCase() === 'DRAFT' &&
                    (item.enabled ?? true),
                );
                if (!hasDraft) {
                  return Promise.reject(
                    new Error('状态条目中必须包含一个启用的 DRAFT 状态'),
                  );
                }
                return Promise.resolve();
              },
            },
          ]}
        >
          <ProFormDependency name={['system']}>
            {() => (
              <Space direction="horizontal" align="start" size={10} wrap>
                <ProFormText
                  name="code"
                  label="状态编码"
                  width="sm"
                  placeholder="如 DRAFT"
                  rules={[{ required: true, message: '请输入状态编码' }]}
                />
                <ProFormText
                  name="label"
                  label="状态名称"
                  width="sm"
                  placeholder="如 草稿"
                  rules={[{ required: true, message: '请输入状态名称' }]}
                />
                <ProFormDigit
                  name="sortOrder"
                  label="排序"
                  width="xs"
                  min={0}
                  fieldProps={{ precision: 0 }}
                />
                <ProFormText
                  name="colorToken"
                  label="颜色标识"
                  width="xs"
                  placeholder="如 success"
                />
                <ProFormSwitch
                  name="system"
                  label="系统状态"
                  initialValue={false}
                />
                <ProFormSwitch
                  name="enabled"
                  label="启用"
                  initialValue={true}
                />
              </Space>
            )}
          </ProFormDependency>
        </ProFormList>
      </ModalForm>

      <ModalForm<PublishFormValues>
        title={`发布状态模板 - ${publishingItem?.name ?? ''}`}
        open={publishModalOpen}
        modalProps={{
          destroyOnClose: true,
          width: 520,
          onCancel: () => setPublishModalOpen(false),
        }}
        onOpenChange={setPublishModalOpen}
        onFinish={async (values) => {
          if (!publishingItem?.id) return false;
          await masterDataServicePublishStatusTemplate(
            { id: publishingItem.id },
            { id: publishingItem.id, isDefault: values.isDefault ?? false },
          );
          message.success('状态模板已成功发布');
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
            { label: '是，发布并立即设为该业务类型的默认模板', value: true },
          ]}
        />
      </ModalForm>
    </>
  );
}

export default function MasterDataPage() {
  const tabItems = useMemo(
    () => [
      {
        key: 'catalog',
        tab: (
          <Space size={6}>
            <DatabaseOutlined />
            <span>主数据目录</span>
          </Space>
        ),
        children: <MasterDataCatalogPanel />,
      },
      {
        key: 'number-rules',
        tab: (
          <Space size={6}>
            <NumberOutlined />
            <span>单据编号规则</span>
          </Space>
        ),
        children: <NumberRulesPanel />,
      },
      {
        key: 'status-templates',
        tab: (
          <Space size={6}>
            <NodeIndexOutlined />
            <span>状态流转模板</span>
          </Space>
        ),
        children: <StatusTemplatesPanel />,
      },
      {
        key: 'milestone-templates',
        tab: (
          <Space size={6}>
            <FlagOutlined />
            <span>履约里程碑模板</span>
          </Space>
        ),
        children: <MilestoneTemplatesPanel />,
      },
    ],
    [],
  );

  const [activeTab, setActiveTab] = useState<string>('catalog');

  const activeContent = tabItems.find((item) => item.key === activeTab)?.children;

  return (
    <PageContainer
      title="主数据管理"
      subTitle="维护物流业务基础字典、单据编号序列规则、状态流转状态机与履约里程碑流水"
      tabList={tabItems.map((item) => ({
        key: item.key,
        tab: item.tab,
      }))}
      tabActiveKey={activeTab}
      onTabChange={(key) => setActiveTab(key)}
    >
      {activeContent}
    </PageContainer>
  );
}
