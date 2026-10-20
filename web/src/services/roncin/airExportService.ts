// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

/** 此处后端没有提供注释 GET /api/v1/air-export/master-waybills */
export async function airExportServiceListAirMasterWaybills(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AirExportServiceListAirMasterWaybillsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListAirMasterWaybillsResponse>(
    "/api/v1/air-export/master-waybills",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/air-export/master-waybills/${param0} */
export async function airExportServiceGetAirMasterWaybill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AirExportServiceGetAirMasterWaybillParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetAirMasterWaybillResponse>(
    `/api/v1/air-export/master-waybills/${param0}`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 PUT /api/v1/air-export/master-waybills/${param0} */
export async function airExportServiceUpdateAirMasterWaybill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AirExportServiceUpdateAirMasterWaybillParams,
  body: API.UpdateAirMasterWaybillRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateAirMasterWaybillResponse>(
    `/api/v1/air-export/master-waybills/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/air-export/master-waybills/${param0}/link-candidates */
export async function airExportServiceListAirLinkCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AirExportServiceListAirLinkCandidatesParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ListAirLinkCandidatesResponse>(
    `/api/v1/air-export/master-waybills/${param0}/link-candidates`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/air-export/master-waybills/${param0}/members */
export async function airExportServiceLinkAirMasterWaybill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AirExportServiceLinkAirMasterWaybillParams,
  body: API.LinkAirMasterWaybillRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.LinkAirMasterWaybillResponse>(
    `/api/v1/air-export/master-waybills/${param0}/members`,
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

/** 此处后端没有提供注释 POST /api/v1/air-export/master-waybills/${param0}/preview */
export async function airExportServicePreviewAirMasterWaybillUpdate(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AirExportServicePreviewAirMasterWaybillUpdateParams,
  body: API.PreviewAirMasterWaybillUpdateRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.PreviewAirMasterWaybillUpdateResponse>(
    `/api/v1/air-export/master-waybills/${param0}/preview`,
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

/** 此处后端没有提供注释 GET /api/v1/air-export/orders/${param0}/house-versions */
export async function airExportServiceListAirHouseWaybillVersions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.AirExportServiceListAirHouseWaybillVersionsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListAirHouseWaybillVersionsResponse>(
    `/api/v1/air-export/orders/${param0}/house-versions`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}
