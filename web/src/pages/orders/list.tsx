import type { ActionType } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { history, useAccess, useLocation } from '@umijs/max';
import { OrderListTemplate, type OrderListItem } from '@/components/ui';
import { App, Result } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import {
  orderServiceListOrders,
  orderServiceListPersonnelOptions,
} from '@/services/roncin/orderService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import {
  MASTER_DATA_KINDS,
  isMasterDataKind,
  parseOrderKind,
  paymentTermOptions,
  seFlowStatusLabels,
  shipmentTypeOptions,
  tradeDirectionOptions,
  tradeTermOptions,
} from './common';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';
import AttachmentDrawer, {
  type AttachmentDrawerRef,
} from './components/drawers/AttachmentDrawer';
import CargoItemDrawer, {
  type CargoItemDrawerRef,
} from './components/drawers/CargoItemDrawer';
import ConsolidationDrawer, {
  type ConsolidationDrawerRef,
} from './components/drawers/ConsolidationDrawer';
import ContainerDrawer, {
  type ContainerDrawerRef,
} from './components/drawers/ContainerDrawer';
import MilestoneDrawer, {
  type MilestoneDrawerRef,
} from './components/drawers/MilestoneDrawer';
import PersonnelDrawer, {
  type PersonnelDrawerRef,
} from './components/drawers/PersonnelDrawer';
import ShippingDocumentDrawer, {
  type ShippingDocumentDrawerRef,
} from './components/drawers/ShippingDocumentDrawer';
import EditOrderModal, {
  type EditOrderModalRef,
} from './components/modals/EditOrderModal';
import TransitionModal, {
  type TransitionModalRef,
} from './components/modals/TransitionModal';

const seStatusTabs = [
  { key: 'all', label: '全部订单' },
  { key: 'draft', label: '草稿/待订舱', badgeColor: '#d9d9d9' },
  { key: 'booked', label: '已订舱', badgeColor: '#1677ff' },
  { key: 'allocated', label: '已配舱', badgeColor: '#13c2c2' },
  { key: 'trucking', label: '拖车安排', badgeColor: '#722ed1' },
  { key: 'cutoff', label: '已截单', badgeColor: '#eb2f96' },
  { key: 'customs', label: '报关放行', badgeColor: '#2f54eb' },
  { key: 'released', label: '已放单', badgeColor: '#52c41a' },
  { key: 'terminating', label: '退关中', badgeColor: '#fa8c16' },
  { key: 'terminated', label: '已退关', badgeColor: '#ff4d4f' },
  { key: 'completed', label: '已完结', badgeColor: '#52c41a' },
  { key: 'abnormal', label: '异常挂起', badgeColor: '#ff4d4f' },
];

const lifecycleFiltersByStage: Record<
  string,
  {
    flowStatus?: number;
    terminationStatus?: number;
    closureStatus?: number;
    hasActiveException?: boolean;
  }
> = {
  draft: { flowStatus: 1 },
  booked: { flowStatus: 2 },
  allocated: { flowStatus: 3 },
  trucking: { flowStatus: 4 },
  cutoff: { flowStatus: 5 },
  customs: { flowStatus: 6 },
  released: { flowStatus: 7 },
  terminating: { terminationStatus: 2 },
  terminated: { terminationStatus: 3 },
  completed: { closureStatus: 2 },
  abnormal: { hasActiveException: true },
  unreturned: { terminationStatus: 1, closureStatus: 1 },
  returned: { terminationStatus: 3 },
};

export default function OrderListPage() {
  const location = useLocation();
  const config = parseOrderKind(location.pathname);

  const actionRef = useRef<ActionType | undefined>(undefined);
  const editOrderModalRef = useRef<EditOrderModalRef | null>(null);
  const transitionModalRef = useRef<TransitionModalRef | null>(null);
  const milestoneDrawerRef = useRef<MilestoneDrawerRef | null>(null);
  const attachmentDrawerRef = useRef<AttachmentDrawerRef | null>(null);
  const personnelDrawerRef = useRef<PersonnelDrawerRef | null>(null);
  const containerDrawerRef = useRef<ContainerDrawerRef | null>(null);
  const consolidationDrawerRef = useRef<ConsolidationDrawerRef | null>(null);
  const cargoItemDrawerRef = useRef<CargoItemDrawerRef | null>(null);
  const shippingDocumentDrawerRef = useRef<ShippingDocumentDrawerRef | null>(
    null,
  );
  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);

  const { message } = App.useApp();
  const access = useAccess();

  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [ports, setPorts] = useState<API.Port[]>([]);
  const [airports, setAirports] = useState<API.Airport[]>([]);
  const [customerMap, setCustomerMap] = useState<Record<string, string>>({});

  useEffect(() => {
    void Promise.all([
      masterDataServiceListOptions(),
      masterDataServiceListPorts({ page: 1, pageSize: 200 }),
      masterDataServiceListAirports({ page: 1, pageSize: 200 }),
      partnerServiceListPartners({ role: 1, page: 1, pageSize: 200 }),
    ])
      .then(
        ([
          optionsResponse,
          portsResponse,
          airportsResponse,
          partnersResponse,
        ]) => {
          setMasterOptions(optionsResponse.data ?? []);
          setPorts(portsResponse.data ?? []);
          setAirports(airportsResponse.data ?? []);
          const partnerList = partnersResponse.data ?? [];
          setCustomerMap((prev) => {
            const next = { ...prev };
            for (const p of partnerList) {
              if (p.id) {
                next[p.id] = p.legalName
                  ? `${p.legalName} (${p.code})`
                  : p.code || p.id;
              }
            }
            return next;
          });
        },
      )
      .catch((error: Error) =>
        message.error(error.message || '订单主数据加载失败'),
      );
  }, [message]);

  if (!config) {
    return (
      <PageContainer>
        <Result
          status="404"
          title="未知的业务类型"
          subTitle="当前路径未匹配到有效的订单业务类型配置"
        />
      </PageContainer>
    );
  }

  const containerSpecOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const containerSpecMap = Object.fromEntries(
    masterOptions
      .filter(
        (item) =>
          isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) &&
          item.id,
      )
      .map((item) => [
        item.id as string,
        item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      ]),
  );

  const serviceTypeOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.SERVICE_TYPE) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const cargoCategoryOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CARGO_CATEGORY) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const regionLocationOptions = masterOptions
    .filter(
      (item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.REGION) &&
        item.enabled !== false,
    )
    .map((item) => ({
      label: item.code ? `${item.name} (${item.code})` : (item.name ?? ''),
      value: item.id ?? '',
    }));

  const locationOptions = [
    ...regionLocationOptions,
    ...ports
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
        value: item.id ?? '',
      })),
    ...airports
      .filter((item) => item.enabled !== false)
      .map((item) => ({
        label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.iataCode})`,
        value: item.id ?? '',
      })),
  ];

  const searchCustomers = async (keyword?: string) => {
    const res = await partnerServiceListPartners({
      role: 1,
      enabled: true,
      keyword,
    });
    const partners = res.data ?? [];
    setCustomerMap((prev) => {
      const next = { ...prev };
      for (const p of partners) {
        if (p.id) {
          next[p.id] = p.legalName
            ? `${p.legalName} (${p.code})`
            : p.code || p.id;
        }
      }
      return next;
    });
    return partners.map((p) => ({
      label: p.legalName ? `${p.legalName} (${p.code})` : p.code || p.id || '',
      value: p.id ?? '',
    }));
  };

  const searchOrderPorts = async (keyword?: string) => {
    const response = await masterDataServiceListPorts({
      page: 1,
      pageSize: 50,
      keyword,
      enabled: true,
    });
    const result = response.data ?? [];
    setPorts((current) => {
      const merged = new Map(
        current.filter((item) => item.id).map((item) => [item.id, item]),
      );
      for (const item of result) {
        if (item.id) merged.set(item.id, item);
      }
      return [...merged.values()];
    });
    return result.map((item) => ({
      label: `${item.nameZh ? `${item.nameZh} / ` : ''}${item.nameEn} (${item.unLocode})`,
      value: item.id ?? '',
    }));
  };

  const searchOrderCarriers = async (keyword?: string) => {
    const response = await partnerServiceListPartners({
      role: 4,
      page: 1,
      pageSize: 50,
      keyword,
      enabled: true,
    });
    return (response.data ?? []).map((item) => ({
      label: item.legalName
        ? `${item.legalName} (${item.code})`
        : item.code || item.id || '',
      value: item.id ?? '',
    }));
  };

  const searchOrderPersonnel = async (keyword?: string) => {
    const response = await orderServiceListPersonnelOptions({
      businessType: config?.businessType ?? 1,
      keyword,
      page: 1,
      pageSize: 50,
    });
    return (response.data ?? [])
      .filter(
        (item) =>
          item.userId &&
          item.displayName &&
          item.organizationId &&
          item.organizationName,
      )
      .map((item) => ({
        userId: item.userId as string,
        displayName: item.displayName as string,
        organizationId: item.organizationId as string,
        organizationName: item.organizationName as string,
      }));
  };

  return (
    <>
      <OrderListTemplate
        actionRef={actionRef}
        orderKind={config.kind as any}
        title={config.title}
        subTitle={`统一维护${config.title}全流程状态、主分单据、箱量配载、费用核算与业务履约轨迹`}
        statusTabs={seStatusTabs}
        options={{
          loadPorts: searchOrderPorts,
          loadPartners: searchCustomers,
          loadCarriers: searchOrderCarriers,
          loadPersonnel: searchOrderPersonnel,
        }}
        queryOrders={async (params) => {
          const lifecycleFilters =
            lifecycleFiltersByStage[params.stage ?? ''] ?? {};
          const response = await orderServiceListOrders({
            page: params.page,
            pageSize: params.pageSize,
            numberType:
              params.numberType === 'order'
                ? 1
                : params.numberType === 'master'
                  ? 2
                  : params.numberType === 'consolidated_master'
                    ? 3
                    : undefined,
            numberKeyword: params.numberKeyword,
            ...lifecycleFilters,
            businessType: config.businessType,
            customerId: params.customerId,
            createdAtFrom: params.createdAtRange?.[0],
            createdAtTo: params.createdAtRange?.[1],
            etdFrom: params.etdRange?.[0],
            etdTo: params.etdRange?.[1],
            etaFrom: params.etaRange?.[0],
            etaTo: params.etaRange?.[1],
            statusTimeFrom: params.statusTimeRange?.[0],
            statusTimeTo: params.statusTimeRange?.[1],
            lockedAtFrom: params.lockedAtRange?.[0],
            lockedAtTo: params.lockedAtRange?.[1],
            originLocationId: params.originLocationId,
            destinationLocationId: params.destinationLocationId,
            carrierId: params.carrierId,
            consigneeShortName: params.consignee,
            shipperShortName: params.shipper,
            operatorId: params.operatorId,
            operatorOrganizationId: params.operatorDeptId,
            salesId: params.salesId,
            salesOrganizationId: params.salesDeptId,
            customerServiceId: params.customerServiceId,
            customerServiceOrganizationId: params.customerServiceDeptId,
            creatorId: params.creatorId,
            creatorOrganizationId: params.creatorDeptId,
            tags: params.tags,
            tagMatchMode:
              params.tagMatchMode === 'fuzzy_or'
                ? 1
                : params.tagMatchMode === 'exact_and'
                  ? 2
                  : undefined,
            isLocked:
              params.isLocked === 'locked'
                ? true
                : params.isLocked === 'unlocked'
                  ? false
                  : undefined,
            isShared:
              params.shareStatus === 'shared'
                ? true
                : params.shareStatus === 'unshared'
                  ? false
                  : undefined,
          });

          const items: OrderListItem[] = (response.data ?? []).map((order) => {
            const originPort = ports.find(
              (p) => p.id === order.originLocationId,
            );
            const destPort = ports.find(
              (p) => p.id === order.destinationLocationId,
            );
            const originAirport = airports.find(
              (a) => a.id === order.originLocationId,
            );
            const destAirport = airports.find(
              (a) => a.id === order.destinationLocationId,
            );

            const originName =
              originPort?.nameZh ||
              originAirport?.nameZh ||
              order.originLocationId;
            const originCode = originPort?.unLocode || originAirport?.iataCode;
            const destName =
              destPort?.nameZh ||
              destAirport?.nameZh ||
              order.destinationLocationId;
            const destCode = destPort?.unLocode || destAirport?.iataCode;

            const containerSummary = (order.containerRequests ?? [])
              .map(
                (req) =>
                  `${req.quantity}×${containerSpecMap[req.containerSpecId ?? ''] || req.containerSpecId}`,
              )
              .join(', ');

            return {
              id: order.id || '',
              orderNo: order.orderNo || '',
              orderKind: config.kind as any,
              businessType: config.title,
              stage:
                order.closureStatus === 2
                  ? '已完结'
                  : order.terminationStatus === 3
                    ? '已退关'
                    : order.terminationStatus === 2
                      ? '退关中'
                      : order.hasActiveException
                        ? '异常挂起'
                        : '正常运作',
              customerName:
                customerMap[order.customerId ?? ''] || order.customerId,
              customerReferenceNo: order.customerReferenceNo,
              createdAt: order.createdAt,
              vesselVoyage: order.vesselVoyage,
              originPortName: originName,
              originPortCode: originCode,
              destinationPortName: destName,
              destinationPortCode: destCode,
              containerSummary: containerSummary || undefined,
              totalPackages: order.totalPackages,
              packageUnit: order.totalPackageUnit,
              grossWeightKg: order.totalGrossWeightKg,
              volumeCbm: order.totalVolumeCbm,
              paymentTerm: paymentTermOptions.find(
                (o) => o.value === order.paymentTerm,
              )?.label,
              tradeTerm: tradeTermOptions.find(
                (o) => o.value === order.tradeTerm,
              )?.label,
              contractNo: order.contractNo,
              shipperName: order.shipperShortName,
              consigneeName: order.consigneeShortName,
              lockedAt: order.lockedAt,
              isLocked: Boolean(order.lockedAt),
              tags: order.tags,
              notes: order.notes,
              statusName:
                seFlowStatusLabels[order.flowStatus ?? 0] ?? '未知状态',
              abnormalLevel: order.hasActiveException ? 'high' : 'normal',
              rawRecord: order,
            };
          });

          return {
            data: items,
            total: response.total ?? 0,
            success: response.success ?? true,
          };
        }}
        onCreateOrder={() => history.push(`/orders/${config.kind}/new`)}
        onViewDetail={(item) =>
          history.push(`/orders/${item.orderKind || config.kind}/${item.id}`)
        }
        onEditOrder={(item) =>
          item.rawRecord && editOrderModalRef.current?.open(item.rawRecord)
        }
        onOpenFees={(item) =>
          item.rawRecord && orderFeePanelRef.current?.open(item.rawRecord)
        }
        onOpenMilestones={(item) =>
          item.rawRecord && milestoneDrawerRef.current?.open(item.rawRecord)
        }
        onOpenDocuments={(item) =>
          item.rawRecord &&
          shippingDocumentDrawerRef.current?.open(item.rawRecord)
        }
        onOpenContainers={(item) =>
          item.rawRecord && containerDrawerRef.current?.open(item.rawRecord)
        }
        onOpenCargo={(item) =>
          item.rawRecord && cargoItemDrawerRef.current?.open(item.rawRecord)
        }
        onOpenAttachments={(item) =>
          item.rawRecord && attachmentDrawerRef.current?.open(item.rawRecord)
        }
        onOpenPersonnel={(item) =>
          item.rawRecord && personnelDrawerRef.current?.open(item.rawRecord)
        }
        onOpenConsolidations={(item) =>
          item.rawRecord && consolidationDrawerRef.current?.open(item.rawRecord)
        }
        onOpenAbnormal={(item) =>
          item.rawRecord && abnormalCasePanelRef.current?.open(item.rawRecord)
        }
        onTransitionStatus={(item) =>
          item.rawRecord && transitionModalRef.current?.open(item.rawRecord)
        }
      />

      <EditOrderModal
        ref={editOrderModalRef}
        category={config.category}
        tradeDirectionOptions={tradeDirectionOptions}
        tradeTermOptions={tradeTermOptions}
        paymentTermOptions={paymentTermOptions}
        shipmentTypeOptions={shipmentTypeOptions}
        locationOptions={locationOptions}
        serviceTypeOptions={serviceTypeOptions}
        cargoCategoryOptions={cargoCategoryOptions}
        containerSpecOptions={containerSpecOptions}
        searchCustomers={searchCustomers}
        onSuccess={() => actionRef.current?.reload()}
      />
      <TransitionModal
        ref={transitionModalRef}
        onSuccess={() => actionRef.current?.reload()}
      />
      <MilestoneDrawer
        ref={milestoneDrawerRef}
        canSet={access.canOrder(config.businessType, 'milestone.set')}
      />
      <AttachmentDrawer
        ref={attachmentDrawerRef}
        canRegister={access.canOrder(
          config.businessType,
          'attachment.register',
        )}
      />
      <PersonnelDrawer
        ref={personnelDrawerRef}
        canAssign={access.canOrder(config.businessType, 'personnel.assign')}
        canRemove={access.canOrder(config.businessType, 'personnel.remove')}
      />
      <ContainerDrawer
        ref={containerDrawerRef}
        canCreate={access.canOrder(config.businessType, 'container.create')}
        canUpdate={access.canOrder(config.businessType, 'container.update')}
        canRemove={access.canOrder(config.businessType, 'container.delete')}
        containerSpecOptions={containerSpecOptions}
        containerSpecMap={containerSpecMap}
      />
      <ConsolidationDrawer ref={consolidationDrawerRef} />
      <CargoItemDrawer
        ref={cargoItemDrawerRef}
        canCreate={access.canOrder(config.businessType, 'cargo_item.create')}
        canUpdate={access.canOrder(config.businessType, 'cargo_item.update')}
        canRemove={access.canOrder(config.businessType, 'cargo_item.delete')}
      />
      <ShippingDocumentDrawer
        ref={shippingDocumentDrawerRef}
        canManage={access.canOrder(config.businessType, 'update')}
        category={config.category}
      />
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
