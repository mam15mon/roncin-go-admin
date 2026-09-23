import type { ProColumns } from '@ant-design/pro-components';
import {
  ProFormDigit,
  ProFormRadio,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { useQueryClient } from '@tanstack/react-query';
import { Alert, Form, Tag } from 'antd';
import React, { useState } from 'react';
import { useAccess } from '@/app/access';
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
  quantityMustBeInteger: boolean;
  enabled: boolean;
  sortOrder: number;
};

function BillingUnitRuleFields({ editing }: { editing: boolean }) {
  const form = Form.useFormInstance<BillingUnitFormValues>();
  return (
    <>
      <ProFormSwitch
        name="isContainerUnit"
        label="是否为箱型单位"
        extra="开启后该单位将作为 20GP/40HQ 等集装箱计量基准"
        fieldProps={{
          onChange: (checked) => {
            if (
              !editing &&
              checked &&
              !form.isFieldTouched('quantityMustBeInteger')
            ) {
              form.setFieldValue('quantityMustBeInteger', true);
            }
          },
        }}
      />
      <ProFormRadio.Group
        name="quantityMustBeInteger"
        label="数量规则"
        options={[
          { label: '允许小数', value: false },
          { label: '整数', value: true },
        ]}
      />
    </>
  );
}

export function BillingUnitsPanel() {
  const access = useAccess();
  const queryClient = useQueryClient();
  // A 型全局主数据：全员同权可见，仅系统管理可维护（权限码 + 组织身份双重收敛）。
  const isSystemWorkspace = access.isSystemWorkspace;
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
    {
      title: '数量规则',
      dataIndex: 'quantityMustBeInteger',
      width: 110,
      render: (_, record) =>
        record.quantityMustBeInteger ? '整数' : '允许小数',
    },
  ];

  return (
    <>
      {!isSystemWorkspace && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          title="计费单位为系统公共基础资料，由系统管理员统一维护与共享，本组织只读"
        />
      )}
      <SettingTableTemplate<API.BillingUnit, BillingUnitFormValues>
        entityName="计费单位"
        columns={columns}
        modalWidth={560}
        labelWidth={145}
        query={feeCatalogServiceListBillingUnits}
        canCreate={access.canCreateFeeSettings && isSystemWorkspace}
        canUpdate={access.canUpdateFeeSettings && isSystemWorkspace}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys),
        }}
        createItem={async (values) => {
          const response = await feeCatalogServiceCreateBillingUnit({
            code: values.code.trim().toUpperCase(),
            name: values.name.trim(),
            isContainerUnit: values.isContainerUnit,
            quantityMustBeInteger: values.quantityMustBeInteger,
            sortOrder: values.sortOrder ?? 100,
          });
          if (!response) throw new Error('创建计费单位失败');
          await queryClient.invalidateQueries({
            queryKey: ['orders', 'fee-options'],
          });
        }}
        updateItem={async (record, values) => {
          if (!record.id) return Promise.resolve();
          const response = await feeCatalogServiceUpdateBillingUnit(
            { id: record.id },
            {
              id: record.id,
              code: values.code.trim().toUpperCase(),
              name: values.name.trim(),
              isContainerUnit: values.isContainerUnit,
              quantityMustBeInteger: values.quantityMustBeInteger,
              sortOrder: values.sortOrder ?? 100,
              enabled: values.enabled,
            },
          );
          if (!response) throw new Error('更新计费单位失败');
          await queryClient.invalidateQueries({
            queryKey: ['orders', 'fee-options'],
          });
        }}
        initialValues={(editing) =>
          editing
            ? {
                code: editing.code,
                name: editing.name,
                isContainerUnit: editing.isContainerUnit ?? false,
                quantityMustBeInteger: editing.quantityMustBeInteger ?? false,
                sortOrder: editing.sortOrder ?? 100,
                enabled: editing.enabled ?? true,
              }
            : {
                isContainerUnit: false,
                quantityMustBeInteger: false,
                sortOrder: 100,
                enabled: true,
              }
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
            <BillingUnitRuleFields editing={Boolean(editing)} />
            {editing && <ProFormSwitch name="enabled" label="启用状态" />}
          </>
        )}
      />
    </>
  );
}

export default BillingUnitsPanel;
