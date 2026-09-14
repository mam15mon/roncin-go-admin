import {
  CloudSyncOutlined,
  DownloadOutlined,
  EditOutlined,
  PlusOutlined,
  StopOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormSelect,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Card, Form, Popconfirm, Space, Tag, Tooltip } from 'antd';
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
import { ExchangeRateSyncModal } from './ExchangeRateSyncModal';

const exchangeRatePattern = /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$/;

type ExchangeRateFormValues = {
  fromCurrency: string;
  toCurrency: string;
  effectiveFrom: string | Dayjs;
  arRate: string;
  apRate: string;
  rate?: string;
};

const rateRule =
  (label: string, required: boolean) => async (_: unknown, value?: string) => {
    if (!value) {
      if (required) throw new Error(`请输入${label}`);
      return;
    }
    if (!isPositiveExactDecimal(value, exchangeRatePattern)) {
      throw new Error(`${label}必须大于 0，最多 10 位整数、8 位小数`);
    }
  };

export function ExchangeRatesPanel() {
  const access = useAccess();
  // 组织身份统一取自 access.ts 的 isHeadquartersOrganization（auth/me kind 契约），
  // 不在面板内重复推导；仅总部可编辑 NULL 基线行。
  const isHeadquartersOrganization = access.isHeadquartersOrganization;
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [form] = Form.useForm<ExchangeRateFormValues>();
  const [modalOpen, setModalOpen] = useState(false);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [syncModalOpen, setSyncModalOpen] = useState(false);
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
      arRate: trimDecimal(record.arRate),
      apRate: trimDecimal(record.apRate),
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
      title: '应收汇率（现汇卖出价）',
      dataIndex: 'arRate',
      align: 'right',
      width: 150,
      render: (_, record) => (
        <span style={{ fontWeight: 600, color: '#1677ff' }}>
          {trimDecimal(record.arRate)}
        </span>
      ),
    },
    {
      title: '应付汇率（现汇买入价）',
      dataIndex: 'apRate',
      align: 'right',
      width: 150,
      render: (_, record) => (
        <span style={{ fontWeight: 600, color: '#52c41a' }}>
          {trimDecimal(record.apRate)}
        </span>
      ),
    },
    {
      title: '基准汇率',
      dataIndex: 'rate',
      align: 'right',
      width: 110,
      render: (_, record) => (
        <Tooltip title="中行折算价口径，内部审计与报表基准">
          <span style={{ color: '#8c8c8c' }}>{trimDecimal(record.rate)}</span>
        </Tooltip>
      ),
    },
    {
      title: '生效周',
      dataIndex: 'effectiveFrom',
      width: 200,
      render: (_, record) =>
        `${formatDate(record.effectiveFrom)} ~ ${formatDate(record.effectiveTo)}`,
    },
    {
      title: '归属',
      dataIndex: 'organizationId',
      width: 110,
      render: (_, record) =>
        record.organizationId ? (
          <Tag color="blue" style={{ margin: 0, fontSize: 11 }}>
            本组织行
          </Tag>
        ) : (
          <Tag color="purple" style={{ margin: 0, fontSize: 11 }}>
            集团基线行
          </Tag>
        ),
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
      render: (_, record) => {
        const isBaseline = !record.organizationId;
        const canEdit =
          record.isActive &&
          access.canUpdateExchangeRates &&
          (isBaseline ? isHeadquartersOrganization : true);
        const canDisable =
          record.isActive &&
          access.canDisableExchangeRates &&
          (isBaseline ? isHeadquartersOrganization : true);
        return (
          <Space size="small">
            {canEdit && (
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
            )}
            {canDisable && (
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
        );
      },
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
        arRate: trimDecimal(editing.arRate),
        apRate: trimDecimal(editing.apRate),
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
          ...(access.canCreateExchangeRates
            ? [
                <Tooltip
                  key="sync-tip"
                  title={
                    baseCurrency === 'CNY'
                      ? '抓取中国银行现汇买卖价，一键维护本周（或预设下周）汇率'
                      : `按本币 ${baseCurrency} 抓取国际直盘或中行交叉盘牌价`
                  }
                >
                  <Button
                    key="sync"
                    type="primary"
                    ghost
                    icon={<CloudSyncOutlined />}
                    onClick={() => setSyncModalOpen(true)}
                  >
                    {baseCurrency === 'CNY'
                      ? '从中国银行同步周汇率'
                      : '一键同步周汇率'}
                  </Button>
                </Tooltip>,
              ]
            : []),
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
        labelCol={{ flex: '150px' }}
        wrapperCol={{ flex: 'auto' }}
        form={form}
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setModalOpen(false),
          width: 620,
        }}
        onOpenChange={(visible) => {
          setModalOpen(visible);
          if (!visible) setEditing(undefined);
        }}
        onFinish={async (values) => {
          const effectiveFrom = dayjs(values.effectiveFrom).format(
            'YYYY-MM-DDTHH:mm:ssZ',
          );
          if (
            values.arRate &&
            values.apRate &&
            !isPositiveExactDecimal(values.arRate, exchangeRatePattern)
          ) {
            message.error('应收汇率格式不正确');
            return false;
          }
          const input = {
            fromCurrency: values.fromCurrency.trim().toUpperCase(),
            toCurrency: values.toCurrency.trim().toUpperCase(),
            effectiveFrom,
            arRate: values.arRate,
            apRate: values.apRate,
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
          disabled
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
        <ProFormDatePicker
          name="effectiveFrom"
          label="生效周"
          extra="任选目标自然周内日期，保存后按该自然周（周一至周日）生效"
          rules={[{ required: true, message: '请选择生效周' }]}
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormText
          name="arRate"
          label="应收汇率"
          extra="现汇卖出价：客户账单（AR）折本币收款使用"
          rules={[
            { required: true, message: '请输入应收汇率' },
            { validator: rateRule('应收汇率', true) },
          ]}
        />
        <ProFormText
          name="apRate"
          label="应付汇率"
          extra="现汇买入价：供应商账单（AP）折本币付款使用"
          rules={[
            { required: true, message: '请输入应付汇率' },
            { validator: rateRule('应付汇率', true) },
          ]}
        />
        <ProFormText
          name="rate"
          label="基准汇率"
          extra="中行折算价口径（可空），缺省按应收/应付中间价记录"
          rules={[{ validator: rateRule('基准汇率', false) }]}
        />
      </ModalForm>

      <ExchangeRateImportModal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onSuccess={() => actionRef.current?.reload()}
      />

      <ExchangeRateSyncModal
        open={syncModalOpen}
        baseCurrency={baseCurrency}
        onClose={() => setSyncModalOpen(false)}
        onSuccess={() => actionRef.current?.reload()}
      />
    </Card>
  );
}

export default ExchangeRatesPanel;
