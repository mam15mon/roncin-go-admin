// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListReleasePods 获取指定订单的放货凭证列表。 GET /api/v1/orders/${param0}/release-pods */
export async function orderReleasePodServiceListReleasePods(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderReleasePodServiceListReleasePodsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListReleasePodsResponse>(
    `/api/v1/orders/${param0}/release-pods`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** AddReleasePod 添加放货凭证，初始状态为待签收。 POST /api/v1/orders/${param0}/release-pods */
export async function orderReleasePodServiceAddReleasePod(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderReleasePodServiceAddReleasePodParams,
  body: API.AddReleasePodRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.AddReleasePodResponse>(
    `/api/v1/orders/${param0}/release-pods`,
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

/** UpdateReleasePod 更新凭证字段，已回单后不可修改。 PUT /api/v1/orders/${param0}/release-pods/${param1} */
export async function orderReleasePodServiceUpdateReleasePod(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderReleasePodServiceUpdateReleasePodParams,
  body: API.UpdateReleasePodRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.UpdateReleasePodResponse>(
    `/api/v1/orders/${param0}/release-pods/${param1}`,
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

/** RemoveReleasePod 移除放货凭证，已回单的凭证不可删除。 DELETE /api/v1/orders/${param0}/release-pods/${param1} */
export async function orderReleasePodServiceRemoveReleasePod(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderReleasePodServiceRemoveReleasePodParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RemoveReleasePodResponse>(
    `/api/v1/orders/${param0}/release-pods/${param1}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** TransitionReleasePodStatus 流转凭证状态，仅允许向前单步流转；
 流转到已签收时由服务端记录签收人与签收时间。 POST /api/v1/orders/${param0}/release-pods/${param1}/transition */
export async function orderReleasePodServiceTransitionReleasePodStatus(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderReleasePodServiceTransitionReleasePodStatusParams,
  body: API.TransitionReleasePodStatusRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.TransitionReleasePodStatusResponse>(
    `/api/v1/orders/${param0}/release-pods/${param1}/transition`,
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
