import {
  CheckOutlined,
  CopyOutlined,
  DollarOutlined,
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
import { orderServiceUpdateOrder } from '@/services/roncin/orderService';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import {
  PARTNER_ROLES,
  parseOrderKind,
  searchPartnersByRole,
} from './common';
import { buildOrderAuditTimelineSection } from './components/detail/OrderAuditTimelineSection';
import OrderDetailHeader from './components/detail/OrderDetailHeader';
import { buildOrderStatusSection } from './components/detail/OrderStatusSection';
import {
  buildInitialValues,
  buildUpdatePayload,
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
    businessType: 1,
    tradeDirection: 1,
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

  // 5. 后置区块：操作记录日志
  const appendSections: OrderFormTemplateSection[] = useMemo(
    () => [buildOrderAuditTimelineSection(order)],
    [order],
  );

  // 6. 保存修改提交处理
  const handleSaveEdit = async (values: any) => {
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
    } catch (error: any) {
      message.error(error.message || '保存订单失败');
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
    order.closureStatus === 2
      ? '已完结'
      : order.terminationStatus === 3
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
      key: 'reload-data',
      icon: <ReloadOutlined />,
      label: '刷新数据',
      onClick: () => void loadData(),
    },
  ];

  return (
    <>
      <OrderFormTemplate<any>
        loading={false}
        readonly={!hasAction(1)}
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
            moreMenuItems={moreMenuItems}
            hasAction={hasAction}
            onSave={() => formRef.current?.submit()}
            onConfirmTermination={confirmTermination}
            onConfirmClosure={confirmClosure}
            onOpenReleasePod={() => releasePodPanelRef.current?.open(order)}
            onOpenAbnormalCase={() =>
              abnormalCasePanelRef.current?.open(order)
            }
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
            {hasAction(1) && (
              <Button
                icon={<UndoOutlined />}
                onClick={() => formRef.current?.setFieldsValue(initialValues)}
              >
                重置修改
              </Button>
            )}
            {hasAction(1) && (
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
    </>
  );
}
