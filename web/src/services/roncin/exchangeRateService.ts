// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** GetExchangeRateCustomSetting 获取组织级汇率自定义策略。 GET /api/v1/finance/exchange-rate-custom-setting */
export async function exchangeRateServiceGetExchangeRateCustomSetting(options?: {
  [key: string]: any;
}) {
  return request<API.GetExchangeRateCustomSettingResponse>(
    "/api/v1/finance/exchange-rate-custom-setting",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** UpdateExchangeRateCustomSetting 更新组织级汇率自定义策略。 PUT /api/v1/finance/exchange-rate-custom-setting */
export async function exchangeRateServiceUpdateExchangeRateCustomSetting(
  body: API.UpdateExchangeRateCustomSettingRequest,
  options?: { [key: string]: any }
) {
  return request<API.UpdateExchangeRateCustomSettingResponse>(
    "/api/v1/finance/exchange-rate-custom-setting",
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      data: body,
      ...(options || {}),
    }
  );
}

/** DownloadExchangeRateImportTemplate 下载当前版本的汇率 Excel 导入模板。 GET /api/v1/finance/exchange-rate-import-template */
export async function exchangeRateServiceDownloadExchangeRateImportTemplate(options?: {
  [key: string]: any;
}) {
  return request<API.DownloadExchangeRateImportTemplateResponse>(
    "/api/v1/finance/exchange-rate-import-template",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** ConfirmExchangeRateImport 使用预检令牌确认整批导入。 POST /api/v1/finance/exchange-rate-imports */
export async function exchangeRateServiceConfirmExchangeRateImport(
  body: API.ConfirmExchangeRateImportRequest,
  options?: { [key: string]: any }
) {
  return request<API.ConfirmExchangeRateImportResponse>(
    "/api/v1/finance/exchange-rate-imports",
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

/** 此处后端没有提供注释 GET /api/v1/finance/exchange-rate-imports/${param0} */
export async function exchangeRateServiceGetExchangeRateImport(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.ExchangeRateServiceGetExchangeRateImportParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetExchangeRateImportResponse>(
    `/api/v1/finance/exchange-rate-imports/${param0}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** PreviewExchangeRateImport 解析并严格预检 Excel，不写入汇率设置。 POST /api/v1/finance/exchange-rate-imports/preview */
export async function exchangeRateServicePreviewExchangeRateImport(
  body: API.PreviewExchangeRateImportRequest,
  options?: { [key: string]: any }
) {
  return request<API.PreviewExchangeRateImportResponse>(
    "/api/v1/finance/exchange-rate-imports/preview",
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

/** 此处后端没有提供注释 GET /api/v1/finance/exchange-rate-time-standards */
export async function exchangeRateServiceListExchangeRateTimeStandards(options?: {
  [key: string]: any;
}) {
  return request<API.ListExchangeRateTimeStandardsResponse>(
    "/api/v1/finance/exchange-rate-time-standards",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 PUT /api/v1/finance/exchange-rate-time-standards */
export async function exchangeRateServiceUpdateExchangeRateTimeStandards(
  body: API.UpdateExchangeRateTimeStandardsRequest,
  options?: { [key: string]: any }
) {
  return request<API.UpdateExchangeRateTimeStandardsResponse>(
    "/api/v1/finance/exchange-rate-time-standards",
    {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      data: body,
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/exchange-rates */
export async function exchangeRateServiceListExchangeRateSettings(options?: {
  [key: string]: any;
}) {
  return request<API.ListExchangeRateSettingsResponse>(
    "/api/v1/finance/exchange-rates",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/exchange-rates */
export async function exchangeRateServiceCreateExchangeRateSetting(
  body: API.CreateExchangeRateSettingRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateExchangeRateSettingResponse>(
    "/api/v1/finance/exchange-rates",
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

/** 此处后端没有提供注释 PUT /api/v1/finance/exchange-rates/${param0} */
export async function exchangeRateServiceUpdateExchangeRateSetting(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.ExchangeRateServiceUpdateExchangeRateSettingParams,
  body: API.UpdateExchangeRateSettingRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateExchangeRateSettingResponse>(
    `/api/v1/finance/exchange-rates/${param0}`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/exchange-rates/${param0}/disable */
export async function exchangeRateServiceDisableExchangeRateSetting(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.ExchangeRateServiceDisableExchangeRateSettingParams,
  body: API.DisableExchangeRateSettingRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.DisableExchangeRateSettingResponse>(
    `/api/v1/finance/exchange-rates/${param0}/disable`,
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
