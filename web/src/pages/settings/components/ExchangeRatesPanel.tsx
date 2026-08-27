import {
  DownloadOutlined,
  EditOutlined,
  HolderOutlined,
  PlusOutlined,
  SettingOutlined,
  StopOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  sortableKeyboardCoordinates,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateTimePicker,
  ProFormSelect,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Card, Form, Modal, Popconfirm, Select, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  exchangeRateServiceCreateExchangeRateSetting,
  exchangeRateServiceDisableExchangeRateSetting,
  exchangeRateServiceDownloadExchangeRateImportTemplate,
  exchangeRateServiceListExchangeRateSettings,
  exchangeRateServiceListExchangeRateTimeStandards,
  exchangeRateServiceUpdateExchangeRateSetting,
  exchangeRateServiceUpdateExchangeRateTimeStandards,
} from '@/services/roncin/exchangeRateService';
import { masterDataServiceListCurrencies } from '@/services/roncin/masterDataService';
import { isPositiveExactDecimal, trimExactDecimal } from '../../orders/order-fee-decimal';
import { ExchangeRateImportModal } from './ExchangeRateImportModal';

const exchangeRatePattern = /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$/;
const exchangeRateTypeLabels: Record<string, string> = {
  BASE_CURRENCY: '汇率（折本币）',
  INVOICE: '开票汇率',
  SETTLEMENT: '结算汇率',
  WRITE_OFF: '核销汇率',
  BILL: '账单汇率',
};
const exchangeRateTypeOptions = Object.entries(exchangeRateTypeLabels).map(([value, label]) => ({
  label,
  value,
}));
const timeStandardLabels: Record<string, string> = {
  ETD_ETA_TRAIN_DATE: 'ETD/ETA/班列日期',
  BUSINESS_TIME: '业务时间',
  BARGE_ETD: '驳船 ETD',
  EXPENSE_TIME: '费用时间',
  ORDER_CREATED_AT: '订单创建时间',
  BILL_DATE: '账单日期',
  BILL_CREATED_AT: '账单创建时间',
  INVOICE_DATE: '开票日期',
  TRANSACTION_DATE: '资金交易日期',
  WRITE_OFF_TIME: '核销时间',
};
const timeStandardsByRateType: Record<string, string[]> = {
  BASE_CURRENCY: [
    'ETD_ETA_TRAIN_DATE',
    'BUSINESS_TIME',
    'BARGE_ETD',
    'EXPENSE_TIME',
    'ORDER_CREATED_AT',
  ],
  BILL: ['BILL_DATE', 'BILL_CREATED_AT'],
  INVOICE: ['INVOICE_DATE'],
  SETTLEMENT: ['TRANSACTION_DATE'],
  WRITE_OFF: ['WRITE_OFF_TIME'],
};

type ExchangeRateFormValues = {
  rateType: string;
  fromCurrency: string;
  toCurrency: string;
  effectiveFrom: string | Dayjs;
  effectiveTo?: string | Dayjs;
  receivableRate: string;
  payableRate: string;
};

type SortableTimeStandardsProps = {
  values: string[];
  onChange: (values: string[]) => void;
};

function SortableTimeStandards({ values, onChange }: SortableTimeStandardsProps) {
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) return;
    const oldIndex = values.indexOf(String(active.id));
    const newIndex = values.indexOf(String(over.id));
    onChange(arrayMove(values, oldIndex, newIndex));
  };

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
      <SortableContext items={values} strategy={verticalListSortingStrategy}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 8 }}>
          {values.map((value) => (
            <SortableItem key={value} value={value} />
          ))}
        </div>
      </SortableContext>
    </DndContext>
  );
}

function SortableItem({ value }: { value: string }) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({
    id: value,
  });
  return (
    <div
      ref={setNodeRef}
      style={{
        alignItems: 'center',
        background: '#fff',
        border: '1px solid #f0f0f0',
        borderRadius: 6,
        display: 'flex',
        justifyContent: 'space-between',
        padding: '6px 8px',
        transform: CSS.Transform.toString(transform),
        transition,
      }}
    >
      <span>{timeStandardLabels[value]}</span>
      <Button
        type="text"
        size="small"
        icon={<HolderOutlined />}
        aria-label={`拖拽排序：${timeStandardLabels[value]}`}
        style={{ cursor: 'grab' }}
        {...attributes}
        {...listeners}
      />
    </div>
  );
}

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
  const [timeStandardsOpen, setTimeStandardsOpen] = useState(false);
  const [timeStandardsSaving, setTimeStandardsSaving] = useState(false);
  const [timeStandards, setTimeStandards] = useState<Record<string, string[]>>({});

  const openCreate = () => {
    setEditing(undefined);
    form.resetFields();
    form.setFieldsValue({
      rateType: 'BASE_CURRENCY',
      toCurrency: baseCurrency,
      effectiveFrom: dayjs(),
    });
    setModalOpen(true);
  };

  const openEdit = (record: API.ExchangeRateSetting) => {
    setEditing(record);
    form.setFieldsValue({
      rateType: record.rateType,
      fromCurrency: record.fromCurrency,
      toCurrency: record.toCurrency,
      effectiveFrom: record.effectiveFrom ? dayjs(record.effectiveFrom) : undefined,
      effectiveTo: record.effectiveTo ? dayjs(record.effectiveTo) : undefined,
      receivableRate: trimExactDecimal(record.receivableRate),
      payableRate: trimExactDecimal(record.payableRate),
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

  const openTimeStandards = async () => {
    const response = await exchangeRateServiceListExchangeRateTimeStandards();
    setTimeStandards(
      Object.fromEntries(
        (response.data ?? []).map((item) => [item.rateType ?? '', item.timeStandards ?? []]),
      ),
    );
    setTimeStandardsOpen(true);
  };

  const saveTimeStandards = async () => {
    setTimeStandardsSaving(true);
    try {
      await exchangeRateServiceUpdateExchangeRateTimeStandards({
        data: Object.entries(timeStandards).map(([rateType, standards]) => ({
          rateType,
          timeStandards: standards,
        })),
      });
      message.success('时间标准配置已更新');
      setTimeStandardsOpen(false);
    } catch {
      message.error('时间标准配置更新失败');
    } finally {
      setTimeStandardsSaving(false);
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
      title: '汇率类型',
      dataIndex: 'rateType',
      width: 140,
      render: (_, record) =>
        record.rateType ? exchangeRateTypeLabels[record.rateType] ?? record.rateType : '-',
    },
    {
      title: '原币种',
      dataIndex: 'fromCurrency',
      width: 110,
      render: (_, record) => {
        const cur = currencies.find((c) => c.code === record.fromCurrency);
        return cur?.name ? `${record.fromCurrency} (${cur.name})` : record.fromCurrency;
      },
    },
    {
      title: '目标币种',
      dataIndex: 'toCurrency',
      width: 110,
      render: (_, record) => {
        const cur = currencies.find((c) => c.code === record.toCurrency);
        return cur?.name ? `${record.toCurrency} (${cur.name})` : record.toCurrency;
      },
    },
    {
      title: '应收汇率',
      dataIndex: 'receivableRate',
      align: 'right',
      width: 110,
      render: (_, record) => (
        <span style={{ fontWeight: 600, color: '#1677ff' }}>
          {trimExactDecimal(record.receivableRate)}
        </span>
      ),
    },
    {
      title: '应付汇率',
      dataIndex: 'payableRate',
      align: 'right',
      width: 110,
      render: (_, record) => (
        <span style={{ fontWeight: 600, color: '#52c41a' }}>
          {trimExactDecimal(record.payableRate)}
        </span>
      ),
    },
    {
      title: '开始时间',
      dataIndex: 'effectiveFrom',
      width: 165,
      render: (_, record) =>
        record.effectiveFrom ? dayjs(record.effectiveFrom).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: '结束时间',
      dataIndex: 'effectiveTo',
      width: 165,
      render: (_, record) => {
        if (!record.effectiveTo) {
          return <Tag color="cyan" style={{ margin: 0, fontSize: 11 }}>长期有效</Tag>;
        }
        return dayjs(record.effectiveTo).format('YYYY-MM-DD HH:mm:ss');
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
      label: `${c.code} - ${c.name || ''} ${c.symbol ? `(${c.symbol})` : ''}`.trim(),
      value: c.code ?? '',
    }));

  const initialValues: Partial<ExchangeRateFormValues> = editing
    ? {
        rateType: editing.rateType,
        fromCurrency: editing.fromCurrency,
        toCurrency: editing.toCurrency,
        effectiveFrom: editing.effectiveFrom ? dayjs(editing.effectiveFrom) : undefined,
        effectiveTo: editing.effectiveTo ? dayjs(editing.effectiveTo) : undefined,
        receivableRate: trimExactDecimal(editing.receivableRate),
        payableRate: trimExactDecimal(editing.payableRate),
      }
    : { rateType: 'BASE_CURRENCY', toCurrency: baseCurrency, effectiveFrom: dayjs() };

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
            masterDataServiceListCurrencies(),
          ]);
          setBaseCurrency(rateResponse.baseCurrency ?? '');
          setCurrencies(currencyResponse.data ?? []);
          return { data: rateResponse.data ?? [], success: rateResponse.success ?? true };
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
          ...(access.canUpdateExchangeRates
            ? [
                <Button
                  key="time-standards"
                  icon={<SettingOutlined />}
                  onClick={openTimeStandards}
                >
                  时间标准配置
                </Button>,
              ]
            : []),
          ...(access.canCreateExchangeRates
            ? [
                <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
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
        form={form}
        modalProps={{ destroyOnHidden: true, onCancel: () => setModalOpen(false), width: 580 }}
        onOpenChange={(visible) => {
          setModalOpen(visible);
          if (!visible) setEditing(undefined);
        }}
        onFinish={async (values) => {
          const effectiveFrom = dayjs(values.effectiveFrom).format('YYYY-MM-DDTHH:mm:ssZ');
          const effectiveTo = values.effectiveTo
            ? dayjs(values.effectiveTo).format('YYYY-MM-DDTHH:mm:ssZ')
            : undefined;
          if (effectiveTo && (dayjs(effectiveTo).isBefore(dayjs(effectiveFrom)) || dayjs(effectiveTo).isSame(dayjs(effectiveFrom)))) {
            message.error('生效结束时间必须晚于生效开始时间');
            return false;
          }
          const fromCurrency = (values.fromCurrency || editing?.fromCurrency || '').trim().toUpperCase();
          const toCurrency = (values.toCurrency || editing?.toCurrency || baseCurrency || '').trim().toUpperCase();
          if (!fromCurrency) {
            message.error('请选择原币币种');
            return false;
          }
          if (!toCurrency) {
            message.error('未能获取目标本币');
            return false;
          }
          const input = {
            rateType: values.rateType,
            fromCurrency,
            toCurrency,
            effectiveFrom,
            effectiveTo,
            receivableRate: values.receivableRate,
            payableRate: values.payableRate,
          };
          try {
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
          } catch (err: any) {
            const msg = err.data?.message || err.response?.data?.message || err.message;
            message.error(msg || '保存汇率失败');
            return false;
          }
        }}
      >
        <ProFormSelect
          name="rateType"
          label="汇率类型"
          initialValue="BASE_CURRENCY"
          options={exchangeRateTypeOptions}
          rules={[{ required: true, message: '请选择汇率类型' }]}
        />
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
          label="目标币（机构本币）"
          disabled
          options={
            currencyOptions.length > 0
              ? currencyOptions
              : [{ label: baseCurrency, value: baseCurrency }]
          }
          rules={[{ required: true, message: '请输入目标币' }]}
        />
        <ProFormText name="receivableRate" label="应收汇率" rules={[{ validator: rateRule }]} />
        <ProFormText name="payableRate" label="应付汇率" rules={[{ validator: rateRule }]} />
        <ProFormDateTimePicker
          name="effectiveFrom"
          label="生效开始时间"
          extra="精确至秒级（左闭区间包含该时刻起生效）"
          rules={[{ required: true, message: '请选择生效开始时间' }]}
          fieldProps={{ style: { width: '100%' }, format: 'YYYY-MM-DD HH:mm:ss' }}
        />
        <ProFormDateTimePicker
          name="effectiveTo"
          label="生效结束时间"
          extra="精确至秒级（右开区间不包含该时刻）；留空表示长期有效"
          fieldProps={{ style: { width: '100%' }, format: 'YYYY-MM-DD HH:mm:ss' }}
        />
      </ModalForm>

      <ExchangeRateImportModal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onSuccess={() => actionRef.current?.reload()}
      />

      <Modal
        title="汇率类型时间标准"
        open={timeStandardsOpen}
        width={720}
        confirmLoading={timeStandardsSaving}
        onCancel={() => setTimeStandardsOpen(false)}
        onOk={saveTimeStandards}
        okText="保存"
      >
        {exchangeRateTypeOptions.map(({ label, value }) => (
          <div key={value} style={{ marginBottom: 20 }}>
            <div style={{ fontWeight: 600, marginBottom: 8 }}>{label}</div>
            <Select
              mode="multiple"
              value={timeStandards[value] ?? []}
              options={timeStandardsByRateType[value].map((standard) => ({
                label: timeStandardLabels[standard],
                value: standard,
              }))}
              style={{ width: '100%' }}
              onChange={(values) =>
                setTimeStandards((current) => ({ ...current, [value]: values }))
              }
            />
            <SortableTimeStandards
              values={timeStandards[value] ?? []}
              onChange={(values) =>
                setTimeStandards((current) => ({ ...current, [value]: values }))
              }
            />
          </div>
        ))}
      </Modal>
    </Card>
  );
}

export default ExchangeRatesPanel;
