import { history, useAccess, useParams } from '@umijs/max';
import { App, Button, Card, Empty, Spin } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { OrderDetailTemplate, type OrderDetailData } from '@/components/ui';
import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { orderAttachmentServiceListAttachments } from '@/services/roncin/orderAttachmentService';
import { orderCargoItemServiceListCargoItems } from '@/services/roncin/orderCargoItemService';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import { orderMilestoneServiceListMilestones } from '@/services/roncin/orderMilestoneService';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import {
  MASTER_DATA_KINDS,
  isMasterDataKind,
  orderPersonnelRoleValueEnum,
  parseOrderKind,
  paymentTermOptions,
  shippingDocumentStatusValueEnum,
  tradeTermOptions,
} from './common';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';

export default function OrderDetailPage() {
  const params = useParams<{ kind: string; id: string }>();
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
  const [order, setOrder] = useState<API.Order>();
  const [milestones, setMilestones] = useState<API.OrderMilestone[]>([]);
  const [shippingDocs, setShippingDocs] = useState<API.OrderShippingDocument[]>([]);
  const [containers, setContainers] = useState<API.OrderContainer[]>([]);
  const [cargoItems, setCargoItems] = useState<API.OrderCargoItem[]>([]);
  const [personnel, setPersonnel] = useState<API.OrderPersonnel[]>([]);
  const [attachments, setAttachments] = useState<API.OrderAttachment[]>([]);

  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [ports, setPorts] = useState<API.Port[]>([]);
  const [airports, setAirports] = useState<API.Airport[]>([]);
  const [partners, setPartners] = useState<API.Partner[]>([]);

  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);

  // 1. 加载订单完整数据
  const loadOrderDetail = async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const [
        orderRes,
        milestonesRes,
        docsRes,
        containersRes,
        cargoRes,
        personnelRes,
        attachmentsRes,
        masterOptionsRes,
        portsRes,
        airportsRes,
        partnersRes,
      ] = await Promise.all([
        orderServiceGetOrder({ id: orderId }),
        orderMilestoneServiceListMilestones({ orderId }).catch(() => ({ data: [] })),
        orderShippingDocumentServiceListShippingDocuments({ orderId }).catch(() => ({ data: [] })),
        orderContainerServiceListContainers({ orderId }).catch(() => ({ data: [] })),
        orderCargoItemServiceListCargoItems({ orderId }).catch(() => ({ data: [] })),
        orderPersonnelServiceListPersonnel({ orderId }).catch(() => ({ data: [] })),
        orderAttachmentServiceListAttachments({ orderId }).catch(() => ({ data: [] })),
        masterDataServiceListOptions().catch(() => ({ data: [] })),
        masterDataServiceListPorts({ page: 1, pageSize: 100 }).catch(() => ({ data: [] })),
        masterDataServiceListAirports({ page: 1, pageSize: 100 }).catch(() => ({ data: [] })),
        partnerServiceListPartners({ page: 1, pageSize: 100 }).catch(() => ({ data: [] })),
      ]);

      setOrder(orderRes.data);
      setMilestones(milestonesRes.data ?? []);
      setShippingDocs(docsRes.data ?? []);
      setContainers(containersRes.data ?? []);
      setCargoItems(cargoRes.data ?? []);
      setPersonnel(personnelRes.data ?? []);
      setAttachments(attachmentsRes.data ?? []);
      setMasterOptions(masterOptionsRes.data ?? []);
      setPorts(portsRes.data ?? []);
      setAirports(airportsRes.data ?? []);
      setPartners(partnersRes.data ?? []);
    } catch (error: any) {
      message.error(error.message || '加载订单详情失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadOrderDetail();
  }, [orderId]);

  // 主数据字典映射
  const partnerMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const p of partners) {
      if (p.id) {
        map[p.id] = p.legalName ? `${p.legalName} (${p.code})` : p.code || p.id;
      }
    }
    return map;
  }, [partners]);

  const locationMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const p of ports) {
      if (p.id) {
        map[p.id] = `${p.nameZh ? `${p.nameZh} / ` : ''}${p.nameEn} (${p.unLocode})`;
      }
    }
    for (const a of airports) {
      if (a.id) {
        map[a.id] = `${a.nameZh ? `${a.nameZh} / ` : ''}${a.nameEn} (${a.iataCode})`;
      }
    }
    return map;
  }, [ports, airports]);

  const containerSpecMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const item of masterOptions) {
      if (isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) && item.id) {
        map[item.id] = item.code ? `${item.name} (${item.code})` : (item.name ?? '');
      }
    }
    return map;
  }, [masterOptions]);

  const serviceTypeMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const item of masterOptions) {
      if (isMasterDataKind(item.kind, MASTER_DATA_KINDS.SERVICE_TYPE) && item.id) {
        map[item.id] = item.name ?? item.code ?? '';
      }
    }
    return map;
  }, [masterOptions]);

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

  // 组装 OrderDetailData 聚合数据
  const detailData: OrderDetailData = {
    id: order.id,
    orderNo: order.orderNo,
    businessTypeTitle: config.title,
    status: order.status,
    canModify: order.canModify,
    createdAt: order.createdAt,
    updatedAt: order.updatedAt,

    customerName: partnerMap[order.customerId ?? ''] || order.customerId,
    customerId: order.customerId,
    customerReferenceNo: order.customerReferenceNo,
    internalReferenceNo: order.internalReferenceNo,
    tradeTermName: tradeTermOptions.find((o) => o.value === order.tradeTerm)?.label || (order.tradeTerm ? String(order.tradeTerm) : undefined),
    paymentTermName: paymentTermOptions.find((o) => o.value === order.paymentTerm)?.label || (order.paymentTerm ? String(order.paymentTerm) : undefined),
    bookingAgentName: partnerMap[order.bookingAgentId ?? ''] || order.bookingAgentId,
    carrierName: partnerMap[order.carrierId ?? ''] || order.carrierId,
    contractNo: order.contractNo,
    serviceTypeNames: (order.serviceTypeIds ?? []).map((id) => serviceTypeMap[id] || id),
    cargoValueWithCurrency: order.cargoValue ? `${order.cargoValue} ${order.cargoCurrency || 'USD'}` : undefined,
    insurancePremiumWithCurrency: order.insurancePremium ? `${order.insurancePremium} ${order.insuranceCurrency || 'CNY'}` : undefined,

    originName: locationMap[order.originLocationId ?? ''] || order.originLocationId,
    destinationName: locationMap[order.destinationLocationId ?? ''] || order.destinationLocationId,
    dischargeName: locationMap[order.dischargeLocationId ?? ''] || order.dischargeLocationId,
    transitName: locationMap[order.transitLocationId ?? ''] || order.transitLocationId,
    vesselVoyage: order.vesselVoyage,
    etd: order.etd,
    eta: order.eta,
    loadingTerms: order.loadingTerms,
    siCutoff: order.siCutoff,
    docCutoff: order.docCutoff,
    customsCutoff: order.customsCutoff,
    vgmCutoff: order.vgmCutoff,

    totalPackages: order.totalPackages,
    packageUnit: order.totalPackageUnit,
    grossWeightKg: order.totalGrossWeightKg,
    volumeCbm: order.totalVolumeCbm,

    bookingNotes: order.bookingNotes,
    allocationNotes: order.allocationNotes,
    operationNotes: order.operationNotes,
    notes: order.notes,

    shippingDocuments: shippingDocs.map((doc) => ({
      id: doc.id,
      masterNo: doc.masterNo,
      houseNo: doc.houseNo,
      masterDocumentType: doc.masterDocumentType,
      masterReleaseMethod: doc.masterReleaseMethod,
      releaseType: doc.releaseType,
      status:
        doc.status !== undefined && shippingDocumentStatusValueEnum[doc.status]
          ? shippingDocumentStatusValueEnum[doc.status]?.text
          : '正常',
      createdAt: doc.createdAt,
    })),

    containers: containers.map((c) => ({
      id: c.id,
      containerNo: c.containerNo,
      sealNo: c.sealNo,
      containerSpecName: containerSpecMap[c.containerSpecId ?? ''] || c.containerSpecId,
      grossWeightKg: c.grossWeightKg,
      volumeCbm: c.volumeCbm,
      note: c.note,
    })),

    cargoItems: cargoItems.map((cargo) => ({
      id: cargo.id,
      cargoName: cargo.cargoName,
      packageCount: cargo.packageCount,
      grossWeightKg: cargo.grossWeightKg,
      volumeCbm: cargo.volumeCbm,
      netWeightKg: cargo.netWeightKg,
      note: cargo.note,
    })),

    milestones: milestones.map((m) => ({
      id: m.id,
      type: m.templateNodeLabel || m.type,
      occurredAt: m.occurredAt,
      note: m.note,
    })),

    attachments: attachments.map((att) => ({
      id: att.id,
      docType: att.docType,
      fileName: att.fileName,
      fileSize: att.fileSize,
      mimeType: att.mimeType,
      createdAt: att.createdAt,
    })),

    personnel: personnel.map((p) => ({
      id: p.id,
      roleName: orderPersonnelRoleValueEnum[p.role ?? 0]?.text || `角色 ${p.role}`,
      userId: p.userId,
    })),
  };

  return (
    <>
      <OrderDetailTemplate
        orderKind={config.kind as any}
        title={config.title}
        data={detailData}
        onBack={() => history.push(`/orders/${kind}`)}
        onRefresh={() => loadOrderDetail()}
        onEdit={
          access.canOrder(config.businessType, 'update') && (order.canModify || order.status === 'DRAFT')
            ? () => {
                message.info('可使用快捷编辑或全表单编辑');
              }
            : undefined
        }
        onOpenFees={
          access.canOrder(config.businessType, 'fee.read')
            ? () => orderFeePanelRef.current?.open(order)
            : undefined
        }
        onOpenReleasePod={
          access.canOrder(config.businessType, 'release_pod.create')
            ? () => releasePodPanelRef.current?.open(order)
            : undefined
        }
        onOpenAbnormal={
          access.canOrder(config.businessType, 'abnormal_case.create')
            ? () => abnormalCasePanelRef.current?.open(order)
            : undefined
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
        masterOptions={masterOptions}
      />
    </>
  );
}
