// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListShippingDocuments 获取指定订单的提单列表。 GET /api/v1/orders/${param0}/shipping-documents */
export async function orderShippingDocumentServiceListShippingDocuments(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderShippingDocumentServiceListShippingDocumentsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.OrderShippingDocumentListReply>(
    `/api/v1/orders/${param0}/shipping-documents`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** AddShippingDocument 添加订单提单，初始状态为草稿。 POST /api/v1/orders/${param0}/shipping-documents */
export async function orderShippingDocumentServiceAddShippingDocument(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderShippingDocumentServiceAddShippingDocumentParams,
  body: API.AddShippingDocumentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.OrderShippingDocumentReply>(
    `/api/v1/orders/${param0}/shipping-documents`,
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

/** UpdateShippingDocument 更新提单字段，草稿与已确认状态可改，已放货后不可改。 PUT /api/v1/orders/${param0}/shipping-documents/${param1} */
export async function orderShippingDocumentServiceUpdateShippingDocument(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderShippingDocumentServiceUpdateShippingDocumentParams,
  body: API.UpdateShippingDocumentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.OrderShippingDocumentReply>(
    `/api/v1/orders/${param0}/shipping-documents/${param1}`,
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

/** RemoveShippingDocument 移除提单，已放货的提单不可删除。 DELETE /api/v1/orders/${param0}/shipping-documents/${param1} */
export async function orderShippingDocumentServiceRemoveShippingDocument(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderShippingDocumentServiceRemoveShippingDocumentParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.OrderShippingDocumentOperationReply>(
    `/api/v1/orders/${param0}/shipping-documents/${param1}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** TransitionShippingDocumentStatus 流转提单状态，仅允许向前单步流转。 POST /api/v1/orders/${param0}/shipping-documents/${param1}/transition */
export async function orderShippingDocumentServiceTransitionShippingDocumentStatus(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderShippingDocumentServiceTransitionShippingDocumentStatusParams,
  body: API.TransitionShippingDocumentStatusRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.OrderShippingDocumentReply>(
    `/api/v1/orders/${param0}/shipping-documents/${param1}/transition`,
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
