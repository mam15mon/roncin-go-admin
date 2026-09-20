import {
  CopyOutlined,
  DollarOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { App, Button, Card, Empty, type MenuProps, Result, Spin } from 'antd';
import React, {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useParams } from 'react-router';
import { useAccess } from '@/app/access';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import type {
  OrderFormTemplateActions,
  OrderFormTemplateSection,
} from '@/components/ui/order-template/types';
import { OrderAllowedAction } from '@/enums.generated';
import { searchShippingLineOptions } from '@/features/master-data/shipping-lines';
import { history } from '@/router/history';
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import { generateUUID } from '@/utils/uuid';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import { PARTNER_ROLES, searchPartnersByRole } from './common';
import { buildOrderAuditTimelineSection } from './components/detail/OrderAuditTimelineSection';
import OrderDetailHeader from './components/detail/OrderDetailHeader';
import { buildOrderStatusSection } from './components/detail/OrderStatusSection';
import OrderPageHeader from './components/OrderPageHeader';
import {
  confirmOrderClosure,
  confirmOrderTermination,
} from './order-detail-transitions';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import { getOrderKindDefinition } from './order-kinds/registry';
import type { OrderDetailFormValues } from './order-kinds/sea-export/form-adapter';
import type {
  OrderDetailFeaturesContext,
  OrderDetailFeaturesProps,
} from './order-kinds/types';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
import { useOrderDetailData } from './use-order-detail-data';
import {
  getOrderBusinessWritePolicy,
  useOrderLockState,
} from './use-order-lock-state';

/** 未提供类型扩展时，详情页只渲染通用布局。 */
const EmptyDetailFeatures: React.ComponentType<OrderDetailFeaturesProps> = ({
  children,
}) => <>{children({})}</>;

export default function OrderDetailPage() {
  const params = useParams<{ kind: string; id: string }>();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  // 草稿更新幂等键：每次提交意图一个键；失败重试沿用同键，成功后重新生成。
  // 后端以「同键 + 同 expectedVersion 重放返回当前草稿」保护超时重试场景。
  const updateIdempotencyKeyRef = useRef(generateUUID());
  const templateActionsRef = useRef<
    OrderFormTemplateActions<OrderDetailFormValues> | undefined
  >(undefined);
  const { message, modal } = App.useApp();
  const access = useAccess();

  const kind = params.kind;
  const orderId = params.id;
  const definition = getOrderKindDefinition(kind);

  const targetOrderId = definition ? orderId : undefined;
  const orderFormIdentity =
    definition && orderId ? `${definition.kind}:${orderId}` : undefined;

  const [saving, setSaving] = useState(false);
  // 显式刷新标记携带发起时的订单身份与令牌；A 的迟到刷新不得操作 B 的模板。
  const pendingExplicitFormRefreshRef = useRef<{
    identity: string;
    token: number;
  } | null>(null);
  // 统一显式刷新令牌：发起时递增；订单身份切换时在 layout effect setup/cleanup 中递增，
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
  } = useOrderDetailData(targetOrderId, definition);

  // 订单身份提交变化时同步作废旧身份的全部在途刷新。
  // 在 layout effect 的 setup 与 cleanup 中均递增 token 并清空 pending：
  // cleanup 在旧身份卸载/切换前同步执行，setup 在新身份挂载/切换后同步执行，
  // 彻底消除 render 阶段变异 ref，且不需要额外的 previousRef。
  useLayoutEffect(() => {
    explicitRefreshTokenRef.current += 1;
    pendingExplicitFormRefreshRef.current = null;
    return () => {
      explicitRefreshTokenRef.current += 1;
      pendingExplicitFormRefreshRef.current = null;
    };
  }, [orderFormIdentity]);

  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);

  const {
    state: lockState,
    loading: lockStateLoading,
    error: lockStateError,
    refresh: refreshLockState,
  } = useOrderLockState(targetOrderId);
  const [synchronizingLockChange, setSynchronizingLockChange] = useState(false);

  useEffect(() => {
    if (
      order?.orderNo &&
      orderId &&
      order.id === orderId &&
      definition?.kind &&
      typeof window !== 'undefined'
    ) {
      window.dispatchEvent(
        new CustomEvent('roncin:update-tab-title', {
          detail: {
            path: `/orders/${definition.kind}/${orderId}`,
            title: `${order.orderNo}_${definition?.title || ''}详情`,
          },
        }),
      );
    }
  }, [order?.orderNo, order?.id, orderId, definition?.kind, definition?.title]);

  // 2. 构造表单初始值
  const initialValues = useMemo(
    () =>
      definition?.form.buildDetailInitialValues(
        order,
        shippingDocs,
        personnel,
      ) ?? {},
    [definition, order, shippingDocs, personnel],
  );

  const lockWritePolicy = getOrderBusinessWritePolicy({
    state: lockState,
    loading: lockStateLoading || synchronizingLockChange,
    error: lockStateError,
    canOperate: access.canOperateOrganization(order?.organizationId),
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
      // 后续刷新发起或订单身份切换都会使当前令牌失效。
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
      organizationId: order?.organizationId,
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
      order?.organizationId,
      loadData,
    ],
  );

  const formSections = useMemo(
    () => definition?.form.buildSections(templateProps) ?? [],
    [definition, templateProps],
  );

  // 4. 海管家风格「订单状态」卡片（作为前置区块）
  const prependSections: OrderFormTemplateSection[] = useMemo(
    () => [buildOrderStatusSection(order)],
    [order],
  );

  // 5. 通用后置区块：操作记录日志（类型专属区块由详情扩展贡献并置于其前）
  const appendSections: OrderFormTemplateSection[] = useMemo(
    () => [buildOrderAuditTimelineSection(order)],
    [order],
  );

  // 6. 保存修改提交处理：成功/失败只由订单更新接口决定，模板统一清草稿与脏状态。
  const handleSaveEdit = async (values: OrderDetailFormValues) => {
    if (!definition || !orderId || !ensureBusinessWriteAllowed()) return false;
    setSaving(true);
    try {
      const payload = definition.form.buildUpdatePayload(
        orderId,
        order?.version || '0',
        values,
      );
      await orderServiceUpdateOrder(
        { id: orderId },
        { ...payload, idempotencyKey: updateIdempotencyKeyRef.current },
      );
      updateIdempotencyKeyRef.current = generateUUID();
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

  if (!definition) {
    return (
      <div style={{ padding: 48, background: '#f5f7fa', minHeight: '100vh' }}>
        <Result
          status="404"
          title="业务类型不存在"
          subTitle={`未知的业务类型路径 "${kind || ''}"，请选择有效业务入口。`}
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
          orderKind={definition.kind}
          navigationTitle={definition.navigationTitle}
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
          orderKind={definition.kind}
          navigationTitle={definition.navigationTitle}
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
          orderKind={definition.kind}
          navigationTitle={definition.navigationTitle}
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

  const hasAction = (action: number) =>
    access.canOperateOrganization(order.organizationId) &&
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

  // 类型详情扩展以同一订单身份重挂载：扩展持有自己的请求与本地状态，
  // 只通过 context 读取业务输入与通用刷新命令。
  const DetailFeatures = definition.DetailFeatures ?? EmptyDetailFeatures;
  const detailFeaturesContext: OrderDetailFeaturesContext = {
    kind: definition.kind,
    orderId: orderId || '',
    order,
    orderFormIdentity: orderFormIdentity || '',
    businessWritesDisabled,
    businessWriteBlockedReason: lockWritePolicy.reason,
    canOrder: (operation) =>
      access.canOrder(definition.businessType, operation),
    searchShippingLines: templateProps.searchShippingLines,
    searchLocations: templateProps.searchLocations,
    containerSpecOptions,
    refreshOrderAndLock: async () => {
      await Promise.all([loadData(), refreshLockState()]);
    },
    ensureBusinessWriteAllowed,
  };

  return (
    <DetailFeatures key={orderFormIdentity} context={detailFeaturesContext}>
      {(features) => {
        const synchronizeLockChange = async () => {
          setSynchronizingLockChange(true);
          try {
            await Promise.all([loadData(), refreshLockState()]);
            await features.refreshTypeState?.();
          } finally {
            setSynchronizingLockChange(false);
          }
        };

        const moreMenuItems: MenuProps['items'] = [
          ...(features.moreMenuItems ?? []),
          {
            key: 'fees-drawer',
            icon: <DollarOutlined />,
            label: '快速费用抽屉',
            disabled: !access.canOrder(definition.businessType, 'fee.read'),
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
            key: 'reload-data',
            icon: <ReloadOutlined />,
            label: '刷新数据',
            onClick: () => {
              void refreshOrderDataAndResetForm();
              void refreshLockState();
              void features.refreshTypeState?.();
            },
          },
        ];

        return (
          <>
            <OrderFormTemplate<OrderDetailFormValues>
              key={orderFormIdentity}
              tabKey={
                definition && orderId
                  ? resolveTabKey(`/orders/${definition.kind}/${orderId}`)
                  : undefined
              }
              draftPathname={
                definition && orderId
                  ? `/orders/${definition.kind}/${orderId}`
                  : undefined
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
                  kind={definition.kind}
                  navigationTitle={definition.navigationTitle}
                  orderId={orderId || ''}
                  order={order}
                  saving={saving}
                  canManageFee={access.canOrder(
                    definition.businessType,
                    'fee.read',
                  )}
                  canCreatePod={access.canOrder(
                    definition.businessType,
                    'release_pod.create',
                  )}
                  canCreateAbnormal={access.canOrder(
                    definition.businessType,
                    'abnormal_case.create',
                  )}
                  businessActions={features.headerActions}
                  moreMenuItems={moreMenuItems}
                  hasAction={hasAction}
                  onSave={() => formRef.current?.submit()}
                  onReset={() =>
                    templateActionsRef.current?.resetTo(initialValues)
                  }
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
              appendSections={[
                ...(features.appendSections ?? []),
                ...appendSections,
              ]}
            />

            {features.overlays}

            {/* 挂载通用功能弹窗 */}
            <ReleasePodPanel
              ref={releasePodPanelRef}
              canManage={
                !businessWritesDisabled &&
                access.canOrder(definition.businessType, 'release_pod.create')
              }
            />
            <OrderFeePanel ref={orderFeePanelRef} />
            <AbnormalCasePanel
              ref={abnormalCasePanelRef}
              canManage={
                !businessWritesDisabled &&
                access.canOrder(definition.businessType, 'abnormal_case.create')
              }
              masterOptions={[]}
            />
          </>
        );
      }}
    </DetailFeatures>
  );
}
