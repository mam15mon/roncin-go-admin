// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListMyCommissions 返回本人提成单与调整明细，服务端分页并支持状态/归属日期过滤。 GET /api/v1/workbench/my-commissions */
export async function workbenchServiceListMyCommissions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.WorkbenchServiceListMyCommissionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListMyCommissionsResponse>(
    "/api/v1/workbench/my-commissions",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** ListMyReceivables 返回与本人提成归属相关的已确认应收未结项，服务端分页。 GET /api/v1/workbench/my-receivables */
export async function workbenchServiceListMyReceivables(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.WorkbenchServiceListMyReceivablesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListMyReceivablesResponse>(
    "/api/v1/workbench/my-receivables",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** ListMyRecentOrders 返回本人真实协作的近期海运出口订单，服务端分页。 GET /api/v1/workbench/my-recent-orders */
export async function workbenchServiceListMyRecentOrders(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.WorkbenchServiceListMyRecentOrdersParams,
  options?: { [key: string]: any }
) {
  return request<API.ListMyRecentOrdersResponse>(
    "/api/v1/workbench/my-recent-orders",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** GetWorkbenchOverview 返回模块门禁、本人提成状态汇总、近期订单、可靠作业
 待办计数与当前用户有权查看的财务/审批摘要。 GET /api/v1/workbench/overview */
export async function workbenchServiceGetWorkbenchOverview(options?: {
  [key: string]: any;
}) {
  return request<API.GetWorkbenchOverviewResponse>(
    "/api/v1/workbench/overview",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}
