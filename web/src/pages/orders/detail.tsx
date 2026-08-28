import {
  AlertOutlined,
  CheckOutlined,
  CopyOutlined,
  DollarOutlined,
  DownOutlined,
  FileDoneOutlined,
  ReloadOutlined,
  SaveOutlined,
  UndoOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import { history, useAccess, useParams } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Col,
  Dropdown,
  Empty,
  Input,
  type MenuProps,
  Select,
  Space,
  Spin,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
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
  const initialValues = useMemo(() => {
    if (!order) return {};

    const personnelRoleMap: Record<
      number,
      { userId?: string; organizationId?: string }
    > = {};
    for (const p of personnel) {
      if (p.role !== undefined) {
        personnelRoleMap[p.role] = {
          userId: p.userId,
          organizationId: p.organizationId,
        };
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
      customsCutoff: order.customsCutoff
        ? dayjs(order.customsCutoff)
        : undefined,
      vgmCutoff: order.vgmCutoff ? dayjs(order.vgmCutoff) : undefined,
      goodsDescription: order.goodsDescription,
      totalPackages: order.totalPackages,
      totalGrossWeightKg: order.totalGrossWeightKg,
      totalVolumeCbm: order.totalVolumeCbm,
      totalPackageUnit: order.totalPackageUnit || 'CTNS',
      orderDate: order.orderDate
        ? dayjs(order.orderDate)
        : dayjs(order.createdAt),
      notes: order.notes,
      bookingNotes: order.bookingNotes,
      allocationNotes: order.allocationNotes,
      operationNotes: order.operationNotes,
      shippingDocuments:
        shippingDocs.length > 0 ? shippingDocs : order.shippingDocuments,
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
  const prependSections: OrderFormTemplateSection[] = useMemo(() => {
    const isUnreturned = order?.terminationStatus !== 3;
    const isUncompleted = order?.closureStatus !== 2;

    const steps = [
      { value: 2, key: 'booked', label: '已订舱' },
      { value: 3, key: 'allocated', label: '已配舱' },
      { value: 4, key: 'trucked', label: '拖车已安排' },
      { value: 5, key: 'si_cutoff', label: '已截单' },
      { value: 6, key: 'customs', label: '报关已安排' },
      { value: 7, key: 'released', label: '已放单' },
    ];

    return [
      {
        key: 'orderStatusSection',
        title: '订单状态',
        collapsible: true,
        extra: (
          <Space size={12} align="center">
            <Tag color="blue">海运出口固定流程</Tag>
            <div
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                padding: '2px 10px',
                borderRadius: 12,
                backgroundColor: isUnreturned ? '#f1f5f9' : '#fee2e2',
                color: isUnreturned ? '#475569' : '#ef4444',
                fontSize: 12,
                userSelect: 'none',
              }}
            >
              <span style={{ fontSize: 10 }}>{isUnreturned ? '⚪' : '🔴'}</span>
              <span>{isUnreturned ? '未退关' : '已退关'}</span>
            </div>
            <div
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                padding: '2px 10px',
                borderRadius: 12,
                backgroundColor: isUncompleted ? '#f1f5f9' : '#dcfce7',
                color: isUncompleted ? '#475569' : '#16a34a',
                fontSize: 12,
                userSelect: 'none',
              }}
            >
              <span style={{ fontSize: 10 }}>
                {isUncompleted ? '⚪' : '🟢'}
              </span>
              <span>{isUncompleted ? '未完结' : '已完结'}</span>
            </div>
          </Space>
        ),
        content: (
          <Col span={24}>
            {/* 海管家极简横向流程节点 */}
            <div
              style={{
                padding: '20px 40px 12px',
                position: 'relative',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              {/* 背景贯穿灰色连接线 */}
              <div
                style={{
                  position: 'absolute',
                  top: 27,
                  left: 60,
                  right: 60,
                  height: 1,
                  backgroundColor: '#cbd5e1',
                  zIndex: 1,
                }}
              />

              {steps.map((st) => {
                const isPassed = Number(order?.flowStatus ?? 0) >= st.value;
                return (
                  <div
                    key={st.key}
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      position: 'relative',
                      zIndex: 2,
                    }}
                  >
                    <div
                      style={{
                        width: 14,
                        height: 14,
                        borderRadius: '50%',
                        backgroundColor: isPassed ? '#1677ff' : '#94a3b8',
                        border: '3px solid #ffffff',
                        boxShadow: '0 0 0 1px #cbd5e1',
                        marginBottom: 8,
                      }}
                    />
                    <span
                      style={{
                        fontSize: 12,
                        color: isPassed ? '#0f172a' : '#64748b',
                        fontWeight: isPassed ? 500 : 400,
                      }}
                    >
                      {st.label}
                    </span>
                  </div>
                );
              })}
            </div>
          </Col>
        ),
      },
    ];
  }, [order]);

  // 5. 后置区块：操作记录日志
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
                        <div
                          style={{
                            fontSize: 12,
                            color: '#94a3b8',
                            marginTop: 2,
                          }}
                        >
                          {order?.createdAt
                            ? dayjs(order.createdAt).format(
                                'YYYY-MM-DD HH:mm:ss',
                              )
                            : '-'}
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
                        <div
                          style={{
                            fontSize: 12,
                            color: '#94a3b8',
                            marginTop: 2,
                          }}
                        >
                          {order?.updatedAt
                            ? dayjs(order.updatedAt).format(
                                'YYYY-MM-DD HH:mm:ss',
                              )
                            : '-'}
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

  // 6. 保存修改提交处理
  const handleSaveEdit = async (values: any) => {
    if (!orderId) return false;
    setSaving(true);
    try {
      const payload: API.UpdateOrderRequest = {
        id: orderId,
        expectedVersion: order?.version || '0',
        customerId: values.customerId,
        customerReferenceNo: values.customerReferenceNo?.trim() || undefined,
        internalReferenceNo: values.internalReferenceNo?.trim() || undefined,
        tradeTerm:
          values.tradeTerm !== undefined ? Number(values.tradeTerm) : undefined,
        paymentTerm:
          values.paymentTerm !== undefined
            ? Number(values.paymentTerm)
            : undefined,
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
        shipmentType:
          values.shipmentType !== undefined
            ? Number(values.shipmentType)
            : undefined,
        containerOwnership:
          values.containerOwnership !== undefined
            ? Number(values.containerOwnership)
            : undefined,
        shipmentMode:
          values.shipmentMode !== undefined
            ? Number(values.shipmentMode)
            : undefined,
        serviceTypeIds: values.serviceTypeIds,
        cargoCategoryIds: values.cargoCategoryIds,
        originLocationId: values.originLocationId || undefined,
        destinationLocationId: values.destinationLocationId || undefined,
        dischargeLocationId: values.dischargeLocationId || undefined,
        transitLocationId: values.transitLocationId || undefined,
        vesselVoyage: values.vesselVoyage?.trim() || undefined,
        etd: values.etd ? dayjs(values.etd).toISOString() : undefined,
        eta: values.eta ? dayjs(values.eta).toISOString() : undefined,
        siCutoff: values.siCutoff
          ? dayjs(values.siCutoff).toISOString()
          : undefined,
        docCutoff: values.docCutoff
          ? dayjs(values.docCutoff).toISOString()
          : undefined,
        customsCutoff: values.customsCutoff
          ? dayjs(values.customsCutoff).toISOString()
          : undefined,
        vgmCutoff: values.vgmCutoff
          ? dayjs(values.vgmCutoff).toISOString()
          : undefined,
        goodsDescription: values.goodsDescription?.trim() || undefined,
        totalPackages:
          values.totalPackages !== undefined
            ? Number(values.totalPackages)
            : undefined,
        totalGrossWeightKg:
          values.totalGrossWeightKg !== undefined
            ? Number(values.totalGrossWeightKg)
            : undefined,
        totalVolumeCbm:
          values.totalVolumeCbm !== undefined
            ? Number(values.totalVolumeCbm)
            : undefined,
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
            reason: reason.trim(),
          },
        );
        setOrder(response.data);
        message.success('订单终止状态已更新');
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
      title: targetStatus === 2 ? '确认完结订单' : '确认反结案',
      content: (
        <Input.TextArea
          style={{ marginTop: 12 }}
          placeholder="请输入原因（必填）"
          maxLength={500}
          showCount
          onChange={(event) => {
            reason = event.target.value;
          }}
        />
      ),
      async onOk() {
        if (!reason.trim()) {
          message.error('请输入原因');
          return Promise.reject();
        }
        const response = await orderServiceTransitionOrderClosure(
          { id: orderID },
          { id: orderID, expectedVersion, targetStatus, reason: reason.trim() },
        );
        setOrder(response.data);
        message.success(targetStatus === 2 ? '订单已完结' : '订单已反结案');
      },
    });
  };

  const moreMenuItems: MenuProps['items'] = [
    {
      key: 'refresh',
      label: '刷新数据',
      icon: <ReloadOutlined />,
      onClick: () => void loadData(),
    },
    {
      key: 'copyOrderNo',
      label: '复制订单编号',
      icon: <CopyOutlined />,
      onClick: () => {
        if (order.orderNo) {
          navigator.clipboard?.writeText(order.orderNo);
          message.success('订单编号已复制到剪贴板');
        }
      },
    },
  ];

  // 纯粹本地已接入的操作工具栏（已彻底移除未接入的通道与推广小图标）
  const cleanHeader = (
    <div style={{ marginBottom: 12 }}>
      {/* 顶部第 1 行：面包屑 */}
      <div style={{ padding: '8px 16px', fontSize: 13, color: '#64748b' }}>
        <Space size={6}>
          <a
            style={{ color: '#64748b' }}
            onClick={() => history.push(`/orders/${kind}`)}
          >
            {config.title}
          </a>
          <span>&gt;</span>
          <span style={{ color: '#1677ff', fontWeight: 500 }}>
            {config.title}详情
          </span>
          {order.orderNo && (
            <span
              style={{
                color: '#0f172a',
                fontWeight: 600,
                marginLeft: 8,
                fontFamily: 'monospace',
              }}
            >
              ({order.orderNo})
            </span>
          )}
        </Space>
      </div>

      {/* 顶部第 2 行：操作工具栏 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-start',
          flexWrap: 'nowrap',
          overflowX: 'auto',
          backgroundColor: '#ffffff',
          padding: '8px 16px',
          borderRadius: 6,
          border: '1px solid #e2e8f0',
          boxShadow: '0 1px 2px rgba(0, 0, 0, 0.02)',
        }}
      >
        <Space size={8} wrap={false}>
          {/* 实心蓝底主保存按钮 */}
          {hasAction(1) && (
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={saving}
              onClick={() => formRef.current?.submit()}
              style={{ fontWeight: 500 }}
            >
              保存
            </Button>
          )}

          {hasAction(3) && (
            <Button danger onClick={() => confirmTermination(2)}>
              发起退关
            </Button>
          )}
          {hasAction(4) && (
            <Button danger type="primary" onClick={() => confirmTermination(3)}>
              完成退关
            </Button>
          )}
          {hasAction(5) && (
            <Button onClick={() => confirmTermination(1)}>取消退关</Button>
          )}
          {hasAction(6) && (
            <Button type="primary" onClick={() => confirmClosure(2)}>
              完结订单
            </Button>
          )}
          {hasAction(7) && (
            <Button onClick={() => confirmClosure(1)}>反结案</Button>
          )}

          {/* 费用录入（直达独立全屏费用工作台页面） */}
          {access.canOrder(config.businessType, 'fee.read') && (
            <Button
              type="primary"
              icon={<DollarOutlined />}
              onClick={() => history.push(`/orders/${kind}/${orderId}/fees`)}
              style={{ fontWeight: 500 }}
            >
              费用录入
            </Button>
          )}

          {/* 导出单证 / 放货凭证 POD */}
          {access.canOrder(config.businessType, 'release_pod.create') && (
            <Button
              style={{ color: '#1677ff', borderColor: '#1677ff' }}
              icon={<FileDoneOutlined />}
              onClick={() => releasePodPanelRef.current?.open(order)}
            >
              导出单证 (POD)
            </Button>
          )}

          {/* 异常情况 */}
          {access.canOrder(config.businessType, 'abnormal_case.create') && (
            <Button
              style={{ color: '#ff4d4f', borderColor: '#ff4d4f' }}
              icon={<AlertOutlined />}
              onClick={() => abnormalCasePanelRef.current?.open(order)}
            >
              异常情况
            </Button>
          )}

          {/* 更多操作 */}
          <Dropdown menu={{ items: moreMenuItems }} trigger={['click']}>
            <Button style={{ color: '#64748b', borderColor: '#d9d9d9' }}>
              更多操作 <DownOutlined style={{ fontSize: 10 }} />
            </Button>
          </Dropdown>
        </Space>
      </div>
    </div>
  );

  return (
    <>
      <OrderFormTemplate<any>
        loading={false}
        readonly={!hasAction(1)}
        formRef={formRef}
        initialValues={initialValues}
        onFinish={handleSaveEdit}
        header={cleanHeader}
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
