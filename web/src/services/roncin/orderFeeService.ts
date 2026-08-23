// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListFeeOptions 获取费用录入所需的结算单位和币种候选项。 GET /api/v1/orders/${param0}/fee-options */
export async function orderFeeServiceListFeeOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceListFeeOptionsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListFeeOptionsResponse>(
    `/api/v1/orders/${param0}/fee-options`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** ListFees 获取指定订单的费用列表。 GET /api/v1/orders/${param0}/fees */
export async function orderFeeServiceListFees(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceListFeesParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListFeesResponse>(`/api/v1/orders/${param0}/fees`, {
    method: "GET",
    params: { ...queryParams },
    ...(options || {}),
  });
}

/** AddFee 录入订单费用，总金额由服务端按数量乘单价精确计算。 POST /api/v1/orders/${param0}/fees */
export async function orderFeeServiceAddFee(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceAddFeeParams,
  body: API.AddFeeRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.AddFeeResponse>(`/api/v1/orders/${param0}/fees`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** UpdateFee 更新订单费用，总金额由服务端重新精确计算。 PUT /api/v1/orders/${param0}/fees/${param1} */
export async function orderFeeServiceUpdateFee(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceUpdateFeeParams,
  body: API.UpdateFeeRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.UpdateFeeResponse>(
    `/api/v1/orders/${param0}/fees/${param1}`,
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

/** RemoveFee 删除尚未进入后续财务流程的订单费用。 DELETE /api/v1/orders/${param0}/fees/${param1} */
export async function orderFeeServiceRemoveFee(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceRemoveFeeParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RemoveFeeResponse>(
    `/api/v1/orders/${param0}/fees/${param1}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}
