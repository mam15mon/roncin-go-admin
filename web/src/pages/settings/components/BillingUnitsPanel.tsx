import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Tag } from 'antd';
import React, { useState } from 'react';
import { SettingTableTemplate } from '@/components/ui';
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

export function BillingUnitsPanel() {
  const access = useAccess();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 表头汇总：多选框、序号、计费单位、是否为箱型单位、操作
  const columns: ProColumns<API.BillingUnit>[] = [
    {
      title: '序号',
      valueType: 'index',
      width: 60,
    },
    {
      title: '计费单位',
      dataIndex: 'name',
      render: (_, record) => `${record.name} (${record.code})`,
    },
    {
      title: '是否为箱型单位',
      dataIndex: 'isContainerUnit',
      width: 150,
      render: (_, record) => (
        <Tag color={record.isContainerUnit ? 'blue' : 'default'}>
          {record.isContainerUnit ? '箱型计量单位' : '常规计量单位'}
        </Tag>
      ),
    },
  ];

  return (
    <SettingTableTemplate<API.BillingUnit, BillingUnitFormValues>
      entityName="计费单位"
      columns={columns}
      query={feeCatalogServiceListBillingUnits}
      canCreate={access.canCreateFeeSettings}
      canUpdate={access.canUpdateFeeSettings}
      rowSelection={{
        selectedRowKeys,
        onChange: (keys) => setSelectedRowKeys(keys),
      }}
      createItem={(values) =>
        feeCatalogServiceCreateBillingUnit({
          code: values.code.trim().toUpperCase(),
          name: values.name.trim(),
          isContainerUnit: values.isContainerUnit,
          sortOrder: values.sortOrder ?? 100,
        })
      }
      updateItem={(record, values) => {
        if (!record.id) return Promise.resolve();
        return feeCatalogServiceUpdateBillingUnit(
          { id: record.id },
          {
            id: record.id,
            code: values.code.trim().toUpperCase(),
            name: values.name.trim(),
            isContainerUnit: values.isContainerUnit,
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
              isContainerUnit: editing.isContainerUnit ?? false,
              sortOrder: editing.sortOrder ?? 100,
              enabled: editing.enabled ?? true,
            }
          : { isContainerUnit: false, sortOrder: 100, enabled: true }
      }
      renderFormItems={(editing) => (
        <>
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
            label="排序权重"
            min={0}
            fieldProps={{ precision: 0 }}
          />
          <ProFormSwitch
            name="isContainerUnit"
            label="是否为箱型单位"
            extra="开启后该单位将作为 20GP/40HQ 等集装箱计量基准"
          />
          {editing && <ProFormSwitch name="enabled" label="启用状态" />}
        </>
      )}
    />
  );
}

export default BillingUnitsPanel;
