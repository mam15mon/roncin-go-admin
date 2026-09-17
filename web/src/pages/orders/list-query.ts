import type { OrderListFilterParams, OrderListItem } from '@/components/ui';
import { orderFlowStatusMeta, statusText } from '@/constants/statusMeta';
import {
  OrderClosureStatus,
  OrderNumberFilterType,
  OrderTerminationStatus,
} from '@/enums.generated';
import { orderServiceListOrders } from '@/services/roncin/orderService';
import { toTableRequest, unwrapPage } from '@/utils/api';
import { paymentTermOptions, tradeTermOptions } from './common';
import { lifecycleFiltersByStage } from './list-constants';
import type { OrderKindDefinition } from './order-kinds/types';

interface OrderListQueryContext {
  ports: API.Port[];
  airports: API.Airport[];
  customerMap: Record<string, string>;
  containerSpecMap: Record<string, string>;
}

/** 将模板筛选参数映射为订单列表查询请求，并把响应整理为模板行数据。 */
export async function queryOrderList(
  params: OrderListFilterParams,
  definition: OrderKindDefinition,
  ctx: OrderListQueryContext,
) {
  const lifecycleFilters = lifecycleFiltersByStage[params.stage ?? ''] ?? {};
  const response = await orderServiceListOrders({
    page: params.page,
    pageSize: params.pageSize,
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

  const page = unwrapPage(response);
  const items: OrderListItem[] = page.data.map((order) => {
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
      originPort?.nameZh || originAirport?.nameZh || order.originLocationId;
    const originCode = originPort?.unLocode || originAirport?.iataCode;
    const destName =
      destPort?.nameZh || destAirport?.nameZh || order.destinationLocationId;
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
      orderKind: definition.kind,
      businessType: definition.navigationTitle,
      stage:
        order.closureStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED
          ? '已完结'
          : order.terminationStatus ===
              OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED
            ? '已退关'
            : order.terminationStatus ===
                OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING
              ? '退关中'
              : order.hasActiveException
                ? '异常挂起'
                : '正常运作',
      customerName: ctx.customerMap[order.customerId ?? ''] || order.customerId,
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
      paymentTerm: paymentTermOptions.find((o) => o.value === order.paymentTerm)
        ?.label,
      tradeTerm: tradeTermOptions.find((o) => o.value === order.tradeTerm)
        ?.label,
      contractNo: order.contractNo,
      shipperName: order.shipperShortName,
      consigneeName: order.consigneeShortName,
      lockedAt: order.lockedAt,
      isLocked: Boolean(order.lockedAt),
      tags: order.tags,
      notes: order.notes,
      statusName: statusText(
        orderFlowStatusMeta,
        order.flowStatus ?? 0,
        '未知状态',
      ),
      abnormalLevel: order.hasActiveException ? 'high' : 'normal',
      rawRecord: order,
    };
  });

  return {
    ...toTableRequest({ ...response, data: items }),
    total: page.total,
  };
}
