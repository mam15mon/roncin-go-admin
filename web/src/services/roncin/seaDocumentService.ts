// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** GetSeaOrderDocuments 获取海运单证聚合信息。 GET /api/v1/orders/${param0}/sea-documents */
export async function seaDocumentServiceGetSeaOrderDocuments(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceGetSeaOrderDocumentsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.GetSeaOrderDocumentsResponse>(
    `/api/v1/orders/${param0}/sea-documents`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** CancelSeaOrderDirect 取消直单标记，回到未确定状态。 POST /api/v1/orders/${param0}/sea-documents/cancel-direct */
export async function seaDocumentServiceCancelSeaOrderDirect(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceCancelSeaOrderDirectParams,
  body: API.CancelSeaOrderDirectRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.CancelSeaOrderDirectResponse>(
    `/api/v1/orders/${param0}/sea-documents/cancel-direct`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** AddSeaHouseBill 添加海运分单（HBL）。 POST /api/v1/orders/${param0}/sea-documents/house-bills */
export async function seaDocumentServiceAddSeaHouseBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceAddSeaHouseBillParams,
  body: API.AddSeaHouseBillRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.AddSeaHouseBillResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bills`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** UpdateSeaHouseBill 更新海运分单。 PUT /api/v1/orders/${param0}/sea-documents/house-bills/${param1} */
export async function seaDocumentServiceUpdateSeaHouseBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceUpdateSeaHouseBillParams,
  body: API.UpdateSeaHouseBillRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.UpdateSeaHouseBillResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bills/${param1}`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** RemoveSeaHouseBill 移除海运分单。 DELETE /api/v1/orders/${param0}/sea-documents/house-bills/${param1} */
export async function seaDocumentServiceRemoveSeaHouseBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceRemoveSeaHouseBillParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RemoveSeaHouseBillResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bills/${param1}`,
    {
      method: "DELETE",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** MarkSeaOrderDirect 明确标记海运订单为直单。 POST /api/v1/orders/${param0}/sea-documents/mark-direct */
export async function seaDocumentServiceMarkSeaOrderDirect(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceMarkSeaOrderDirectParams,
  body: API.MarkSeaOrderDirectRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.MarkSeaOrderDirectResponse>(
    `/api/v1/orders/${param0}/sea-documents/mark-direct`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}

/** UpdateSeaMasterBillContent 更新共享 MBL 提单内容。 PUT /api/v1/orders/${param0}/sea-documents/master-bill-content */
export async function seaDocumentServiceUpdateSeaMasterBillContent(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceUpdateSeaMasterBillContentParams,
  body: API.UpdateSeaMasterBillContentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.UpdateSeaMasterBillContentResponse>(
    `/api/v1/orders/${param0}/sea-documents/master-bill-content`,
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      params: { ...queryParams },
      data: body,
      ...(options || {}),
    }
  );
}
