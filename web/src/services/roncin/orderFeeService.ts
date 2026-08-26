// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ResolveFeeExchangeRate 按汇率（折本币）的时间标准、币种和收付方向预览汇率。 GET /api/v1/orders/${param0}/fee-exchange-rate */
export async function orderFeeServiceResolveFeeExchangeRate(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceResolveFeeExchangeRateParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ResolveFeeExchangeRateResponse>(
    `/api/v1/orders/${param0}/fee-exchange-rate`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** ListFeeOptions 获取费用录入所需的费用设置、计费单位、结算单位和币种候选项。 GET /api/v1/orders/${param0}/fee-options */
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

/** RemoveFee 作废尚未进入账单的订单费用，并保留完整历史数据。 DELETE /api/v1/orders/${param0}/fees/${param1} */
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
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** ConfirmFee 确认费用；确认后方可进入账单，修改前必须先撤回确认。 POST /api/v1/orders/${param0}/fees/${param1}/confirm */
export async function orderFeeServiceConfirmFee(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceConfirmFeeParams,
  body: API.ConfirmFeeRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.ConfirmFeeResponse>(
    `/api/v1/orders/${param0}/fees/${param1}/confirm`,
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

/** ReopenFee 撤回尚未进入账单的已确认费用，使其重新可编辑。 POST /api/v1/orders/${param0}/fees/${param1}/reopen */
export async function orderFeeServiceReopenFee(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceReopenFeeParams,
  body: API.ReopenFeeRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.ReopenFeeResponse>(
    `/api/v1/orders/${param0}/fees/${param1}/reopen`,
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
