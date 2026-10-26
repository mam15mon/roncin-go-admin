import { OrderBusinessType, SeaDocumentStructure } from '@/enums.generated';
import type { SeaHouseOrderCreateSource } from './templates/types';

export function buildSeaHouseOrderCreatePath(orderId: string): string {
  return `/orders/sea-export/new?sourceOrderId=${encodeURIComponent(orderId)}`;
}

/** 草稿资源只取业务类型及明确支持的来源参数，不受无关查询参数影响。 */
export function resolveSeaHouseOrderCreateRequest(
  kind: string,
  search: string,
) {
  const values = new URLSearchParams(search).getAll('sourceOrderId');
  const pathname = `/orders/${kind}/new`;
  if (values.length === 0) return { draftPathname: pathname };
  if (kind !== 'sea-export' || values.length !== 1 || !values[0].trim()) {
    return {
      draftPathname: `${pathname}?${new URLSearchParams({ sourceOrderId: values.join(',') })}`,
      error: '来源订单参数无效，请从海运 HOUSE 订单的主单区域重新进入。',
    };
  }
  const sourceOrderId = values[0].trim();
  return {
    sourceOrderId,
    draftPathname: buildSeaHouseOrderCreatePath(sourceOrderId),
  };
}

/** 只提取共享身份；本票客户、分单、单证正文及货物不得从来源复制。 */
export function getSeaHouseOrderCreateSource(
  order: API.Order | undefined,
  sourceOrderId: string,
  organizationId: string,
): SeaHouseOrderCreateSource {
  if (!order?.id || order.id !== sourceOrderId) {
    throw new Error('来源订单读取失败，请重试。');
  }
  if (order.organizationId !== organizationId) {
    throw new Error('来源订单不属于当前公司，请在对应公司工作台操作。');
  }
  if (
    order.businessType !== OrderBusinessType.BUSINESS_TYPE_SE ||
    order.seaDocumentStructure !==
      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE
  ) {
    throw new Error('来源必须是海运出口 HOUSE 订单。');
  }
  const master = order.seaMasterBill;
  if (
    !master?.masterBillId ||
    !master.masterNo?.trim() ||
    !master.shippingLineId ||
    !master.transportExecutionId ||
    master.status === 'VOIDED' ||
    order.shippingLineId !== master.shippingLineId
  ) {
    throw new Error('来源订单缺少有效的主单或航程，请返回来源订单检查。');
  }
  return {
    orderId: order.id,
    orderNo: order.orderNo,
    masterBillId: master.masterBillId,
    masterNo: master.masterNo,
    shippingLineId: master.shippingLineId,
    shippingLineName: master.shippingLineName,
    transportExecutionId: master.transportExecutionId,
  };
}
