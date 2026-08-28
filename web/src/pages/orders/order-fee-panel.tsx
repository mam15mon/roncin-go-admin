import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
} from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Drawer,
  Input,
  Popconfirm,
  Space,
  Tag,
} from 'antd';
import dayjs from 'dayjs';
import React, {
  forwardRef,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import FeeFormModal, {
  type FeeFormValues,
} from './components/fees/FeeFormModal';
import QuickAddFeeModal from './components/fees/QuickAddFeeModal';
import QuickAddPartnerModal from './components/fees/QuickAddPartnerModal';
import {
  FEE_BILLED,
  FEE_CANCELLED,
  FEE_CONFIRMED,
  FEE_DRAFT,
  PAYABLE,
  RECEIVABLE,
  feeDirectionCode,
  feeStatusCode,
} from './components/fees/feeConstants';
import { trimExactDecimal } from './order-fee-decimal';
import {
  orderFeeServiceAddFee,
  orderFeeServiceConfirmFee,
  orderFeeServiceListFeeOptions,
  orderFeeServiceListFees,
  orderFeeServiceRemoveFee,
  orderFeeServiceReopenFee,
  orderFeeServiceResolveFeeExchangeRate,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';

type ExchangeRateStatus = 'idle' | 'loading' | 'resolved' | 'missing' | 'error';

type FeeRequestError = Error & {
  data?: { message?: string; reason?: string };
  response?: { data?: { message?: string; reason?: string } };
};

export type OrderFeePanelRef = {
  open: (order: API.Order) => void;
};

const OrderFeePanel = forwardRef<OrderFeePanelRef>(
  function OrderFeePanel(_, ref) {
    const access = useAccess();
    const { message, modal } = App.useApp();
    const actionRef = useRef<ActionType | undefined>(undefined);
    const exchangeRateRequestRef = useRef(0);
    const createIdempotencyKeyRef = useRef(globalThis.crypto.randomUUID());
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
    const [feeSettings, setFeeSettings] = useState<API.OrderFeeSettingOption[]>(
      [],
    );
    const [billingUnits, setBillingUnits] = useState<
      API.OrderFeeBillingUnitOption[]
    >([]);
    const [_selectedFeeSetting, setSelectedFeeSetting] =
      useState<API.OrderFeeSettingOption>();
    const [totalPreview, setTotalPreview] = useState<string>();
    const [exchangeRatePreview, setExchangeRatePreview] = useState<string>();
    const [exchangeRateStatus, setExchangeRateStatus] =
      useState<ExchangeRateStatus>('idle');
    const [manualExchangeRate, setManualExchangeRate] = useState(false);
    const [financeLocked, setFinanceLocked] = useState(false);
    const [financeLockReason, setFinanceLockReason] = useState('');
    const [financeLockCommissionNos, setFinanceLockCommissionNos] = useState<
      string[]
    >([]);
    const [customerName, setCustomerName] = useState('');
    const [quickAddFeeModalOpen, setQuickAddFeeModalOpen] = useState(false);
    const [quickAddPartnerModalOpen, setQuickAddPartnerModalOpen] =
      useState(false);
    const [taxableServices] = useState<API.TaxableService[]>([]);

    const resolveExchangeRate = (
      orderId: string,
      direction: number,
      currency: string,
      expenseDate: string,
    ) => {
      const currentRequestId = ++exchangeRateRequestRef.current;
      setExchangeRateStatus('loading');
      orderFeeServiceResolveFeeExchangeRate({
        orderId,
        direction,
        currency,
        expenseDate,
      })
        .then((response) => {
          if (currentRequestId !== exchangeRateRequestRef.current) return;
          if (response.success && response.exchangeRate) {
            setExchangeRateStatus('resolved');
            setExchangeRatePreview(trimExactDecimal(response.exchangeRate));
            if (!editingFee) {
              setManualExchangeRate(false);
            }
          } else {
            setExchangeRateStatus('missing');
            setExchangeRatePreview(undefined);
            setManualExchangeRate(true);
          }
        })
        .catch(() => {
          if (currentRequestId !== exchangeRateRequestRef.current) return;
          setExchangeRateStatus('error');
          setExchangeRatePreview(undefined);
          setManualExchangeRate(true);
        });
    };

    const handleValuesChange = () => {
      // values change handler for fee total calculation
    };

    const openCreate = () => {
      createIdempotencyKeyRef.current = globalThis.crypto.randomUUID();
      setEditingFee(undefined);
      setSelectedFeeSetting(undefined);
      setTotalPreview(undefined);
      setExchangeRatePreview(undefined);
      setExchangeRateStatus('idle');
      setManualExchangeRate(false);
      setModalOpen(true);
    };

    const openEdit = (fee: API.OrderFee) => {
      setEditingFee(fee);
      const setting = feeSettings.find((item) => item.id === fee.feeSettingId);
      setSelectedFeeSetting(setting);
      setTotalPreview(trimExactDecimal(fee.totalAmount));
      setExchangeRatePreview(
        fee.exchangeRate ? trimExactDecimal(fee.exchangeRate) : undefined,
      );
      setExchangeRateStatus(fee.exchangeRate ? 'resolved' : 'missing');
      setManualExchangeRate(fee.exchangeRateSource === 'MANUAL');
      setModalOpen(true);
    };

    useImperativeHandle(ref, () => ({
      open: async (targetOrder) => {
        setOrder(targetOrder);
        setDrawerOpen(true);
        if (!targetOrder?.id) return;
        try {
          const response = await orderFeeServiceListFeeOptions({
            orderId: targetOrder.id,
          });
          setCurrencies(response.currencies ?? []);
          setSettlementParties(response.settlementParties ?? []);
          setFeeSettings(response.feeSettings ?? []);
          setBillingUnits(response.billingUnits ?? []);
          setFinanceLocked(Boolean(response.financeLocked));
          setFinanceLockReason(response.financeLockReason ?? '');
          setFinanceLockCommissionNos(response.financeLockCommissionNos ?? []);
          setCustomerName(response.customerName ?? '');
        } catch {
          message.error('加载费用基础选项失败');
        }
      },
    }));

    const businessType = order?.businessType;
    const canCreate =
      !financeLocked &&
      businessType !== undefined &&
      access.canOrder(businessType, 'fee.create');
    const canUpdate =
      !financeLocked &&
      businessType !== undefined &&
      access.canOrder(businessType, 'fee.update');
    const canDelete =
      !financeLocked &&
      businessType !== undefined &&
      access.canOrder(businessType, 'fee.delete');

    const requestReason = (
      title: string,
      onSubmit: (reason: string) => Promise<void>,
    ) => {
      let reason = '';
      modal.confirm({
        title,
        content: (
          <Input.TextArea
            autoFocus
            maxLength={500}
            showCount
            placeholder="请输入操作原因（必填）"
            onChange={(event) => {
              reason = event.target.value.trim();
            }}
          />
        ),
        onOk: async () => {
          if (!reason) {
            message.warning('请输入操作原因');
            throw new Error('操作原因不能为空');
          }
          await onSubmit(reason);
        },
      });
    };

    const columns: ProColumns<API.OrderFee>[] = [
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        render: (_, record) => {
          if (feeStatusCode(record.status) === FEE_CONFIRMED)
            return <Tag color="green">已确认</Tag>;
          if (feeStatusCode(record.status) === FEE_BILLED)
            return <Tag color="blue">已进账单</Tag>;
          if (feeStatusCode(record.status) === FEE_CANCELLED)
            return <Tag>已作废</Tag>;
          return <Tag color="gold">草稿</Tag>;
        },
      },
      {
        title: '收付方向',
        dataIndex: 'direction',
        width: 90,
        render: (_, record) =>
          feeDirectionCode(record.direction) === PAYABLE ? (
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
            {canUpdate &&
              (feeStatusCode(record.status) === FEE_DRAFT ||
                feeStatusCode(record.status) === FEE_BILLED) && (
                <Button
                  type="link"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openEdit(record)}
                >
                  编辑
                </Button>
              )}
            {canUpdate && feeStatusCode(record.status) === FEE_DRAFT && (
              <Popconfirm
                title="确认后该费用才能进入账单，确定继续？"
                onConfirm={async () => {
                  if (!order?.id || !record.id || !record.version) return;
                  await orderFeeServiceConfirmFee(
                    { orderId: order.id, id: record.id },
                    {
                      orderId: order.id,
                      id: record.id,
                      expectedVersion: record.version,
                    },
                  );
                  message.success('费用已确认');
                  actionRef.current?.reload();
                }}
              >
                <Button type="link" size="small">
                  确认
                </Button>
              </Popconfirm>
            )}
            {canUpdate && feeStatusCode(record.status) === FEE_CONFIRMED && (
              <Button
                type="link"
                size="small"
                onClick={() =>
                  requestReason('撤回费用确认？', async (reason) => {
                    if (!order?.id || !record.id || !record.version) return;
                    await orderFeeServiceReopenFee(
                      { orderId: order.id, id: record.id },
                      {
                        orderId: order.id,
                        id: record.id,
                        expectedVersion: record.version,
                        reason,
                      },
                    );
                    message.success('费用已撤回为草稿');
                    actionRef.current?.reload();
                  })
                }
              >
                撤回
              </Button>
            )}
            {canDelete &&
              (feeStatusCode(record.status) === FEE_DRAFT ||
                feeStatusCode(record.status) === FEE_CONFIRMED) && (
                <Button
                  type="link"
                  danger
                  size="small"
                  onClick={() =>
                    requestReason('确认作废该费用？', async (reason) => {
                      if (!order?.id || !record.id || !record.version) return;
                      await orderFeeServiceRemoveFee({
                        orderId: order.id,
                        id: record.id,
                        expectedVersion: record.version,
                        reason,
                      });
                      message.success('费用已作废并保留历史记录');
                      actionRef.current?.reload();
                    })
                  }
                >
                  作废
                </Button>
              )}
          </Space>
        ),
      },
    ];

    const handleModalSubmit = async (values: FeeFormValues) => {
      if (!order?.id) return false;
      const expenseDate = dayjs(values.expenseDate).format('YYYY-MM-DD');
      const exchangeRateOverride = manualExchangeRate
        ? values.exchangeRateOverride?.trim() || undefined
        : undefined;
      const direction = values.direction ?? RECEIVABLE;

      try {
        if (editingFee?.id) {
          await orderFeeServiceUpdateFee(
            { orderId: order.id, id: editingFee.id },
            {
              orderId: order.id,
              id: editingFee.id,
              expectedVersion: editingFee.version ?? '0',
              direction,
              feeSettingId: values.feeSettingId,
              settlementPartyId: values.settlementPartyId,
              billingUnitId: values.billingUnitId,
              quantity: values.quantity,
              unitPrice: values.unitPrice,
              currency: values.currency,
              expenseDate,
              note: values.note?.trim() || undefined,
              exchangeRateOverride,
            },
          );
          message.success('费用更新成功');
        } else {
          await orderFeeServiceAddFee(
            { orderId: order.id },
            {
              orderId: order.id,
              idempotencyKey: createIdempotencyKeyRef.current,
              direction,
              feeSettingId: values.feeSettingId,
              settlementPartyId: values.settlementPartyId,
              billingUnitId: values.billingUnitId,
              quantity: values.quantity,
              unitPrice: values.unitPrice,
              currency: values.currency,
              expenseDate,
              note: values.note?.trim() || undefined,
              exchangeRateOverride,
            },
            {
              headers: {
                'X-Idempotency-Key': createIdempotencyKeyRef.current,
              },
            },
          );
          message.success('费用录入成功');
        }
        setModalOpen(false);
        actionRef.current?.reload();
        return true;
      } catch (error) {
        const err = error as FeeRequestError;
        message.error(
          err.data?.message || err.response?.data?.message || '操作费用失败',
        );
        return false;
      }
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
            setFeeSettings([]);
            setBillingUnits([]);
            setSelectedFeeSetting(undefined);
            setFinanceLocked(false);
            setFinanceLockReason('');
            setFinanceLockCommissionNos([]);
            setCustomerName('');
          }}
          size={1280}
          destroyOnHidden
        >
          {customerName && (
            <Alert
              type="info"
              showIcon
              title={`委托单位：${customerName}`}
              style={{ marginBottom: 16 }}
            />
          )}
          {financeLocked && (
            <Alert
              type="warning"
              showIcon
              title="该订单费用已进入财务锁定"
              description={`${financeLockReason || '关联提成已确认或已发放，原费用事实不可再修改。'}${financeLockCommissionNos.length > 0 ? ` 关联提成：${financeLockCommissionNos.join('、')}。` : ''} 后续差异请在提成管理中新增独立调整记录。`}
              style={{ marginBottom: 16 }}
            />
          )}
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

        <FeeFormModal
          open={modalOpen}
          onOpenChange={setModalOpen}
          editingFee={editingFee}
          modalDirection={
            editingFee
              ? feeDirectionCode(editingFee.direction)
              : RECEIVABLE
          }
          isFeeBilled={feeStatusCode(editingFee?.status) === FEE_BILLED}
          feeSettings={feeSettings}
          settlementParties={settlementParties}
          currencies={currencies}
          billingUnits={billingUnits}
          totalPreview={totalPreview}
          exchangeRateStatus={exchangeRateStatus}
          exchangeRatePreview={exchangeRatePreview}
          manualExchangeRate={manualExchangeRate}
          setManualExchangeRate={setManualExchangeRate}
          onOpenQuickAddFee={() => setQuickAddFeeModalOpen(true)}
          onOpenQuickAddPartner={() => setQuickAddPartnerModalOpen(true)}
          onValuesChange={handleValuesChange}
          onFeeSettingSelect={(setting) => {
            setSelectedFeeSetting(setting);
            if (setting?.defaultCurrency && order?.id) {
              resolveExchangeRate(
                order.id,
                editingFee ? feeDirectionCode(editingFee.direction) : RECEIVABLE,
                setting.defaultCurrency,
                dayjs().format('YYYY-MM-DD'),
              );
            }
          }}
          onSubmit={handleModalSubmit}
        />

        <QuickAddFeeModal
          open={quickAddFeeModalOpen}
          onCancel={() => setQuickAddFeeModalOpen(false)}
          currencies={currencies}
          billingUnits={billingUnits}
          taxableServices={taxableServices}
          onSuccess={(created) => {
            const newOption: API.OrderFeeSettingOption = {
              id: created.id,
              nameZh: created.nameZh,
              feeCode: created.feeCode,
              defaultBillingUnitId: created.billingUnitId,
              defaultCurrency: created.defaultCurrency,
              taxRate: created.taxRate,
            };
            setFeeSettings((prev) => [newOption, ...prev]);
            setSelectedFeeSetting(newOption);
            message.success(
              `已成功新建费用科目【${created.nameZh}】并自动选用`,
            );
            setQuickAddFeeModalOpen(false);
          }}
        />

        <QuickAddPartnerModal
          open={quickAddPartnerModalOpen}
          onCancel={() => setQuickAddPartnerModalOpen(false)}
          onSuccess={(newOption) => {
            setSettlementParties((prev) => [newOption, ...prev]);
            message.success(`已成功新建往来单位【${newOption.name}】并自动选用`);
            setQuickAddPartnerModalOpen(false);
          }}
        />
      </>
    );
  },
);

export default OrderFeePanel;
