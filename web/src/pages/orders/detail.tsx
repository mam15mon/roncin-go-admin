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
  Input,
  type MenuProps,
  Select,
  Space,
  Spin,
  Typography,
} from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { StickyFooterBar } from '@/components/ui';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import type { OrderFormTemplateSection } from '@/components/ui/order-template/types';
import {
  orderServiceGetOrder,
  orderServiceListPersonnelOptions,
  orderServiceTransitionOrderClosure,
  orderServiceTransitionOrderTermination,
  orderServiceUpdateOrder,
} from '@/services/roncin/orderService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import { orderCargoItemServiceListCargoItems } from '@/services/roncin/orderCargoItemService';
import { orderMilestoneServiceListMilestones } from '@/services/roncin/orderMilestoneService';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import {
  fetchOrderMasterData,
  isMasterDataKind,
  MASTER_DATA_KINDS,
  PARTNER_ROLES,
  parseOrderKind,
  searchPartnersByRole,
  seaServiceTypeNames,
} from './common';
import { buildOrderAuditTimelineSection } from './components/detail/OrderAuditTimelineSection';
import OrderDetailHeader from './components/detail/OrderDetailHeader';
import { buildOrderStatusSection } from './components/detail/OrderStatusSection';
import {
  buildInitialValues,
  buildUpdatePayload,
} from './components/detail/orderDetailHelpers';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
import {
  getAirTemplateSections,
  getSeaTemplateSections,
  type SelectOption,
} from './templates';

const { Text } = Typography;

export default function OrderDetailPage() {
  const params = useParams<{ kind: string; id: string }>();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const { message, modal } = App.useApp();
  const access = useAccess();

  const kind = params.kind || 'sea-export';
  const orderId = params.id;
  const config = parseOrderKind(kind) || {
    kind: 'sea-export',
    title: '海运出口',
    businessType: 1,
    category: 'sea',
  };

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const [order, setOrder] = useState<API.Order>();
  const [shippingDocs, setShippingDocs] = useState<API.OrderShippingDocument[]>(
    [],
  );
  const [_containers, setContainers] = useState<API.OrderContainer[]>([]);
  const [_cargoItems, setCargoItems] = useState<API.OrderCargoItem[]>([]);
  const [_milestones, setMilestones] = useState<API.OrderMilestone[]>([]);
  const [personnel, setPersonnel] = useState<API.OrderPersonnel[]>([]);

  const [serviceTypeOptions, setServiceTypeOptions] = useState<SelectOption[]>(
    [],
  );
  const [cargoCategoryOptions, setCargoCategoryOptions] = useState<
    SelectOption[]
  >([]);
  const [locationOptions, setLocationOptions] = useState<SelectOption[]>([]);
  const [currencyOptions, setCurrencyOptions] = useState<SelectOption[]>([]);
  const [containerSpecOptions, setContainerSpecOptions] = useState<
    SelectOption[]
  >([]);
  const [personnelOptions, setPersonnelOptions] = useState<
    API.OrderPersonnelOption[]
  >([]);

  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);

  // 1. 加载主数据与订单数据
  const loadData = async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const [
        masterData,
        personnelOptRes,
        orderRes,
        docsRes,
        cntrsRes,
        cargoRes,
        milestonesRes,
        personnelRes,
      ] = await Promise.all([
        fetchOrderMasterData(),
        config.category === 'sea'
          ? orderServiceListPersonnelOptions({
              businessType: config.businessType,
              page: 1,
              pageSize: 200,
            }).catch(() => ({ data: [] }))
          : Promise.resolve({ data: [] }),
        orderServiceGetOrder({ id: orderId }),
        orderShippingDocumentServiceListShippingDocuments({ orderId }).catch(
          () => ({ data: [] }),
        ),
        orderContainerServiceListContainers({ orderId }).catch(() => ({
          data: [],
        })),
        orderCargoItemServiceListCargoItems({ orderId }).catch(() => ({
          data: [],
        })),
        orderMilestoneServiceListMilestones({ orderId }).catch(() => ({
          data: [],
        })),
        orderPersonnelServiceListPersonnel({ orderId }).catch(() => ({
          data: [],
        })),
      ]);

      const nextServiceTypeOptions =
        config.category === 'sea'
          ? seaServiceTypeNames.map((name) => {
              const option = masterData.serviceTypeOptions.find(
                (item) => item.label === name,
              );
              return option || { label: name, value: name };
            })
          : masterData.serviceTypeOptions;

      setServiceTypeOptions(nextServiceTypeOptions);
      setCargoCategoryOptions(masterData.cargoCategoryOptions);
      setLocationOptions(
        config.category === 'sea'
          ? masterData.seaLocationOptions
          : masterData.airLocationOptions,
      );
      setCurrencyOptions(masterData.currencyOptions);
      setContainerSpecOptions(
        masterData.masterOptions
          .filter(
            (item) =>
              isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
              item.enabled !== false,
          )
          .map((item) => ({
            label: item.code
              ? `${item.name ?? item.code} (${item.code})`
              : (item.name ?? ''),
            value: item.id ?? '',
          }))
          .filter((item) => item.value !== ''),
      );
      setPersonnelOptions(personnelOptRes.data ?? []);

      setOrder(orderRes.data);
      setShippingDocs(docsRes.data ?? []);
      setContainers(cntrsRes.data ?? []);
      setCargoItems(cargoRes.data ?? []);
      setMilestones(milestonesRes.data ?? []);
      setPersonnel(personnelRes.data ?? []);
    } catch (error: any) {
      message.error(error.message || '加载订单数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadData();
  }, [orderId]);

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
      currencyOptions,
      containerSpecOptions,
      isDetail: true,
      searchCustomers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CUSTOMER, keyword),
      searchCarriers: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.CARRIER, keyword),
      searchBookingAgents: (keyword?: string) =>
        searchPartnersByRole(PARTNER_ROLES.BOOKING_AGENT, keyword),
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

  const confirmTermination = (targetStatus: number) => {
    const orderID = order.id;
    const expectedVersion = order.version;
    if (!orderID || expectedVersion === undefined) {
      message.error('订单数据不完整，请刷新后重试');
      return;
    }
    let reason = '';
    let terminationType = 3;
    modal.confirm({
      title:
        targetStatus === 1
          ? '取消退关/终止'
          : targetStatus === 2
            ? '发起退关/终止'
            : '完成退关/终止',
      content: (
        <Space vertical style={{ width: '100%', marginTop: 12 }}>
          {targetStatus !== 1 && (
            <Select
              defaultValue={3}
              style={{ width: '100%' }}
              options={[
                { label: '客户撤单', value: 1 },
                { label: '承运人取消', value: 2 },
                { label: '海关退关', value: 3 },
                { label: '操作取消', value: 4 },
                { label: '其他', value: 5 },
              ]}
              onChange={(value) => {
                terminationType = value;
              }}
            />
          )}
          <Input.TextArea
            placeholder="请输入原因（必填）"
            maxLength={500}
            showCount
            onChange={(event) => {
              reason = event.target.value;
            }}
          />
        </Space>
      ),
      async onOk() {
        if (!reason.trim()) {
          message.error('请输入原因');
          return Promise.reject();
        }
        const response = await orderServiceTransitionOrderTermination(
          { id: orderID },
          {
            id: orderID,
            expectedVersion,
            targetStatus,
            terminationType: targetStatus === 1 ? undefined : terminationType,
            reason,
          },
        );
        if (response.data) {
          message.success('更新退关状态成功');
          await loadData();
        }
      },
    });
  };

  const confirmClosure = (targetStatus: number) => {
    const orderID = order.id;
    const expectedVersion = order.version;
    if (!orderID || expectedVersion === undefined) {
      message.error('订单数据不完整，请刷新后重试');
      return;
    }
    let reason = '';
    modal.confirm({
      title: targetStatus === 1 ? '反结案/重新激活订单' : '完结订单',
      content: (
        <Space vertical style={{ width: '100%', marginTop: 12 }}>
          <Input.TextArea
            placeholder={
              targetStatus === 1
                ? '请输入反结案原因（必填）'
                : '请输入完结原因（选填）'
            }
            maxLength={500}
            showCount
            onChange={(event) => {
              reason = event.target.value;
            }}
          />
        </Space>
      ),
      async onOk() {
        if (targetStatus === 1 && !reason.trim()) {
          message.error('请输入反结案原因');
          return Promise.reject();
        }
        const response = await orderServiceTransitionOrderClosure(
          { id: orderID },
          {
            id: orderID,
            expectedVersion,
            targetStatus,
            reason: reason.trim(),
          },
        );
        if (response.data) {
          message.success(targetStatus === 1 ? '反结案成功' : '完结订单成功');
          await loadData();
        }
      },
    });
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
