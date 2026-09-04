// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** LockOrder 锁定海运出口订单并固定单证不可变版本。 POST /api/v1/orders/${param0}/lock */
export async function orderLockServiceLockOrder(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderLockServiceLockOrderParams,
  body: API.LockOrderRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.LockOrderResponse>(`/api/v1/orders/${param0}/lock`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** GetOrderLockState 获取订单当前锁定状态与可执行动作。 GET /api/v1/orders/${param0}/lock-state */
export async function orderLockServiceGetOrderLockState(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderLockServiceGetOrderLockStateParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.GetOrderLockStateResponse>(
    `/api/v1/orders/${param0}/lock-state`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** RequestOrderUnlock 请求或直接解锁订单（根据调用人角色分流）。 POST /api/v1/orders/${param0}/unlock */
export async function orderLockServiceRequestOrderUnlock(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderLockServiceRequestOrderUnlockParams,
  body: API.RequestOrderUnlockRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.RequestOrderUnlockResponse>(
    `/api/v1/orders/${param0}/unlock`,
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

/** ListOrderUnlockRequests 查询订单解锁请求历史。 GET /api/v1/orders/${param0}/unlock-requests */
export async function orderLockServiceListOrderUnlockRequests(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderLockServiceListOrderUnlockRequestsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListOrderUnlockRequestsResponse>(
    `/api/v1/orders/${param0}/unlock-requests`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** GetOrderUnlockRequest 获取单个解锁请求详情。 GET /api/v1/orders/${param0}/unlock-requests/${param1} */
export async function orderLockServiceGetOrderUnlockRequest(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderLockServiceGetOrderUnlockRequestParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, requestId: param1, ...queryParams } = params;
  return request<API.GetOrderUnlockRequestResponse>(
    `/api/v1/orders/${param0}/unlock-requests/${param1}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}
