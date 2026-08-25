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
  masterDataServiceCreateItem,
  masterDataServiceListItems,
  masterDataServiceUpdateItem,
} from '@/services/roncin/masterDataService';

const ABNORMAL_CASE_KIND = 10;

type AbnormalCaseFormValues = {
  code: string;
  name: string;
  nameEn?: string;
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

export function AbnormalCasesPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.MasterDataItem>();

  const columns: ProColumns<API.MasterDataItem>[] = [
    { title: '异常代码', dataIndex: 'code', width: 180, copyable: true },
    { title: '异常名称', dataIndex: 'name' },
    { title: '英文名称', dataIndex: 'nameEn' },
    { title: '排序', dataIndex: 'sortOrder', width: 90 },
    enabledColumn<API.MasterDataItem>(),
    {
      title: '操作',
      valueType: 'option',
      width: 90,
      render: (_, record) =>
        access.canUpdateMasterDataItems ? (
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
      <ProTable<API.MasterDataItem>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        pagination={false}
        request={async () => {
          const response = await masterDataServiceListItems({
            kind: ABNORMAL_CASE_KIND,
            page: 1,
            pageSize: 100,
          });
          return {
            data: response.data ?? [],
            success: response.success ?? true,
          };
        }}
        toolBarRender={() =>
          access.canCreateMasterDataItems
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
                  新建异常情况
                </Button>,
              ]
            : []
        }
      />
      <ModalForm<AbnormalCaseFormValues>
        title={editing ? '编辑异常情况' : '新建异常情况'}
        open={modalOpen}
        initialValues={
          editing
            ? {
                code: editing.code,
                name: editing.name,
                nameEn: editing.nameEn,
                sortOrder: editing.sortOrder ?? 100,
                enabled: editing.enabled ?? true,
              }
            : { sortOrder: 100, enabled: true }
        }
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          if (editing?.id) {
            await masterDataServiceUpdateItem(
              { id: editing.id },
              {
                id: editing.id,
                kind: ABNORMAL_CASE_KIND,
                name: values.name.trim(),
                nameEn: values.nameEn?.trim() || undefined,
                source: editing.source || 'manual',
                sortOrder: values.sortOrder,
                enabled: values.enabled,
              },
            );
            message.success('异常情况更新成功');
          } else {
            await masterDataServiceCreateItem({
              kind: ABNORMAL_CASE_KIND,
              code: values.code.trim().toUpperCase(),
              name: values.name.trim(),
              nameEn: values.nameEn?.trim() || undefined,
              source: 'manual',
              sortOrder: values.sortOrder,
            });
            message.success('异常情况创建成功');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          name="code"
          label="异常代码"
          disabled={Boolean(editing)}
          rules={[{ required: true, message: '请输入异常代码' }]}
          fieldProps={{ maxLength: 64 }}
        />
        <ProFormText
          name="name"
          label="异常名称"
          rules={[{ required: true, message: '请输入异常名称' }]}
          fieldProps={{ maxLength: 200 }}
        />
        <ProFormText
          name="nameEn"
          label="英文名称"
          fieldProps={{ maxLength: 200 }}
        />
        <ProFormDigit
          name="sortOrder"
          label="排序"
          min={0}
          fieldProps={{ precision: 0 }}
        />
        {editing && <ProFormSwitch name="enabled" label="启用状态" />}
      </ModalForm>
    </Card>
  );
}

export default AbnormalCasesPanel;
