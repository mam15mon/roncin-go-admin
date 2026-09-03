// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListCargoItems 获取指定订单的货物明细列表。 GET /api/v1/orders/${param0}/cargo-items */
export async function orderCargoItemServiceListCargoItems(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderCargoItemServiceListCargoItemsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListCargoItemsResponse>(
    `/api/v1/orders/${param0}/cargo-items`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** AddCargoItem 添加订单货物明细。 POST /api/v1/orders/${param0}/cargo-items */
export async function orderCargoItemServiceAddCargoItem(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderCargoItemServiceAddCargoItemParams,
  body: API.AddCargoItemRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.AddCargoItemResponse>(
    `/api/v1/orders/${param0}/cargo-items`,
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

/** UpdateCargoItem 更新订单货物明细，采用全量字段替换语义。 PUT /api/v1/orders/${param0}/cargo-items/${param1} */
export async function orderCargoItemServiceUpdateCargoItem(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderCargoItemServiceUpdateCargoItemParams,
  body: API.UpdateCargoItemRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.UpdateCargoItemResponse>(
    `/api/v1/orders/${param0}/cargo-items/${param1}`,
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

/** RemoveCargoItem 移除订单货物明细。 DELETE /api/v1/orders/${param0}/cargo-items/${param1} */
export async function orderCargoItemServiceRemoveCargoItem(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderCargoItemServiceRemoveCargoItemParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RemoveCargoItemResponse>(
    `/api/v1/orders/${param0}/cargo-items/${param1}`,
    {
      method: "DELETE",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}
