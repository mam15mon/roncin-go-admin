import {
  CheckOutlined,
  DollarOutlined,
  FileDoneOutlined,
  LockOutlined,
  ReloadOutlined,
  UndoOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { history, useAccess, useParams } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Col,
  Empty,
  Space,
  Spin,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { PageHeaderShell, StickyFooterBar } from '@/components/ui';
import { OrderFormTemplate } from '@/components/ui/order-template/OrderFormTemplate';
import type { OrderFormTemplateSection } from '@/components/ui/order-template/types';
import {
  orderServiceGetOrder,
  orderServiceListPersonnelOptions,
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
  loadStatusTemplatesByBusinessType,
  MASTER_DATA_KINDS,
  PARTNER_ROLES,
  parseOrderKind,
  searchPartnersByRole,
  seaServiceTypeNames,
} from './common';
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
  const { message } = App.useApp();
  const access = useAccess();

  const kind = params.kind || 'sea-export';
  const orderId = params.id;
  const config = parseOrderKind(kind) || {
    kind: 'sea-export',
    title: '海运出口订单',
    businessType: 1,
    category: 'sea',
  };

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const [order, setOrder] = useState<API.Order>();
  const [shippingDocs, setShippingDocs] = useState<API.OrderShippingDocument[]>([]);
  const [_containers, setContainers] = useState<API.OrderContainer[]>([]);
  const [_cargoItems, setCargoItems] = useState<API.OrderCargoItem[]>([]);
  const [_milestones, setMilestones] = useState<API.OrderMilestone[]>([]);
  const [personnel, setPersonnel] = useState<API.OrderPersonnel[]>([]);

  const [_statusTemplateOptions, setStatusTemplateOptions] = useState<{ label: string; value: string }[]>([]);
  const [serviceTypeOptions, setServiceTypeOptions] = useState<SelectOption[]>([]);
  const [cargoCategoryOptions, setCargoCategoryOptions] = useState<SelectOption[]>([]);
  const [locationOptions, setLocationOptions] = useState<SelectOption[]>([]);
  const [currencyOptions, setCurrencyOptions] = useState<SelectOption[]>([]);
  const [containerSpecOptions, setContainerSpecOptions] = useState<SelectOption[]>([]);
  const [personnelOptions, setPersonnelOptions] = useState<API.OrderPersonnelOption[]>([]);

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
        templates,
        personnelOptRes,
        orderRes,
        docsRes,
        cntrsRes,
        cargoRes,
        milestonesRes,
        personnelRes,
      ] = await Promise.all([
        fetchOrderMasterData(),
        loadStatusTemplatesByBusinessType(config.businessType),
        config.category === 'sea'
          ? orderServiceListPersonnelOptions({ businessType: config.businessType }).catch(() => ({ data: [] }))
          : Promise.resolve({ data: [] }),
        orderServiceGetOrder({ id: orderId }),
        orderShippingDocumentServiceListShippingDocuments({ orderId }).catch(() => ({ data: [] })),
        orderContainerServiceListContainers({ orderId }).catch(() => ({ data: [] })),
        orderCargoItemServiceListCargoItems({ orderId }).catch(() => ({ data: [] })),
        orderMilestoneServiceListMilestones({ orderId }).catch(() => ({ data: [] })),
        orderPersonnelServiceListPersonnel({ orderId }).catch(() => ({ data: [] })),
      ]);

      const nextServiceTypeOptions =
        config.category === 'sea'
          ? seaServiceTypeNames.map((name) => {
              const option = masterData.serviceTypeOptions.find((item) => item.label === name);
              return option || { label: name, value: name };
            })
          : masterData.serviceTypeOptions;

      setServiceTypeOptions(nextServiceTypeOptions);
      setCargoCategoryOptions(masterData.cargoCategoryOptions);
      setLocationOptions(
        config.category === 'sea' ? masterData.seaLocationOptions : masterData.airLocationOptions,
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
      setStatusTemplateOptions(templates);
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

  // 2. 构造表单初始值（将订单数据与子明细映射到表单字段）
  const initialValues = useMemo(() => {
    if (!order) return {};

    const personnelRoleMap: Record<number, { userId?: string; organizationId?: string }> = {};
    for (const p of personnel) {
      if (p.role !== undefined) {
        personnelRoleMap[p.role] = { userId: p.userId, organizationId: p.organizationId };
      }
    }

    return {
      orderNo: order.orderNo,
      customerId: order.customerId,
      customerReferenceNo: order.customerReferenceNo,
      internalReferenceNo: order.internalReferenceNo,
      tradeTerm: order.tradeTerm,
      paymentTerm: order.paymentTerm,
      carrierId: order.carrierId,
      bookingAgentId: order.bookingAgentId,
      foreignAgentId: order.foreignAgentId,
      shippingAgentId: order.shippingAgentId,
      contractNo: order.contractNo,
      cargoValue: order.cargoValue,
      cargoCurrency: order.cargoCurrency || 'USD',
      insurancePremium: order.insurancePremium,
      insuranceCurrency: order.insuranceCurrency || 'CNY',
      loadingTerms: order.loadingTerms,
      shipmentType: order.shipmentType ?? 1,
      containerOwnership: order.containerOwnership ?? 1,
      shipmentMode: order.shipmentMode ?? 1,
      serviceTypeIds: order.serviceTypeIds ?? [],
      cargoCategoryIds: order.cargoCategoryIds ?? [],
      originLocationId: order.originLocationId,
      destinationLocationId: order.destinationLocationId,
      dischargeLocationId: order.dischargeLocationId,
      transitLocationId: order.transitLocationId,
      vesselVoyage: order.vesselVoyage,
      etd: order.etd ? dayjs(order.etd) : undefined,
      eta: order.eta ? dayjs(order.eta) : undefined,
      siCutoff: order.siCutoff ? dayjs(order.siCutoff) : undefined,
      docCutoff: order.docCutoff ? dayjs(order.docCutoff) : undefined,
      customsCutoff: order.customsCutoff ? dayjs(order.customsCutoff) : undefined,
      vgmCutoff: order.vgmCutoff ? dayjs(order.vgmCutoff) : undefined,
      goodsDescription: order.goodsDescription,
      totalPackages: order.totalPackages,
      totalGrossWeightKg: order.totalGrossWeightKg,
      totalVolumeCbm: order.totalVolumeCbm,
      totalPackageUnit: order.totalPackageUnit || 'CTNS',
      orderDate: order.orderDate ? dayjs(order.orderDate) : dayjs(order.createdAt),
      notes: order.notes,
      bookingNotes: order.bookingNotes,
      allocationNotes: order.allocationNotes,
      operationNotes: order.operationNotes,
      shippingDocuments: shippingDocs.length > 0 ? shippingDocs : order.shippingDocuments,
      containerRequests: order.containerRequests,
      creatorUserId: personnelRoleMap[0]?.userId,
      creatorOrganizationId: personnelRoleMap[0]?.organizationId,
      operatorUserId: personnelRoleMap[1]?.userId,
      operatorOrganizationId: personnelRoleMap[1]?.organizationId,
      salesUserId: personnelRoleMap[2]?.userId,
      salesOrganizationId: personnelRoleMap[2]?.organizationId,
      customerServiceUserId: personnelRoleMap[3]?.userId,
      customerServiceOrganizationId: personnelRoleMap[3]?.organizationId,
      documentUserId: personnelRoleMap[4]?.userId,
      documentOrganizationId: personnelRoleMap[4]?.organizationId,
      commercialUserId: personnelRoleMap[6]?.userId,
      commercialOrganizationId: personnelRoleMap[6]?.organizationId,
      associateUserId: personnelRoleMap[7]?.userId,
      associateOrganizationId: personnelRoleMap[7]?.organizationId,
      associate2UserId: personnelRoleMap[8]?.userId,
      associate2OrganizationId: personnelRoleMap[8]?.organizationId,
    };
  }, [order, shippingDocs, personnel]);

  // 当重新加载数据时同步更新表单值
  useEffect(() => {
    if (formRef.current && order) {
      formRef.current.setFieldsValue(initialValues);
    }
  }, [initialValues]);

  // 3. 复用与新建页 100% 相同的一套分节构建器（传入 isDetail: true 锁定订单编号与创建人员）
  const templateProps = useMemo(
    () => ({
      serviceTypeOptions,
      cargoCategoryOptions,
      locationOptions,
      currencyOptions,
      containerSpecOptions,
      isDetail: true,
      searchCustomers: (keyword?: string) => searchPartnersByRole(PARTNER_ROLES.CUSTOMER, keyword),
      searchCarriers: (keyword?: string) => searchPartnersByRole(PARTNER_ROLES.CARRIER, keyword),
      searchBookingAgents: (keyword?: string) => searchPartnersByRole(PARTNER_ROLES.BOOKING_AGENT, keyword),
      searchForeignAgents: (keyword?: string) => searchPartnersByRole(PARTNER_ROLES.FOREIGN_AGENT, keyword),
      searchShippingAgents: (keyword?: string) => searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword),
      setCustomerCode: (code?: string) => formRef.current?.setFieldValue('customerCode', code ?? ''),
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

  // 4. 后置区块：操作记录日志
  const appendSections: OrderFormTemplateSection[] = useMemo(() => {
    return [
      {
        key: 'order-operation-logs',
        title: '操作记录与历史流转日志',
        extra: <Tag color="geekblue">全生命周期审计</Tag>,
        content: (
          <Col span={24}>
            <div style={{ padding: '8px 12px' }}>
              <Timeline
                items={[
                  {
                    color: 'green',
                    children: (
                      <div>
                        <Space>
                          <Text strong>初始建单成功</Text>
                          <Tag color="default">系统录入</Tag>
                        </Space>
                        <div style={{ fontSize: 12, color: '#94a3b8', marginTop: 2 }}>
                          {order?.createdAt ? dayjs(order.createdAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
                        </div>
                      </div>
                    ),
                  },
                  {
                    color: 'blue',
                    children: (
                      <div>
                        <Space>
                          <Text strong>业务信息与配舱已录入</Text>
                          <Tag color="processing">主操作员</Tag>
                        </Space>
                        <div style={{ fontSize: 12, color: '#94a3b8', marginTop: 2 }}>
                          {order?.updatedAt ? dayjs(order.updatedAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
                        </div>
                      </div>
                    ),
                  },
                ]}
              />
            </div>
          </Col>
        ),
      },
    ];
  }, [order]);

  // 5. 保存修改提交处理
  const handleSaveEdit = async (values: any) => {
    if (!orderId) return false;
    setSaving(true);
    try {
      const payload: API.UpdateOrderRequest = {
        id: orderId,
        expectedStatus: order?.status || 'DRAFT',
        customerId: values.customerId,
        customerReferenceNo: values.customerReferenceNo?.trim() || undefined,
        internalReferenceNo: values.internalReferenceNo?.trim() || undefined,
        tradeTerm: values.tradeTerm !== undefined ? Number(values.tradeTerm) : undefined,
        paymentTerm: values.paymentTerm !== undefined ? Number(values.paymentTerm) : undefined,
        carrierId: values.carrierId || undefined,
        bookingAgentId: values.bookingAgentId || undefined,
        foreignAgentId: values.foreignAgentId || undefined,
        shippingAgentId: values.shippingAgentId || undefined,
        contractNo: values.contractNo?.trim() || undefined,
        cargoValue: values.cargoValue?.trim() || undefined,
        cargoCurrency: values.cargoCurrency || undefined,
        insurancePremium: values.insurancePremium?.trim() || undefined,
        insuranceCurrency: values.insuranceCurrency || undefined,
        loadingTerms: values.loadingTerms?.trim() || undefined,
        shipmentType: values.shipmentType !== undefined ? Number(values.shipmentType) : undefined,
        containerOwnership: values.containerOwnership !== undefined ? Number(values.containerOwnership) : undefined,
        shipmentMode: values.shipmentMode !== undefined ? Number(values.shipmentMode) : undefined,
        serviceTypeIds: values.serviceTypeIds,
        cargoCategoryIds: values.cargoCategoryIds,
        originLocationId: values.originLocationId || undefined,
        destinationLocationId: values.destinationLocationId || undefined,
        dischargeLocationId: values.dischargeLocationId || undefined,
        transitLocationId: values.transitLocationId || undefined,
        vesselVoyage: values.vesselVoyage?.trim() || undefined,
        etd: values.etd ? dayjs(values.etd).toISOString() : undefined,
        eta: values.eta ? dayjs(values.eta).toISOString() : undefined,
        siCutoff: values.siCutoff ? dayjs(values.siCutoff).toISOString() : undefined,
        docCutoff: values.docCutoff ? dayjs(values.docCutoff).toISOString() : undefined,
        customsCutoff: values.customsCutoff ? dayjs(values.customsCutoff).toISOString() : undefined,
        vgmCutoff: values.vgmCutoff ? dayjs(values.vgmCutoff).toISOString() : undefined,
        goodsDescription: values.goodsDescription?.trim() || undefined,
        totalPackages: values.totalPackages !== undefined ? Number(values.totalPackages) : undefined,
        totalGrossWeightKg: values.totalGrossWeightKg !== undefined ? Number(values.totalGrossWeightKg) : undefined,
        totalVolumeCbm: values.totalVolumeCbm !== undefined ? Number(values.totalVolumeCbm) : undefined,
        totalPackageUnit: values.totalPackageUnit?.trim() || undefined,
        notes: values.notes?.trim() || undefined,
        bookingNotes: values.bookingNotes?.trim() || undefined,
        allocationNotes: values.allocationNotes?.trim() || undefined,
        operationNotes: values.operationNotes?.trim() || undefined,
        shippingDocuments: values.shippingDocuments
          ?.map((doc: any) => ({
            ...doc,
            masterNo: doc.masterNo?.trim() || '',
            houseNo: doc.houseNo?.trim() || '',
          }))
          .filter((doc: any) => doc.masterNo || doc.houseNo),
        containerRequests: values.containerRequests,
      };

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
      <div style={{ textAlign: 'center', padding: '120px 0', background: '#f5f7fa', minHeight: '100vh' }}>
        <Spin size="large" tip="正在加载订单详情..." />
      </div>
    );
  }

  if (!order) {
    return (
      <div style={{ padding: 48, background: '#f5f7fa', minHeight: '100vh' }}>
        <Card bordered={false} style={{ borderRadius: 8, textAlign: 'center', padding: 32 }}>
          <Empty description="未找到对应的订单档案" />
          <Button type="primary" onClick={() => history.push(`/orders/${kind}`)} style={{ marginTop: 16 }}>
            返回订单列表
          </Button>
        </Card>
      </div>
    );
  }

  const progressStage =
    order.status === 'COMPLETED' ? '已完结' : order.status === 'CANCELLED' ? '已退关' : '未退关';
  const progressColor =
    progressStage === '已完结' ? 'success' : progressStage === '已退关' ? 'error' : 'processing';

  return (
    <>
      <OrderFormTemplate<any>
        loading={false}
        readonly={false}
        formRef={formRef}
        initialValues={initialValues}
        onFinish={handleSaveEdit}
        header={
          <PageHeaderShell
            title={
              <Space size={8}>
                <span style={{ fontSize: 20, fontWeight: 700, fontFamily: 'monospace', color: '#0f172a' }}>
                  {order.orderNo || order.id}
                </span>
                <Tag color={progressColor}>
                  {progressStage}
                </Tag>
                {order.canModify === false && order.status !== 'DRAFT' && (
                  <Tag color="warning" icon={<LockOutlined />}>
                    已锁定
                  </Tag>
                )}
              </Space>
            }
            subTitle={`${config.title} · 订单详情维护`}
            breadcrumbs={[
              { label: '订单管理' },
              { label: config.title, onClick: () => history.push(`/orders/${kind}`) },
              { label: '订单详情' },
            ]}
            onBack={() => history.push(`/orders/${kind}`)}
            extra={[
              <Button key="refresh" icon={<ReloadOutlined />} onClick={() => loadData()}>
                刷新
              </Button>,
              access.canOrder(config.businessType, 'fee.read') && (
                <Button key="fee" icon={<DollarOutlined />} onClick={() => orderFeePanelRef.current?.open(order)}>
                  费用核算
                </Button>
              ),
              access.canOrder(config.businessType, 'release_pod.create') && (
                <Button key="pod" icon={<FileDoneOutlined />} onClick={() => releasePodPanelRef.current?.open(order)}>
                  放货凭证 (POD)
                </Button>
              ),
              access.canOrder(config.businessType, 'abnormal_case.create') && (
                <Button
                  key="abnormal"
                  icon={<WarningOutlined />}
                  danger
                  onClick={() => abnormalCasePanelRef.current?.open(order)}
                >
                  异常登记
                </Button>
              ),
            ].filter(Boolean)}
          />
        }
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
            <Button
              icon={<UndoOutlined />}
              onClick={() => formRef.current?.setFieldsValue(initialValues)}
            >
              重置修改
            </Button>
            <Button
              type="primary"
              icon={<CheckOutlined />}
              loading={saving}
              onClick={() => formRef.current?.submit()}
            >
              保存修改
            </Button>
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
