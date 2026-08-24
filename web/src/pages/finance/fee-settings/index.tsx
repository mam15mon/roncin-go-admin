import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Space, Tabs, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  feeCatalogServiceCreateBillingUnit,
  feeCatalogServiceCreateFeeSetting,
  feeCatalogServiceCreateTaxableService,
  feeCatalogServiceListBillingUnits,
  feeCatalogServiceListFeeSettings,
  feeCatalogServiceListTaxableServices,
  feeCatalogServiceUpdateBillingUnit,
  feeCatalogServiceUpdateFeeSetting,
  feeCatalogServiceUpdateTaxableService,
} from '@/services/roncin/feeCatalogService';
import {
  masterDataServiceCreateItem,
  masterDataServiceListCurrencies,
  masterDataServiceListItems,
  masterDataServiceListOptions,
  masterDataServiceUpdateItem,
} from '@/services/roncin/masterDataService';

const SERVICE_TYPE_KIND = 8;
const ABNORMAL_CASE_KIND = 10;
const codePattern = /^[A-Z0-9_]{2,32}$/;
const taxRatePattern =
  /^(100(?:\.0{1,2})?|(?:[0-9]|[1-9][0-9])(?:\.[0-9]{1,2})?)$/;

type FeeSettingFormValues = {
  feeCode: string;
  nameZh: string;
  nameEn?: string;
  aliasName?: string;
  serviceTypeId?: string;
  defaultCurrency: string;
  billingUnitId: string;
  abnormalCaseId?: string;
  taxRate: string;
  taxableServiceId: string;
  enabled: boolean;
  sortOrder: number;
};

type BillingUnitFormValues = {
  code: string;
  name: string;
  enabled: boolean;
  sortOrder: number;
};

type TaxableServiceFormValues = {
  name: string;
  shortName?: string;
  goodsCode?: string;
  defaultTaxRate: string;
  enabled: boolean;
};

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

function FeeSettingsPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.FeeSetting>();
  const [billingUnits, setBillingUnits] = useState<API.BillingUnit[]>([]);
  const [taxableServices, setTaxableServices] = useState<API.TaxableService[]>(
    [],
  );
  const [currencies, setCurrencies] = useState<API.Currency[]>([]);
  const [serviceTypes, setServiceTypes] = useState<API.MasterDataItem[]>([]);
  const [abnormalCases, setAbnormalCases] = useState<API.MasterDataItem[]>([]);

  const columns: ProColumns<API.FeeSetting>[] = [
    { title: '费用代码', dataIndex: 'feeCode', width: 150, copyable: true },
    { title: '费用名称', dataIndex: 'nameZh', width: 150 },
    { title: '英文名称', dataIndex: 'nameEn', width: 180, ellipsis: true },
    {
      title: '服务类型',
      dataIndex: 'serviceTypeName',
      width: 120,
      renderText: (value) => value || '通用',
    },
    { title: '币种', dataIndex: 'defaultCurrency', width: 80 },
    { title: '计费单位', dataIndex: 'billingUnitName', width: 100 },
    {
      title: '异常情况',
      dataIndex: 'abnormalCaseName',
      width: 130,
      renderText: (value) => value || '不限',
    },
    {
      title: '税率',
      dataIndex: 'taxRate',
      width: 80,
      renderText: (value) => `${value ?? '0.00'}%`,
    },
    {
      title: '货物或应税劳务',
      dataIndex: 'taxableServiceName',
      width: 180,
      ellipsis: true,
    },
    { title: '排序', dataIndex: 'sortOrder', width: 70 },
    enabledColumn<API.FeeSetting>(),
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

  const initialValues: Partial<FeeSettingFormValues> = editing
    ? {
        feeCode: editing.feeCode,
        nameZh: editing.nameZh,
        nameEn: editing.nameEn,
        aliasName: editing.aliasName,
        serviceTypeId: editing.serviceTypeId,
        defaultCurrency: editing.defaultCurrency,
        billingUnitId: editing.billingUnitId,
        abnormalCaseId: editing.abnormalCaseId,
        taxRate: editing.taxRate,
        taxableServiceId: editing.taxableServiceId,
        enabled: editing.enabled ?? true,
        sortOrder: editing.sortOrder ?? 100,
      }
    : { enabled: true, sortOrder: 100 };

  return (
    <>
      <ProTable<API.FeeSetting>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        pagination={false}
        scroll={{ x: 1350 }}
        request={async () => {
          const [
            feeResponse,
            unitResponse,
            taxableResponse,
            currencyResponse,
            optionResponse,
          ] = await Promise.all([
            feeCatalogServiceListFeeSettings(),
            feeCatalogServiceListBillingUnits(),
            feeCatalogServiceListTaxableServices(),
            masterDataServiceListCurrencies(),
            masterDataServiceListOptions(),
          ]);
          setBillingUnits(
            (unitResponse.data ?? []).filter((item) => item.enabled),
          );
          setTaxableServices(
            (taxableResponse.data ?? []).filter((item) => item.enabled),
          );
          setCurrencies(
            (currencyResponse.data ?? []).filter((item) => item.enabled),
          );
          setServiceTypes(
            (optionResponse.data ?? []).filter(
              (item) => item.kind === SERVICE_TYPE_KIND,
            ),
          );
          setAbnormalCases(
            (optionResponse.data ?? []).filter(
              (item) => item.kind === ABNORMAL_CASE_KIND,
            ),
          );
          return {
            data: feeResponse.data ?? [],
            success: feeResponse.success ?? true,
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
                  新建费用设置
                </Button>,
              ]
            : []
        }
      />

      <ModalForm<FeeSettingFormValues>
        title={editing ? '编辑费用设置' : '新建费用设置'}
        open={modalOpen}
        initialValues={initialValues}
        grid
        modalProps={{
          destroyOnHidden: true,
          width: 820,
          onCancel: () => setModalOpen(false),
        }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          const input = {
            feeCode: values.feeCode.trim().toUpperCase(),
            nameZh: values.nameZh.trim(),
            nameEn: values.nameEn?.trim() || undefined,
            aliasName: values.aliasName?.trim() || undefined,
            serviceTypeId: values.serviceTypeId || undefined,
            defaultCurrency: values.defaultCurrency,
            billingUnitId: values.billingUnitId,
            abnormalCaseId: values.abnormalCaseId || undefined,
            taxRate: values.taxRate,
            taxableServiceId: values.taxableServiceId,
            sortOrder: values.sortOrder,
          };
          if (editing?.id) {
            await feeCatalogServiceUpdateFeeSetting(
              { id: editing.id },
              { id: editing.id, ...input, enabled: values.enabled },
            );
            message.success('费用设置更新成功');
          } else {
            await feeCatalogServiceCreateFeeSetting(input);
            message.success('费用设置创建成功');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText
          colProps={{ span: 12 }}
          name="feeCode"
          label="费用代码"
          disabled={Boolean(editing)}
          rules={[
            { required: true, message: '请输入费用代码' },
            {
              pattern: codePattern,
              message: '请输入 2–32 位大写字母、数字或下划线',
            },
          ]}
        />
        <ProFormText
          colProps={{ span: 12 }}
          name="nameZh"
          label="费用名称"
          rules={[{ required: true, message: '请输入费用名称' }]}
          fieldProps={{ maxLength: 64 }}
        />
        <ProFormText
          colProps={{ span: 12 }}
          name="nameEn"
          label="费用名称（英文）"
          fieldProps={{ maxLength: 128 }}
        />
        <ProFormText
          colProps={{ span: 12 }}
          name="aliasName"
          label="费用别名"
          fieldProps={{ maxLength: 64 }}
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="serviceTypeId"
          label="对应服务类型"
          allowClear
          options={serviceTypes.map((item) => ({
            label: `${item.name} (${item.code})`,
            value: item.id,
          }))}
          placeholder="不选择表示通用费用"
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="defaultCurrency"
          label="默认币种"
          rules={[{ required: true, message: '请选择默认币种' }]}
          showSearch
          options={currencies.map((item) => ({
            label: `${item.code} - ${item.name}`,
            value: item.code,
          }))}
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="billingUnitId"
          label="默认计费单位"
          rules={[{ required: true, message: '请选择计费单位' }]}
          showSearch
          options={billingUnits.map((item) => ({
            label: `${item.name} (${item.code})`,
            value: item.id,
          }))}
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="abnormalCaseId"
          label="对应异常情况"
          allowClear
          options={abnormalCases.map((item) => ({
            label: `${item.name} (${item.code})`,
            value: item.id,
          }))}
          placeholder="不选择表示不限异常情况"
        />
        <ProFormText
          colProps={{ span: 12 }}
          name="taxRate"
          label="税率（%）"
          rules={[
            { required: true, message: '请输入税率' },
            { pattern: taxRatePattern, message: '请输入 0–100，最多两位小数' },
          ]}
          fieldProps={{ inputMode: 'decimal' }}
        />
        <ProFormSelect
          colProps={{ span: 12 }}
          name="taxableServiceId"
          label="货物或应税劳务名称"
          rules={[{ required: true, message: '请选择货物或应税劳务名称' }]}
          showSearch
          options={taxableServices.map((item) => ({
            label: item.name,
            value: item.id,
          }))}
        />
        <ProFormDigit
          colProps={{ span: 12 }}
          name="sortOrder"
          label="排序"
          min={0}
          fieldProps={{ precision: 0 }}
        />
        {editing && (
          <ProFormSwitch
            colProps={{ span: 12 }}
            name="enabled"
            label="启用状态"
          />
        )}
      </ModalForm>
    </>
  );
}

function BillingUnitsPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.BillingUnit>();
  const columns: ProColumns<API.BillingUnit>[] = [
    { title: '单位代码', dataIndex: 'code', width: 160, copyable: true },
    { title: '单位名称', dataIndex: 'name' },
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
    <>
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
          const input = {
            code: values.code.trim().toUpperCase(),
            name: values.name.trim(),
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
        {editing && <ProFormSwitch name="enabled" label="启用状态" />}
      </ModalForm>
    </>
  );
}

function TaxableServicesPanel() {
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
    <>
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
    </>
  );
}

function AbnormalCasesPanel() {
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
    <>
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
    </>
  );
}

export default function FeeSettingsPage() {
  const access = useAccess();
  const items = [
    { key: 'fee-settings', label: '费用设置', children: <FeeSettingsPanel /> },
    {
      key: 'billing-units',
      label: '计费单位',
      children: <BillingUnitsPanel />,
    },
    {
      key: 'taxable-services',
      label: '货物或应税劳务',
      children: <TaxableServicesPanel />,
    },
    access.canReadMasterDataItems
      ? {
          key: 'abnormal-cases',
          label: '异常情况',
          children: <AbnormalCasesPanel />,
        }
      : null,
  ].filter((item): item is NonNullable<typeof item> => item !== null);
  return (
    <PageContainer
      title="费用设置"
      subTitle="统一维护订单录费使用的费用代码、默认计费属性和异常情况"
    >
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Tabs type="card" items={items} />
      </Space>
    </PageContainer>
  );
}
