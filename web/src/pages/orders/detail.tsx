import {
  CheckOutlined,
  CopyOutlined,
  DollarOutlined,
  HistoryOutlined,
  ReloadOutlined,
  UndoOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { history, useAccess, useParams } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Empty,
  type MenuProps,
  Result,
  Space,
  Spin,
  Typography,
} from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { StickyFooterBar } from '@/components/ui';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import type { OrderFormTemplateSection } from '@/components/ui/order-template/types';
import {
  OrderAllowedAction,
  OrderClosureStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import { searchPartnerOptions } from '@/utils/options';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import { PARTNER_ROLES, parseOrderKind, searchPartnersByRole } from './common';
import { buildOrderAuditTimelineSection } from './components/detail/OrderAuditTimelineSection';
import OrderDetailHeader from './components/detail/OrderDetailHeader';
import OrderPageHeader from './components/OrderPageHeader';
import { buildOrderStatusSection } from './components/detail/OrderStatusSection';
import {
  buildInitialValues,
  buildUpdatePayload,
  type OrderDetailFormValues,
} from './components/detail/orderDetailHelpers';
import SeaOrderChangeHistoryDrawer, {
  SeaOrderChangeHistorySection,
} from './components/drawers/SeaOrderChangeHistoryDrawer';
import SeaOrderReassignmentModal from './components/drawers/SeaOrderReassignmentModal';
import {
  confirmOrderClosure,
  confirmOrderTermination,
} from './order-detail-transitions';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
import { getAirTemplateSections, getSeaTemplateSections } from './templates';
import { useOrderDetailData } from './use-order-detail-data';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

const { Text } = Typography;

export default function OrderDetailPage() {
  const params = useParams<{ kind: string; id: string }>();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message, modal } = App.useApp();
  const access = useAccess();

  const kind = params.kind;
  const orderId = params.id;
  const config = parseOrderKind(kind);

  const [saving, setSaving] = useState(false);

  const {
    loading,
    order,
    shippingDocs,
    personnel,
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    searchLocations,
    currencyOptions,
    containerSpecOptions,
    personnelOptions,
    loadData,
  } = useOrderDetailData(orderId, config);

  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);

  const [changeActions, setChangeActions] =
    useState<API.SeaOrderChangeActionsData | null>(null);
  const [reassignModalOpen, setReassignModalOpen] = useState(false);
  const [historyDrawerOpen, setHistoryDrawerOpen] = useState(false);

  const loadChangeActions = async () => {
    if (!orderId || config?.category !== 'sea') return;
    try {
      const resp = await seaOrderChangeServiceGetSeaOrderChangeActions({
        orderId,
      });
      if (resp?.data) {
        setChangeActions(resp.data);
      }
    } catch (error: unknown) {
      message.error(
        error instanceof Error ? error.message : '加载拆票与改配动作失败',
      );
    }
  };

  const {
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
    refresh: refreshLockState,
  } = useOrderLockState(orderId);
  const [synchronizingLockChange, setSynchronizingLockChange] = useState(false);

  useEffect(() => {
    if (orderId && config?.category === 'sea') {
      loadChangeActions();
    }
  }, [orderId, order?.version]);

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
            path: `/orders/${config.kind}/${orderId}`,
            title: `${order.orderNo}_${config?.title || ''}详情`,
          },
        }),
      );
    }
  }, [order?.orderNo, orderId, config?.kind, config?.title]);

  // 2. 构造表单初始值
  const initialValues = useMemo(
    () => buildInitialValues(order, shippingDocs, personnel),
    [order, shippingDocs, personnel],
  );

  useEffect(() => {
    if (formRef.current && order) {
      formRef.current.setFieldsValue(initialValues);
    }
  }, [initialValues]);

  const lockWritePolicy = getOrderBusinessWritePolicy({
    state: lockState,
    loading: lockStateLoading || synchronizingLockChange,
    error: lockStateError,
  });
  const businessWritesDisabled = lockWritePolicy.disabled;
  const businessWritePolicyRef = useRef(lockWritePolicy);
  businessWritePolicyRef.current = lockWritePolicy;

  const ensureBusinessWriteAllowed = () => {
    const currentPolicy = businessWritePolicyRef.current;
    if (!currentPolicy.disabled) return true;
    message.warning(currentPolicy.reason || '订单当前不可编辑');
    return false;
  };

  const synchronizeLockChange = async () => {
    setSynchronizingLockChange(true);
    try {
      await Promise.all([loadData(), refreshLockState()]);
      if (config?.category === 'sea') {
        await loadChangeActions();
      }
    } finally {
      setSynchronizingLockChange(false);
    }
  };

  useEffect(() => {
    if (businessWritesDisabled) {
      setReassignModalOpen(false);
    }
  }, [businessWritesDisabled]);

  // 3. 复用与新建页 100% 相同的一套分节构建器（传入 isDetail: true）
  const templateProps = useMemo(
    () => ({
      serviceTypeOptions,
      cargoCategoryOptions,
      locationOptions,
      searchLocations,
      currencyOptions,
      containerSpecOptions,
      isDetail: true,
      searchCustomers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CUSTOMER, keyword),
      searchCarriers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CARRIER, keyword),
      searchBookingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword),
      searchForeignAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.FOREIGN_AGENT, keyword),
      searchShippingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword),
      searchIssuers: (keyword?: string) => searchPartnerOptions(keyword),
      setCustomerCode: (code?: string) =>
        formRef.current?.setFieldValue('customerCode', code ?? ''),
      checkCustomerReferenceNo: async () => {},
      checkInternalReferenceNo: async () => {},
      personnelOptions,
      readonly: businessWritesDisabled,
      onOrderDataChanged: loadData,
    }),
    [
      serviceTypeOptions,
      cargoCategoryOptions,
      locationOptions,
      searchLocations,
      currencyOptions,
      containerSpecOptions,
      personnelOptions,
      businessWritesDisabled,
      loadData,
    ],
  );

  const formSections = useMemo(() => {
    if (config?.category === 'air') {
      return getAirTemplateSections(templateProps);
    }
    return getSeaTemplateSections(templateProps);
  }, [config?.category, templateProps]);

  // 4. 海管家风格「订单状态」卡片（作为前置区块）
  const prependSections: OrderFormTemplateSection[] = useMemo(
    () => [buildOrderStatusSection(order)],
    [order],
  );

  // 5. 后置区块：拆票/改配历史与操作记录日志
  const appendSections: OrderFormTemplateSection[] = useMemo(
    () => [
      ...(config?.category === 'sea' && orderId
        ? [
            {
              key: 'sea-order-change-history',
              title: '拆票与改配记录',
              content: (
                <SeaOrderChangeHistorySection
                  orderId={orderId}
                  onOpenAll={() => setHistoryDrawerOpen(true)}
                />
              ),
            },
          ]
        : []),
      buildOrderAuditTimelineSection(order),
    ],
    [config?.category, order, orderId],
  );

  // 6. 保存修改提交处理
  const handleSaveEdit = async (values: OrderDetailFormValues) => {
    if (!orderId || !ensureBusinessWriteAllowed()) return false;
    setSaving(true);
    try {
      const payload = buildUpdatePayload(
        orderId,
        order?.version || '0',
        values,
      );
      await orderServiceUpdateOrder({ id: orderId }, payload);
      message.success('保存订单成功');
      await Promise.all([loadData(), refreshLockState()]);
      return true;
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '保存订单失败');
      return false;
    } finally {
      setSaving(false);
    }
  };

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
          page="detail"
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
          <Spin size="large" description="正在加载订单详情..." />
        </div>
      </div>
    );
  }

  if (!order) {
    return (
      <div style={{ background: '#f5f7fa', minHeight: '100vh' }}>
        <OrderPageHeader
          page="detail"
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

  const progressStage =
    order.closureStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED
      ? '已完结'
      : order.terminationStatus ===
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED
        ? '已退关'
        : '进行中';

  const hasAction = (action: number) =>
    order.allowedActions?.includes(action) === true;

  const confirmTermination = (targetStatus: number) => {
    if (!ensureBusinessWriteAllowed()) return;
    confirmOrderTermination(
      { modal, message },
      order,
      targetStatus,
      async () => {
        await Promise.all([loadData(), refreshLockState()]);
      },
      ensureBusinessWriteAllowed,
    );
  };

  const confirmClosure = (targetStatus: number) => {
    if (!ensureBusinessWriteAllowed()) return;
    confirmOrderClosure(
      { modal, message },
      order,
      targetStatus,
      async () => {
        await Promise.all([loadData(), refreshLockState()]);
      },
      ensureBusinessWriteAllowed,
    );
  };

  const moreMenuItems: MenuProps['items'] = [
    {
      key: 'fees-drawer',
      icon: <DollarOutlined />,
      label: '快速费用抽屉',
      disabled: !access.canOrder(config.businessType, 'fee.read'),
      onClick: () => orderFeePanelRef.current?.open(order),
    },
    {
      key: 'copy-orderno',
      icon: <CopyOutlined />,
      label: '复制订单号',
      onClick: () => {
        if (order.orderNo) {
          navigator.clipboard.writeText(order.orderNo);
          message.success('已复制订单号');
        }
      },
    },
    {
      key: 'change-history',
      icon: <HistoryOutlined />,
      label: '拆票与改配历史',
      onClick: () => setHistoryDrawerOpen(true),
    },
    {
      key: 'reload-data',
      icon: <ReloadOutlined />,
      label: '刷新数据',
      onClick: () => {
        void loadData();
        void refreshLockState();
      },
    },
  ];

  return (
    <>
      <OrderFormTemplate<OrderDetailFormValues>
        loading={false}
        readonly={
          !hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) ||
          businessWritesDisabled
        }
        formRef={formRef}
        initialValues={initialValues}
        onFinish={handleSaveEdit}
        header={
          <OrderDetailHeader
            kind={config.kind}
            orderId={orderId || ''}
            configTitle={config.title}
            order={order}
            saving={saving}
            canManageFee={access.canOrder(config.businessType, 'fee.read')}
            canCreatePod={access.canOrder(
              config.businessType,
              'release_pod.create',
            )}
            canCreateAbnormal={access.canOrder(
              config.businessType,
              'abnormal_case.create',
            )}
            canSplit={
              config.category === 'sea' &&
              access.canOrder(config.businessType, 'split')
            }
            canReassign={
              config.category === 'sea' &&
              access.canOrder(config.businessType, 'reassign')
            }
            splitDisabled={!changeActions?.canSplit}
            splitBlockedReasons={changeActions?.splitBlockedReasons}
            reassignDisabled={!changeActions?.canReassign}
            reassignBlockedReasons={changeActions?.reassignBlockedReasons}
            moreMenuItems={moreMenuItems}
            hasAction={hasAction}
            onSave={() => formRef.current?.submit()}
            onConfirmTermination={confirmTermination}
            onConfirmClosure={confirmClosure}
            onOpenReleasePod={() => {
              if (ensureBusinessWriteAllowed()) {
                releasePodPanelRef.current?.open(order);
              }
            }}
            onOpenAbnormalCase={() => {
              if (ensureBusinessWriteAllowed()) {
                abnormalCasePanelRef.current?.open(order);
              }
            }}
            onOpenSplit={() => {
              if (ensureBusinessWriteAllowed()) {
                history.push(`/orders/sea-export/${orderId}/split`);
              }
            }}
            onOpenReassign={() => {
              if (ensureBusinessWriteAllowed()) {
                setReassignModalOpen(true);
              }
            }}
            lockState={lockState}
            lockStateLoading={lockStateLoading || synchronizingLockChange}
            lockStateError={lockStateError}
            businessWritesDisabled={businessWritesDisabled}
            businessWriteBlockedReason={lockWritePolicy.reason}
            onRetryLockState={refreshLockState}
            onSynchronizeLockChange={synchronizeLockChange}
          />
        }
        prependSections={prependSections}
        sections={formSections}
        appendSections={appendSections}
        footer={
          <StickyFooterBar
            info={
              <Space>
                <Text strong>{order.orderNo}</Text>
                <Text type="secondary">{progressStage}</Text>
              </Space>
            }
          >
            {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) &&
              !businessWritesDisabled && (
                <Button
                  icon={<UndoOutlined />}
                  onClick={() => formRef.current?.setFieldsValue(initialValues)}
                >
                  重置修改
                </Button>
              )}
            {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) &&
              !businessWritesDisabled && (
                <Button
                  type="primary"
                  icon={<CheckOutlined />}
                  loading={saving}
                  onClick={() => formRef.current?.submit()}
                >
                  保存修改
                </Button>
              )}
          </StickyFooterBar>
        }
      />

      {/* 挂载功能弹窗 */}
      <ReleasePodPanel
        ref={releasePodPanelRef}
        canManage={
          !businessWritesDisabled &&
          access.canOrder(config.businessType, 'release_pod.create')
        }
      />
      <OrderFeePanel ref={orderFeePanelRef} />
      <AbnormalCasePanel
        ref={abnormalCasePanelRef}
        canManage={
          !businessWritesDisabled &&
          access.canOrder(config.businessType, 'abnormal_case.create')
        }
        masterOptions={[]}
      />

      {orderId && (
        <>
          <SeaOrderReassignmentModal
            orderId={orderId}
            orderNo={order?.orderNo}
            open={reassignModalOpen}
            disabled={businessWritesDisabled}
            disabledReason={lockWritePolicy.reason}
            onClose={() => setReassignModalOpen(false)}
            onSuccess={async () => {
              await Promise.all([loadData(), refreshLockState()]);
              await loadChangeActions();
            }}
            searchCarriers={templateProps.searchCarriers}
            searchIssuers={templateProps.searchIssuers}
            searchLocations={templateProps.searchLocations}
          />
          <SeaOrderChangeHistoryDrawer
            orderId={orderId}
            open={historyDrawerOpen}
            onClose={() => setHistoryDrawerOpen(false)}
          />
        </>
      )}
    </>
  );
}
