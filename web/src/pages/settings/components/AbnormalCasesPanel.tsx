import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import React, { useState } from 'react';
import { SettingTableTemplate } from '@/components/ui';
import {
  masterDataServiceCreateItem,
  masterDataServiceListItems,
  masterDataServiceUpdateItem,
} from '@/services/roncin/masterDataService';
import { toTableRequest } from '@/utils/api';

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
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 表头汇总：多选框（当前）、序号、异常情况、操作
  const columns: ProColumns<API.MasterDataItem>[] = [
    {
      title: '序号',
      valueType: 'index',
      width: 60,
    },
    {
      title: '异常情况',
      dataIndex: 'name',
      render: (_, record) =>
        record.nameEn ? `${record.name} (${record.nameEn})` : record.name,
    },
  ];

  return (
    <SettingTableTemplate<API.MasterDataItem, AbnormalCaseFormValues>
      entityName="异常情况"
      columns={columns}
      modalWidth={540}
      labelWidth={130}
      rowSelection={{
        selectedRowKeys,
        onChange: (keys) => setSelectedRowKeys(keys),
      }}
      query={async () => {
        const res = await masterDataServiceListItems({
          kind: ABNORMAL_CASE_KIND,
          page: 1,
          pageSize: 200,
        });
        return toTableRequest(res);
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
          sortOrder: values.sortOrder ?? 100,
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
            sortOrder: values.sortOrder ?? 100,
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
            label="排序权重"
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
