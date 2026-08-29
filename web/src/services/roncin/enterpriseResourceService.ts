// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 POST /api/v1/enterprise-resource-address-types/batch-assign */
export async function enterpriseResourceServiceBatchAssignAddressTypes(
  body: API.BatchAssignAddressTypesRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchAssignAddressTypesResponse>(
    "/api/v1/enterprise-resource-address-types/batch-assign",
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

/** 此处后端没有提供注释 POST /api/v1/enterprise-resource-address-types/batch-remove */
export async function enterpriseResourceServiceBatchRemoveAddressTypes(
  body: API.BatchRemoveAddressTypesRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchRemoveAddressTypesResponse>(
    "/api/v1/enterprise-resource-address-types/batch-remove",
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

/** 此处后端没有提供注释 POST /api/v1/enterprise-resource-assignees/batch-assign */
export async function enterpriseResourceServiceBatchAssignAssignees(
  body: API.BatchAssignAssigneesRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchAssignAssigneesResponse>(
    "/api/v1/enterprise-resource-assignees/batch-assign",
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

/** 此处后端没有提供注释 POST /api/v1/enterprise-resource-assignees/batch-remove */
export async function enterpriseResourceServiceBatchRemoveAssignees(
  body: API.BatchRemoveAssigneesRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchRemoveAssigneesResponse>(
    "/api/v1/enterprise-resource-assignees/batch-remove",
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

/** 此处后端没有提供注释 POST /api/v1/enterprise-resource-associations/batch-create */
export async function enterpriseResourceServiceBatchCreateAssociations(
  body: API.BatchCreateAssociationsRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchCreateAssociationsResponse>(
    "/api/v1/enterprise-resource-associations/batch-create",
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

/** 此处后端没有提供注释 POST /api/v1/enterprise-resource-associations/batch-delete */
export async function enterpriseResourceServiceBatchDeleteAssociations(
  body: API.BatchDeleteAssociationsRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchDeleteAssociationsResponse>(
    "/api/v1/enterprise-resource-associations/batch-delete",
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

/** 此处后端没有提供注释 GET /api/v1/enterprise-resource-capabilities */
export async function enterpriseResourceServiceGetEnterpriseResourceCapabilities(options?: {
  [key: string]: any;
}) {
  return request<API.GetEnterpriseResourceCapabilitiesResponse>(
    "/api/v1/enterprise-resource-capabilities",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/enterprise-resource-image-uploads */
export async function enterpriseResourceServicePrepareEnterpriseResourceImageUpload(
  body: API.PrepareEnterpriseResourceImageUploadRequest,
  options?: { [key: string]: any }
) {
  return request<API.PrepareEnterpriseResourceImageUploadResponse>(
    "/api/v1/enterprise-resource-image-uploads",
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

/** 此处后端没有提供注释 GET /api/v1/enterprise-resource-options/assignees */
export async function enterpriseResourceServiceSearchEnterpriseResourceAssigneeOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceSearchEnterpriseResourceAssigneeOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.SearchEnterpriseResourceAssigneeOptionsResponse>(
    "/api/v1/enterprise-resource-options/assignees",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/enterprise-resource-options/partners */
export async function enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceSearchEnterpriseResourcePartnerOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.SearchEnterpriseResourcePartnerOptionsResponse>(
    "/api/v1/enterprise-resource-options/partners",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/enterprise-resource-options/regions */
export async function enterpriseResourceServiceListEnterpriseResourceRegionOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceListEnterpriseResourceRegionOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListEnterpriseResourceRegionOptionsResponse>(
    "/api/v1/enterprise-resource-options/regions",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/enterprise-resources */
export async function enterpriseResourceServiceListEnterpriseResources(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceListEnterpriseResourcesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListEnterpriseResourcesResponse>(
    "/api/v1/enterprise-resources",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/enterprise-resources */
export async function enterpriseResourceServiceCreateEnterpriseResource(
  body: API.CreateEnterpriseResourceRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateEnterpriseResourceResponse>(
    "/api/v1/enterprise-resources",
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

/** 此处后端没有提供注释 GET /api/v1/enterprise-resources/${param0} */
export async function enterpriseResourceServiceGetEnterpriseResource(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceGetEnterpriseResourceParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetEnterpriseResourceResponse>(
    `/api/v1/enterprise-resources/${param0}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 PUT /api/v1/enterprise-resources/${param0} */
export async function enterpriseResourceServiceUpdateEnterpriseResource(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceUpdateEnterpriseResourceParams,
  body: API.UpdateEnterpriseResourceRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateEnterpriseResourceResponse>(
    `/api/v1/enterprise-resources/${param0}`,
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

/** 此处后端没有提供注释 DELETE /api/v1/enterprise-resources/${param0} */
export async function enterpriseResourceServiceDeleteEnterpriseResource(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceDeleteEnterpriseResourceParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.DeleteEnterpriseResourceResponse>(
    `/api/v1/enterprise-resources/${param0}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/enterprise-resources/${param0}/image-access */
export async function enterpriseResourceServiceGetEnterpriseResourceImageAccess(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceGetEnterpriseResourceImageAccessParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetEnterpriseResourceImageAccessResponse>(
    `/api/v1/enterprise-resources/${param0}/image-access`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/enterprise-resources/import-commit */
export async function enterpriseResourceServiceCommitEnterpriseResourceImport(
  body: API.CommitEnterpriseResourceImportRequest,
  options?: { [key: string]: any }
) {
  return request<API.CommitEnterpriseResourceImportResponse>(
    "/api/v1/enterprise-resources/import-commit",
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

/** 此处后端没有提供注释 POST /api/v1/enterprise-resources/import-preview */
export async function enterpriseResourceServicePreviewEnterpriseResourceImport(
  body: API.PreviewEnterpriseResourceImportRequest,
  options?: { [key: string]: any }
) {
  return request<API.PreviewEnterpriseResourceImportResponse>(
    "/api/v1/enterprise-resources/import-preview",
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

/** 此处后端没有提供注释 GET /api/v1/enterprise-tag-groups */
export async function enterpriseResourceServiceListEnterpriseTagGroups(options?: {
  [key: string]: any;
}) {
  return request<API.ListEnterpriseTagGroupsResponse>(
    "/api/v1/enterprise-tag-groups",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/enterprise-tag-groups */
export async function enterpriseResourceServiceCreateEnterpriseTagGroup(
  body: API.CreateEnterpriseTagGroupRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateEnterpriseTagGroupResponse>(
    "/api/v1/enterprise-tag-groups",
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

/** 此处后端没有提供注释 PUT /api/v1/enterprise-tag-groups/${param0} */
export async function enterpriseResourceServiceUpdateEnterpriseTagGroup(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceUpdateEnterpriseTagGroupParams,
  body: API.UpdateEnterpriseTagGroupRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateEnterpriseTagGroupResponse>(
    `/api/v1/enterprise-tag-groups/${param0}`,
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

/** 此处后端没有提供注释 DELETE /api/v1/enterprise-tag-groups/${param0} */
export async function enterpriseResourceServiceDeleteEnterpriseTagGroup(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.EnterpriseResourceServiceDeleteEnterpriseTagGroupParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.DeleteEnterpriseTagGroupResponse>(
    `/api/v1/enterprise-tag-groups/${param0}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}
