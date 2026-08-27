// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/master-data/airlines */
export async function masterDataServiceListAirlines(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListAirlinesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListAirlinesResponse>("/api/v1/master-data/airlines", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/master-data/airlines */
export async function masterDataServiceCreateAirline(
  body: API.CreateAirlineRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateAirlineResponse>("/api/v1/master-data/airlines", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/master-data/airlines/${param0} */
export async function masterDataServiceUpdateAirline(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceUpdateAirlineParams,
  body: API.UpdateAirlineRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateAirlineResponse>(
    `/api/v1/master-data/airlines/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/master-data/airports */
export async function masterDataServiceListAirports(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListAirportsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListAirportsResponse>("/api/v1/master-data/airports", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/master-data/airports */
export async function masterDataServiceCreateAirport(
  body: API.CreateAirportRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateAirportResponse>("/api/v1/master-data/airports", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/master-data/airports/${param0} */
export async function masterDataServiceUpdateAirport(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceUpdateAirportParams,
  body: API.UpdateAirportRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateAirportResponse>(
    `/api/v1/master-data/airports/${param0}`,
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

/** 此处后端没有提供注释 POST /api/v1/master-data/import */
export async function masterDataServiceImportItems(
  body: API.ImportItemsRequest,
  options?: { [key: string]: any }
) {
  return request<API.ImportItemsResponse>("/api/v1/master-data/import", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/master-data/items */
export async function masterDataServiceListItems(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListItemsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListItemsResponse>("/api/v1/master-data/items", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/master-data/items */
export async function masterDataServiceCreateItem(
  body: API.CreateItemRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateItemResponse>("/api/v1/master-data/items", {
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
  body: API.UpdateItemRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateItemResponse>(
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

/** 此处后端没有提供注释 GET /api/v1/master-data/milestone-templates */
export async function masterDataServiceListMilestoneTemplates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListMilestoneTemplatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListMilestoneTemplatesResponse>(
    "/api/v1/master-data/milestone-templates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/master-data/milestone-templates */
export async function masterDataServiceCreateMilestoneTemplate(
  body: API.CreateMilestoneTemplateRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateMilestoneTemplateResponse>(
    "/api/v1/master-data/milestone-templates",
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

/** 此处后端没有提供注释 POST /api/v1/master-data/milestone-templates/${param0}/publish */
export async function masterDataServicePublishMilestoneTemplate(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServicePublishMilestoneTemplateParams,
  body: API.PublishMilestoneTemplateRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.PublishMilestoneTemplateResponse>(
    `/api/v1/master-data/milestone-templates/${param0}/publish`,
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

/** 此处后端没有提供注释 POST /api/v1/master-data/milestone-templates/${param0}/set-default */
export async function masterDataServiceSetDefaultMilestoneTemplate(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceSetDefaultMilestoneTemplateParams,
  body: API.SetDefaultMilestoneTemplateRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.SetDefaultMilestoneTemplateResponse>(
    `/api/v1/master-data/milestone-templates/${param0}/set-default`,
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

/** 此处后端没有提供注释 GET /api/v1/master-data/number-rules */
export async function masterDataServiceListNumberRules(options?: {
  [key: string]: any;
}) {
  return request<API.ListNumberRulesResponse>(
    "/api/v1/master-data/number-rules",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/master-data/number-rules */
export async function masterDataServiceCreateNumberRule(
  body: API.CreateNumberRuleRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateNumberRuleResponse>(
    "/api/v1/master-data/number-rules",
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

/** 此处后端没有提供注释 PUT /api/v1/master-data/number-rules/${param0} */
export async function masterDataServiceUpdateNumberRule(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceUpdateNumberRuleParams,
  body: API.UpdateNumberRuleRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateNumberRuleResponse>(
    `/api/v1/master-data/number-rules/${param0}`,
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
  return request<API.ListOptionsResponse>("/api/v1/master-data/options", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/master-data/ports */
export async function masterDataServiceListPorts(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListPortsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListPortsResponse>("/api/v1/master-data/ports", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/master-data/ports */
export async function masterDataServiceCreatePort(
  body: API.CreatePortRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreatePortResponse>("/api/v1/master-data/ports", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/master-data/ports/${param0} */
export async function masterDataServiceUpdatePort(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceUpdatePortParams,
  body: API.UpdatePortRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdatePortResponse>(
    `/api/v1/master-data/ports/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/master-data/shipping-lines */
export async function masterDataServiceListShippingLines(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListShippingLinesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListShippingLinesResponse>(
    "/api/v1/master-data/shipping-lines",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/master-data/shipping-lines */
export async function masterDataServiceCreateShippingLine(
  body: API.CreateShippingLineRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateShippingLineResponse>(
    "/api/v1/master-data/shipping-lines",
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

/** 此处后端没有提供注释 PUT /api/v1/master-data/shipping-lines/${param0} */
export async function masterDataServiceUpdateShippingLine(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceUpdateShippingLineParams,
  body: API.UpdateShippingLineRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateShippingLineResponse>(
    `/api/v1/master-data/shipping-lines/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/reference/administrative-regions */
export async function masterDataServiceListAdministrativeRegions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListAdministrativeRegionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListAdministrativeRegionsResponse>(
    "/api/v1/reference/administrative-regions",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/reference/currencies */
export async function masterDataServiceListCurrencies(options?: {
  [key: string]: any;
}) {
  return request<API.ListCurrenciesResponse>("/api/v1/reference/currencies", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/reference/currencies/search */
export async function masterDataServiceSearchCurrencies(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceSearchCurrenciesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCurrenciesResponse>(
    "/api/v1/reference/currencies/search",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
