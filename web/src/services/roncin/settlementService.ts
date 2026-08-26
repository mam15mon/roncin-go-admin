// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/finance/bills */
export async function settlementServiceListBills(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListBillsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListBillsResponse>("/api/v1/finance/bills", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/bills */
export async function settlementServiceCreateBill(
  body: API.CreateBillRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateBillResponse>("/api/v1/finance/bills", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/finance/bills/${param0} */
export async function settlementServiceGetBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceGetBillParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetBillResponse>(`/api/v1/finance/bills/${param0}`, {
    method: "GET",
    params: { ...queryParams },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/finance/bills/${param0} */
export async function settlementServiceUpdateBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceUpdateBillParams,
  body: API.UpdateBillRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.UpdateBillResponse>(`/api/v1/finance/bills/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/bills/${param0}/cancel */
export async function settlementServiceCancelBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCancelBillParams,
  body: API.CancelBillRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CancelBillResponse>(
    `/api/v1/finance/bills/${param0}/cancel`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/bills/${param0}/confirm */
export async function settlementServiceConfirmBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceConfirmBillParams,
  body: API.ConfirmBillRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ConfirmBillResponse>(
    `/api/v1/finance/bills/${param0}/confirm`,
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

/** ListFeeLedger 获取当前组织全部业务线的应收应付费用总台账。 GET /api/v1/finance/fees */
export async function settlementServiceListFeeLedger(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListFeeLedgerParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFeeLedgerResponse>("/api/v1/finance/fees", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}
