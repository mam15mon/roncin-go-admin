// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListContainers 获取指定订单的集装箱列表。 GET /api/v1/orders/${param0}/containers */
export async function orderContainerServiceListContainers(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderContainerServiceListContainersParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListContainersResponse>(
    `/api/v1/orders/${param0}/containers`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** AddContainer 添加订单集装箱。 POST /api/v1/orders/${param0}/containers */
export async function orderContainerServiceAddContainer(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderContainerServiceAddContainerParams,
  body: API.AddContainerRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.AddContainerResponse>(
    `/api/v1/orders/${param0}/containers`,
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

/** UpdateContainer 更新订单集装箱，采用全量字段替换语义。 PUT /api/v1/orders/${param0}/containers/${param1} */
export async function orderContainerServiceUpdateContainer(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderContainerServiceUpdateContainerParams,
  body: API.UpdateContainerRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.UpdateContainerResponse>(
    `/api/v1/orders/${param0}/containers/${param1}`,
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

/** RemoveContainer 移除订单集装箱。 DELETE /api/v1/orders/${param0}/containers/${param1} */
export async function orderContainerServiceRemoveContainer(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderContainerServiceRemoveContainerParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RemoveContainerResponse>(
    `/api/v1/orders/${param0}/containers/${param1}`,
    {
      method: "DELETE",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}
