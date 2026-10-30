// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

/** ListPersonnel 获取指定订单的协作人员列表。 GET /api/v1/orders/${param0}/personnel */
export async function orderPersonnelServiceListPersonnel(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderPersonnelServiceListPersonnelParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListPersonnelResponse>(
    `/api/v1/orders/${param0}/personnel`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}
