// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** GetSeaOrderChangeActions 获取订单可执行动作及阻断原因。 GET /api/v1/orders/${param0}/sea-order-change/actions */
export async function seaOrderChangeServiceGetSeaOrderChangeActions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaOrderChangeServiceGetSeaOrderChangeActionsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.GetSeaOrderChangeActionsResponse>(
    `/api/v1/orders/${param0}/sea-order-change/actions`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** ListSeaOrderChangeEvents 查询订单拆票与改配事件历史。 GET /api/v1/orders/${param0}/sea-order-change/events */
export async function seaOrderChangeServiceListSeaOrderChangeEvents(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaOrderChangeServiceListSeaOrderChangeEventsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListSeaOrderChangeEventsResponse>(
    `/api/v1/orders/${param0}/sea-order-change/events`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** GetSeaOrderChangeEvent 获取单个拆票或改配事件详情。 GET /api/v1/orders/${param0}/sea-order-change/events/${param1} */
export async function seaOrderChangeServiceGetSeaOrderChangeEvent(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaOrderChangeServiceGetSeaOrderChangeEventParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, eventId: param1, ...queryParams } = params;
  return request<API.GetSeaOrderChangeEventResponse>(
    `/api/v1/orders/${param0}/sea-order-change/events/${param1}`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** ExecuteSeaOrderReassignment 执行整体改配。 POST /api/v1/orders/${param0}/sea-order-change/reassignment/execute */
export async function seaOrderChangeServiceExecuteSeaOrderReassignment(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaOrderChangeServiceExecuteSeaOrderReassignmentParams,
  body: API.ExecuteSeaOrderReassignmentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ExecuteSeaOrderReassignmentResponse>(
    `/api/v1/orders/${param0}/sea-order-change/reassignment/execute`,
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

/** PreviewSeaOrderReassignment 预览整体改配航程差异。 POST /api/v1/orders/${param0}/sea-order-change/reassignment/preview */
export async function seaOrderChangeServicePreviewSeaOrderReassignment(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaOrderChangeServicePreviewSeaOrderReassignmentParams,
  body: API.PreviewSeaOrderReassignmentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.PreviewSeaOrderReassignmentResponse>(
    `/api/v1/orders/${param0}/sea-order-change/reassignment/preview`,
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

/** GetSeaOrderSplitContext 获取拆票上下文事实与可分配数据。 GET /api/v1/orders/${param0}/sea-order-change/split-context */
export async function seaOrderChangeServiceGetSeaOrderSplitContext(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaOrderChangeServiceGetSeaOrderSplitContextParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.GetSeaOrderSplitContextResponse>(
    `/api/v1/orders/${param0}/sea-order-change/split-context`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** ExecuteSeaOrderSplit 执行部分拆票。 POST /api/v1/orders/${param0}/sea-order-change/split/execute */
export async function seaOrderChangeServiceExecuteSeaOrderSplit(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaOrderChangeServiceExecuteSeaOrderSplitParams,
  body: API.ExecuteSeaOrderSplitRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ExecuteSeaOrderSplitResponse>(
    `/api/v1/orders/${param0}/sea-order-change/split/execute`,
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

/** PreviewSeaOrderSplit 预览拆票结果、守恒计算与计划重算。 POST /api/v1/orders/${param0}/sea-order-change/split/preview */
export async function seaOrderChangeServicePreviewSeaOrderSplit(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaOrderChangeServicePreviewSeaOrderSplitParams,
  body: API.PreviewSeaOrderSplitRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.PreviewSeaOrderSplitResponse>(
    `/api/v1/orders/${param0}/sea-order-change/split/preview`,
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
