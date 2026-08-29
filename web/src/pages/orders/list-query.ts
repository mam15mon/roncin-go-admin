import type {
  OrderListFilterParams,
  OrderListItem,
} from '@/components/ui';
import { orderServiceListOrders } from '@/services/roncin/orderService';
import {
  paymentTermOptions,
  seFlowStatusLabels,
  tradeTermOptions,
  type OrderKindConfig,
} from './common';
import { lifecycleFiltersByStage } from './list-constants';

interface OrderListQueryContext {
  ports: API.Port[];
  airports: API.Airport[];
  customerMap: Record<string, string>;
  containerSpecMap: Record<string, string>;
}

/** 将模板筛选参数映射为订单列表查询请求，并把响应整理为模板行数据。 */
export async function queryOrderList(
  params: OrderListFilterParams,
  config: OrderKindConfig,
  ctx: OrderListQueryContext,
) {
  const lifecycleFilters = lifecycleFiltersByStage[params.stage ?? ''] ?? {};
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
    tagIds: params.tagIds,
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
    const originPort = ctx.ports.find(
      (p) => p.id === order.originLocationId,
    );
    const destPort = ctx.ports.find(
      (p) => p.id === order.destinationLocationId,
    );
    const originAirport = ctx.airports.find(
      (a) => a.id === order.originLocationId,
    );
    const destAirport = ctx.airports.find(
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
          `${req.quantity}×${ctx.containerSpecMap[req.containerSpecId ?? ''] || req.containerSpecId}`,
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
        ctx.customerMap[order.customerId ?? ''] || order.customerId,
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
}
