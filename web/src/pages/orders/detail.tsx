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
  OrderBusinessType,
  OrderClosureStatus,
  OrderTerminationStatus,
  TradeDirection,
} from '@/enums.generated';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import {
  PARTNER_ROLES,
  parseOrderKind,
  searchPartnersByRole,
} from './common';
import { searchPartnerOptions } from '@/utils/options';
import { buildOrderAuditTimelineSection } from './components/detail/OrderAuditTimelineSection';
import OrderDetailHeader from './components/detail/OrderDetailHeader';
import { buildOrderStatusSection } from './components/detail/OrderStatusSection';
import {
  buildInitialValues,
  buildUpdatePayload,
  type OrderDetailFormValues,
} from './components/detail/orderDetailHelpers';
import {
  confirmOrderClosure,
  confirmOrderTermination,
} from './order-detail-transitions';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
import {
  getAirTemplateSections,
  getSeaTemplateSections,
} from './templates';
import { useOrderDetailData } from './use-order-detail-data';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import SeaOrderReassignmentModal from './components/drawers/SeaOrderReassignmentModal';
import SeaOrderChangeHistoryDrawer, {
  SeaOrderChangeHistorySection,
} from './components/drawers/SeaOrderChangeHistoryDrawer';

const { Text } = Typography;

export default function OrderDetailPage() {
  const params = useParams<{ kind: string; id: string }>();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message, modal } = App.useApp();
  const access = useAccess();

  const kind = params.kind || 'sea-export';
  const orderId = params.id;
  const config = parseOrderKind(kind) || {
    kind: 'sea-export' as const,
    title: '海运出口',
    businessType: OrderBusinessType.BUSINESS_TYPE_SE,
    tradeDirection: TradeDirection.TRADE_DIRECTION_EXPORT,
    category: 'sea' as const,
  };

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
    if (!orderId || config.category !== 'sea') return;
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

  useEffect(() => {
    if (orderId && config.category === 'sea') {
      loadChangeActions();
    }
  }, [orderId, order?.version]);

  useEffect(() => {
    if (order?.orderNo && typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: window.location.pathname,
            title: `${order.orderNo}_${config.title}详情`,
          },
        }),
      );
    }
  }, [order?.orderNo]);

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
      searchIssuers: (keyword?: string) =>
        searchPartnerOptions(keyword),
      setCustomerCode: (code?: string) =>
        formRef.current?.setFieldValue('customerCode', code ?? ''),
      checkCustomerReferenceNo: async () => {},
      checkInternalReferenceNo: async () => {},
      personnelOptions,
    }),
    [
      serviceTypeOptions,
      cargoCategoryOptions,
      locationOptions,
      searchLocations,
      currencyOptions,
      containerSpecOptions,
      personnelOptions,
    ],
  );

  const formSections = useMemo(() => {
    if (config.category === 'air') {
      return getAirTemplateSections(templateProps);
    }
    return getSeaTemplateSections(templateProps);
  }, [config.category, templateProps]);

  // 4. 海管家风格「订单状态」卡片（作为前置区块）
  const prependSections: OrderFormTemplateSection[] = useMemo(
    () => [buildOrderStatusSection(order)],
    [order],
  );

  // 5. 后置区块：拆票/改配历史与操作记录日志
  const appendSections: OrderFormTemplateSection[] = useMemo(
    () => [
      ...(config.category === 'sea' && orderId
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
    [config.category, order, orderId],
  );

  // 6. 保存修改提交处理
  const handleSaveEdit = async (values: OrderDetailFormValues) => {
    if (!orderId) return false;
    setSaving(true);
    try {
      const payload = buildUpdatePayload(
        orderId,
        order?.version || '0',
        values,
      );
      await orderServiceUpdateOrder({ id: orderId }, payload);
      message.success('保存订单成功');
      await loadData();
      return true;
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '保存订单失败');
      return false;
    } finally {
      setSaving(false);
    }
  };

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
        <Spin size="large" description="正在加载订单详情..." />
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

  const progressStage =
    order.closureStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED
      ? '已完结'
      : order.terminationStatus ===
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED
        ? '已退关'
        : '进行中';

  const hasAction = (action: number) =>
    order.allowedActions?.includes(action) === true;

  const confirmTermination = (targetStatus: number) =>
    confirmOrderTermination(
      { modal, message },
      order,
      targetStatus,
      loadData,
    );

  const confirmClosure = (targetStatus: number) =>
    confirmOrderClosure(
      { modal, message },
      order,
      targetStatus,
      loadData,
    );

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
      onClick: () => void loadData(),
    },
  ];

  return (
    <>
      <OrderFormTemplate<OrderDetailFormValues>
        loading={false}
        readonly={!hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT)}
        formRef={formRef}
        initialValues={initialValues}
        onFinish={handleSaveEdit}
        header={
          <OrderDetailHeader
            kind={kind}
            orderId={orderId || ''}
            configTitle={config.title}
            businessType={String(config.businessType)}
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
            onOpenReleasePod={() => releasePodPanelRef.current?.open(order)}
            onOpenAbnormalCase={() =>
              abnormalCasePanelRef.current?.open(order)
            }
            onOpenSplit={() =>
              history.push(`/orders/sea-export/${orderId}/split`)
            }
            onOpenReassign={() => setReassignModalOpen(true)}
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
            {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) && (
              <Button
                icon={<UndoOutlined />}
                onClick={() => formRef.current?.setFieldsValue(initialValues)}
              >
                重置修改
              </Button>
            )}
            {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) && (
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
        canManage={access.canOrder(config.businessType, 'release_pod.create')}
      />
      <OrderFeePanel ref={orderFeePanelRef} />
      <AbnormalCasePanel
        ref={abnormalCasePanelRef}
        canManage={access.canOrder(config.businessType, 'abnormal_case.create')}
        masterOptions={[]}
      />

      {orderId && (
        <>
          <SeaOrderReassignmentModal
            orderId={orderId}
            orderNo={order?.orderNo}
            open={reassignModalOpen}
            onClose={() => setReassignModalOpen(false)}
            onSuccess={async () => {
              await loadData();
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
