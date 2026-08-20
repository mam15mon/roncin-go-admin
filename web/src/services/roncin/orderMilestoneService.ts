// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListMilestones 获取指定订单的里程碑列表。 GET /api/v1/orders/${param0}/milestones */
export async function orderMilestoneServiceListMilestones(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderMilestoneServiceListMilestonesParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.OrderMilestoneListReply>(
    `/api/v1/orders/${param0}/milestones`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** SetMilestone 设置指定订单的单个里程碑。 PUT /api/v1/orders/${param0}/milestones/${param1} */
export async function orderMilestoneServiceSetMilestone(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderMilestoneServiceSetMilestoneParams,
  body: API.SetMilestoneRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, type: param1, ...queryParams } = params;
  return request<API.OrderMilestoneReply>(
    `/api/v1/orders/${param0}/milestones/${param1}`,
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
