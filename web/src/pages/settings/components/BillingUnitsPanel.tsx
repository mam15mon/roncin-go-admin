import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Card, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  feeCatalogServiceCreateBillingUnit,
  feeCatalogServiceListBillingUnits,
  feeCatalogServiceUpdateBillingUnit,
} from '@/services/roncin/feeCatalogService';

const codePattern = /^[A-Z0-9_]{2,32}$/;

type BillingUnitFormValues = {
  code: string;
  name: string;
  isContainerUnit: boolean;
  enabled: boolean;
  sortOrder: number;
};

const enabledColumn = <T extends { enabled?: boolean }>(): ProColumns<T> => ({
  title: '状态',
  dataIndex: 'enabled',
  width: 80,
  render: (_, record) =>
    record.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
});

export function BillingUnitsPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.BillingUnit>();

  const columns: ProColumns<API.BillingUnit>[] = [
    { title: '单位代码', dataIndex: 'code', width: 160, copyable: true },
    { title: '单位名称', dataIndex: 'name' },
    {
      title: '是否为箱型单位',
      dataIndex: 'isContainerUnit',
      width: 140,
      render: (_, record) => (
        <Tag color={record.isContainerUnit ? 'blue' : 'default'}>
          {record.isContainerUnit ? '箱型计量单位' : '常规计量单位'}
        </Tag>
      ),
    },
    { title: '排序', dataIndex: 'sortOrder', width: 90 },
    enabledColumn<API.BillingUnit>(),
    {
      title: '操作',
      valueType: 'option',
      width: 90,
      render: (_, record) =>
        access.canUpdateFeeSettings ? (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditing(record);
              setModalOpen(true);
            }}
          >
            编辑
          </Button>
        ) : null,
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
      <ProTable<API.BillingUnit>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        pagination={false}
        request={async () => {
          const response = await feeCatalogServiceListBillingUnits();
          return {
            data: response.data ?? [],
            success: response.success ?? true,
          };
        }}
        toolBarRender={() =>
          access.canCreateFeeSettings
            ? [
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setEditing(undefined);
                    setModalOpen(true);
                  }}
                >
                  新建计费单位
                </Button>,
              ]
            : []
        }
      />
      <ModalForm<BillingUnitFormValues>
        title={editing ? '编辑计费单位' : '新建计费单位'}
        open={modalOpen}
        initialValues={
          editing
            ? {
                code: editing.code,
                name: editing.name,
                isContainerUnit: editing.isContainerUnit ?? false,
                sortOrder: editing.sortOrder ?? 100,
                enabled: editing.enabled ?? true,
              }
            : { isContainerUnit: false, sortOrder: 100, enabled: true }
        }
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          const input = {
            code: values.code.trim().toUpperCase(),
            name: values.name.trim(),
            isContainerUnit: values.isContainerUnit,
            sortOrder: values.sortOrder,
          };
          if (editing?.id) {
            await feeCatalogServiceUpdateBillingUnit(
              { id: editing.id },
              { id: editing.id, ...input, enabled: values.enabled },
            );
            message.success('计费单位更新成功');
          } else {
            await feeCatalogServiceCreateBillingUnit(input);
            message.success('计费单位创建成功');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="单位代码"
          disabled={Boolean(editing)}
          rules={[
            { required: true, message: '请输入单位代码' },
            {
              pattern: codePattern,
              message: '请输入 2–32 位大写字母、数字或下划线',
            },
          ]}
        />
        <ProFormText
          name="name"
          label="单位名称"
          rules={[{ required: true, message: '请输入单位名称' }]}
          fieldProps={{ maxLength: 64 }}
        />
        <ProFormDigit
          name="sortOrder"
          label="排序"
          min={0}
          fieldProps={{ precision: 0 }}
        />
        <ProFormSwitch
          name="isContainerUnit"
          label="是否为箱型单位"
          extra="开启后该单位将作为 20GP/40HQ 等集装箱计量基准"
        />
        {editing && <ProFormSwitch name="enabled" label="启用状态" />}
      </ModalForm>
    </Card>
  );
}

export default BillingUnitsPanel;
