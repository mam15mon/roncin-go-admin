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
  exchangeRatePattern,
  isPositiveExactDecimal,
  quantityOrPricePattern,
  trimExactDecimal,
} from './order-fee-decimal';

const RECEIVABLE = 1;
const PAYABLE = 2;

type FeeFormValues = {
  direction: number;
  feeSettingId: string;
  settlementPartyId: string;
  billingUnitId: string;
  quantity: string;
  unitPrice: string;
  currency: string;
  expenseDate: string | Dayjs;
  note?: string;
  exchangeRateOverride?: string;
};

type ExchangeRateStatus = 'idle' | 'loading' | 'resolved' | 'missing' | 'error';

type FeeRequestError = Error & {
  data?: { message?: string; reason?: string };
  response?: { data?: { message?: string; reason?: string } };
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
    const [feeSettings, setFeeSettings] = useState<
      API.OrderFeeSettingOption[]
    >([]);
    const [billingUnits, setBillingUnits] = useState<
      API.OrderFeeBillingUnitOption[]
    >([]);
    const [selectedFeeSetting, setSelectedFeeSetting] =
      useState<API.OrderFeeSettingOption>();
    const [totalPreview, setTotalPreview] = useState<string>();
    const [exchangeRatePreview, setExchangeRatePreview] = useState<string>();
    const [exchangeRateStatus, setExchangeRateStatus] =
      useState<ExchangeRateStatus>('idle');
    const [manualExchangeRate, setManualExchangeRate] = useState(false);

    useImperativeHandle(ref, () => ({
      open: (record) => {
        setOrder(record);
        setDrawerOpen(true);
        void orderFeeServiceListFeeOptions({ orderId: record.id as string })
          .then((response) => {
            setCurrencies(response.currencies ?? []);
            setSettlementParties(response.settlementParties ?? []);
            setFeeSettings(response.feeSettings ?? []);
            setBillingUnits(response.billingUnits ?? []);
          })
          .catch((error: Error) =>
            message.error(error.message || '费用录入选项加载失败'),
          );
      },
    }));

    const resolveExchangeRate = (
      orderId: string,
      direction: number,
      currency: string,
      expenseDate: string,
    ) => {
      const requestSequence = ++exchangeRateRequestRef.current;
      setExchangeRateStatus('loading');
      setExchangeRatePreview(undefined);
      setManualExchangeRate(false);
      formRef.current?.setFieldValue('exchangeRateOverride', undefined);
      void orderFeeServiceResolveFeeExchangeRate(
        { orderId, direction, currency, expenseDate },
        { skipErrorHandler: true },
      )
        .then((response) => {
          if (requestSequence !== exchangeRateRequestRef.current) return;
          if (!response.exchangeRate) {
            setExchangeRateStatus('error');
            message.error('汇率解析结果不完整');
            return;
          }
          setExchangeRatePreview(trimExactDecimal(response.exchangeRate));
          setExchangeRateStatus('resolved');
        })
        .catch((rawError: unknown) => {
          if (requestSequence !== exchangeRateRequestRef.current) return;
          const error = rawError as FeeRequestError;
          const envelope = error.data ?? error.response?.data;
          if (envelope?.reason === 'FEE_EXCHANGE_RATE_MISSING') {
            setExchangeRateStatus('missing');
            setManualExchangeRate(access.canOverrideFeeExchangeRate);
            return;
          }
          setExchangeRateStatus('error');
          message.error(envelope?.message || error.message || '汇率解析失败');
        });
    };

    const openCreate = () => {
      setEditingFee(undefined);
      setSelectedFeeSetting(undefined);
      setTotalPreview(undefined);
      setExchangeRatePreview(undefined);
      setExchangeRateStatus('idle');
      setManualExchangeRate(false);
      formRef.current?.resetFields();
      setModalOpen(true);
    };

    const openEdit = (fee: API.OrderFee) => {
      setEditingFee(fee);
      setSelectedFeeSetting(
        feeSettings.find((item) => item.id === fee.feeSettingId),
      );
      setTotalPreview(calculateExactFeeTotal(fee.quantity, fee.unitPrice));
      setExchangeRatePreview(undefined);
      setExchangeRateStatus('idle');
      setManualExchangeRate(false);
      setModalOpen(true);
      if (order?.id && fee.direction && fee.currency && fee.expenseDate) {
        resolveExchangeRate(
          order.id,
          fee.direction,
          fee.currency,
          fee.expenseDate,
        );
      }
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
        width: 160,
        align: 'right',
        render: (_, record) => (
          <Space size={4}>
            {trimExactDecimal(record.exchangeRate)}
            {record.exchangeRateSource === 'MANUAL' && (
              <Tag color="gold">手工</Tag>
            )}
            {record.exchangeRateSource === 'SYSTEM' && (
              <Tag color="blue">系统</Tag>
            )}
            {record.exchangeRateSource === 'BASE_CURRENCY' && <Tag>本币</Tag>}
          </Space>
        ),
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
          feeSettingId: editingFee.feeSettingId,
          settlementPartyId: editingFee.settlementPartyId,
          billingUnitId: editingFee.billingUnitId,
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
          quantity: '1',
          expenseDate: dayjs(),
        };

    const exchangeRateSubmissionBlocked =
      exchangeRateStatus === 'idle' ||
      exchangeRateStatus === 'loading' ||
      exchangeRateStatus === 'error' ||
      (exchangeRateStatus === 'missing' && !manualExchangeRate);
    const exchangeRateDisplay =
      exchangeRateStatus === 'loading'
        ? '解析中'
        : exchangeRateStatus === 'missing'
          ? '未配置'
          : exchangeRateStatus === 'error'
            ? '解析失败'
            : (exchangeRatePreview ?? '待解析');

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
            setFeeSettings([]);
            setBillingUnits([]);
            setSelectedFeeSetting(undefined);
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
          submitter={{
            submitButtonProps: { disabled: exchangeRateSubmissionBlocked },
          }}
          onValuesChange={(changed, values) => {
            if ('feeSettingId' in changed) {
              const setting = feeSettings.find(
                (item) => item.id === values.feeSettingId,
              );
              setSelectedFeeSetting(setting);
              formRef.current?.setFieldsValue({
                billingUnitId: setting?.defaultBillingUnitId,
                currency: setting?.defaultCurrency,
              });
              if (
                setting?.defaultCurrency &&
                order?.id &&
                values.direction &&
                values.expenseDate
              ) {
                resolveExchangeRate(
                  order.id,
                  values.direction,
                  setting.defaultCurrency,
                  dayjs(values.expenseDate).format('YYYY-MM-DD'),
                );
              } else {
                exchangeRateRequestRef.current++;
                setExchangeRateStatus('idle');
                setExchangeRatePreview(undefined);
                setManualExchangeRate(false);
              }
            }
            if (
              'currency' in changed ||
              'direction' in changed ||
              'expenseDate' in changed
            ) {
              if (
                order?.id &&
                values.currency &&
                values.direction &&
                values.expenseDate
              ) {
                resolveExchangeRate(
                  order.id,
                  values.direction,
                  values.currency,
                  dayjs(values.expenseDate).format('YYYY-MM-DD'),
                );
              } else {
                exchangeRateRequestRef.current++;
                setExchangeRateStatus('idle');
                setExchangeRatePreview(undefined);
                setManualExchangeRate(false);
                formRef.current?.setFieldValue(
                  'exchangeRateOverride',
                  undefined,
                );
              }
            }
            setTotalPreview(
              calculateExactFeeTotal(values.quantity, values.unitPrice),
            );
          }}
          onFinish={async (values) => {
            if (!order?.id) return false;
            if (exchangeRateSubmissionBlocked) {
              message.warning('请先确认当前费用的结算汇率');
              return false;
            }
            const expenseDate = dayjs(values.expenseDate).format('YYYY-MM-DD');
            const payload = {
              orderId: order.id,
              direction: values.direction,
              feeSettingId: values.feeSettingId,
              settlementPartyId: values.settlementPartyId,
              billingUnitId: values.billingUnitId,
              quantity: values.quantity,
              unitPrice: values.unitPrice,
              currency: values.currency,
              expenseDate,
              note: values.note?.trim() || undefined,
              exchangeRateOverride: manualExchangeRate
                ? values.exchangeRateOverride
                : undefined,
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
          <ProFormSelect
            colProps={{ span: 24 }}
            name="feeSettingId"
            label="费用设置"
            rules={[{ required: true, message: '请选择费用设置' }]}
            showSearch
            options={feeSettings.map((item) => ({
              label: `${item.feeCode} - ${item.nameZh}${item.aliasName ? `（${item.aliasName}）` : ''}`,
              value: item.id,
            }))}
            placeholder="请选择适用于当前订单的费用"
          />
          <ProFormText
            colProps={{ span: 12 }}
            label="费用代码"
            fieldProps={{
              value: selectedFeeSetting?.feeCode ?? '',
              disabled: true,
            }}
          />
          <ProFormText
            colProps={{ span: 12 }}
            label="费用名称"
            fieldProps={{
              value: selectedFeeSetting?.nameZh ?? '',
              disabled: true,
            }}
          />
          <ProFormText
            colProps={{ span: 12 }}
            label="费用名称（英文）"
            fieldProps={{
              value: selectedFeeSetting?.nameEn ?? '',
              disabled: true,
            }}
          />
          <ProFormText
            colProps={{ span: 12 }}
            label="税率"
            fieldProps={{
              value: selectedFeeSetting?.taxRate
                ? `${trimExactDecimal(selectedFeeSetting.taxRate)}%`
                : '',
              disabled: true,
            }}
          />
          <ProFormText
            colProps={{ span: 12 }}
            label="货物或应税劳务名称"
            fieldProps={{
              value: selectedFeeSetting?.taxableServiceName ?? '',
              disabled: true,
            }}
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
          <ProFormSelect
            colProps={{ span: 12 }}
            name="billingUnitId"
            label="计费单位"
            rules={[{ required: true, message: '请选择计费单位' }]}
            options={billingUnits.map((item) => ({
              label: `${item.name} (${item.code})`,
              value: item.id,
            }))}
            showSearch
            placeholder="请选择计费单位"
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
          {manualExchangeRate ? (
            <ProFormText
              key="manual-exchange-rate"
              colProps={{ span: 12 }}
              name="exchangeRateOverride"
              label="手工结算汇率"
              rules={[
                { required: true, message: '请输入手工结算汇率' },
                {
                  validator: positiveDecimalRule(
                    exchangeRatePattern,
                    '汇率最多 10 位整数、8 位小数',
                  ),
                },
              ]}
              fieldProps={{ inputMode: 'decimal' }}
              placeholder="最多 8 位小数"
              extra={
                <Space size="small">
                  <span>仅覆盖当前这笔费用</span>
                  {exchangeRateStatus !== 'missing' && (
                    <Button
                      type="link"
                      size="small"
                      style={{ padding: 0 }}
                      onClick={() => {
                        setManualExchangeRate(false);
                        formRef.current?.setFieldValue(
                          'exchangeRateOverride',
                          undefined,
                        );
                      }}
                    >
                      取消覆盖
                    </Button>
                  )}
                </Space>
              }
            />
          ) : (
            <ProFormText
              key="system-exchange-rate"
              colProps={{ span: 12 }}
              label="结算汇率"
              fieldProps={{ value: exchangeRateDisplay, disabled: true }}
              extra={
                access.canOverrideFeeExchangeRate &&
                exchangeRateStatus === 'resolved' ? (
                  <Button
                    type="link"
                    size="small"
                    style={{ padding: 0 }}
                    onClick={() => {
                      setManualExchangeRate(true);
                      formRef.current?.setFieldValue(
                        'exchangeRateOverride',
                        exchangeRatePreview,
                      );
                    }}
                  >
                    手工覆盖
                  </Button>
                ) : null
              }
            />
          )}
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
          {exchangeRateStatus === 'missing' && (
            <Alert
              style={{ gridColumn: '1 / -1' }}
              type="warning"
              showIcon
              message="当前费用日期所在期间未配置生效汇率"
              description={
                access.canOverrideFeeExchangeRate
                  ? '你可以仅为当前这笔费用录入手工汇率，该值不会改动公司汇率主数据。'
                  : '请联系拥有全公司汇率维护权限的人员配置该期间汇率。'
              }
            />
          )}
          {exchangeRateStatus === 'error' && (
            <Alert
              style={{ gridColumn: '1 / -1' }}
              type="error"
              showIcon
              message="汇率解析失败，暂时不能提交费用"
            />
          )}
          <Alert
            style={{ gridColumn: '1 / -1' }}
            type="info"
            showIcon
            icon={<DollarOutlined />}
            message={
              totalPreview
                ? `精确总金额：${trimExactDecimal(totalPreview)} ${formRef.current?.getFieldValue('currency') || ''}；结算汇率：${manualExchangeRate ? '手工录入' : exchangeRateDisplay}`
                : '总金额由服务端使用精确十进制计算，汇率默认从公司汇率数据带入。'
            }
          />
        </ModalForm>
      </>
    );
  },
);

export default OrderFeePanel;
