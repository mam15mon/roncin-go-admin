// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/master-data/airlines */
export async function masterDataServiceListAirlines(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListAirlinesParams,
  options?: { [key: string]: any }
) {
  return request<API.AirlineListReply>("/api/v1/master-data/airlines", {
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
  return request<API.AirlineReply>("/api/v1/master-data/airlines", {
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
  return request<API.AirlineReply>(`/api/v1/master-data/airlines/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/master-data/airports */
export async function masterDataServiceListAirports(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListAirportsParams,
  options?: { [key: string]: any }
) {
  return request<API.AirportListReply>("/api/v1/master-data/airports", {
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
  return request<API.AirportReply>("/api/v1/master-data/airports", {
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
  return request<API.AirportReply>(`/api/v1/master-data/airports/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/master-data/import */
export async function masterDataServiceImportItems(
  body: API.ImportMasterDataItemsRequest,
  options?: { [key: string]: any }
) {
  return request<API.MasterDataImportReply>("/api/v1/master-data/import", {
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

/** 此处后端没有提供注释 GET /api/v1/master-data/milestone-templates */
export async function masterDataServiceListMilestoneTemplates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListMilestoneTemplatesParams,
  options?: { [key: string]: any }
) {
  return request<API.MilestoneTemplateListReply>(
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
  return request<API.MilestoneTemplateReply>(
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
  return request<API.MilestoneTemplateReply>(
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
  return request<API.MilestoneTemplateReply>(
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
  return request<API.NumberRuleListReply>("/api/v1/master-data/number-rules", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/master-data/number-rules */
export async function masterDataServiceCreateNumberRule(
  body: API.CreateNumberRuleRequest,
  options?: { [key: string]: any }
) {
  return request<API.NumberRuleReply>("/api/v1/master-data/number-rules", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/master-data/number-rules/${param0} */
export async function masterDataServiceUpdateNumberRule(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceUpdateNumberRuleParams,
  body: API.UpdateNumberRuleRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.NumberRuleReply>(
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
  return request<API.MasterDataOptionsReply>("/api/v1/master-data/options", {
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
  return request<API.PortListReply>("/api/v1/master-data/ports", {
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
  return request<API.PortReply>("/api/v1/master-data/ports", {
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
  return request<API.PortReply>(`/api/v1/master-data/ports/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/master-data/shipping-lines */
export async function masterDataServiceListShippingLines(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListShippingLinesParams,
  options?: { [key: string]: any }
) {
  return request<API.ShippingLineListReply>(
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
  return request<API.ShippingLineReply>("/api/v1/master-data/shipping-lines", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/master-data/shipping-lines/${param0} */
export async function masterDataServiceUpdateShippingLine(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceUpdateShippingLineParams,
  body: API.UpdateShippingLineRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ShippingLineReply>(
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

/** 此处后端没有提供注释 GET /api/v1/master-data/status-templates */
export async function masterDataServiceListStatusTemplates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListStatusTemplatesParams,
  options?: { [key: string]: any }
) {
  return request<API.StatusTemplateListReply>(
    "/api/v1/master-data/status-templates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/master-data/status-templates */
export async function masterDataServiceCreateStatusTemplate(
  body: API.CreateStatusTemplateRequest,
  options?: { [key: string]: any }
) {
  return request<API.StatusTemplateReply>(
    "/api/v1/master-data/status-templates",
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

/** 此处后端没有提供注释 POST /api/v1/master-data/status-templates/${param0}/publish */
export async function masterDataServicePublishStatusTemplate(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServicePublishStatusTemplateParams,
  body: API.PublishStatusTemplateRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.StatusTemplateReply>(
    `/api/v1/master-data/status-templates/${param0}/publish`,
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

/** 此处后端没有提供注释 POST /api/v1/master-data/status-templates/${param0}/set-default */
export async function masterDataServiceSetDefaultStatusTemplate(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceSetDefaultStatusTemplateParams,
  body: API.SetDefaultStatusTemplateRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.StatusTemplateReply>(
    `/api/v1/master-data/status-templates/${param0}/set-default`,
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

/** 此处后端没有提供注释 GET /api/v1/reference/administrative-regions */
export async function masterDataServiceListAdministrativeRegions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.MasterDataServiceListAdministrativeRegionsParams,
  options?: { [key: string]: any }
) {
  return request<API.AdministrativeRegionListReply>(
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
  return request<API.CurrencyListReply>("/api/v1/reference/currencies", {
    method: "GET",
    ...(options || {}),
  });
}
