// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/orders/sea-shared-container-candidates */
export async function seaSharedContainerServiceListSeaSharedContainerCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaSharedContainerServiceListSeaSharedContainerCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListSeaSharedContainerCandidatesResponse>(
    "/api/v1/orders/sea-shared-container-candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/orders/sea-shared-containers */
export async function seaSharedContainerServiceListSeaSharedContainers(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaSharedContainerServiceListSeaSharedContainersParams,
  options?: { [key: string]: any }
) {
  return request<API.ListSeaSharedContainersResponse>(
    "/api/v1/orders/sea-shared-containers",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/orders/sea-shared-containers */
export async function seaSharedContainerServiceCreateSeaSharedContainer(
  body: API.CreateSeaSharedContainerRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateSeaSharedContainerResponse>(
    "/api/v1/orders/sea-shared-containers",
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

/** 此处后端没有提供注释 GET /api/v1/orders/sea-shared-containers/${param0} */
export async function seaSharedContainerServiceGetSeaSharedContainer(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaSharedContainerServiceGetSeaSharedContainerParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetSeaSharedContainerResponse>(
    `/api/v1/orders/sea-shared-containers/${param0}`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 PUT /api/v1/orders/sea-shared-containers/${param0} */
export async function seaSharedContainerServiceUpdateSeaSharedContainer(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaSharedContainerServiceUpdateSeaSharedContainerParams,
  body: API.UpdateSeaSharedContainerRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateSeaSharedContainerResponse>(
    `/api/v1/orders/sea-shared-containers/${param0}`,
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

/** 此处后端没有提供注释 DELETE /api/v1/orders/sea-shared-containers/${param0} */
export async function seaSharedContainerServiceDeleteSeaSharedContainer(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaSharedContainerServiceDeleteSeaSharedContainerParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.DeleteSeaSharedContainerResponse>(
    `/api/v1/orders/sea-shared-containers/${param0}`,
    {
      method: "DELETE",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 PUT /api/v1/orders/sea-shared-containers/${param0}/allocations/draft */
export async function seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaSharedContainerServiceSaveSeaSharedContainerAllocationsDraftParams,
  body: API.SaveSeaSharedContainerAllocationsDraftRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.SaveSeaSharedContainerAllocationsDraftResponse>(
    `/api/v1/orders/sea-shared-containers/${param0}/allocations/draft`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/sea-shared-containers/${param0}/confirm */
export async function seaSharedContainerServiceConfirmSeaSharedContainer(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaSharedContainerServiceConfirmSeaSharedContainerParams,
  body: API.ConfirmSeaSharedContainerRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ConfirmSeaSharedContainerResponse>(
    `/api/v1/orders/sea-shared-containers/${param0}/confirm`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/sea-shared-containers/${param0}/withdraw */
export async function seaSharedContainerServiceWithdrawSeaSharedContainer(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaSharedContainerServiceWithdrawSeaSharedContainerParams,
  body: API.WithdrawSeaSharedContainerRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.WithdrawSeaSharedContainerResponse>(
    `/api/v1/orders/sea-shared-containers/${param0}/withdraw`,
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
