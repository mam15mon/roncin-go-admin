// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListPersonnel 获取指定订单的协作人员列表。 GET /api/v1/orders/${param0}/personnel */
export async function orderPersonnelServiceListPersonnel(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderPersonnelServiceListPersonnelParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.OrderPersonnelListReply>(
    `/api/v1/orders/${param0}/personnel`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** AssignPersonnel 分配订单协作人员。 POST /api/v1/orders/${param0}/personnel */
export async function orderPersonnelServiceAssignPersonnel(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderPersonnelServiceAssignPersonnelParams,
  body: API.AssignPersonnelRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.OrderPersonnelReply>(
    `/api/v1/orders/${param0}/personnel`,
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

/** RemovePersonnel 移除订单协作人员。 DELETE /api/v1/orders/${param0}/personnel/${param1} */
export async function orderPersonnelServiceRemovePersonnel(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderPersonnelServiceRemovePersonnelParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.OrderPersonnelOperationReply>(
    `/api/v1/orders/${param0}/personnel/${param1}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}
