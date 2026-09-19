// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListMyApplicationCandidates 返回本人截至上一自然月末、尚未进入任何申请的
 合格提成候选，按提成归属月过滤并服务端分页；候选由服务端按现有计提口径
 全量解析，不信任客户端传入的员工/组织/金额。 GET /api/v1/workbench/application-candidates */
export async function workbenchServiceListMyApplicationCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.WorkbenchServiceListMyApplicationCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListMyApplicationCandidatesResponse>(
    "/api/v1/workbench/application-candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** ListMyCommissionApplications 返回本人月度提成申请历史，服务端分页并支持状态过滤。 GET /api/v1/workbench/commission-applications */
export async function workbenchServiceListMyCommissionApplications(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.WorkbenchServiceListMyCommissionApplicationsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListMyCommissionApplicationsResponse>(
    "/api/v1/workbench/commission-applications",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** GetMyCommissionApplication 返回本人单张申请详情：申请头、提交/决策版本审计
 与明细快照；查询他人申请稳定返回不存在。 GET /api/v1/workbench/commission-applications/${param0} */
export async function workbenchServiceGetMyCommissionApplication(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.WorkbenchServiceGetMyCommissionApplicationParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetMyCommissionApplicationResponse>(
    `/api/v1/workbench/commission-applications/${param0}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** SubmitMyCommissionApplication 提交本人月度提成申请：无业务参数，服务端以
 当前会话组织与本人身份全量重算候选并固化申请头与明细快照；当前自然月、
 空候选与已进入有效申请的提成将被拒绝。 POST /api/v1/workbench/commission-applications/submit */
export async function workbenchServiceSubmitMyCommissionApplication(
  body: API.SubmitMyCommissionApplicationRequest,
  options?: { [key: string]: any }
) {
  return request<API.SubmitMyCommissionApplicationResponse>(
    "/api/v1/workbench/commission-applications/submit",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      data: body,
      ...(options || {}),
    }
  );
}

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
