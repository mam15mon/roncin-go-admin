import type { OrderListItem } from '@/components/ui';
import type { OrderListAcceptedSnapshot } from '@/components/ui/order-list-template/types';
import { orderServiceExportOrders } from '@/services/roncin/orderService';
import { type CsvColumn, csvAmount, csvDecimal } from '@/utils/csv';
import { formatDate } from '@/utils/format';
import {
  buildOrderListRequestFilters,
  mapOrderListItems,
  type OrderListQueryContext,
} from './list-query';
import type { OrderKindDefinition } from './order-kinds/types';

export async function fetchOrderListExport(
  snapshot: OrderListAcceptedSnapshot,
  definition: OrderKindDefinition,
  context: OrderListQueryContext,
  isCurrent: () => boolean = () => true,
) {
  if (!context.organizationId) throw new Error('当前公司尚未就绪。');
  const response = await orderServiceExportOrders(
    buildOrderListRequestFilters(snapshot.filters, definition),
  );
  if (!isCurrent()) throw new Error('查询身份已经变化，已取消本次导出。');
  if (response?.success !== true) throw new Error('订单完整导出失败，请重试。');
  const orders = response.data ?? [];
  const total = response.total ?? 0;
  if (!Number.isSafeInteger(total) || total < 0 || total !== orders.length)
    throw new Error('订单导出结果不完整，请重试。');
  if (!response.generatedAt || formatDate(response.generatedAt) === '-')
    throw new Error('订单导出缺少有效生成时间。');
  if (
    orders.some(
      (order) =>
        !order.id ||
        !order.orderNo ||
        order.organizationId !== context.organizationId ||
        order.businessType !== definition.businessType,
    )
  )
    throw new Error('订单导出响应与当前公司或业务类型不一致。');
  return {
    rows: await mapOrderListItems(orders, definition, context),
    generatedAt: response.generatedAt,
  };
}

const dateValue = (value: unknown, dateOnly = false) =>
  typeof value === 'string' && value
    ? formatDate(value, dateOnly ? 'date' : 'minute')
    : '';

/** 固定业务列映射独立于React表格渲染；受保护子资源按真实权限增列。 */
export function orderExportColumns(access: {
  containers: boolean;
  attachments: boolean;
}): CsvColumn<OrderListItem>[] {
  return [
    { header: '订单编号', value: (row) => row.orderNo },
    { header: '业务类型', value: (row) => row.businessType },
    { header: '进程', value: (row) => row.stage?.join('；') },
    { header: '操作进度', value: (row) => row.statusName },
    { header: '委托单位', value: (row) => row.customerName },
    { header: '客户业务号', value: (row) => row.customerReferenceNo },
    { header: '企业内部编号', value: (row) => row.internalReferenceNo },
    { header: '主单号', value: (row) => row.masterBlNo },
    { header: '分单号', value: (row) => row.houseBlNo },
    {
      header: '主分单模式',
      value: (row) =>
        row.documentMode === 'HOUSE'
          ? '主分单'
          : row.documentMode === 'DIRECT'
            ? '直单'
            : '',
    },
    { header: '航司/船公司', value: (row) => row.shippingLineName },
    { header: '航班/船名航次', value: (row) => row.vesselVoyage },
    { header: '起运地', value: (row) => row.originPortName },
    { header: '目的地', value: (row) => row.destinationPortName },
    { header: '共同段卸货港', value: (row) => row.dischargePortName },
    { header: '中转港', value: (row) => row.transitPortName },
    { header: 'ETD', value: (row) => dateValue(row.etd) },
    { header: 'ETA', value: (row) => dateValue(row.eta) },
    { header: '业务日期', value: (row) => dateValue(row.orderDate, true) },
    { header: '订舱编号', value: (row) => row.bookingNo },
    { header: '订舱代理', value: (row) => row.bookingAgentName },
    { header: '国外代理', value: (row) => row.foreignAgentName },
    { header: '船代', value: (row) => row.shippingAgentName },
    { header: '接单时间', value: (row) => dateValue(row.receivedAt) },
    { header: '货好时间', value: (row) => dateValue(row.cargoReadyAt) },
    {
      header: '件数',
      kind: 'number',
      value: (row) => csvDecimal(row.totalPackages),
    },
    { header: '包装单位', value: (row) => row.packageUnit },
    {
      header: '毛重(KG)',
      kind: 'number',
      value: (row) => csvDecimal(row.grossWeightKg),
    },
    {
      header: '体积(m³)',
      kind: 'number',
      value: (row) => csvDecimal(row.volumeCbm),
    },
    { header: '箱量计划', value: (row) => row.containerSummary },
    ...(access.containers
      ? [
          {
            header: '实际箱号',
            value: (row: OrderListItem) => row.containerNos?.join('；'),
          },
        ]
      : []),
    { header: '业务付款方式', value: (row) => row.paymentTerm },
    { header: '贸易条款', value: (row) => row.tradeTerm },
    { header: '合约号', value: (row) => row.contractNo },
    { header: '发货人简称', value: (row) => row.shipperName },
    { header: '收货人简称', value: (row) => row.consigneeName },
    { header: '工厂', value: (row) => row.factoryName },
    { header: '品名', value: (row) => row.goodsDescription },
    {
      header: '货值',
      kind: 'number',
      value: (row) => csvAmount(row.cargoValue),
    },
    { header: '货值币种', value: (row) => row.cargoCurrency },
    { header: 'UN NO.', value: (row) => row.unNumber },
    { header: '危险品类别', value: (row) => row.hazardClass },
    { header: '操作人员', value: (row) => row.operatorName },
    { header: '业务人员', value: (row) => row.salesName },
    { header: '客服人员', value: (row) => row.customerServiceName },
    { header: '创建人员', value: (row) => row.creatorName },
    ...(access.attachments
      ? [
          {
            header: '附件数',
            kind: 'number' as const,
            value: (row: OrderListItem) => csvDecimal(row.attachmentCount),
          },
        ]
      : []),
    {
      header: '锁定状态',
      value: (row) => (row.isLocked ? '已锁定' : '未锁定'),
    },
    { header: '锁定时间', value: (row) => dateValue(row.lockedAt) },
    { header: '创建时间', value: (row) => dateValue(row.createdAt) },
    { header: '订舱备注', value: (row) => row.bookingNotes },
    { header: '配舱备注', value: (row) => row.spaceNotes },
    { header: '操作备注', value: (row) => row.operationNotes },
    {
      header: '备注',
      value: (row) => (typeof row.notes === 'string' ? row.notes : ''),
    },
  ];
}

export function orderExportFileName(title: string, generatedAt: string) {
  return `${title}_${formatDate(generatedAt).replaceAll(':', '-')}.csv`;
}
