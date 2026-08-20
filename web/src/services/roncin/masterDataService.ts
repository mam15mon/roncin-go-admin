// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/master-data/items */
export async function masterDataServiceListItems(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListItemsParams,
  options?: { [key: string]: any }
) {
  return request<API.MasterDataItemListReply>("/api/v1/master-data/items", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/master-data/items */
export async function masterDataServiceCreateItem(
  body: API.CreateMasterDataItemRequest,
  options?: { [key: string]: any }
) {
  return request<API.MasterDataItemReply>("/api/v1/master-data/items", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/master-data/items/${param0} */
export async function masterDataServiceUpdateItem(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceUpdateItemParams,
  body: API.UpdateMasterDataItemRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.MasterDataItemReply>(
    `/api/v1/master-data/items/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/master-data/options */
export async function masterDataServiceListOptions(options?: {
  [key: string]: any;
}) {
  return request<API.MasterDataOptionsReply>("/api/v1/master-data/options", {
    method: "GET",
    ...(options || {}),
  });
}
