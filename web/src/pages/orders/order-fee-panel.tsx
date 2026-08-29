import { PlusOutlined } from '@ant-design/icons';
import type { ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Alert, App, Button, Drawer } from 'antd';
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
  RECEIVABLE,
  feeDirectionCode,
  feeStatusCode,
} from './components/fees/feeConstants';
import { confirmWithReason } from './fee-reason-confirm';
import { buildOrderFeePanelColumns } from './order-fee-panel-columns';
import {
  orderFeeServiceAddFee,
  orderFeeServiceConfirmFee,
  orderFeeServiceListFeeOptions,
  orderFeeServiceListFees,
  orderFeeServiceRemoveFee,
  orderFeeServiceReopenFee,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import { useOrderFeePanelExchangeRate } from './use-order-fee-panel-rate';

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

    const {
      totalPreview,
      exchangeRatePreview,
      exchangeRateStatus,
      manualExchangeRate,
      setManualExchangeRate,
      resetPreview,
      seedFromFee,
      resolveExchangeRate,
    } = useOrderFeePanelExchangeRate(editingFee);

    const handleValuesChange = () => {
      // values change handler for fee total calculation
    };

    const openCreate = () => {
      createIdempotencyKeyRef.current = globalThis.crypto.randomUUID();
      setEditingFee(undefined);
      setSelectedFeeSetting(undefined);
      resetPreview();
      setModalOpen(true);
    };

    const openEdit = (fee: API.OrderFee) => {
      setEditingFee(fee);
      const setting = feeSettings.find((item) => item.id === fee.feeSettingId);
      setSelectedFeeSetting(setting);
      seedFromFee(fee);
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

    const handleConfirmFee = async (record: API.OrderFee) => {
      const orderId = order?.id;
      if (!orderId || !record.id || !record.version) return;
      await orderFeeServiceConfirmFee(
        { orderId, id: record.id },
        {
          orderId,
          id: record.id,
          expectedVersion: record.version,
        },
      );
      message.success('费用已确认');
      actionRef.current?.reload();
    };

    const handleReopenFee = (record: API.OrderFee) => {
      const orderId = order?.id;
      const feeId = record.id;
      const version = record.version;
      if (!orderId || !feeId || !version) return;
      confirmWithReason({ modal, message }, '撤回费用确认？', async (reason) => {
        await orderFeeServiceReopenFee(
          { orderId, id: feeId },
          {
            orderId,
            id: feeId,
            expectedVersion: version,
            reason,
          },
        );
        message.success('费用已撤回为草稿');
        actionRef.current?.reload();
      });
    };

    const handleCancelFee = (record: API.OrderFee) => {
      const orderId = order?.id;
      const feeId = record.id;
      const version = record.version;
      if (!orderId || !feeId || !version) return;
      confirmWithReason({ modal, message }, '确认作废该费用？', async (reason) => {
        await orderFeeServiceRemoveFee({
          orderId,
          id: feeId,
          expectedVersion: version,
          reason,
        });
        message.success('费用已作废并保留历史记录');
        actionRef.current?.reload();
      });
    };

    const columns = buildOrderFeePanelColumns({
      canUpdate,
      canDelete,
      onEdit: openEdit,
      onConfirmFee: handleConfirmFee,
      onReopenFee: handleReopenFee,
      onCancelFee: handleCancelFee,
    });

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
