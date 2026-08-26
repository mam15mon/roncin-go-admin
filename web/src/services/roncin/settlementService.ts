// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 POST /api/v1/finance/bill-batches */
export async function settlementServiceCreateBillBatch(
  body: API.CreateBillBatchRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateBillBatchResponse>("/api/v1/finance/bill-batches", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/bill-batches/${param0}/confirm */
export async function settlementServiceConfirmBillBatch(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceConfirmBillBatchParams,
  body: API.ConfirmBillBatchRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ConfirmBillBatchResponse>(
    `/api/v1/finance/bill-batches/${param0}/confirm`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/bill-batches/preview */
export async function settlementServicePreviewBillBatch(
  body: API.PreviewBillBatchRequest,
  options?: { [key: string]: any }
) {
  return request<API.PreviewBillBatchResponse>(
    "/api/v1/finance/bill-batches/preview",
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

/** 此处后端没有提供注释 GET /api/v1/finance/cashflows */
export async function settlementServiceListCashflows(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCashflowsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCashflowsResponse>("/api/v1/finance/cashflows", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/cashflows */
export async function settlementServiceCreateCashflow(
  body: API.CreateCashflowRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateCashflowResponse>("/api/v1/finance/cashflows", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/cashflows/${param0}/cancel */
export async function settlementServiceCancelCashflow(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCancelCashflowParams,
  body: API.CancelCashflowRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CancelCashflowResponse>(
    `/api/v1/finance/cashflows/${param0}/cancel`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/cashflows/${param0}/confirm */
export async function settlementServiceConfirmCashflow(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceConfirmCashflowParams,
  body: API.ConfirmCashflowRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ConfirmCashflowResponse>(
    `/api/v1/finance/cashflows/${param0}/confirm`,
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

/** 此处后端没有提供注释 GET /api/v1/finance/commission-rules */
export async function settlementServiceListCommissionRules(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCommissionRulesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCommissionRulesResponse>(
    "/api/v1/finance/commission-rules",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/commission-rules */
export async function settlementServiceCreateCommissionRule(
  body: API.CreateCommissionRuleRequest,
  options?: { [key: string]: any }
) {
  return request<API.CommissionRuleResponse>(
    "/api/v1/finance/commission-rules",
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

/** 此处后端没有提供注释 PUT /api/v1/finance/commission-rules/${param0} */
export async function settlementServiceUpdateCommissionRule(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceUpdateCommissionRuleParams,
  body: API.UpdateCommissionRuleRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CommissionRuleResponse>(
    `/api/v1/finance/commission-rules/${param0}`,
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

/** 此处后端没有提供注释 GET /api/v1/finance/commissions */
export async function settlementServiceListCommissions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCommissionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCommissionsResponse>("/api/v1/finance/commissions", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/commissions */
export async function settlementServiceCreateCommission(
  body: API.CreateCommissionRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateCommissionResponse>("/api/v1/finance/commissions", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/finance/commissions/${param0} */
export async function settlementServiceGetCommission(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceGetCommissionParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CommissionResponse>(
    `/api/v1/finance/commissions/${param0}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/commissions/${param0}/cancel */
export async function settlementServiceCancelCommission(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCancelCommissionParams,
  body: API.CancelCommissionRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CommissionResponse>(
    `/api/v1/finance/commissions/${param0}/cancel`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/commissions/${param0}/confirm */
export async function settlementServiceConfirmCommission(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceConfirmCommissionParams,
  body: API.CommissionTransitionRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CommissionResponse>(
    `/api/v1/finance/commissions/${param0}/confirm`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/commissions/${param0}/paid */
export async function settlementServiceMarkCommissionPaid(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceMarkCommissionPaidParams,
  body: API.CommissionTransitionRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CommissionResponse>(
    `/api/v1/finance/commissions/${param0}/paid`,
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

/** 此处后端没有提供注释 GET /api/v1/finance/commissions/candidates */
export async function settlementServiceListCommissionCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCommissionCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCommissionEmployeesResponse>(
    "/api/v1/finance/commissions/candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/commissions/employees */
export async function settlementServiceListCommissionEmployees(options?: {
  [key: string]: any;
}) {
  return request<API.ListCommissionEmployeesResponse>(
    "/api/v1/finance/commissions/employees",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/commissions/preview */
export async function settlementServicePreviewCommission(
  body: API.PreviewCommissionRequest,
  options?: { [key: string]: any }
) {
  return request<API.PreviewCommissionResponse>(
    "/api/v1/finance/commissions/preview",
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

/** 此处后端没有提供注释 GET /api/v1/finance/invoices */
export async function settlementServiceListInvoices(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListInvoicesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListInvoicesResponse>("/api/v1/finance/invoices", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/invoices */
export async function settlementServiceCreateInvoice(
  body: API.CreateInvoiceRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateInvoiceResponse>("/api/v1/finance/invoices", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/finance/invoices/${param0} */
export async function settlementServiceGetInvoice(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceGetInvoiceParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetInvoiceResponse>(`/api/v1/finance/invoices/${param0}`, {
    method: "GET",
    params: { ...queryParams },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/invoices/${param0}/cancel */
export async function settlementServiceCancelInvoice(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCancelInvoiceParams,
  body: API.CancelInvoiceRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CancelInvoiceResponse>(
    `/api/v1/finance/invoices/${param0}/cancel`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/invoices/${param0}/issue */
export async function settlementServiceIssueInvoice(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceIssueInvoiceParams,
  body: API.IssueInvoiceRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.IssueInvoiceResponse>(
    `/api/v1/finance/invoices/${param0}/issue`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/invoices/${param0}/red-flush */
export async function settlementServiceRedFlushInvoice(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceRedFlushInvoiceParams,
  body: API.RedFlushInvoiceRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.RedFlushInvoiceResponse>(
    `/api/v1/finance/invoices/${param0}/red-flush`,
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

/** 此处后端没有提供注释 GET /api/v1/finance/verifications */
export async function settlementServiceListVerifications(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListVerificationsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListVerificationsResponse>(
    "/api/v1/finance/verifications",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/verifications */
export async function settlementServiceCreateVerification(
  body: API.CreateVerificationRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateVerificationResponse>(
    "/api/v1/finance/verifications",
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

/** 此处后端没有提供注释 POST /api/v1/finance/verifications/${param0}/reverse */
export async function settlementServiceReverseVerification(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceReverseVerificationParams,
  body: API.ReverseVerificationRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ReverseVerificationResponse>(
    `/api/v1/finance/verifications/${param0}/reverse`,
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
