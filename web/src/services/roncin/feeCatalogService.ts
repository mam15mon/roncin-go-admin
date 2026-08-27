// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/finance/billing-units */
export async function feeCatalogServiceListBillingUnits(options?: {
  [key: string]: any;
}) {
  return request<API.ListBillingUnitsResponse>(
    "/api/v1/finance/billing-units",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/billing-units */
export async function feeCatalogServiceCreateBillingUnit(
  body: API.CreateBillingUnitRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateBillingUnitResponse>(
    "/api/v1/finance/billing-units",
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

/** 此处后端没有提供注释 PUT /api/v1/finance/billing-units/${param0} */
export async function feeCatalogServiceUpdateBillingUnit(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FeeCatalogServiceUpdateBillingUnitParams,
  body: API.UpdateBillingUnitRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateBillingUnitResponse>(
    `/api/v1/finance/billing-units/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/finance/billing-units/search */
export async function feeCatalogServiceSearchBillingUnits(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FeeCatalogServiceSearchBillingUnitsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListBillingUnitsResponse>(
    "/api/v1/finance/billing-units/search",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/fee-settings */
export async function feeCatalogServiceListFeeSettings(options?: {
  [key: string]: any;
}) {
  return request<API.ListFeeSettingsResponse>("/api/v1/finance/fee-settings", {
    method: "GET",
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/fee-settings */
export async function feeCatalogServiceCreateFeeSetting(
  body: API.CreateFeeSettingRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateFeeSettingResponse>("/api/v1/finance/fee-settings", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/finance/fee-settings/${param0} */
export async function feeCatalogServiceUpdateFeeSetting(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FeeCatalogServiceUpdateFeeSettingParams,
  body: API.UpdateFeeSettingRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateFeeSettingResponse>(
    `/api/v1/finance/fee-settings/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/finance/fee-settings/search */
export async function feeCatalogServiceSearchFeeSettings(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FeeCatalogServiceSearchFeeSettingsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFeeSettingsResponse>(
    "/api/v1/finance/fee-settings/search",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/taxable-services */
export async function feeCatalogServiceListTaxableServices(options?: {
  [key: string]: any;
}) {
  return request<API.ListTaxableServicesResponse>(
    "/api/v1/finance/taxable-services",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/taxable-services */
export async function feeCatalogServiceCreateTaxableService(
  body: API.CreateTaxableServiceRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateTaxableServiceResponse>(
    "/api/v1/finance/taxable-services",
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

/** 此处后端没有提供注释 PUT /api/v1/finance/taxable-services/${param0} */
export async function feeCatalogServiceUpdateTaxableService(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FeeCatalogServiceUpdateTaxableServiceParams,
  body: API.UpdateTaxableServiceRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateTaxableServiceResponse>(
    `/api/v1/finance/taxable-services/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/finance/taxable-services/search */
export async function feeCatalogServiceSearchTaxableServices(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FeeCatalogServiceSearchTaxableServicesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListTaxableServicesResponse>(
    "/api/v1/finance/taxable-services/search",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
