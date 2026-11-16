import type { OrderListFilterParams, OrderListItem } from '@/components/ui';
import { getOrderFlowStatusText } from '@/constants/statusMeta';
import {
  OrderClosureStatus,
  OrderNumberFilterType,
  OrderTerminationStatus,
  SeaDocumentStructure,
} from '@/enums.generated';
import { orderServiceListOrders } from '@/services/roncin/orderService';
import { ensureListResponse, toTableRequest, unwrapPage } from '@/utils/api';
import { paymentTermOptions, tradeTermOptions } from './common';
import { lifecycleFiltersByStage } from './list-constants';
import type { OrderKindDefinition } from './order-kinds/types';

export interface OrderListQueryContext {
  organizationId: string | undefined;
  ports: API.Port[];
  airports: API.Airport[];
  customerMap: Record<string, string>;
}

/** 列表与完整导出共用业务筛选，分页只由列表调用方补入。 */
export function buildOrderListRequestFilters(
  params: OrderListFilterParams,
  definition: OrderKindDefinition,
) {
  const lifecycleFilters = lifecycleFiltersByStage[params.stage ?? ''] ?? {};
  return {
    sortBy: params.sorterOrder ? params.sorterField : undefined,
    sortOrder: params.sorterOrder,
    numberType:
      params.numberType === 'order'
        ? OrderNumberFilterType.ORDER_NUMBER_FILTER_TYPE_ORDER
        : params.numberType === 'master'
          ? OrderNumberFilterType.ORDER_NUMBER_FILTER_TYPE_MASTER
          : params.numberType === 'consolidated_master'
            ? OrderNumberFilterType.ORDER_NUMBER_FILTER_TYPE_CONSOLIDATED_MASTER
            : params.numberType === 'customer_reference'
              ? OrderNumberFilterType.ORDER_NUMBER_FILTER_TYPE_CUSTOMER_REFERENCE
              : params.numberType === 'booking'
                ? OrderNumberFilterType.ORDER_NUMBER_FILTER_TYPE_BOOKING
                : undefined,
    numberKeyword: params.numberKeyword,
    ...lifecycleFilters,
    businessType: definition.businessType,
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
    shippingLineId: params.shippingLineId,
    consigneeShortName: params.consignee,
    shipperShortName: params.shipper,
    operatorId: params.operatorId,
    salesId: params.salesId,
    customerServiceId: params.customerServiceId,
    creatorId: params.creatorId,
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
  };
}

/** 将API订单投影成列表和导出共用的稳定行模型。 */
export async function mapOrderListItems(
  orders: API.Order[],
  definition: OrderKindDefinition,
  ctx: OrderListQueryContext,
): Promise<OrderListItem[]> {
  const items: OrderListItem[] = orders.map((order) => {
    if (!order.id) throw new Error('订单响应缺少业务标识。');
    const originPort = ctx.ports.find((p) => p.id === order.originLocationId);
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
      order.airExport?.master?.originAirportName ||
      order.originLocationName ||
      originPort?.nameZh ||
      originAirport?.nameZh ||
      '-';
    // 服务端名称已含地点代码（如「上海港 (CNSHA)」），本地代码仅作缺失兜底。
    const originCode = order.originLocationName
      ? undefined
      : originPort?.unLocode || originAirport?.iataCode;
    const destName =
      order.airExport?.master?.destinationAirportName ||
      order.destinationLocationName ||
      destPort?.nameZh ||
      destAirport?.nameZh ||
      '-';
    const destCode = order.destinationLocationName
      ? undefined
      : destPort?.unLocode || destAirport?.iataCode;

    const containerSummary = (order.containerRequests ?? [])
      .map((req) => {
        // 箱计划名称随订单一起由服务端投影，完整导出不另查变化中的字典。
        const label =
          req.containerSpecCode?.trim() || req.containerSpecName?.trim();
        if (!req.containerSpecId || !label)
          throw new Error('订单箱计划缺少有效箱型投影。');
        return `${req.quantity ?? 0}×${label}`;
      })
      .join(', ');

    const stage: NonNullable<OrderListItem['stage']> = [];
    if (
      order.terminationStatus ===
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED
    ) {
      stage.push('已退关');
    } else if (
      order.terminationStatus ===
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING
    ) {
      stage.push('退关中');
    }
    if (
      order.closureStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED
    ) {
      stage.push('已完结');
    }
    if (stage.length === 0) stage.push('正常运作');

    return {
      id: order.id || '',
      orderNo: order.orderNo || '',
      orderKind: definition.kind,
      businessType: definition.navigationTitle,
      stage,
      // 委托单位名称以服务端投影为准；本地候选缓存（前50启用客户）仅作缺失兜底，任何情况不展示原始 ID。
      customerName:
        order.customerName || ctx.customerMap[order.customerId ?? ''] || '-',
      customerReferenceNo: order.customerReferenceNo,
      masterBlNo:
        order.airExport?.master?.content?.masterNo ||
        order.seaMasterBill?.masterNo,
      houseBlNo:
        order.airExport?.house?.houseNo || order.seaDocumentSummary?.houseNo,
      documentMode:
        order.airExport?.documentMode === 'HOUSE' ||
        order.seaDocumentSummary?.documentStructure ===
          SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
          ? 'HOUSE'
          : order.airExport?.documentMode === 'DIRECT' ||
              order.seaDocumentSummary?.documentStructure ===
                SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT
            ? 'DIRECT'
            : undefined,
      billRelation:
        definition.transportMode === 'sea'
          ? {
              kind:
                order.seaDocumentSummary?.documentStructure ===
                SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
                  ? 'house'
                  : order.seaDocumentSummary?.documentStructure ===
                      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT
                    ? 'direct'
                    : 'unknown',
              houseNo: order.seaDocumentSummary?.houseNo,
              memberCount: order.seaMasterBill?.memberCount ?? 0,
            }
          : undefined,
      bookingAgentName: order.bookingAgentName,
      shippingLineName:
        order.airExport?.master?.airlineName || order.shippingLineName,
      etd: order.etd,
      eta: order.eta,
      siCutoff: order.siCutoff,
      docCutoff: order.docCutoff,
      customsCutoff: order.customsCutoff,
      vgmCutoff: order.vgmCutoff,
      declarationCutoffAt: order.declarationCutoffAt,
      bookingNo: order.bookingNo,
      dischargePortName: order.dischargeLocationName,
      transitPortName: order.transitLocationName,
      foreignAgentName: order.foreignAgentName,
      shippingAgentName: order.shippingAgentName,
      internalReferenceNo: order.internalReferenceNo,
      orderDate: order.orderDate,
      receivedAt: order.receivedAt,
      cargoReadyAt: order.cargoReadyAt,
      factoryName: order.factoryName,
      goodsDescription: order.goodsDescription,
      cargoValue: order.cargoValue,
      cargoCurrency: order.cargoCurrency,
      unNumber: order.unNumber,
      hazardClass: order.hazardClass,
      containerNos: order.containerNos ?? [],
      attachmentCount: order.attachmentCount,
      customerServiceName: order.customerServiceName,
      spaceNotes: order.allocationNotes,
      bookingNotes: order.bookingNotes,
      operationNotes: order.operationNotes,
      createdAt: order.createdAt,
      vesselVoyage:
        order.airExport?.master?.content?.flightNo || order.vesselVoyage,
      originPortName: originName,
      originPortCode: originCode,
      destinationPortName: destName,
      destinationPortCode: destCode,
      containerSummary: containerSummary || undefined,
      totalPackages: order.totalPackages,
      packageUnit: order.totalPackageUnit,
      grossWeightKg: order.totalGrossWeightKg,
      volumeCbm: order.totalVolumeCbm,
      paymentTerm: paymentTermOptions.find((o) => o.value === order.paymentTerm)
        ?.label,
      tradeTerm: tradeTermOptions.find((o) => o.value === order.tradeTerm)
        ?.label,
      contractNo: order.contractNo,
      shipperName: order.shipperShortName,
      consigneeName: order.consigneeShortName,
      // 人员名称同样来自服务端投影；未分配时保持 undefined，由模板渲染 '-'。
      operatorName: order.operatorName || undefined,
      operatorBranch: order.operatorBranch || undefined,
      salesName: order.salesName || undefined,
      salesBranch: order.salesBranch || undefined,
      creatorName: order.creatorName || undefined,
      lockedAt: order.lockedAt,
      isLocked: Boolean(order.lockedAt),
      notes: order.notes,
      statusName: getOrderFlowStatusText(order.flowStatus, order.businessType),
      abnormalLevel: order.hasActiveException ? 'high' : 'normal',
      rawRecord: order,
    };
  });

  return items;
}

/** 分页请求与数据流仍交给官方ProTable request引擎。 */
export async function queryOrderList(
  params: OrderListFilterParams,
  definition: OrderKindDefinition,
  ctx: OrderListQueryContext,
) {
  if (!ctx.organizationId) throw new Error('当前公司尚未就绪。');
  const response = await orderServiceListOrders({
    ...buildOrderListRequestFilters(params, definition),
    page: params.page,
    pageSize: params.pageSize,
  });
  ensureListResponse(response, '订单列表加载失败。');
  if (response?.success !== true) throw new Error('订单列表加载失败。');
  const page = unwrapPage(response);
  const items = await mapOrderListItems(page.data, definition, ctx);
  return {
    ...toTableRequest({ ...response, data: items }),
    total: page.total,
  };
}
