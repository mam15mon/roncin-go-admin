import { PlusOutlined, TagOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { useQueryClient } from '@tanstack/react-query';
import { Alert, App, Button, Drawer } from 'antd';
import dayjs from 'dayjs';
import React, {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import { useAccess } from '@/app/access';
import { BusinessTagModal } from '@/components/business-tag/BusinessTagModal';
import { useModalOwnerSignal } from '@/components/layout/useModalOwnerSignal';
import { ResponsiveProTable as ProTable } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { PartnerRoleType } from '@/enums.generated';
import { useOrderFeeReview } from '@/features/orders/fee-review';
import {
  orderFeeServiceAddFee,
  orderFeeServiceBatchAssignOrderFeeTags,
  orderFeeServiceBatchRemoveOrderFeeTags,
  orderFeeServiceListFeeOptions,
  orderFeeServiceListFees,
  orderFeeServiceListOrderFeeTagOptions,
  orderFeeServiceRemoveFee,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import { isWriteResponse, toTableRequest } from '@/utils/api';
import { confirmWithReason } from '@/utils/confirmWithReason';
import { normalizeDecimalInput } from '@/utils/decimal';
import { generateUUID } from '@/utils/uuid';
import FeeFormModal, {
  type FeeFormValues,
} from './components/fees/FeeFormModal';
import { OrderFeeReviewSection } from './components/fees/fee-review/OrderFeeReview';
import { feeDirectionCode, RECEIVABLE } from './components/fees/feeConstants';
import QuickAddFeeModal from './components/fees/QuickAddFeeModal';
import QuickAddPartnerModal from './components/fees/QuickAddPartnerModal';
import { buildOrderFeePanelColumns } from './order-fee-panel-columns';
import { useOrderFeePanelExchangeRate } from './use-order-fee-panel-rate';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

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
    const queryClient = useQueryClient();
    const getModalOwnerSignal = useModalOwnerSignal();
    const { message, modal } = App.useApp();
    const actionRef = useRef<ActionType | undefined>(undefined);
    const feeOptionsRequestRef = useRef(0);
    const createIdempotencyKeyRef = useRef(generateUUID());
    const [drawerOpen, setDrawerOpen] = useState(false);
    const [selectedFeeIds, setSelectedFeeIds] = useState<React.Key[]>([]);
    const [selectedFees, setSelectedFees] = useState<API.OrderFee[]>([]);
    const [tagModalOpen, setTagModalOpen] = useState(false);
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
    const [_financeLockReason, setFinanceLockReason] = useState('');
    const [_financeLockCommissionNos, setFinanceLockCommissionNos] = useState<
      string[]
    >([]);
    const [customerName, setCustomerName] = useState('');
    const [feeOptionsReady, setFeeOptionsReady] = useState(false);
    const [feeOptionsError, setFeeOptionsError] = useState('');
    const [quickAddFeeModalOpen, setQuickAddFeeModalOpen] = useState(false);
    const [quickAddPartnerModalOpen, setQuickAddPartnerModalOpen] =
      useState(false);
    const [taxableServices] = useState<API.TaxableService[]>([]);
    const {
      state: lockState,
      loading: lockStateLoading,
      error: lockStateError,
    } = useOrderLockState(order?.id);
    const lockWritePolicy = getOrderBusinessWritePolicy({
      state: lockState,
      loading: lockStateLoading,
      error: lockStateError,
      canOperate: access.canOperateOrganization(order?.organizationId),
    });
    const feeReview = useOrderFeeReview(order?.id, drawerOpen);
    const feeWritesDisabled =
      financeLocked ||
      lockWritePolicy.disabled ||
      feeReview.isError ||
      feeReview.data?.canEdit !== true;
    const feeWritePolicyRef = useRef({ financeLocked, lockWritePolicy });
    feeWritePolicyRef.current = { financeLocked, lockWritePolicy };

    const ensureFeeWriteAllowed = () => {
      if (feeReview.isError || feeReview.data?.canEdit !== true) {
        message.warning(
          feeReview.data?.blockedReasons?.join('；') ||
            '费用审核信息未就绪或费用已冻结',
        );
        return false;
      }
      const currentPolicy = feeWritePolicyRef.current;
      if (currentPolicy.lockWritePolicy.disabled) {
        message.warning(
          currentPolicy.lockWritePolicy.reason || '订单业务费用当前不可编辑',
        );
        return false;
      }
      if (currentPolicy.financeLocked) {
        message.warning('存在有效提成，原费用不能直接修改；新增成本请申请补录');
        return false;
      }
      return true;
    };

    useEffect(() => {
      if (feeWritesDisabled) {
        setModalOpen(false);
        setTagModalOpen(false);
        setQuickAddFeeModalOpen(false);
        setQuickAddPartnerModalOpen(false);
      }
    }, [feeWritesDisabled]);

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
      if (!ensureFeeWriteAllowed()) return;
      createIdempotencyKeyRef.current = generateUUID();
      setEditingFee(undefined);
      setSelectedFeeSetting(undefined);
      resetPreview();
      setModalOpen(true);
    };

    const openEdit = (fee: API.OrderFee) => {
      if (!ensureFeeWriteAllowed()) return;
      setEditingFee(fee);
      const setting = feeSettings.find((item) => item.id === fee.feeSettingId);
      setSelectedFeeSetting(setting);
      seedFromFee(fee);
      setModalOpen(true);
    };

    useImperativeHandle(ref, () => ({
      open: async (targetOrder) => {
        const requestSequence = ++feeOptionsRequestRef.current;
        setOrder(targetOrder);
        setDrawerOpen(true);
        setSelectedFeeIds([]);
        setSelectedFees([]);
        setTagModalOpen(false);
        setModalOpen(false);
        setEditingFee(undefined);
        setCurrencies([]);
        setSettlementParties([]);
        setFeeSettings([]);
        setBillingUnits([]);
        setSelectedFeeSetting(undefined);
        setFinanceLocked(false);
        setFinanceLockReason('');
        setFinanceLockCommissionNos([]);
        setCustomerName('');
        setFeeOptionsReady(false);
        setFeeOptionsError('');
        resetPreview();
        if (!targetOrder?.id) return;
        try {
          const response = await orderFeeServiceListFeeOptions({
            orderId: targetOrder.id,
          });
          if (requestSequence !== feeOptionsRequestRef.current) return;
          setCurrencies(response.currencies ?? []);
          setSettlementParties(response.settlementParties ?? []);
          setFeeSettings(response.feeSettings ?? []);
          setBillingUnits(response.billingUnits ?? []);
          setFinanceLocked(Boolean(response.financeLocked));
          setFinanceLockReason(response.financeLockReason ?? '');
          setFinanceLockCommissionNos(response.financeLockCommissionNos ?? []);
          setCustomerName(response.customerName ?? '');
          setFeeOptionsReady(true);
        } catch (error) {
          if (requestSequence !== feeOptionsRequestRef.current) return;
          const requestError = error as FeeRequestError;
          const errorMessage =
            requestError.data?.message ??
            requestError.response?.data?.message ??
            '加载费用基础选项失败';
          setFeeOptionsError(errorMessage);
          message.error('加载费用基础选项失败');
        }
      },
    }));

    const businessType = order?.businessType;
    const canCreate =
      feeOptionsReady &&
      !feeWritesDisabled &&
      businessType !== undefined &&
      access.canOrder(businessType, 'fee.create');
    const canUpdate =
      feeOptionsReady &&
      !feeWritesDisabled &&
      businessType !== undefined &&
      access.canOrder(businessType, 'fee.update');
    const canManageTags =
      feeOptionsReady &&
      !feeWritesDisabled &&
      businessType !== undefined &&
      access.canOrder(businessType, 'fee.update');
    const canDelete =
      feeOptionsReady &&
      !feeWritesDisabled &&
      businessType !== undefined &&
      access.canOrder(businessType, 'fee.delete');

    const handleCancelFee = (record: API.OrderFee) => {
      const orderId = order?.id;
      const feeId = record.id;
      const version = record.version;
      if (!orderId || !feeId || !version) return;
      confirmWithReason(
        { modal, message },
        '确认删除该笔费用？',
        async (reason): Promise<false | undefined> => {
          if (!ensureFeeWriteAllowed()) return false;
          const writeResponse1 = await orderFeeServiceRemoveFee({
            orderId,
            id: feeId,
            expectedVersion: version,
            reason: reason || undefined,
          });
          if (!isWriteResponse(writeResponse1)) return false;
          message.success('费用已删除');
          actionRef.current?.reload();
          void queryClient.invalidateQueries({
            queryKey: ['order-fee-review'],
          });

          return undefined;
        },
        { optional: true, signal: getModalOwnerSignal() },
      );
    };

    const columns = buildOrderFeePanelColumns({
      canUpdate,
      canDelete,
      onEdit: openEdit,
      onCancelFee: handleCancelFee,
    });

    const columnSettings = useColumnSettings<ProColumns<API.OrderFee>>({
      tableKey: 'orders:fee-panel',
      columns,
    });

    const handleModalSubmit = async (values: FeeFormValues) => {
      if (!order?.id || !ensureFeeWriteAllowed()) return false;
      const expenseDate = dayjs(values.expenseDate).format('YYYY-MM-DD HH:mm');
      const direction = values.direction ?? RECEIVABLE;
      const preservesManualSnapshot =
        editingFee?.exchangeRateSource === 'MANUAL' &&
        !manualExchangeRate &&
        values.currency === editingFee.currency &&
        expenseDate ===
          dayjs(editingFee.expenseDate).format('YYYY-MM-DD HH:mm') &&
        direction === editingFee.direction;
      if (
        exchangeRateStatus === 'missing' &&
        !manualExchangeRate &&
        !preservesManualSnapshot
      ) {
        message.error('请先维护汇率');
        return false;
      }
      const exchangeRateOverride = manualExchangeRate
        ? values.exchangeRateOverride?.trim() || undefined
        : undefined;

      try {
        if (editingFee?.id) {
          const writeResponse2 = await orderFeeServiceUpdateFee(
            { orderId: order.id, id: editingFee.id },
            {
              orderId: order.id,
              id: editingFee.id,
              expectedVersion: editingFee.version ?? '0',
              direction,
              feeSettingId: values.feeSettingId,
              settlementPartyId: values.settlementPartyId,
              billingUnitId: values.billingUnitId,
              quantity: normalizeDecimalInput(String(values.quantity ?? '')),
              unitPrice: normalizeDecimalInput(String(values.unitPrice ?? '')),
              currency: values.currency,
              expenseDate,
              note: values.note?.trim() || undefined,
              exchangeRateOverride,
            },
          );
          if (!isWriteResponse(writeResponse2, true)) return false;
          message.success('费用更新成功');
        } else {
          const writeResponse3 = await orderFeeServiceAddFee(
            { orderId: order.id },
            {
              orderId: order.id,
              idempotencyKey: createIdempotencyKeyRef.current,
              direction,
              feeSettingId: values.feeSettingId,
              settlementPartyId: values.settlementPartyId,
              billingUnitId: values.billingUnitId,
              quantity: normalizeDecimalInput(String(values.quantity ?? '')),
              unitPrice: normalizeDecimalInput(String(values.unitPrice ?? '')),
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
          if (!isWriteResponse(writeResponse3, true)) return false;
          message.success('费用录入成功');
        }
        setModalOpen(false);
        actionRef.current?.reload();
        void queryClient.invalidateQueries({ queryKey: ['order-fee-review'] });
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
            feeOptionsRequestRef.current += 1;
            setDrawerOpen(false);
            setOrder(undefined);
            setSelectedFeeIds([]);
            setSelectedFees([]);
            setTagModalOpen(false);
            setModalOpen(false);
            setQuickAddFeeModalOpen(false);
            setQuickAddPartnerModalOpen(false);
            setCurrencies([]);
            setSettlementParties([]);
            setFeeSettings([]);
            setBillingUnits([]);
            setSelectedFeeSetting(undefined);
            setFinanceLocked(false);
            setFinanceLockReason('');
            setFinanceLockCommissionNos([]);
            setCustomerName('');
            setFeeOptionsReady(false);
            setFeeOptionsError('');
          }}
          size={1280}
          destroyOnHidden
        >
          {order?.id && (
            <OrderFeeReviewSection
              key={order.id}
              orderId={order.id}
              onChanged={() => actionRef.current?.reload()}
            />
          )}
          {feeOptionsError && (
            <Alert
              type="error"
              showIcon
              title="费用基础选项加载失败"
              description={feeOptionsError}
              style={{ marginBottom: 16 }}
            />
          )}
          {customerName && (
            <Alert
              type="info"
              showIcon
              title={`委托单位：${customerName}`}
              style={{ marginBottom: 16 }}
            />
          )}
          {order && (
            <ProTable<API.OrderFee>
              actionRef={actionRef}
              rowKey="id"
              columns={columnSettings.columns}
              rowSelection={{
                selectedRowKeys: selectedFeeIds,
                onChange: (keys, rows) => {
                  setSelectedFeeIds(keys);
                  setSelectedFees(rows);
                },
              }}
              bordered
              search={false}
              pagination={false}
              options={{ reload: true, density: true, setting: false }}
              request={async () => {
                const response = await orderFeeServiceListFees({
                  orderId: order.id as string,
                });
                return toTableRequest(response);
              }}
              toolBarRender={() => [
                columnSettings.entry,
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
                canManageTags && selectedFeeIds.length > 0 && (
                  <Button
                    key="tags"
                    icon={<TagOutlined />}
                    onClick={() => {
                      if (ensureFeeWriteAllowed()) setTagModalOpen(true);
                    }}
                  >
                    标签管理
                  </Button>
                ),
              ]}
            />
          )}
        </Drawer>

        <BusinessTagModal
          open={tagModalOpen}
          loadOptions={(params) =>
            orderFeeServiceListOrderFeeTagOptions({
              ...params,
              orderId: order?.id as string,
            })
          }
          targetCount={selectedFeeIds.length}
          existingTags={selectedFees.flatMap((fee) => fee.tags ?? [])}
          canQuickCreate={Boolean(access.canCreateEnterpriseResources)}
          onSubmit={async (mode, tagIds): Promise<false | undefined> => {
            if (
              !ensureFeeWriteAllowed() ||
              !order?.id ||
              selectedFeeIds.length === 0
            )
              return;
            const feeIds = selectedFeeIds.map(String);
            if (mode === 'assign') {
              const writeResponse4 =
                await orderFeeServiceBatchAssignOrderFeeTags(
                  { orderId: order.id as string },
                  { orderId: order.id as string, feeIds, tagIds },
                );
              if (!isWriteResponse(writeResponse4, 'count')) return false;
            } else {
              const writeResponse5 =
                await orderFeeServiceBatchRemoveOrderFeeTags(
                  { orderId: order.id as string },
                  { orderId: order.id as string, feeIds, tagIds },
                );
              if (!isWriteResponse(writeResponse5, 'count')) return false;
            }
            setSelectedFeeIds([]);
            setSelectedFees([]);
            actionRef.current?.reload();
            void queryClient.invalidateQueries({
              queryKey: ['order-fee-review'],
            });

            return undefined;
          }}
          onCancel={() => setTagModalOpen(false)}
        />

        <FeeFormModal
          open={modalOpen}
          onOpenChange={setModalOpen}
          editingFee={editingFee}
          modalDirection={
            editingFee ? feeDirectionCode(editingFee.direction) : RECEIVABLE
          }
          isFeeBilled={editingFee?.hasActiveBill === true}
          feeSettings={feeSettings}
          settlementParties={settlementParties}
          currencies={currencies}
          billingUnits={billingUnits}
          totalPreview={totalPreview}
          exchangeRateStatus={exchangeRateStatus}
          exchangeRatePreview={exchangeRatePreview}
          manualExchangeRate={manualExchangeRate}
          setManualExchangeRate={setManualExchangeRate}
          onOpenQuickAddFee={() => {
            if (ensureFeeWriteAllowed()) setQuickAddFeeModalOpen(true);
          }}
          onOpenQuickAddPartner={() => {
            if (ensureFeeWriteAllowed()) setQuickAddPartnerModalOpen(true);
          }}
          onValuesChange={handleValuesChange}
          onFeeSettingSelect={(setting) => {
            setSelectedFeeSetting(setting);
            if (setting?.defaultCurrency && order?.id) {
              resolveExchangeRate(
                order.id,
                editingFee
                  ? feeDirectionCode(editingFee.direction)
                  : RECEIVABLE,
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
          defaultRole={
            (editingFee
              ? feeDirectionCode(editingFee.direction)
              : RECEIVABLE) === RECEIVABLE
              ? PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER
              : PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER
          }
          onCancel={() => setQuickAddPartnerModalOpen(false)}
          onSuccess={(newOption) => {
            setSettlementParties((prev) => [newOption, ...prev]);
            message.success(
              `已成功新建往来单位【${newOption.name}】并自动选用`,
            );
            setQuickAddPartnerModalOpen(false);
          }}
        />
      </>
    );
  },
);

export default OrderFeePanel;
