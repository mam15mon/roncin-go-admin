import {
  CheckOutlined,
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
import { App, Button, Card, Popconfirm, Space, Tabs, Tag } from 'antd';
import React, { useRef, useState } from 'react';
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

const kindOptions = [
  { label: '币种', value: 1 },
  { label: '国家', value: 2 },
  { label: '地区', value: 3 },
  { label: '港口', value: 4 },
  { label: '机场', value: 5 },
  { label: '承运人', value: 6 },
  { label: '箱型', value: 7 },
  { label: '服务类型', value: 8 },
  { label: '货物类别', value: 9 },
];

const kindLabels = Object.fromEntries(
  kindOptions.map((item) => [item.value, item.label]),
);

const documentTypeOptions = [
  { label: '订单', value: 1 },
  { label: '订舱', value: 2 },
  { label: 'HBL', value: 3 },
  { label: 'MBL', value: 4 },
  { label: '账单', value: 5 },
  { label: '对账单', value: 6 },
  { label: '付款', value: 7 },
  { label: '发票', value: 8 },
];

const documentTypeLabels = Object.fromEntries(
  documentTypeOptions.map((item) => [item.value, item.label]),
);

const dateFormatOptions = [
  { label: 'yyyyMMdd', value: 1 },
  { label: 'yyyyMM', value: 2 },
  { label: 'yyyy', value: 3 },
  { label: '无日期', value: 4 },
];

const dateFormatLabels = Object.fromEntries(
  dateFormatOptions.map((item) => [item.value, item.label]),
);

const resetPolicyOptions = [
  { label: '每日', value: 1 },
  { label: '每月', value: 2 },
  { label: '每年', value: 3 },
  { label: '永不', value: 4 },
];

const resetPolicyLabels = Object.fromEntries(
  resetPolicyOptions.map((item) => [item.value, item.label]),
);

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
      render: (_, record) => (
        <Tag>{kindLabels[record.kind ?? 0] ?? '未知'}</Tag>
      ),
    },
    { title: '编码', dataIndex: 'code', width: 160, copyable: true },
    { title: '名称', dataIndex: 'name', width: 220, ellipsis: true },
    {
      title: '英文名称',
      dataIndex: 'nameEn',
      width: 220,
      ellipsis: true,
      search: false,
    },
    { title: '上级编码', dataIndex: 'parentCode', width: 140, search: false },
    {
      title: '运输方式',
      dataIndex: 'transportMode',
      width: 120,
      search: false,
    },
    { title: 'TEU 系数', dataIndex: 'teuFactor', width: 110, search: false },
    { title: '来源', dataIndex: 'source', width: 120, search: false },
    { title: '排序', dataIndex: 'sortOrder', width: 80, search: false },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      valueType: 'select',
      valueEnum: { true: { text: '启用' }, false: { text: '停用' } },
      render: (_, record) =>
        record.enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 100,
      render: (_, record) =>
        access.canManageMasterData ? (
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
    <>
      <ProTable<API.MasterDataItem>
        headerTitle="主数据目录"
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        scroll={{ x: 1400 }}
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
        title={editing ? '编辑主数据' : '新增主数据'}
        open={modalOpen}
        formRef={formRef}
        initialValues={{ source: 'manual', sortOrder: 100, ...editing }}
        modalProps={{
          destroyOnClose: true,
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
            message.success('主数据已更新');
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
            message.success('主数据已创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          name="kind"
          label="类型"
          options={kindOptions}
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请选择主数据类型' }]}
        />
        <ProFormText
          name="code"
          label="编码"
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请输入编码' }]}
        />
        <ProFormText
          name="name"
          label="名称"
          rules={[{ required: true, message: '请输入名称' }]}
        />
        <ProFormText name="nameEn" label="英文名称" />
        <ProFormDependency name={['kind']}>
          {({ kind }) =>
            kind === 3 || kind === 4 || kind === 5 ? (
              <ProFormText name="parentCode" label="上级编码" />
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
                  { label: '海运', value: 'SEA' },
                  { label: '空运', value: 'AIR' },
                  { label: '陆运', value: 'LAND' },
                  { label: '铁路', value: 'RAIL' },
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
                label="TEU 系数"
                rules={[
                  { pattern: /^\d+(\.\d+)?$/, message: '请输入大于 0 的数字' },
                ]}
              />
            ) : null
          }
        </ProFormDependency>
        <ProFormText
          name="source"
          label="来源"
          rules={[{ required: true, message: '请输入来源' }]}
        />
        <ProFormDigit
          name="sortOrder"
          label="排序"
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
      width: 140,
      render: (_, record) => (
        <Tag>{documentTypeLabels[record.documentType ?? 0] ?? '未知'}</Tag>
      ),
    },
    { title: '前缀', dataIndex: 'prefix', width: 120 },
    {
      title: '日期格式',
      dataIndex: 'dateFormat',
      width: 140,
      render: (_, record) => dateFormatLabels[record.dateFormat ?? 0] ?? '未知',
    },
    { title: '序号长度', dataIndex: 'sequenceLength', width: 100 },
    {
      title: '重置周期',
      dataIndex: 'resetPolicy',
      width: 120,
      render: (_, record) =>
        resetPolicyLabels[record.resetPolicy ?? 0] ?? '未知',
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      render: (_, record) =>
        record.enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
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
      width: 100,
      render: (_, record) =>
        access.canManageMasterData ? (
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
    <>
      <ProTable<API.NumberRule>
        headerTitle="编号规则"
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
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
        title={editing ? '编辑编号规则' : '新增编号规则'}
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
            message.success('编号规则已更新');
          } else {
            await masterDataServiceCreateNumberRule({
              documentType: values.documentType ?? 1,
              prefix: values.prefix ?? '',
              dateFormat: values.dateFormat ?? 1,
              sequenceLength: values.sequenceLength ?? 4,
              resetPolicy: values.resetPolicy ?? 1,
            });
            message.success('编号规则已创建');
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
        <ProFormText name="prefix" label="前缀" placeholder="如 ORD、BKG" />
        <ProFormSelect
          name="dateFormat"
          label="日期格式"
          options={dateFormatOptions}
          rules={[{ required: true, message: '请选择日期格式' }]}
        />
        <ProFormDigit
          name="sequenceLength"
          label="序号长度"
          min={1}
          max={12}
          fieldProps={{ precision: 0 }}
          rules={[{ required: true, message: '请输入序号长度(1-12)' }]}
        />
        <ProFormSelect
          name="resetPolicy"
          label="重置周期"
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
      render: (_, record) => (
        <Tag>{businessTypeLabels[record.businessType ?? 0] ?? '未知'}</Tag>
      ),
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
      title: '状态数量',
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
        API.StatusTemplate,
        API.MasterDataServiceListStatusTemplatesParams
      >
        headerTitle="状态模板"
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        pagination={false}
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
        title="新增状态模板"
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
          width: 840,
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
          message.success('状态模板已创建');
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="编码"
          placeholder="如 SE_DEFAULT"
          rules={[{ required: true, message: '请输入模板编码' }]}
        />
        <ProFormText
          name="name"
          label="名称"
          placeholder="如 海运出口默认状态流程"
          rules={[{ required: true, message: '请输入模板名称' }]}
        />
        <ProFormSelect
          name="businessType"
          label="业务类型"
          options={businessTypeOptions}
          rules={[{ required: true, message: '请选择业务类型' }]}
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
          label="状态条目列表"
          creatorButtonProps={{
            creatorButtonText: '添加状态条目',
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
              <Space direction="horizontal" align="start" size={8} wrap>
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
          onCancel: () => setPublishModalOpen(false),
        }}
        onOpenChange={setPublishModalOpen}
        onFinish={async (values) => {
          if (!publishingItem?.id) return false;
          await masterDataServicePublishStatusTemplate(
            { id: publishingItem.id },
            { id: publishingItem.id, isDefault: values.isDefault ?? false },
          );
          message.success('状态模板发布成功');
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
            { label: '是，发布并设为该业务类型的默认模板', value: true },
          ]}
        />
      </ModalForm>
    </>
  );
}

export default function MasterDataPage() {
  const tabItems = [
    {
      key: 'catalog',
      label: '主数据目录',
      children: <MasterDataCatalogPanel />,
    },
    {
      key: 'number-rules',
      label: '编号规则',
      children: <NumberRulesPanel />,
    },
    {
      key: 'status-templates',
      label: '状态模板',
      children: <StatusTemplatesPanel />,
    },
    {
      key: 'milestone-templates',
      label: '里程碑模板',
      children: <MilestoneTemplatesPanel />,
    },
  ];

  return (
    <Card>
      <Tabs items={tabItems} />
    </Card>
  );
}
