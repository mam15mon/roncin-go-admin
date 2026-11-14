// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

/** 此处后端没有提供注释 GET /api/v1/finance/reports/aging */
export async function financeReportServiceListAgingReport(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FinanceReportServiceListAgingReportParams,
  options?: { [key: string]: any }
) {
  return request<API.ListAgingReportResponse>("/api/v1/finance/reports/aging", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/finance/reports/aging/export */
export async function financeReportServiceExportAgingReport(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FinanceReportServiceExportAgingReportParams,
  options?: { [key: string]: any }
) {
  return request<API.ExportAgingReportResponse>(
    "/api/v1/finance/reports/aging/export",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/reports/cashflows */
export async function financeReportServiceListCashflowReport(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FinanceReportServiceListCashflowReportParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCashflowReportResponse>(
    "/api/v1/finance/reports/cashflows",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/reports/cashflows/export */
export async function financeReportServiceExportCashflowReport(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FinanceReportServiceExportCashflowReportParams,
  options?: { [key: string]: any }
) {
  return request<API.ExportCashflowReportResponse>(
    "/api/v1/finance/reports/cashflows/export",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/reports/party-options */
export async function financeReportServiceListFinanceReportPartyOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FinanceReportServiceListFinanceReportPartyOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFinanceReportPartyOptionsResponse>(
    "/api/v1/finance/reports/party-options",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/reports/personnel-options */
export async function financeReportServiceListFinanceReportPersonnelOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FinanceReportServiceListFinanceReportPersonnelOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFinanceReportPersonnelOptionsResponse>(
    "/api/v1/finance/reports/personnel-options",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/reports/profits */
export async function financeReportServiceListProfitReport(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FinanceReportServiceListProfitReportParams,
  options?: { [key: string]: any }
) {
  return request<API.ListProfitReportResponse>(
    "/api/v1/finance/reports/profits",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/reports/profits/export */
export async function financeReportServiceExportProfitReport(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.FinanceReportServiceExportProfitReportParams,
  options?: { [key: string]: any }
) {
  return request<API.ExportProfitReportResponse>(
    "/api/v1/finance/reports/profits/export",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
