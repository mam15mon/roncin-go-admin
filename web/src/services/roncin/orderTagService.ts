// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/order-tag-options */
export async function orderTagServiceListOrderTagOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderTagServiceListOrderTagOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListOrderTagOptionsResponse>("/api/v1/order-tag-options", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/order-tags/batch-assign */
export async function orderTagServiceBatchAssignOrderTags(
  body: API.BatchAssignOrderTagsRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchAssignOrderTagsResponse>(
    "/api/v1/order-tags/batch-assign",
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

/** 此处后端没有提供注释 POST /api/v1/order-tags/batch-remove */
export async function orderTagServiceBatchRemoveOrderTags(
  body: API.BatchRemoveOrderTagsRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchRemoveOrderTagsResponse>(
    "/api/v1/order-tags/batch-remove",
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
