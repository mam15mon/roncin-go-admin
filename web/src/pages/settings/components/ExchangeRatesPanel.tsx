import {
  DownloadOutlined,
  EditOutlined,
  PlusOutlined,
  StopOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateTimePicker,
  ProFormSelect,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Card, Form, Popconfirm, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  exchangeRateServiceCreateExchangeRateSetting,
  exchangeRateServiceDisableExchangeRateSetting,
  exchangeRateServiceDownloadExchangeRateImportTemplate,
  exchangeRateServiceListExchangeRateSettings,
  exchangeRateServiceUpdateExchangeRateSetting,
} from '@/services/roncin/exchangeRateService';
import { toTableRequest } from '@/utils/api';
import { isPositiveExactDecimal } from '@/utils/decimal';
import { formatDate, trimDecimal } from '@/utils/format';
import { getCurrencies } from '@/utils/options';
import { ExchangeRateImportModal } from './ExchangeRateImportModal';

const exchangeRatePattern = /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$/;

type ExchangeRateFormValues = {
  fromCurrency: string;
  toCurrency: string;
  effectiveFrom: string | Dayjs;
  effectiveTo?: string | Dayjs;
  rate: string;
};

const rateRule = async (_: unknown, value?: string) => {
  if (!value) throw new Error('请输入汇率');
  if (!isPositiveExactDecimal(value, exchangeRatePattern)) {
    throw new Error('汇率必须大于 0，最多 10 位整数、8 位小数');
  }
};

export function ExchangeRatesPanel() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [form] = Form.useForm<ExchangeRateFormValues>();
  const [modalOpen, setModalOpen] = useState(false);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [downloadingTemplate, setDownloadingTemplate] = useState(false);
  const [editing, setEditing] = useState<API.ExchangeRateSetting>();
  const [baseCurrency, setBaseCurrency] = useState('');
  const [currencies, setCurrencies] = useState<API.Currency[]>([]);

  const openCreate = () => {
    setEditing(undefined);
    form.resetFields();
    form.setFieldsValue({
      toCurrency: baseCurrency,
      effectiveFrom: dayjs(),
    });
    setModalOpen(true);
  };

  const openEdit = (record: API.ExchangeRateSetting) => {
    setEditing(record);
    form.setFieldsValue({
      fromCurrency: record.fromCurrency,
      toCurrency: record.toCurrency,
      effectiveFrom: record.effectiveFrom
        ? dayjs(record.effectiveFrom)
        : undefined,
      effectiveTo: record.effectiveTo ? dayjs(record.effectiveTo) : undefined,
      rate: trimDecimal(record.rate),
    });
    setModalOpen(true);
  };

  const handleDownloadTemplate = async () => {
    setDownloadingTemplate(true);
    try {
      const res = await exchangeRateServiceDownloadExchangeRateImportTemplate();
      const base64Data = res.content;
      if (!base64Data) {
        message.error('下载模板失败：文件内容为空');
        return;
      }
      const byteCharacters = atob(base64Data);
      const byteNumbers = new Array(byteCharacters.length);
      for (let i = 0; i < byteCharacters.length; i++) {
        byteNumbers[i] = byteCharacters.charCodeAt(i);
      }
      const byteArray = new Uint8Array(byteNumbers);
      const blob = new Blob([byteArray], {
        type:
          res.contentType ||
          'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = res.fileName || '汇率导入模板.xlsx';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      message.success('导入模板下载成功');
    } catch (e: any) {
      message.error(e.message || '下载导入模板失败');
    } finally {
      setDownloadingTemplate(false);
    }
  };

  const columns: ProColumns<API.ExchangeRateSetting>[] = [
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 55,
      align: 'center',
    },
    {
      title: '币种对',
      dataIndex: 'fromCurrency',
      width: 220,
      render: (_, record) => {
        const from = currencies.find((c) => c.code === record.fromCurrency);
        const to = currencies.find((c) => c.code === record.toCurrency);
        const fromLabel = from?.name
          ? `${record.fromCurrency} (${from.name})`
          : record.fromCurrency;
        const toLabel = to?.name
          ? `${record.toCurrency} (${to.name})`
          : record.toCurrency;
        return `${fromLabel} → ${toLabel}`;
      },
    },
    {
      title: '折本币汇率',
      dataIndex: 'rate',
      align: 'right',
      width: 120,
      render: (_, record) => (
        <span style={{ fontWeight: 600, color: '#1677ff' }}>
          {trimDecimal(record.rate)}
        </span>
      ),
    },
    {
      title: '生效起始日',
      dataIndex: 'effectiveFrom',
      width: 165,
      render: (_, record) => formatDate(record.effectiveFrom),
    },
    {
      title: '失效日',
      dataIndex: 'effectiveTo',
      width: 165,
      render: (_, record) => {
        if (!record.effectiveTo) {
          return (
            <Tag color="cyan" style={{ margin: 0, fontSize: 11 }}>
              长期有效
            </Tag>
          );
        }
        return formatDate(record.effectiveTo);
      },
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
              onClick={() => openEdit(record)}
            >
              编辑
            </Button>
          )}
          {record.isActive && access.canDisableExchangeRates && (
            <Popconfirm
              title="确定停用该汇率设置？"
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

  const currencyOptions = currencies
    .filter((c) => c.enabled !== false)
    .map((c) => ({
      label:
        `${c.code} - ${c.name || ''} ${c.symbol ? `(${c.symbol})` : ''}`.trim(),
      value: c.code ?? '',
    }));

  const initialValues: Partial<ExchangeRateFormValues> = editing
    ? {
        fromCurrency: editing.fromCurrency,
        toCurrency: editing.toCurrency,
        effectiveFrom: editing.effectiveFrom
          ? dayjs(editing.effectiveFrom)
          : undefined,
        effectiveTo: editing.effectiveTo
          ? dayjs(editing.effectiveTo)
          : undefined,
        rate: trimDecimal(editing.rate),
      }
    : { toCurrency: baseCurrency, effectiveFrom: dayjs() };

  return (
    <Card
      variant="borderless"
      style={{
        borderRadius: 8,
        border: '1px solid #f0f0f0',
        backgroundColor: '#ffffff',
      }}
      styles={{ body: { padding: '12px 16px' } }}
    >
      <ProTable<API.ExchangeRateSetting>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        pagination={false}
        cardProps={false}
        tableAlertRender={false}
        tableAlertOptionRender={false}
        request={async () => {
          const [rateResponse, currencyResponse] = await Promise.all([
            exchangeRateServiceListExchangeRateSettings(),
            getCurrencies(),
          ]);
          setBaseCurrency(rateResponse.baseCurrency ?? '');
          setCurrencies(currencyResponse);
          return toTableRequest(rateResponse);
        }}
        toolBarRender={() => [
          <Button
            key="download-template"
            icon={<DownloadOutlined />}
            loading={downloadingTemplate}
            onClick={handleDownloadTemplate}
          >
            下载模板
          </Button>,
          ...(access.canCreateExchangeRates
            ? [
                <Button
                  key="import"
                  icon={<UploadOutlined />}
                  onClick={() => setImportModalOpen(true)}
                >
                  批量导入
                </Button>,
              ]
            : []),
          ...(access.canCreateExchangeRates
            ? [
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openCreate}
                >
                  新建汇率
                </Button>,
              ]
            : []),
        ]}
      />

      <ModalForm<ExchangeRateFormValues>
        title={editing ? '编辑汇率' : '新建汇率'}
        open={modalOpen}
        initialValues={initialValues}
        layout="horizontal"
        labelAlign="right"
        labelCol={{ flex: '110px' }}
        wrapperCol={{ flex: 'auto' }}
        form={form}
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setModalOpen(false),
          width: 580,
        }}
        onOpenChange={(visible) => {
          setModalOpen(visible);
          if (!visible) setEditing(undefined);
        }}
        onFinish={async (values) => {
          const effectiveFrom = dayjs(values.effectiveFrom).format(
            'YYYY-MM-DDTHH:mm:ssZ',
          );
          const effectiveTo = values.effectiveTo
            ? dayjs(values.effectiveTo).format('YYYY-MM-DDTHH:mm:ssZ')
            : undefined;
          if (
            effectiveTo &&
            (dayjs(effectiveTo).isBefore(dayjs(effectiveFrom)) ||
              dayjs(effectiveTo).isSame(dayjs(effectiveFrom)))
          ) {
            message.error('生效结束时间必须晚于生效开始时间');
            return false;
          }
          const input = {
            fromCurrency: values.fromCurrency.trim().toUpperCase(),
            toCurrency: values.toCurrency.trim().toUpperCase(),
            effectiveFrom,
            effectiveTo,
            rate: values.rate,
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
          name="fromCurrency"
          label="原币"
          showSearch
          options={currencyOptions}
          placeholder="请选择原币币种（支持代码/名称搜索）"
          rules={[
            { required: true, message: '请选择原币币种' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (value && value === getFieldValue('toCurrency')) {
                  return Promise.reject(new Error('原币不能与目标本币相同'));
                }
                return Promise.resolve();
              },
            }),
          ]}
        />
        <ProFormSelect
          name="toCurrency"
          label="本币"
          showSearch
          options={
            currencyOptions.length > 0
              ? currencyOptions
              : [{ label: baseCurrency, value: baseCurrency }]
          }
          placeholder="请选择本币币种"
          rules={[
            { required: true, message: '请选择本币币种' },
            () => ({
              validator(_, value) {
                if (value && baseCurrency && value !== baseCurrency) {
                  return Promise.reject(
                    new Error(
                      `当前组织本币为 ${baseCurrency}，本币必须与组织本币一致`,
                    ),
                  );
                }
                return Promise.resolve();
              },
            }),
          ]}
        />
        <ProFormText
          name="rate"
          label="折本币汇率"
          rules={[{ validator: rateRule }]}
        />
        <ProFormDateTimePicker
          name="effectiveFrom"
          label="生效开始时间"
          extra="精确至秒级（左闭区间包含该时刻起生效）"
          rules={[{ required: true, message: '请选择生效开始时间' }]}
          fieldProps={{
            style: { width: '100%' },
            format: 'YYYY-MM-DD HH:mm:ss',
          }}
        />
        <ProFormDateTimePicker
          name="effectiveTo"
          label="生效结束时间"
          extra="精确至秒级（右开区间不包含该时刻）；留空表示长期有效"
          fieldProps={{
            style: { width: '100%' },
            format: 'YYYY-MM-DD HH:mm:ss',
            showTime: { defaultValue: dayjs('23:59:59', 'HH:mm:ss') },
          }}
        />
      </ModalForm>

      <ExchangeRateImportModal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onSuccess={() => actionRef.current?.reload()}
      />
    </Card>
  );
}

export default ExchangeRatesPanel;
