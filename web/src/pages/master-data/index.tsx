import { EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDependency,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
  masterDataServiceUpdateItem,
} from '@/services/roncin/masterDataService';

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

const kindLabels = Object.fromEntries(kindOptions.map((item) => [item.value, item.label]));

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

export default function MasterDataPage() {
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
      valueEnum: Object.fromEntries(kindOptions.map((item) => [item.value, { text: item.label }])),
      render: (_, record) => <Tag>{kindLabels[record.kind ?? 0] ?? '未知'}</Tag>,
    },
    { title: '编码', dataIndex: 'code', width: 160, copyable: true },
    { title: '名称', dataIndex: 'name', width: 220, ellipsis: true },
    { title: '英文名称', dataIndex: 'nameEn', width: 220, ellipsis: true, search: false },
    { title: '上级编码', dataIndex: 'parentCode', width: 140, search: false },
    { title: '运输方式', dataIndex: 'transportMode', width: 120, search: false },
    { title: 'TEU 系数', dataIndex: 'teuFactor', width: 110, search: false },
    { title: '来源', dataIndex: 'source', width: 120, search: false },
    { title: '排序', dataIndex: 'sortOrder', width: 80, search: false },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 100,
      valueType: 'select',
      valueEnum: { true: { text: '启用' }, false: { text: '停用' } },
      render: (_, record) => record.enabled ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 100,
      render: (_, record) => access.canManageMasterData ? <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>编辑</Button> : null,
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
          const response = await masterDataServiceListItems({ page: params.current, pageSize: params.pageSize, kind: params.kind, keyword: params.keyword, enabled: params.enabled });
          return { data: response.data ?? [], success: response.success ?? true, total: response.total ?? 0 };
        }}
        toolBarRender={() => [
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>刷新</Button>,
          access.canManageMasterData ? <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增主数据</Button> : null,
        ].filter(Boolean) as React.ReactNode[]}
      />
      <ModalForm<MasterDataFormValues>
        title={editing ? '编辑主数据' : '新增主数据'}
        open={modalOpen}
        formRef={formRef}
        initialValues={{ source: 'manual', sortOrder: 100, ...editing }}
        modalProps={{ destroyOnClose: true, onCancel: () => setModalOpen(false) }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editing?.id) {
            await masterDataServiceUpdateItem({ id: editing.id }, { id: editing.id, kind: values.kind ?? 0, name: values.name ?? '', nameEn: values.nameEn, parentCode: values.parentCode, transportMode: values.transportMode, teuFactor: values.teuFactor, source: values.source, sortOrder: values.sortOrder, enabled: values.enabled ?? true });
            message.success('主数据已更新');
          } else {
            await masterDataServiceCreateItem({ kind: values.kind ?? 0, code: values.code ?? '', name: values.name ?? '', nameEn: values.nameEn, parentCode: values.parentCode, transportMode: values.transportMode, teuFactor: values.teuFactor, source: values.source, sortOrder: values.sortOrder });
            message.success('主数据已创建');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect name="kind" label="类型" options={kindOptions} disabled={Boolean(editing)} rules={[{ required: true, message: '请选择主数据类型' }]} />
        <ProFormText name="code" label="编码" disabled={Boolean(editing)} rules={[{ required: true, message: '请输入编码' }]} />
        <ProFormText name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]} />
        <ProFormText name="nameEn" label="英文名称" />
        <ProFormDependency name={['kind']}>
          {({ kind }) => kind === 3 || kind === 4 || kind === 5 ? <ProFormText name="parentCode" label="上级编码" /> : null}
        </ProFormDependency>
        <ProFormDependency name={['kind']}>
          {({ kind }) => kind === 6 ? <ProFormSelect name="transportMode" label="运输方式" options={[{ label: '海运', value: 'SEA' }, { label: '空运', value: 'AIR' }, { label: '陆运', value: 'LAND' }, { label: '铁路', value: 'RAIL' }]} /> : null}
        </ProFormDependency>
        <ProFormDependency name={['kind']}>
          {({ kind }) => kind === 7 ? <ProFormText name="teuFactor" label="TEU 系数" rules={[{ pattern: /^\d+(\.\d+)?$/, message: '请输入大于 0 的数字' }]} /> : null}
        </ProFormDependency>
        <ProFormText name="source" label="来源" rules={[{ required: true, message: '请输入来源' }]} />
        <ProFormDigit name="sortOrder" label="排序" min={0} fieldProps={{ precision: 0 }} />
        {editing ? <ProFormSwitch name="enabled" label="启用状态" /> : null}
      </ModalForm>
    </>
  );
}
