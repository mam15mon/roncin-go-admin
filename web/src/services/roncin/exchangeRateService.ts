// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

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

/** SyncExchangeRates 将财务终审微调后的牌价按自然周幂等 Upsert 入库生效。 POST /api/v1/finance/exchange-rate-syncs */
export async function exchangeRateServiceSyncExchangeRates(
  body: API.SyncExchangeRatesRequest,
  options?: { [key: string]: any }
) {
  return request<API.SyncExchangeRatesResponse>(
    "/api/v1/finance/exchange-rate-syncs",
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

/** FetchExchangeRates 按当前组织本币与目标周抓取官方/市场牌价，返回结构化预览，
 不落库；数据来源与换算路径在预览中明示，抓取失败返回业务错误。 POST /api/v1/finance/exchange-rate-syncs/fetch */
export async function exchangeRateServiceFetchExchangeRates(
  body: API.FetchExchangeRatesRequest,
  options?: { [key: string]: any }
) {
  return request<API.FetchExchangeRatesResponse>(
    "/api/v1/finance/exchange-rate-syncs/fetch",
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

/** 此处后端没有提供注释 GET /api/v1/finance/exchange-rates */
export async function exchangeRateServiceListExchangeRateSettings(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.ExchangeRateServiceListExchangeRateSettingsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListExchangeRateSettingsResponse>(
    "/api/v1/finance/exchange-rates",
    {
      method: "GET",
      params: {
        ...params,
      },
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
