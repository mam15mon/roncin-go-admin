import {
  LockOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { history, useAccess, useParams } from '@umijs/max';
import { App, Button, Card, Empty, Result, Spin, Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { FinanceSummaryBoard } from '@/components/ui';
import { OrderFlowStatus } from '@/enums.generated';
import BillCreationWorkbench from '@/pages/finance/bills/components/BillCreationWorkbench';
import { feeCatalogServiceListTaxableServices } from '@/services/roncin/feeCatalogService';
import {
  orderFeeServiceAddFee,
  orderFeeServiceConfirmFee,
  orderFeeServiceRemoveFee,
  orderFeeServiceReopenFee,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import { unwrapList } from '@/utils/api';
import { confirmWithReason } from '@/utils/confirmWithReason';
import { trimDecimal } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import { parseOrderKind } from './common';
import FeeFormModal, {
  type FeeFormValues,
} from './components/fees/FeeFormModal';
import {
  FEE_BILLED,
  feeStatusCode,
  RECEIVABLE,
} from './components/fees/feeConstants';
import OrderFeeHeader from './components/fees/OrderFeeHeader';
import OrderPageHeader from './components/OrderPageHeader';
import OrderFeeTableTabs from './components/fees/OrderFeeTableTabs';
import { getOrderFeeTableColumns } from './components/fees/orderFeeColumns';
import QuickAddFeeModal from './components/fees/QuickAddFeeModal';
import QuickAddPartnerModal from './components/fees/QuickAddPartnerModal';
import { useFeeExchangePreview } from './use-fee-exchange-preview';
import { useOrderFeeOptions } from './use-order-fee-options';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

export default function OrderFeesPage() {
  const params = useParams<{ kind: string; id: string }>();
  const access = useAccess();
  const { message, modal } = App.useApp();

  const kind = params.kind;
  const orderId = params.id;
  const config = parseOrderKind(kind);

  const receivableActionRef = useRef<ActionType | undefined>(undefined);
  const payableActionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance<FeeFormValues> | undefined>(undefined);
  const createIdempotencyKeyRef = useRef(generateUUID());

  const {
    loading,
    order,
    currencies,
    settlementParties,
    setSettlementParties,
    feeSettings,
    setFeeSettings,
    billingUnits,
    financeLocked,
    financeLockReason,
    financeLockCommissionNos,
    customerName,
    loadData,
  } = useOrderFeeOptions(orderId);
  const {
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
    refresh: refreshLockState,
  } = useOrderLockState(orderId);

  const {
    totalPreview,
    exchangeRatePreview,
    exchangeRateStatus,
    manualExchangeRate,
    setManualExchangeRate,
    resetPreview,
    seedFromFee,
    handleValuesChange,
  } = useFeeExchangePreview(orderId, formRef);

  const [modalOpen, setModalOpen] = useState(false);
  const [modalDirection, setModalDirection] = useState<number>(RECEIVABLE);
  const [editingFee, setEditingFee] = useState<API.OrderFee>();
  const [billWorkbenchOpen, setBillWorkbenchOpen] = useState(false);
  const [billWorkbenchFeeIds, setBillWorkbenchFeeIds] = useState<string[]>([]);
  const [selectedReceivableFeeIds, setSelectedReceivableFeeIds] = useState<
    React.Key[]
  >([]);
  const [selectedPayableFeeIds, setSelectedPayableFeeIds] = useState<
    React.Key[]
  >([]);
  const [allReceivableItems, setAllReceivableItems] = useState<API.OrderFee[]>(
    [],
  );
  const [allPayableItems, setAllPayableItems] = useState<API.OrderFee[]>([]);
  const [_selectedFeeSetting, setSelectedFeeSetting] =
    useState<API.OrderFeeSettingOption>();

  // 汇总统计数据
  const [receivableSummary, setReceivableSummary] = useState<{
    totalAmount: number;
    count: number;
  }>({ totalAmount: 0, count: 0 });
  const [payableSummary, setPayableSummary] = useState<{
    totalAmount: number;
    count: number;
  }>({ totalAmount: 0, count: 0 });

  // 快捷新增费目状态
  const [quickAddFeeModalOpen, setQuickAddFeeModalOpen] = useState(false);
  const [taxableServices, setTaxableServices] = useState<API.TaxableService[]>(
    [],
  );

  // 快捷新建结算单位状态
  const [quickAddPartnerModalOpen, setQuickAddPartnerModalOpen] =
    useState(false);

  const lockWritePolicy = getOrderBusinessWritePolicy({
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
  });
  const feeWritesDisabled = financeLocked || lockWritePolicy.disabled;
  const feeWritePolicyRef = useRef({
    financeLocked,
    lockWritePolicy,
  });
  feeWritePolicyRef.current = { financeLocked, lockWritePolicy };

  const ensureFeeWriteAllowed = () => {
    const currentPolicy = feeWritePolicyRef.current;
    if (currentPolicy.lockWritePolicy.disabled) {
      message.warning(
        currentPolicy.lockWritePolicy.reason || '订单业务费用当前不可编辑',
      );
      return false;
    }
    if (currentPolicy.financeLocked) {
      message.warning('订单财务已锁定，请在提成管理中创建独立调整记录');
      return false;
    }
    return true;
  };

  useEffect(() => {
    if (feeWritesDisabled) {
      setModalOpen(false);
      setQuickAddFeeModalOpen(false);
      setQuickAddPartnerModalOpen(false);
    }
  }, [feeWritesDisabled]);

  useEffect(() => {
    if (
      order?.orderNo &&
      orderId &&
      config?.kind &&
      typeof window !== 'undefined'
    ) {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: `/orders/${config.kind}/${orderId}/fees`,
            title: `${order.orderNo}_费用录入`,
          },
        }),
      );
    }
  }, [order?.orderNo, orderId, config?.kind]);

  const handleOpenQuickAddFee = async () => {
    if (!ensureFeeWriteAllowed()) return;
    setQuickAddFeeModalOpen(true);
    try {
      const res = await feeCatalogServiceListTaxableServices({
        skipErrorHandler: true,
      });
      setTaxableServices(unwrapList(res));
    } catch {
      // ignore
    }
  };

  const handleOpenQuickAddPartner = () => {
    if (!ensureFeeWriteAllowed()) return;
    setQuickAddPartnerModalOpen(true);
  };

  const openFeeModal = (direction: number, fee?: API.OrderFee) => {
    if (!ensureFeeWriteAllowed()) return;
    if (!fee) createIdempotencyKeyRef.current = generateUUID();
    setEditingFee(fee);
    setModalDirection(direction);
    setSelectedFeeSetting(undefined);
    resetPreview();
    setModalOpen(true);

    if (fee) {
      const setting = feeSettings.find((item) => item.id === fee.feeSettingId);
      setSelectedFeeSetting(setting);
      seedFromFee(fee);
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          direction: fee.direction ?? direction,
          feeSettingId: fee.feeSettingId ?? '',
          settlementPartyId: fee.settlementPartyId ?? '',
          billingUnitId: fee.billingUnitId ?? '',
          quantity: fee.quantity ? trimDecimal(fee.quantity) : '',
          unitPrice: fee.unitPrice ? trimDecimal(fee.unitPrice) : '',
          currency: fee.currency ?? '',
          expenseDate: fee.expenseDate ? dayjs(fee.expenseDate) : dayjs(),
          note: fee.note ?? '',
        });
      }, 0);
    } else {
      const defaultParty =
        direction === RECEIVABLE
          ? order?.customerId
          : order?.bookingAgentId || order?.carrierId;
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          direction,
          settlementPartyId: defaultParty ?? '',
          currency: 'CNY',
          quantity: '1',
          expenseDate: dayjs(),
        });
        handleValuesChange();
      }, 0);
    }
  };

  const handleModalSubmit = async (values: FeeFormValues) => {
    if (!orderId || !ensureFeeWriteAllowed()) return false;
    const body = {
      direction: values.direction ?? modalDirection,
      feeSettingId: values.feeSettingId,
      settlementPartyId: values.settlementPartyId,
      billingUnitId: values.billingUnitId,
      quantity: values.quantity,
      unitPrice: values.unitPrice,
      currency: values.currency,
      expenseDate: dayjs(values.expenseDate).format('YYYY-MM-DD'),
      note: values.note,
      exchangeRateOverride: manualExchangeRate
        ? values.exchangeRateOverride
        : undefined,
      taxInclusive: true,
    };

    try {
      if (editingFee?.id) {
        if (!editingFee.version)
          throw new Error('费用版本信息缺失，请刷新后重试');
        await orderFeeServiceUpdateFee(
          { orderId, id: editingFee.id },
          {
            ...body,
            orderId,
            id: editingFee.id,
            expectedVersion: editingFee.version,
          },
        );
        message.success('费用更新成功');
      } else {
        await orderFeeServiceAddFee(
          { orderId },
          { ...body, orderId, idempotencyKey: createIdempotencyKeyRef.current },
        );
        createIdempotencyKeyRef.current = generateUUID();
        message.success('费用录入成功');
      }
      setModalOpen(false);
      receivableActionRef.current?.reload();
      payableActionRef.current?.reload();
      return true;
    } catch (error: any) {
      message.error(error.message || '保存费用失败');
      return false;
    }
  };

  const reloadFeeTables = () => {
    receivableActionRef.current?.reload();
    payableActionRef.current?.reload();
  };

  const handleCancelFee = (fee: API.OrderFee) => {
    const feeId = fee.id;
    const version = fee.version;
    if (!orderId || !feeId || !version) return;
    confirmWithReason(
      { modal, message },
      '确认作废该笔费用？',
      async (reason) => {
        if (!ensureFeeWriteAllowed()) return;
        await orderFeeServiceRemoveFee({
          orderId,
          id: feeId,
          expectedVersion: version,
          reason,
        });
        message.success('费用已作废并保留历史记录');
        reloadFeeTables();
      },
    );
  };

  const handleConfirmFee = async (fee: API.OrderFee) => {
    if (!ensureFeeWriteAllowed() || !orderId || !fee.id || !fee.version) return;
    try {
      await orderFeeServiceConfirmFee(
        { orderId, id: fee.id },
        { orderId, id: fee.id, expectedVersion: fee.version },
      );
      message.success('费用已确认，可以进入账单');
      reloadFeeTables();
    } catch (error: any) {
      message.error(error.message || '确认费用失败');
    }
  };

  const handleReopenFee = (fee: API.OrderFee) => {
    const feeId = fee.id;
    const version = fee.version;
    if (!orderId || !feeId || !version) return;
    confirmWithReason({ modal, message }, '撤回费用确认？', async (reason) => {
      if (!ensureFeeWriteAllowed()) return;
      await orderFeeServiceReopenFee(
        { orderId, id: feeId },
        { orderId, id: feeId, expectedVersion: version, reason },
      );
      message.success('费用已撤回为草稿');
      reloadFeeTables();
    });
  };

  const getTableColumns = (direction: number): ProColumns<API.OrderFee>[] =>
    getOrderFeeTableColumns({
      direction,
      feeWritesDisabled,
      onOpenModal: openFeeModal,
      onConfirmFee: handleConfirmFee,
      onReopenFee: handleReopenFee,
      onCancelFee: handleCancelFee,
    });

  if (!config) {
    return (
      <div style={{ padding: 48, background: '#f5f7fa', minHeight: '100vh' }}>
        <Result
          status="404"
          title="业务类型不存在"
          subTitle={`未知的业务类型路径 "${params.kind || ''}"，请选择有效业务入口。`}
          extra={
            <Button
              type="primary"
              onClick={() => history.push('/orders/sea-export')}
            >
              返回海运出口订单
            </Button>
          }
        />
      </div>
    );
  }

  if (loading) {
    return (
      <div style={{ background: '#f5f7fa', minHeight: '100vh' }}>
        <OrderPageHeader
          page="fees"
          orderKind={config.kind}
          orderId={orderId}
          orderNo={order?.orderNo}
        />
        <div
          style={{
            textAlign: 'center',
            padding: '120px 0',
          }}
        >
          <Spin size="large" description="正在加载费用工作台..." />
        </div>
      </div>
    );
  }

  if (!order) {
    return (
      <div style={{ background: '#f5f7fa', minHeight: '100vh' }}>
        <OrderPageHeader
          page="fees"
          orderKind={config.kind}
          orderId={orderId}
          orderNo={orderId}
        />
        <div style={{ padding: 48 }}>
          <Card
            variant="borderless"
            style={{ borderRadius: 8, textAlign: 'center', padding: 32 }}
          >
            <Empty description="未找到对应的订单档案" />
            <Button
              type="primary"
              onClick={() => history.push(`/orders/${config.kind}`)}
              style={{ marginTop: 16 }}
            >
              返回订单列表
            </Button>
          </Card>
        </div>
      </div>
    );
  }

  const profitCny = receivableSummary.totalAmount - payableSummary.totalAmount;
  const profitRate =
    receivableSummary.totalAmount > 0
      ? ((profitCny / receivableSummary.totalAmount) * 100).toFixed(1)
      : '0.0';

  return (
    <div
      style={{ padding: '0 0 40px', background: '#f5f7fa', minHeight: '100vh' }}
    >
      <OrderPageHeader
        page="fees"
        orderKind={config.kind}
        orderId={orderId}
        orderNo={order.orderNo}
        tags={
          <>
            {order.canModify === false &&
              order.flowStatus !== OrderFlowStatus.ORDER_FLOW_STATUS_DRAFT && (
                <Tag color="warning" icon={<LockOutlined />}>
                  已锁单
                </Tag>
              )}
            {financeLocked && (
              <Tag color="red" icon={<LockOutlined />}>
                财务已关账
              </Tag>
            )}
          </>
        }
        extra={
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              void loadData();
              void refreshLockState();
              receivableActionRef.current?.reload();
              payableActionRef.current?.reload();
            }}
          >
            刷新数据
          </Button>
        }
      />

      <div style={{ maxWidth: 1440, margin: '16px auto 0', padding: '0 24px' }}>
        <OrderFeeHeader
          order={order}
          kind={config.kind}
          orderId={orderId || ''}
          configTitle={config.title}
          customerName={customerName}
          financeLocked={financeLocked}
          financeLockReason={financeLockReason}
          financeLockCommissionNos={financeLockCommissionNos}
          lockWritePolicy={lockWritePolicy}
          onRetryLockState={refreshLockState}
          receivableSummary={receivableSummary}
          payableSummary={payableSummary}
          profitCny={profitCny}
          profitRate={profitRate}
        />

        {/* 3. 费用表格工作区 */}
        <OrderFeeTableTabs
          orderId={orderId || ''}
          receivableActionRef={receivableActionRef}
          payableActionRef={payableActionRef}
          receivableSummary={receivableSummary}
          payableSummary={payableSummary}
          selectedReceivableFeeIds={selectedReceivableFeeIds}
          setSelectedReceivableFeeIds={setSelectedReceivableFeeIds}
          selectedPayableFeeIds={selectedPayableFeeIds}
          setSelectedPayableFeeIds={setSelectedPayableFeeIds}
          setAllReceivableItems={setAllReceivableItems}
          setAllPayableItems={setAllPayableItems}
          setReceivableSummary={setReceivableSummary}
          setPayableSummary={setPayableSummary}
          canCreateFinanceBills={Boolean(access.canCreateFinanceBills)}
          feeWritesDisabled={feeWritesDisabled}
          onOpenBillWorkbench={(feeIds) => {
            setBillWorkbenchFeeIds(feeIds);
            setBillWorkbenchOpen(true);
          }}
          onOpenFeeModal={openFeeModal}
          getTableColumns={getTableColumns}
        />

        {/* 底部双层多币种动态汇总看板 */}
        <FinanceSummaryBoard
          selectedRows={[...allReceivableItems, ...allPayableItems].filter(
            (f) =>
              Boolean(f.id) &&
              (selectedReceivableFeeIds.includes(f.id || '') ||
                selectedPayableFeeIds.includes(f.id || '')),
          )}
          allRows={[...allReceivableItems, ...allPayableItems]}
        />
      </div>

      <BillCreationWorkbench
        open={billWorkbenchOpen}
        initialFeeIds={billWorkbenchFeeIds}
        sourceLabel={`订单 ${order.orderNo || order.id}`}
        onClose={() => setBillWorkbenchOpen(false)}
        onCreated={() => {
          setSelectedReceivableFeeIds([]);
          setSelectedPayableFeeIds([]);
          receivableActionRef.current?.reload();
          payableActionRef.current?.reload();
        }}
      />

      {/* 4. 费用录入/编辑 ModalForm */}
      <FeeFormModal
        open={modalOpen}
        onOpenChange={setModalOpen}
        editingFee={editingFee}
        modalDirection={modalDirection}
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
        onOpenQuickAddFee={handleOpenQuickAddFee}
        onOpenQuickAddPartner={handleOpenQuickAddPartner}
        onValuesChange={handleValuesChange}
        onFeeSettingSelect={setSelectedFeeSetting}
        onSubmit={handleModalSubmit}
      />

      {/* 快捷新增费用科目 Modal */}
      <QuickAddFeeModal
        open={quickAddFeeModalOpen}
        onCancel={() => setQuickAddFeeModalOpen(false)}
        currencies={currencies}
        billingUnits={billingUnits}
        taxableServices={taxableServices}
        onSuccess={(created) => {
          const newOption: API.OrderFeeSettingOption = {
            id: created.id,
            feeCode: created.feeCode,
            nameZh: created.nameZh,
            nameEn: created.nameEn,
            defaultCurrency: created.defaultCurrency,
            defaultBillingUnitId: created.billingUnitId,
            defaultBillingUnitName: billingUnits.find(
              (b) => b.id === created.billingUnitId,
            )?.name,
            taxRate: created.taxRate,
          };
          setFeeSettings((prev) => [newOption, ...prev]);
          formRef.current?.setFieldValue('feeSettingId', created.id);
          setSelectedFeeSetting(newOption);
          if (created.billingUnitId) {
            formRef.current?.setFieldValue(
              'billingUnitId',
              created.billingUnitId,
            );
          }
          if (created.defaultCurrency) {
            formRef.current?.setFieldValue('currency', created.defaultCurrency);
          }
          handleValuesChange();
          message.success(`已成功新建费用科目【${created.nameZh}】并自动选用`);
          setQuickAddFeeModalOpen(false);
        }}
      />

      {/* 快捷新建往来单位 Modal */}
      <QuickAddPartnerModal
        open={quickAddPartnerModalOpen}
        onCancel={() => setQuickAddPartnerModalOpen(false)}
        onSuccess={(newOption) => {
          setSettlementParties((prev) => [newOption, ...prev]);
          formRef.current?.setFieldValue('settlementPartyId', newOption.id);
          message.success(`已成功新建往来单位【${newOption.name}】并自动选用`);
          setQuickAddPartnerModalOpen(false);
        }}
      />
    </div>
  );
}
