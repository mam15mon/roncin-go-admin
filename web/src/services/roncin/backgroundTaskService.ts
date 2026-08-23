// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** ListBackgroundTasks 查询当前组织的后台任务。 GET /api/v1/background-tasks */
export async function backgroundTaskServiceListBackgroundTasks(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.BackgroundTaskServiceListBackgroundTasksParams,
  options?: { [key: string]: any }
) {
  return request<API.ListBackgroundTasksResponse>("/api/v1/background-tasks", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** GetBackgroundTask 查询单个后台任务。 GET /api/v1/background-tasks/${param0} */
export async function backgroundTaskServiceGetBackgroundTask(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.BackgroundTaskServiceGetBackgroundTaskParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetBackgroundTaskResponse>(
    `/api/v1/background-tasks/${param0}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** RequeueBackgroundTask 回放失败或死信任务：重置为待执行并清空租约。 POST /api/v1/background-tasks/${param0}/requeue */
export async function backgroundTaskServiceRequeueBackgroundTask(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.BackgroundTaskServiceRequeueBackgroundTaskParams,
  body: API.RequeueBackgroundTaskRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.RequeueBackgroundTaskResponse>(
    `/api/v1/background-tasks/${param0}/requeue`,
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
