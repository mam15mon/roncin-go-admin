// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListAbnormalCases 获取指定订单的异常标记列表。 GET /api/v1/orders/${param0}/abnormal-cases */
export async function orderAbnormalCaseServiceListAbnormalCases(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAbnormalCaseServiceListAbnormalCasesParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.OrderAbnormalCaseListReply>(
    `/api/v1/orders/${param0}/abnormal-cases`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** MarkAbnormalCase 标记订单异常；同类型已解决的标记会重新激活，进行中的重复标记返回冲突。 POST /api/v1/orders/${param0}/abnormal-cases */
export async function orderAbnormalCaseServiceMarkAbnormalCase(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAbnormalCaseServiceMarkAbnormalCaseParams,
  body: API.MarkAbnormalCaseRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.OrderAbnormalCaseReply>(
    `/api/v1/orders/${param0}/abnormal-cases`,
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

/** RemoveAbnormalCase 移除订单异常标记。 DELETE /api/v1/orders/${param0}/abnormal-cases/${param1} */
export async function orderAbnormalCaseServiceRemoveAbnormalCase(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAbnormalCaseServiceRemoveAbnormalCaseParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.OrderAbnormalCaseOperationReply>(
    `/api/v1/orders/${param0}/abnormal-cases/${param1}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** ResolveAbnormalCase 解决订单异常，仅进行中的异常可解决。 POST /api/v1/orders/${param0}/abnormal-cases/${param1}/resolve */
export async function orderAbnormalCaseServiceResolveAbnormalCase(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAbnormalCaseServiceResolveAbnormalCaseParams,
  body: API.ResolveAbnormalCaseRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.OrderAbnormalCaseReply>(
    `/api/v1/orders/${param0}/abnormal-cases/${param1}/resolve`,
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
