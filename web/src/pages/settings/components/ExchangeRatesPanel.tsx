import { EditOutlined, HolderOutlined, PlusOutlined, SettingOutlined, StopOutlined } from '@ant-design/icons';
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
  ProFormDatePicker,
  ProFormSelect,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Card, Modal, Popconfirm, Select, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  exchangeRateServiceCreateExchangeRateSetting,
  exchangeRateServiceDisableExchangeRateSetting,
  exchangeRateServiceListExchangeRateTimeStandards,
  exchangeRateServiceListExchangeRateSettings,
  exchangeRateServiceUpdateExchangeRateTimeStandards,
  exchangeRateServiceUpdateExchangeRateSetting,
} from '@/services/roncin/exchangeRateService';
import { isPositiveExactDecimal, trimExactDecimal } from '../../orders/order-fee-decimal';

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
  BILL_CREATED_AT: '账单创建时间',
  WRITE_OFF_TIME: '核销时间',
};
const timeStandardsByRateType: Record<string, string[]> = {
  BASE_CURRENCY: ['ETD_ETA_TRAIN_DATE', 'BUSINESS_TIME', 'BARGE_ETD', 'ORDER_CREATED_AT'],
  INVOICE: ['BILL_CREATED_AT'],
  SETTLEMENT: [
    'ETD_ETA_TRAIN_DATE',
    'BUSINESS_TIME',
    'BARGE_ETD',
    'EXPENSE_TIME',
    'ORDER_CREATED_AT',
  ],
  WRITE_OFF: ['WRITE_OFF_TIME'],
  BILL: ['BILL_CREATED_AT'],
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
        <div style={{ display: 'grid', gap: 8, marginTop: 8 }}>
          {values.map((value) => (
            <SortableTimeStandard key={value} value={value} />
          ))}
        </div>
      </SortableContext>
    </DndContext>
  );
}

function SortableTimeStandard({ value }: { value: string }) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: value });
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
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<API.ExchangeRateSetting>();
  const [baseCurrency, setBaseCurrency] = useState('');
  const [timeStandardsOpen, setTimeStandardsOpen] = useState(false);
  const [timeStandardsSaving, setTimeStandardsSaving] = useState(false);
  const [timeStandards, setTimeStandards] = useState<Record<string, string[]>>({});

  const openCreate = () => {
    setEditing(undefined);
    setModalOpen(true);
  };

  const openTimeStandards = async () => {
    const response = await exchangeRateServiceListExchangeRateTimeStandards();
    setTimeStandards(
      Object.fromEntries(
        (response.data ?? []).map((setting) => [
          setting.rateType ?? '',
          setting.timeStandards ?? [],
        ]),
      ),
    );
    setTimeStandardsOpen(true);
  };

  const saveTimeStandards = async () => {
    if (exchangeRateTypeOptions.some(({ value }) => !timeStandards[value]?.length)) {
      message.error('每种汇率类型至少选择一个时间标准');
      return;
    }
    setTimeStandardsSaving(true);
    try {
      await exchangeRateServiceUpdateExchangeRateTimeStandards({
        data: exchangeRateTypeOptions.map(({ value }) => ({
          rateType: value,
          timeStandards: timeStandards[value],
        })),
      });
      message.success('汇率类型时间标准已更新');
      setTimeStandardsOpen(false);
    } finally {
      setTimeStandardsSaving(false);
    }
  };

  const columns: ProColumns<API.ExchangeRateSetting>[] = [
    {
      title: '汇率类型',
      dataIndex: 'rateType',
      width: 130,
      render: (_, record) => record.rateType && exchangeRateTypeLabels[record.rateType],
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
        request={async () => {
          const response = await exchangeRateServiceListExchangeRateSettings();
          setBaseCurrency(response.baseCurrency ?? '');
          return { data: response.data ?? [], success: response.success ?? true };
        }}
        toolBarRender={() => [
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
      </ModalForm>

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
