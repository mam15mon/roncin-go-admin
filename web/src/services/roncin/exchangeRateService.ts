// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

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
