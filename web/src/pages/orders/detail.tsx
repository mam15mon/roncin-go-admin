import {
  CheckOutlined,
  CopyOutlined,
  DollarOutlined,
  HistoryOutlined,
  ReloadOutlined,
  ShareAltOutlined,
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
import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { StickyFooterBar } from '@/components/ui';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import type {
  OrderFormTemplateActions,
  OrderFormTemplateSection,
} from '@/components/ui/order-template/types';
import {
  OrderAllowedAction,
  OrderClosureStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import { searchShippingLineOptions } from '@/utils/options';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import { PARTNER_ROLES, parseOrderKind, searchPartnersByRole } from './common';
import { buildOrderAuditTimelineSection } from './components/detail/OrderAuditTimelineSection';
import OrderDetailHeader from './components/detail/OrderDetailHeader';
import { buildOrderStatusSection } from './components/detail/OrderStatusSection';
import {
  buildInitialValues,
  buildUpdatePayload,
  type OrderDetailFormValues,
} from './components/detail/orderDetailHelpers';
import SameBatchOrdersSection from './components/detail/SameBatchOrdersSection';
import SeaOrderChangeHistoryDrawer, {
  SeaOrderChangeHistorySection,
} from './components/drawers/SeaOrderChangeHistoryDrawer';
import SeaOrderReassignmentModal from './components/drawers/SeaOrderReassignmentModal';
import SeaSharedContainerDrawer from './components/drawers/SeaSharedContainerDrawer';
import SeaTransportExecutionUpdateModal from './components/drawers/SeaTransportExecutionUpdateModal';
import OrderPageHeader from './components/OrderPageHeader';
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
  const templateActionsRef = useRef<
    OrderFormTemplateActions<OrderDetailFormValues> | undefined
  >(undefined);
  const { message, modal } = App.useApp();
  const access = useAccess();

  const kind = params.kind;
  const orderId = params.id;
  const config = parseOrderKind(kind);

  const targetOrderId = config ? orderId : undefined;
  const orderFormIdentity =
    config && orderId ? `${config.kind}:${orderId}` : undefined;

  const [saving, setSaving] = useState(false);
  // 显式刷新标记携带发起时的订单身份与令牌；A 的迟到刷新不得操作 B 的模板。
  const pendingExplicitFormRefreshRef = useRef<{
    identity: string;
    token: number;
  } | null>(null);
  // 统一显式刷新令牌：发起时递增；订单身份提交变化时再递增一次，
  // 使旧身份的全部在途刷新立即失效（覆盖 A→B 与 A→B→回 A 往返）。
  const explicitRefreshTokenRef = useRef(0);
  const [explicitFormRefreshVersion, setExplicitFormRefreshVersion] =
    useState(0);

  const {
    loading,
    error,
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
    draftScope,
    loadData,
  } = useOrderDetailData(targetOrderId, config);

  // 订单身份提交变化时同步作废旧身份的全部在途刷新。在 layout effect 中
  // 执行而非渲染期：render 可能被并发模式重试或放弃，试探性渲染不得作废
  // 真实请求；layout effect 先于消费 passive effect 运行，可一并清掉已写入
  // 的 pending，堵住「完成写入 pending 后、消费前身份已切换」的间隙。
  const previousOrderFormIdentityRef = useRef(orderFormIdentity);
  useLayoutEffect(() => {
    if (previousOrderFormIdentityRef.current === orderFormIdentity) return;
    previousOrderFormIdentityRef.current = orderFormIdentity;
    explicitRefreshTokenRef.current += 1;
    pendingExplicitFormRefreshRef.current = null;
  }, [orderFormIdentity]);

  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);

  const changeActionsTargetKey =
    orderId && config?.category === 'sea'
      ? orderFormIdentity
      : undefined;
  const activeChangeActionsTargetRef = useRef(changeActionsTargetKey);
  activeChangeActionsTargetRef.current = changeActionsTargetKey;
  const changeActionsRequestIdRef = useRef(0);
  const [changeActionsState, setChangeActionsState] = useState<{
    targetKey: string;
    data: API.SeaOrderChangeActionsData;
  } | null>(null);
  const changeActions =
    changeActionsState &&
    changeActionsState.targetKey === changeActionsTargetKey
      ? changeActionsState.data
      : null;
  const [reassignModalOpen, setReassignModalOpen] = useState(false);
  const [voyageUpdateModalOpen, setVoyageUpdateModalOpen] = useState(false);
  const [historyDrawerOpen, setHistoryDrawerOpen] = useState(false);
  const [sharedContainerDrawerOpen, setSharedContainerDrawerOpen] =
    useState(false);
  const [sharedContainerTEId, setSharedContainerTEId] = useState<
    string | undefined
  >(undefined);
  const [sharedContainerOrderId, setSharedContainerOrderId] = useState<
    string | undefined
  >(undefined);

  // 路由切换到其他业务类型或记录时立即关闭共享箱工作台并清空旧运输执行，
  // 防止“新订单 + 旧运输执行”形成错误业务上下文；工作区切换由上层 OrganizationWorkspace 卸载兜底。
  useEffect(() => {
    setSharedContainerDrawerOpen(false);
    setSharedContainerTEId(undefined);
    setSharedContainerOrderId(undefined);
  }, [orderFormIdentity]);

  const loadChangeActions = useCallback(async () => {
    const requestOrderId = orderId;
    const requestTargetKey = changeActionsTargetKey;
    if (!requestOrderId || !requestTargetKey) {
      changeActionsRequestIdRef.current += 1;
      setChangeActionsState(null);
      return;
    }
    // 等待其他刷新任务结束的旧闭包不得使当前订单请求失效。
    if (requestTargetKey !== activeChangeActionsTargetRef.current) return;
    const requestId = ++changeActionsRequestIdRef.current;
    try {
      const resp = await seaOrderChangeServiceGetSeaOrderChangeActions({
        orderId: requestOrderId,
      });
      if (
        requestId !== changeActionsRequestIdRef.current ||
        requestTargetKey !== activeChangeActionsTargetRef.current
      ) {
        return;
      }

      setChangeActionsState(
        resp?.data ? { targetKey: requestTargetKey, data: resp.data } : null,
      );
    } catch (error: unknown) {
      if (
        requestId !== changeActionsRequestIdRef.current ||
        requestTargetKey !== activeChangeActionsTargetRef.current
      ) {
        return;
      }
      setChangeActionsState(null);
      message.error(
        error instanceof Error ? error.message : '加载拆票与改配动作失败',
      );
    }
  }, [changeActionsTargetKey, message, orderId]);

  const {
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
    refresh: refreshLockState,
  } = useOrderLockState(targetOrderId);
  const [synchronizingLockChange, setSynchronizingLockChange] = useState(false);

  useEffect(() => {
    setChangeActionsState(null);
    void loadChangeActions();

    return () => {
      changeActionsRequestIdRef.current += 1;
    };
  }, [loadChangeActions, order?.version]);

  useEffect(() => {
    if (
      order?.orderNo &&
      orderId &&
      order.id === orderId &&
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
  }, [order?.orderNo, order?.id, orderId, config?.kind, config?.title]);

  // 2. 构造表单初始值
  const initialValues = useMemo(
    () => buildInitialValues(order, shippingDocs, personnel),
    [order, shippingDocs, personnel],
  );

  const lockWritePolicy = getOrderBusinessWritePolicy({
    state: lockState,
    loading: lockStateLoading || synchronizingLockChange,
    error: lockStateError,
  });
  const businessWritesDisabled = lockWritePolicy.disabled;

  // 有效只读 = 无编辑动作权限或业务写入关闭；分节构建器与模板壳必须使用同一判定。
  const effectiveReadonly =
    order?.allowedActions?.includes(
      OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT,
    ) !== true || businessWritesDisabled;

  /**
   * OrderFormTemplate 独占草稿与脏状态生命周期；显式刷新成功后由页面
   * 通过模板动作接口 resetTo 回填最新服务端值。刷新完成必须校验发起身份：
   * A 的迟到刷新不得清 B 的草稿或重置 B 的表单。
   */
  useEffect(() => {
    const pending = pendingExplicitFormRefreshRef.current;
    if (!pending) return;
    pendingExplicitFormRefreshRef.current = null;
    // 令牌与身份双重复核：覆盖「完成写入 pending 后、消费 effect 运行前」
    // 又有刷新发起或身份切换的窗口；刷新后无当前订单（如详情加载失败被
    // 清空）时同样保留草稿与脏状态。
    if (
      pending.token !== explicitRefreshTokenRef.current ||
      pending.identity !== orderFormIdentity ||
      !order
    ) {
      return;
    }
    templateActionsRef.current?.resetTo(initialValues);
  }, [explicitFormRefreshVersion, initialValues, order, orderFormIdentity]);

  const refreshOrderDataAndResetForm = useCallback(async () => {
    const requestedIdentity = orderFormIdentity;
    if (!requestedIdentity) return;
    const token = ++explicitRefreshTokenRef.current;
    try {
      await loadData();
      // 后续刷新发起或订单身份提交变化都会使当前令牌失效。
      if (token !== explicitRefreshTokenRef.current) return;
      // loadData 完成后再触发本次显式刷新重置，让 React 先用最新服务端响应重算 initialValues。
      pendingExplicitFormRefreshRef.current = {
        identity: requestedIdentity,
        token,
      };
      setExplicitFormRefreshVersion((version) => version + 1);
    } catch {
      // 刷新失败保留草稿与当前表单，不触发显式重置
    }
  }, [loadData, orderFormIdentity]);

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
      searchShippingLines: searchShippingLineOptions,
      searchBookingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword),
      searchForeignAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.FOREIGN_AGENT, keyword),
      searchShippingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword),
      setCustomerCode: (code?: string) =>
        formRef.current?.setFieldValue('customerCode', code ?? ''),
      checkCustomerReferenceNo: async () => {},
      checkInternalReferenceNo: async () => {},
      personnelOptions,
      readonly: effectiveReadonly,
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
      effectiveReadonly,
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
              key: 'same-batch-orders',
              title: '同批订单',
              content: (
                <SameBatchOrdersSection
                  orderId={orderId}
                  orderKind={config.kind}
                />
              ),
            },
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

  // 6. 保存修改提交处理：成功/失败只由订单更新接口决定，模板统一清草稿与脏状态。
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
      // 详情与锁状态刷新是 best-effort 后台任务：不阻塞成功返回，
      // 也不把已落库的保存改判为失败；Hook 已呈现普通请求错误，
      // 这里只兜底意外泄漏的 reject。
      Promise.all([loadData(), refreshLockState()]).catch(() => {
        message.warning('订单已保存，但最新数据刷新失败，请手动刷新');
      });
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

  if (error && !loading) {
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
            style={{
              borderRadius: 8,
              border: '1px solid #f0f0f0',
              backgroundColor: '#ffffff',
            }}
          >
            <Result
              status="warning"
              title="加载订单详情失败"
              subTitle={
                error.message || '无法获取订单详情数据，请检查网络或重试。'
              }
              extra={
                <Button type="primary" onClick={() => void loadData()}>
                  重新加载
                </Button>
              }
            />
          </Card>
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
    ...(config.category === 'sea' &&
    access.canOrder(config.businessType, 'reassign')
      ? [
          {
            key: 'shared-voyage-update',
            icon: <ReloadOutlined />,
            label: '共享航次调整',
            onClick: () => setVoyageUpdateModalOpen(true),
          },
        ]
      : []),
    ...(config.category === 'sea'
      ? [
          {
            key: 'shared-container-workbench',
            icon: <ShareAltOutlined />,
            label: '跨订单共享箱工作台',
            disabled: !access.canOrder(config.businessType, 'container.read'),
            onClick: () => {
              const teId = order.seaMasterBill?.transportExecutionId;
              if (!teId) {
                message.warning(
                  '当前订单尚未关联实际运输执行，无法开展跨订单拼箱',
                );
                return;
              }
              setSharedContainerTEId(teId);
              setSharedContainerOrderId(orderId);
              setSharedContainerDrawerOpen(true);
            },
          },
        ]
      : []),
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
        void refreshOrderDataAndResetForm();
        void refreshLockState();
        void loadChangeActions();
      },
    },
  ];

  return (
    <>
      <OrderFormTemplate<OrderDetailFormValues>
        key={orderFormIdentity}
        tabKey={
          config && orderId
            ? resolveTabKey(`/orders/${config.kind}/${orderId}`)
            : undefined
        }
        draftPathname={
          config && orderId ? `/orders/${config.kind}/${orderId}` : undefined
        }
        draftScope={draftScope}
        loading={false}
        readonly={effectiveReadonly}
        formRef={formRef}
        actionsRef={templateActionsRef}
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
              setReassignModalOpen(true);
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
                  onClick={() =>
                    templateActionsRef.current?.resetTo(initialValues)
                  }
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
            disabled={changeActions?.canReassign === false}
            disabledReason={changeActions?.reassignBlockedReasons?.join('；')}
            onClose={() => setReassignModalOpen(false)}
            onSuccess={async () => {
              await Promise.all([loadData(), refreshLockState()]);
              await loadChangeActions();
            }}
            searchShippingLines={templateProps.searchShippingLines}
            searchLocations={templateProps.searchLocations}
            initialShippingLineId={order.shippingLineId}
            initialShippingLineName={order.seaMasterBill?.shippingLineName}
          />
          <SeaTransportExecutionUpdateModal
            order={order}
            open={voyageUpdateModalOpen}
            onClose={() => setVoyageUpdateModalOpen(false)}
            onSuccess={async () => {
              await Promise.all([loadData(), refreshLockState()]);
              await loadChangeActions();
            }}
            searchLocations={templateProps.searchLocations}
          />
          <SeaOrderChangeHistoryDrawer
            orderId={orderId}
            open={historyDrawerOpen}
            onClose={() => setHistoryDrawerOpen(false)}
          />
          <SeaSharedContainerDrawer
            key={`shared-container:${orderId}:${sharedContainerTEId ?? ''}`}
            open={
              sharedContainerDrawerOpen &&
              sharedContainerOrderId === orderId &&
              !!sharedContainerTEId
            }
            onClose={() => setSharedContainerDrawerOpen(false)}
            transportExecutionId={sharedContainerTEId}
            orderId={orderId}
            orderNo={order?.orderNo}
            canCreate={
              !businessWritesDisabled &&
              access.canOrder(config.businessType, 'container.create')
            }
            canUpdate={
              !businessWritesDisabled &&
              access.canOrder(config.businessType, 'container.update')
            }
            canDelete={
              !businessWritesDisabled &&
              access.canOrder(config.businessType, 'container.delete')
            }
            containerSpecOptions={containerSpecOptions}
          />
        </>
      )}
    </>
  );
}
