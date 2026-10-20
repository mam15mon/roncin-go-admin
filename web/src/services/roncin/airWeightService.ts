// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

/** 此处后端没有提供注释 POST /api/v1/air-weights/confirm */
export async function airWeightServiceConfirmAirWeight(
  body: API.ConfirmAirWeightRequest,
  options?: { [key: string]: any }
) {
  return request<API.ConfirmAirWeightResponse>("/api/v1/air-weights/confirm", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/air-weights/history */
export async function airWeightServiceListAirWeightConfirmations(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AirWeightServiceListAirWeightConfirmationsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListAirWeightConfirmationsResponse>(
    "/api/v1/air-weights/history",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/air-weights/preview */
export async function airWeightServicePreviewAirWeight(
  body: API.PreviewAirWeightRequest,
  options?: { [key: string]: any }
) {
  return request<API.PreviewAirWeightResponse>("/api/v1/air-weights/preview", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}
