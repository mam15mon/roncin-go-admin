import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Tag } from 'antd';
import React from 'react';
import { SettingTableTemplate } from '@/components/ui';
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

export function AbnormalCasesPanel() {
  const access = useAccess();

  const columns: ProColumns<API.MasterDataItem>[] = [
    { title: '异常代码', dataIndex: 'code', width: 180, copyable: true },
    { title: '异常名称', dataIndex: 'name' },
    { title: '英文名称', dataIndex: 'nameEn' },
    { title: '排序', dataIndex: 'sortOrder', width: 90 },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 80,
      render: (_, record) =>
        record.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
    },
  ];

  return (
    <SettingTableTemplate<API.MasterDataItem, AbnormalCaseFormValues>
      entityName="异常情况"
      columns={columns}
      query={async () => {
        const res = await masterDataServiceListItems({
          kind: ABNORMAL_CASE_KIND,
          page: 1,
          pageSize: 100,
        });
        return {
          data: res.data ?? [],
          success: res.success ?? true,
        };
      }}
      canCreate={access.canCreateMasterDataItems}
      canUpdate={access.canUpdateMasterDataItems}
      createItem={(values) =>
        masterDataServiceCreateItem({
          kind: ABNORMAL_CASE_KIND,
          code: values.code.trim().toUpperCase(),
          name: values.name.trim(),
          nameEn: values.nameEn?.trim() || undefined,
          source: 'manual',
          sortOrder: values.sortOrder,
        })
      }
      updateItem={(record, values) => {
        if (!record.id) return Promise.resolve();
        return masterDataServiceUpdateItem(
          { id: record.id },
          {
            id: record.id,
            kind: ABNORMAL_CASE_KIND,
            name: values.name.trim(),
            nameEn: values.nameEn?.trim() || undefined,
            source: record.source || 'manual',
            sortOrder: values.sortOrder,
            enabled: values.enabled,
          },
        );
      }}
      initialValues={(editing) =>
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
      renderFormItems={(editing) => (
        <>
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
        </>
      )}
    />
  );
}

export default AbnormalCasesPanel;
