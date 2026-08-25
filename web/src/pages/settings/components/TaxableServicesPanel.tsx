import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Card, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  feeCatalogServiceCreateTaxableService,
  feeCatalogServiceListTaxableServices,
  feeCatalogServiceUpdateTaxableService,
} from '@/services/roncin/feeCatalogService';

const taxRatePattern =
  /^(100(?:\.0{1,2})?|(?:[0-9]|[1-9][0-9])(?:\.[0-9]{1,2})?)$/;

type TaxableServiceFormValues = {
  name: string;
  shortName?: string;
  goodsCode?: string;
  defaultTaxRate: string;
  enabled: boolean;
};

const enabledColumn = <T extends { enabled?: boolean }>(): ProColumns<T> => ({
  title: '状态',
  dataIndex: 'enabled',
  width: 80,
  render: (_, record) =>
    record.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
});

export function TaxableServicesPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.TaxableService>();

  const columns: ProColumns<API.TaxableService>[] = [
    { title: '货物或应税劳务名称', dataIndex: 'name' },
    { title: '简称', dataIndex: 'shortName', width: 160 },
    { title: '商品编码', dataIndex: 'goodsCode', width: 160 },
    {
      title: '默认税率',
      dataIndex: 'defaultTaxRate',
      width: 110,
      renderText: (value) => `${value ?? '0.00'}%`,
    },
    enabledColumn<API.TaxableService>(),
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
      <ProTable<API.TaxableService>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        pagination={false}
        request={async () => {
          const response = await feeCatalogServiceListTaxableServices();
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
                  新建货物或应税劳务
                </Button>,
              ]
            : []
        }
      />
      <ModalForm<TaxableServiceFormValues>
        title={editing ? '编辑货物或应税劳务' : '新建货物或应税劳务'}
        open={modalOpen}
        initialValues={
          editing
            ? {
                name: editing.name,
                shortName: editing.shortName,
                goodsCode: editing.goodsCode,
                defaultTaxRate: editing.defaultTaxRate,
                enabled: editing.enabled ?? true,
              }
            : { defaultTaxRate: '0.00', enabled: true }
        }
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          const input = {
            name: values.name.trim(),
            shortName: values.shortName?.trim() || undefined,
            goodsCode: values.goodsCode?.trim() || undefined,
            defaultTaxRate: values.defaultTaxRate,
          };
          if (editing?.id) {
            await feeCatalogServiceUpdateTaxableService(
              { id: editing.id },
              { id: editing.id, ...input, enabled: values.enabled },
            );
            message.success('货物或应税劳务更新成功');
          } else {
            await feeCatalogServiceCreateTaxableService(input);
            message.success('货物或应税劳务创建成功');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="name"
          label="货物或应税劳务名称"
          rules={[{ required: true, message: '请输入名称' }]}
          fieldProps={{ maxLength: 128 }}
        />
        <ProFormText
          name="shortName"
          label="简称"
          fieldProps={{ maxLength: 64 }}
        />
        <ProFormText
          name="goodsCode"
          label="商品编码"
          fieldProps={{ maxLength: 64 }}
        />
        <ProFormText
          name="defaultTaxRate"
          label="默认税率（%）"
          rules={[
            { required: true, message: '请输入默认税率' },
            { pattern: taxRatePattern, message: '请输入 0–100，最多两位小数' },
          ]}
          fieldProps={{ inputMode: 'decimal' }}
        />
        {editing && <ProFormSwitch name="enabled" label="启用状态" />}
      </ModalForm>
    </Card>
  );
}

export default TaxableServicesPanel;
