import { EditOutlined, PlusOutlined, StopOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormDatePicker,
  ProFormSelect,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Popconfirm, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  exchangeRateServiceCreateExchangeRateSetting,
  exchangeRateServiceDisableExchangeRateSetting,
  exchangeRateServiceListExchangeRateSettings,
  exchangeRateServiceUpdateExchangeRateSetting,
} from '@/services/roncin/exchangeRateService';
import { isPositiveExactDecimal, trimExactDecimal } from '../../orders/order-fee-decimal';

const exchangeRatePattern = /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$/;
const exchangeRateTypeOptions = [
  { label: '汇率（折本币）', value: 'BASE_CURRENCY' },
  { label: '开票汇率', value: 'INVOICE' },
  { label: '结算汇率', value: 'SETTLEMENT' },
  { label: '核销汇率', value: 'WRITE_OFF' },
  { label: '账单汇率', value: 'BILL' },
];

type ExchangeRateFormValues = {
  rateType: string;
  fromCurrency: string;
  toCurrency: string;
  effectiveFrom: string | Dayjs;
  effectiveTo?: string | Dayjs;
  receivableRate: string;
  payableRate: string;
  matchingRule: string;
};

const rateRule = async (_: unknown, value?: string) => {
  if (!value) throw new Error('请输入汇率');
  if (!isPositiveExactDecimal(value, exchangeRatePattern)) {
    throw new Error('汇率必须大于 0，最多 10 位整数、8 位小数');
  }
};

export default function ExchangeRatesPage() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.ExchangeRateSetting>();
  const [baseCurrency, setBaseCurrency] = useState('');

  const openCreate = () => {
    setEditing(undefined);
    setModalOpen(true);
  };

  const columns: ProColumns<API.ExchangeRateSetting>[] = [
    {
      title: '汇率类型',
      dataIndex: 'rateType',
      width: 130,
      render: (_, record) =>
        exchangeRateTypeOptions.find((option) => option.value === record.rateType)!.label,
    },
    { title: '原币', dataIndex: 'fromCurrency', width: 90 },
    { title: '目标币', dataIndex: 'toCurrency', width: 90 },
    {
      title: '应收汇率',
      dataIndex: 'receivableRate',
      align: 'right',
      render: (_, record) => trimExactDecimal(record.receivableRate),
    },
    {
      title: '应付汇率',
      dataIndex: 'payableRate',
      align: 'right',
      render: (_, record) => trimExactDecimal(record.payableRate),
    },
    { title: '生效开始', dataIndex: 'effectiveFrom', width: 120 },
    {
      title: '生效结束（不含）',
      dataIndex: 'effectiveTo',
      width: 150,
      render: (_, record) => record.effectiveTo || '长期有效',
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      width: 80,
      render: (_, record) =>
        record.isActive ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 150,
      render: (_, record) => (
        <Space size="small">
          {record.isActive && access.canUpdateExchangeRates && (
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
          )}
          {record.isActive && access.canDisableExchangeRates && (
            <Popconfirm
              title="确定停用该汇率？"
              description="停用后新费用不能再匹配它，历史费用快照不受影响。"
              onConfirm={async () => {
                if (!record.id) return;
                await exchangeRateServiceDisableExchangeRateSetting(
                  { id: record.id },
                  { id: record.id },
                );
                message.success('汇率已停用');
                actionRef.current?.reload();
              }}
            >
              <Button type="link" danger size="small" icon={<StopOutlined />}>
                停用
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const initialValues: Partial<ExchangeRateFormValues> = editing
    ? {
        rateType: editing.rateType,
        fromCurrency: editing.fromCurrency,
        toCurrency: editing.toCurrency,
        effectiveFrom: editing.effectiveFrom
          ? dayjs(editing.effectiveFrom)
          : undefined,
        effectiveTo: editing.effectiveTo
          ? dayjs(editing.effectiveTo)
          : undefined,
        receivableRate: trimExactDecimal(editing.receivableRate),
        payableRate: trimExactDecimal(editing.payableRate),
      }
    : { toCurrency: baseCurrency, effectiveFrom: dayjs() };

  return (
    <PageContainer
      title="汇率设置"
      subTitle="全公司共用一套汇率主数据；授权人员可在所属分公司维护，当前按费用日期匹配"
    >
      <ProTable<API.ExchangeRateSetting>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        pagination={false}
        request={async () => {
          const response = await exchangeRateServiceListExchangeRateSettings();
          setBaseCurrency(response.baseCurrency ?? '');
          return { data: response.data ?? [], success: response.success ?? true };
        }}
        toolBarRender={() =>
          access.canCreateExchangeRates
            ? [
                <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                  新建汇率
                </Button>,
              ]
            : []
        }
      />

      <ModalForm<ExchangeRateFormValues>
        title={editing ? '编辑汇率' : '新建汇率'}
        open={modalOpen}
        initialValues={initialValues}
        modalProps={{ destroyOnHidden: true, onCancel: () => setModalOpen(false) }}
        onOpenChange={setModalOpen}
        onFinish={async (values) => {
          const effectiveFrom = dayjs(values.effectiveFrom).format('YYYY-MM-DD');
          const effectiveTo = values.effectiveTo
            ? dayjs(values.effectiveTo).format('YYYY-MM-DD')
            : undefined;
          if (effectiveTo && effectiveTo <= effectiveFrom) {
            message.error('生效结束日期必须晚于生效开始日期');
            return false;
          }
          const input = {
            rateType: values.rateType,
            fromCurrency: values.fromCurrency.trim().toUpperCase(),
            toCurrency: values.toCurrency.trim().toUpperCase(),
            timeStandard: 'EXPENSE_DATE',
            effectiveFrom,
            effectiveTo,
            receivableRate: values.receivableRate,
            payableRate: values.payableRate,
          };
          if (editing?.id) {
            await exchangeRateServiceUpdateExchangeRateSetting(
              { id: editing.id },
              { id: editing.id, ...input },
            );
            message.success('汇率更新成功');
          } else {
            await exchangeRateServiceCreateExchangeRateSetting(input);
            message.success('汇率创建成功');
          }
          setModalOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormSelect
          name="rateType"
          label="汇率类型"
          options={exchangeRateTypeOptions}
          rules={[{ required: true, message: '请选择汇率类型' }]}
        />
        <ProFormText
          name="fromCurrency"
          label="原币"
          placeholder="例如 USD"
          rules={[{ required: true, pattern: /^[A-Za-z]{3}$/, message: '请输入 3 位币种代码' }]}
          fieldProps={{ maxLength: 3 }}
        />
        <ProFormText
          name="toCurrency"
          label="目标币"
          disabled
          rules={[{ required: true, message: '请输入目标币' }]}
        />
        <ProFormText name="receivableRate" label="应收汇率" rules={[{ validator: rateRule }]} />
        <ProFormText name="payableRate" label="应付汇率" rules={[{ validator: rateRule }]} />
        <ProFormDatePicker
          name="effectiveFrom"
          label="生效开始日期"
          rules={[{ required: true, message: '请选择生效开始日期' }]}
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDatePicker
          name="effectiveTo"
          label="生效结束日期（不含）"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormSelect
          name="matchingRule"
          label="匹配规则"
          initialValue="EXPENSE_DATE"
          disabled
          options={[{ label: '按费用日期', value: 'EXPENSE_DATE' }]}
        />
      </ModalForm>
    </PageContainer>
  );
}
