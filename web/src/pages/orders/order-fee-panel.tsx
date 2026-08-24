import { DollarOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Alert, App, Button, Drawer, Popconfirm, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, {
  forwardRef,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import {
  orderFeeServiceAddFee,
  orderFeeServiceListFeeOptions,
  orderFeeServiceListFees,
  orderFeeServiceRemoveFee,
  orderFeeServiceResolveFeeExchangeRate,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import {
  calculateExactFeeTotal,
  isPositiveExactDecimal,
  quantityOrPricePattern,
  trimExactDecimal,
} from './order-fee-decimal';

const RECEIVABLE = 1;
const PAYABLE = 2;

type FeeFormValues = {
  direction: number;
  feeCode: string;
  feeName: string;
  settlementPartyId: string;
  billingUnit: string;
  quantity: string;
  unitPrice: string;
  currency: string;
  expenseDate: string | Dayjs;
  note?: string;
};

export type OrderFeePanelRef = {
  open: (order: API.Order) => void;
};

function positiveDecimalRule(pattern: RegExp, precisionMessage: string) {
  return async (_: unknown, value?: string) => {
    if (!value) throw new Error('请输入数值');
    if (!pattern.test(value)) throw new Error(precisionMessage);
    if (!isPositiveExactDecimal(value, pattern)) {
      throw new Error('数值必须大于 0');
    }
  };
}

const OrderFeePanel = forwardRef<OrderFeePanelRef>(
  function OrderFeePanel(_, ref) {
    const access = useAccess();
    const { message } = App.useApp();
    const actionRef = useRef<ActionType | undefined>(undefined);
    const exchangeRateRequestRef = useRef(0);
    const formRef = useRef<ProFormInstance<FeeFormValues> | undefined>(
      undefined,
    );
    const [drawerOpen, setDrawerOpen] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const [order, setOrder] = useState<API.Order>();
    const [editingFee, setEditingFee] = useState<API.OrderFee>();
    const [currencies, setCurrencies] = useState<API.OrderFeeCurrencyOption[]>(
      [],
    );
    const [settlementParties, setSettlementParties] = useState<
      API.OrderFeeSettlementPartyOption[]
    >([]);
    const [baseCurrency, setBaseCurrency] = useState('');
    const [totalPreview, setTotalPreview] = useState<string>();
    const [exchangeRatePreview, setExchangeRatePreview] = useState<string>();

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDrawerOpen(true);
        void orderFeeServiceListFeeOptions({ orderId: record.id as string })
          .then((response) => {
            setCurrencies(response.currencies ?? []);
            setSettlementParties(response.settlementParties ?? []);
            setBaseCurrency(response.baseCurrency ?? '');
          })
          .catch((error: Error) =>
            message.error(error.message || '费用录入选项加载失败'),
          );
      },
    }));

    const openCreate = () => {
      setEditingFee(undefined);
      setTotalPreview(undefined);
      setExchangeRatePreview('1');
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const openEdit = (fee: API.OrderFee) => {
      setEditingFee(fee);
      setTotalPreview(calculateExactFeeTotal(fee.quantity, fee.unitPrice));
      setExchangeRatePreview(trimExactDecimal(fee.exchangeRate));
      setModalOpen(true);
    };

    const businessType = order?.businessType;
    const canCreate =
      businessType !== undefined && access.canOrder(businessType, 'fee.create');
    const canUpdate =
      businessType !== undefined && access.canOrder(businessType, 'fee.update');
    const canDelete =
      businessType !== undefined && access.canOrder(businessType, 'fee.delete');

    const columns: ProColumns<API.OrderFee>[] = [
      {
        title: '收付方向',
        dataIndex: 'direction',
        width: 90,
        render: (_, record) =>
          record.direction === PAYABLE ? (
            <Tag color="volcano">应付</Tag>
          ) : (
            <Tag color="green">应收</Tag>
          ),
      },
      {
        title: '费用代码',
        dataIndex: 'feeCode',
        width: 130,
        copyable: true,
      },
      {
        title: '费用名称',
        dataIndex: 'feeName',
        width: 150,
        ellipsis: true,
      },
      {
        title: '结算单位',
        dataIndex: 'settlementPartyName',
        width: 190,
        ellipsis: true,
      },
      {
        title: '计费单位',
        dataIndex: 'billingUnit',
        width: 90,
      },
      {
        title: '数量',
        dataIndex: 'quantity',
        width: 110,
        align: 'right',
        render: (_, record) => trimExactDecimal(record.quantity),
      },
      {
        title: '单价',
        dataIndex: 'unitPrice',
        width: 130,
        align: 'right',
        render: (_, record) => trimExactDecimal(record.unitPrice),
      },
      {
        title: '总金额',
        dataIndex: 'totalAmount',
        width: 150,
        align: 'right',
        render: (_, record) => (
          <strong>
            {trimExactDecimal(record.totalAmount)} {record.currency}
          </strong>
        ),
      },
      {
        title: '汇率',
        dataIndex: 'exchangeRate',
        width: 110,
        align: 'right',
        render: (_, record) => trimExactDecimal(record.exchangeRate),
      },
      {
        title: '费用日期',
        dataIndex: 'expenseDate',
        width: 110,
      },
      {
        title: '备注',
        dataIndex: 'note',
        width: 180,
        ellipsis: true,
        render: (_, record) => record.note || '-',
      },
      {
        title: '操作',
        valueType: 'option',
        width: 120,
        fixed: 'right',
        render: (_, record) => (
          <Space size="small">
            {canUpdate && (
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => openEdit(record)}
              >
                编辑
              </Button>
            )}
            {canDelete && (
              <Popconfirm
                title="确定删除该费用？"
                description="当前仅有费用录入，删除后不可恢复。"
                onConfirm={async () => {
                  if (!order?.id || !record.id) return;
                  await orderFeeServiceRemoveFee({
                    orderId: order.id,
                    id: record.id,
                  });
                  message.success('删除费用成功');
                  actionRef.current?.reload();
                }}
              >
                <Button type="link" danger size="small">
                  删除
                </Button>
              </Popconfirm>
            )}
          </Space>
        ),
      },
    ];

    const initialValues: Partial<FeeFormValues> = editingFee
      ? {
          direction: editingFee.direction,
          feeCode: editingFee.feeCode,
          feeName: editingFee.feeName,
          settlementPartyId: editingFee.settlementPartyId,
          billingUnit: editingFee.billingUnit,
          quantity: trimExactDecimal(editingFee.quantity),
          unitPrice: trimExactDecimal(editingFee.unitPrice),
          currency: editingFee.currency,
          expenseDate: editingFee.expenseDate
            ? dayjs(editingFee.expenseDate)
            : undefined,
          note: editingFee.note,
        }
      : {
          direction: RECEIVABLE,
          billingUnit: '票',
          quantity: '1',
          currency: baseCurrency,
          expenseDate: dayjs(),
        };

    return (
      <>
        <Drawer
          title={order ? `费用录入 - ${order.orderNo || order.id}` : '费用录入'}
          open={drawerOpen}
          onClose={() => {
            setDrawerOpen(false);
            setOrder(undefined);
            setCurrencies([]);
            setSettlementParties([]);
            setBaseCurrency('');
          }}
          width={1280}
          destroyOnHidden
        >
          {order?.id && (
            <ProTable<API.OrderFee>
              actionRef={actionRef}
              rowKey="id"
              columns={columns}
              bordered
              search={false}
              pagination={false}
              scroll={{ x: 1500 }}
              request={async () => {
                const response = await orderFeeServiceListFees({
                  orderId: order.id as string,
                });
                return {
                  data: response.data ?? [],
                  success: response.success ?? true,
                };
              }}
              toolBarRender={() => [
                canCreate && (
                  <Button
                    key="create"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openCreate}
                  >
                    录入费用
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <ModalForm<FeeFormValues>
          title={editingFee ? '编辑费用' : '录入费用'}
          open={modalOpen}
          formRef={formRef}
          initialValues={initialValues}
          grid
          modalProps={{
            destroyOnHidden: true,
            width: 760,
            onCancel: () => setModalOpen(false),
          }}
          onOpenChange={setModalOpen}
          onValuesChange={(changed, values) => {
            if (
              ('currency' in changed ||
                'direction' in changed ||
                'expenseDate' in changed) &&
              order?.id &&
              values.currency &&
              values.direction &&
              values.expenseDate
            ) {
              const expenseDate = dayjs(values.expenseDate).format('YYYY-MM-DD');
              const requestSequence = ++exchangeRateRequestRef.current;
              setExchangeRatePreview(undefined);
              void orderFeeServiceResolveFeeExchangeRate({
                orderId: order.id,
                direction: values.direction,
                currency: values.currency,
                expenseDate,
              })
                .then((response) => {
                  if (requestSequence === exchangeRateRequestRef.current) {
                    setExchangeRatePreview(
                      trimExactDecimal(response.exchangeRate),
                    );
                  }
                })
                .catch((error: Error) => {
                  if (requestSequence === exchangeRateRequestRef.current) {
                    message.error(error.message || '汇率解析失败');
                  }
                });
            }
            setTotalPreview(
              calculateExactFeeTotal(values.quantity, values.unitPrice),
            );
          }}
          onFinish={async (values) => {
            if (!order?.id) return false;
            const expenseDate = dayjs(values.expenseDate).format('YYYY-MM-DD');
            const payload = {
              orderId: order.id,
              direction: values.direction,
              feeCode: values.feeCode.trim(),
              feeName: values.feeName.trim(),
              settlementPartyId: values.settlementPartyId,
              billingUnit: values.billingUnit.trim(),
              quantity: values.quantity,
              unitPrice: values.unitPrice,
              currency: values.currency,
              expenseDate,
              note: values.note?.trim() || undefined,
            };
            if (editingFee?.id) {
              await orderFeeServiceUpdateFee(
                { orderId: order.id, id: editingFee.id },
                { ...payload, id: editingFee.id },
              );
              message.success('更新费用成功');
            } else {
              await orderFeeServiceAddFee({ orderId: order.id }, payload);
              message.success('录入费用成功');
            }
            setModalOpen(false);
            actionRef.current?.reload();
            return true;
          }}
        >
          <ProFormSelect
            colProps={{ span: 12 }}
            name="direction"
            label="应收 / 应付"
            rules={[{ required: true, message: '请选择收付方向' }]}
            options={[
              { label: '应收', value: RECEIVABLE },
              { label: '应付', value: PAYABLE },
            ]}
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="feeCode"
            label="费用代码"
            rules={[{ required: true, message: '请输入费用代码' }]}
            fieldProps={{ maxLength: 30 }}
            placeholder="例如 OCEAN_FREIGHT"
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="feeName"
            label="费用名称"
            rules={[{ required: true, message: '请输入费用名称' }]}
            fieldProps={{ maxLength: 80 }}
            placeholder="例如 海运费"
          />
          <ProFormSelect
            colProps={{ span: 12 }}
            name="settlementPartyId"
            label="结算单位"
            rules={[{ required: true, message: '请选择结算单位' }]}
            showSearch
            options={settlementParties.map((item) => ({
              label: `${item.name} (${item.code})`,
              value: item.id,
            }))}
            placeholder="搜索往来单位"
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="billingUnit"
            label="计费单位"
            rules={[{ required: true, message: '请输入计费单位' }]}
            fieldProps={{ maxLength: 32 }}
            placeholder="例如 票、箱、公斤"
          />
          <ProFormSelect
            colProps={{ span: 12 }}
            name="currency"
            label="币种"
            rules={[{ required: true, message: '请选择币种' }]}
            options={currencies.map((item) => ({
              label: `${item.code} - ${item.name}`,
              value: item.code,
            }))}
            showSearch
            placeholder="请选择币种"
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="quantity"
            label="数量"
            rules={[
              { required: true, message: '请输入数量' },
              {
                validator: positiveDecimalRule(
                  quantityOrPricePattern,
                  '数量最多 10 位整数、4 位小数',
                ),
              },
            ]}
            fieldProps={{ inputMode: 'decimal' }}
            placeholder="最多 4 位小数"
          />
          <ProFormText
            colProps={{ span: 12 }}
            name="unitPrice"
            label="单价"
            rules={[
              { required: true, message: '请输入单价' },
              {
                validator: positiveDecimalRule(
                  quantityOrPricePattern,
                  '单价最多 10 位整数、4 位小数',
                ),
              },
            ]}
            fieldProps={{ inputMode: 'decimal' }}
            placeholder="最多 4 位小数"
          />
          <ProFormText
            colProps={{ span: 12 }}
            label="结算汇率"
            fieldProps={{ value: exchangeRatePreview ?? '待解析', disabled: true }}
          />
          <ProFormDatePicker
            colProps={{ span: 12 }}
            name="expenseDate"
            label="费用日期"
            rules={[{ required: true, message: '请选择费用日期' }]}
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormTextArea
            colProps={{ span: 24 }}
            name="note"
            label="备注"
            fieldProps={{ maxLength: 500, showCount: true }}
          />
          <Alert
            style={{ gridColumn: '1 / -1' }}
            type="info"
            showIcon
            icon={<DollarOutlined />}
            message={
              totalPreview
                ? `精确总金额：${trimExactDecimal(totalPreview)} ${formRef.current?.getFieldValue('currency') || ''}；结算汇率：${exchangeRatePreview ?? '待解析'}`
                : '总金额和汇率均由服务端精确计算、解析，不接受客户端覆盖。'
            }
          />
        </ModalForm>
      </>
    );
  },
);

export default OrderFeePanel;
