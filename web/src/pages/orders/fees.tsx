import {
  ArrowLeftOutlined,
  LockOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { history, useAccess, useParams } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Empty,
  Space,
  Spin,
  Tag,
} from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { FinanceSummaryBoard } from '@/components/ui';
import BillCreationWorkbench from '@/pages/finance/bills/components/BillCreationWorkbench';
import { unwrapList } from '@/utils/api';
import FeeFormModal, {
  type FeeFormValues,
} from './components/fees/FeeFormModal';
import OrderFeeHeader from './components/fees/OrderFeeHeader';
import { getOrderFeeTableColumns } from './components/fees/orderFeeColumns';
import OrderFeeTableTabs from './components/fees/OrderFeeTableTabs';
import QuickAddFeeModal from './components/fees/QuickAddFeeModal';
import QuickAddPartnerModal from './components/fees/QuickAddPartnerModal';
import { feeCatalogServiceListTaxableServices } from '@/services/roncin/feeCatalogService';
import {
  orderFeeServiceAddFee,
  orderFeeServiceConfirmFee,
  orderFeeServiceRemoveFee,
  orderFeeServiceReopenFee,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import { parseOrderKind } from './common';
import { trimExactDecimal } from '@/utils/decimal';
import {
  FEE_BILLED,
  RECEIVABLE,
  feeStatusCode,
} from './components/fees/feeConstants';
import { confirmWithReason } from '@/utils/confirmWithReason';
import { useFeeExchangePreview } from './use-fee-exchange-preview';
import { useOrderFeeOptions } from './use-order-fee-options';

export default function OrderFeesPage() {
  const params = useParams<{ kind: string; id: string }>();
  const access = useAccess();
  const { message, modal } = App.useApp();

  const kind = params.kind || 'sea-export';
  const orderId = params.id;
  const config = parseOrderKind(kind) || {
    kind: 'sea-export',
    title: '海运出口',
    businessType: 1,
    category: 'sea',
  };

  const receivableActionRef = useRef<ActionType | undefined>(undefined);
  const payableActionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance<FeeFormValues> | undefined>(undefined);
  const createIdempotencyKeyRef = useRef(globalThis.crypto.randomUUID());

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
  const [taxableServices, setTaxableServices] = useState<API.TaxableService[]>([]);

  // 快捷新建结算单位状态
  const [quickAddPartnerModalOpen, setQuickAddPartnerModalOpen] = useState(false);

  useEffect(() => {
    if (order?.orderNo && typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: window.location.pathname,
            title: `${order.orderNo}_费用录入`,
          },
        }),
      );
    }
  }, [order?.orderNo]);

  const handleOpenQuickAddFee = async () => {
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
    setQuickAddPartnerModalOpen(true);
  };

  const openFeeModal = (direction: number, fee?: API.OrderFee) => {
    if (financeLocked) {
      message.warning('订单财务已锁定，请在提成管理中创建独立调整记录');
      return;
    }
    if (!fee) createIdempotencyKeyRef.current = globalThis.crypto.randomUUID();
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
          quantity: fee.quantity ? trimExactDecimal(fee.quantity) : '',
          unitPrice: fee.unitPrice ? trimExactDecimal(fee.unitPrice) : '',
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
    if (!orderId) return false;
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
        createIdempotencyKeyRef.current = globalThis.crypto.randomUUID();
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
    confirmWithReason({ modal, message }, '确认作废该笔费用？', async (reason) => {
      await orderFeeServiceRemoveFee({
        orderId,
        id: feeId,
        expectedVersion: version,
        reason,
      });
      message.success('费用已作废并保留历史记录');
      reloadFeeTables();
    });
  };

  const handleConfirmFee = async (fee: API.OrderFee) => {
    if (!orderId || !fee.id || !fee.version) return;
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
      financeLocked,
      onOpenModal: openFeeModal,
      onConfirmFee: handleConfirmFee,
      onReopenFee: handleReopenFee,
      onCancelFee: handleCancelFee,
    });

  if (loading) {
    return (
      <div
        style={{
          textAlign: 'center',
          padding: '120px 0',
          background: '#f5f7fa',
          minHeight: '100vh',
        }}
      >
        <Spin size="large" description="正在加载费用工作台..." />
      </div>
    );
  }

  if (!order) {
    return (
      <div style={{ padding: 48, background: '#f5f7fa', minHeight: '100vh' }}>
        <Card
          variant="borderless"
          style={{ borderRadius: 8, textAlign: 'center', padding: 32 }}
        >
          <Empty description="未找到对应的订单档案" />
          <Button
            type="primary"
            onClick={() => history.push(`/orders/${kind}`)}
            style={{ marginTop: 16 }}
          >
            返回订单列表
          </Button>
        </Card>
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
      {/* 顶部面包屑与快捷返回 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '10px 24px',
          background: '#ffffff',
          borderBottom: '1px solid #e2e8f0',
          marginBottom: 16,
        }}
      >
        <Space size={8}>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => history.push(`/orders/${kind}/${orderId}`)}
          >
            返回订单详情
          </Button>
          <span style={{ color: '#cbd5e1' }}>|</span>
          <span style={{ color: '#64748b' }}>{config.title}</span>
          <span>&gt;</span>
          <a
            style={{
              color: '#1677ff',
              fontWeight: 600,
              fontFamily: 'monospace',
            }}
            onClick={() => history.push(`/orders/${kind}/${orderId}`)}
          >
            {order.orderNo || order.id}
          </a>
          <span>&gt;</span>
          <span style={{ fontWeight: 600, color: '#0f172a' }}>费用录入</span>
          {order.canModify === false && order.flowStatus !== 1 && (
            <Tag color="warning" icon={<LockOutlined />}>
              已锁单
            </Tag>
          )}
          {financeLocked && (
            <Tag color="red" icon={<LockOutlined />}>
              财务已关账
            </Tag>
          )}
        </Space>

        <Space size={8}>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              void loadData();
              receivableActionRef.current?.reload();
              payableActionRef.current?.reload();
            }}
          >
            刷新数据
          </Button>
          <Button
            type="primary"
            onClick={() => history.push(`/orders/${kind}/${orderId}`)}
          >
            回到订单详情
          </Button>
        </Space>
      </div>

      <div style={{ maxWidth: 1440, margin: '0 auto', padding: '0 24px' }}>
        <OrderFeeHeader
          order={order}
          kind={kind}
          orderId={orderId || ''}
          configTitle={config.title}
          customerName={customerName}
          financeLocked={financeLocked}
          financeLockReason={financeLockReason}
          financeLockCommissionNos={financeLockCommissionNos}
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
          financeLocked={financeLocked}
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
            formRef.current?.setFieldValue('billingUnitId', created.billingUnitId);
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
