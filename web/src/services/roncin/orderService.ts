// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/order-reference-check */
export async function orderServiceCheckOrderReference(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderServiceCheckOrderReferenceParams,
  options?: { [key: string]: any }
) {
  return request<API.CheckOrderReferenceResponse>(
    "/api/v1/order-reference-check",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/orders */
export async function orderServiceListOrders(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderServiceListOrdersParams,
  options?: { [key: string]: any }
) {
  return request<API.ListOrdersResponse>("/api/v1/orders", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/orders */
export async function orderServiceCreateOrder(
  body: API.CreateOrderRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateOrderResponse>("/api/v1/orders", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/orders/${param0} */
export async function orderServiceGetOrder(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderServiceGetOrderParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetOrderResponse>(`/api/v1/orders/${param0}`, {
    method: "GET",
    params: { ...queryParams },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/orders/${param0} */
export async function orderServiceUpdateOrder(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderServiceUpdateOrderParams,
  body: API.UpdateOrderRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateOrderResponse>(`/api/v1/orders/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/status */
export async function orderServiceTransitionOrderStatus(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderServiceTransitionOrderStatusParams,
  body: API.TransitionOrderStatusRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.TransitionOrderStatusResponse>(
    `/api/v1/orders/${param0}/status`,
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
