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

/** 此处后端没有提供注释 GET /api/v1/finance/bill-settlement-account-candidates */
export async function settlementServiceListBillSettlementAccountCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListBillSettlementAccountCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListBillSettlementAccountCandidatesResponse>(
    "/api/v1/finance/bill-settlement-account-candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/bill-settlement-account-update-candidates */
export async function settlementServiceListBillSettlementAccountUpdateCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListBillSettlementAccountUpdateCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListBillSettlementAccountUpdateCandidatesResponse>(
    "/api/v1/finance/bill-settlement-account-update-candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** ListFinanceBillTagAssignmentOptions 仅为账单标签写入提供候选，按 bill.update 可写组织过滤。 GET /api/v1/finance/bill-tag-assignment-options */
export async function settlementServiceListFinanceBillTagAssignmentOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListFinanceBillTagAssignmentOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFinanceBillTagAssignmentOptionsResponse>(
    "/api/v1/finance/bill-tag-assignment-options",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/bill-tag-options */
export async function settlementServiceListFinanceBillTagOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListFinanceBillTagOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFinanceBillTagOptionsResponse>(
    "/api/v1/finance/bill-tag-options",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/bill-tags/batch-assign */
export async function settlementServiceBatchAssignFinanceBillTags(
  body: API.BatchAssignFinanceBillTagsRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchAssignFinanceBillTagsResponse>(
    "/api/v1/finance/bill-tags/batch-assign",
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

/** 此处后端没有提供注释 POST /api/v1/finance/bill-tags/batch-remove */
export async function settlementServiceBatchRemoveFinanceBillTags(
  body: API.BatchRemoveFinanceBillTagsRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchRemoveFinanceBillTagsResponse>(
    "/api/v1/finance/bill-tags/batch-remove",
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

/** 此处后端没有提供注释 GET /api/v1/finance/bills/creation-candidates */
export async function settlementServiceListBillCreationCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListBillCreationCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListBillCreationCandidatesResponse>(
    "/api/v1/finance/bills/creation-candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
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

/** ListCommissionAdjustments 财务调整列表：服务端分页，支持状态、来源、员工与
 订单号/提成号关键字过滤，默认 created_at 倒序；组织范围按 commission.read 解析。 GET /api/v1/finance/commission-adjustments */
export async function settlementServiceListCommissionAdjustments(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCommissionAdjustmentsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCommissionAdjustmentsResponse>(
    "/api/v1/finance/commission-adjustments",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/commission-adjustments/${param0}/cancel */
export async function settlementServiceCancelCommissionAdjustment(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCancelCommissionAdjustmentParams,
  body: API.CancelCommissionAdjustmentRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CancelCommissionAdjustmentResponse>(
    `/api/v1/finance/commission-adjustments/${param0}/cancel`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/commission-adjustments/${param0}/confirm */
export async function settlementServiceConfirmCommissionAdjustment(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceConfirmCommissionAdjustmentParams,
  body: API.ConfirmCommissionAdjustmentRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ConfirmCommissionAdjustmentResponse>(
    `/api/v1/finance/commission-adjustments/${param0}/confirm`,
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

/** GetMyFeeSupplementAdjustmentSource 员工本人专属冲减来源最小详情：只返回
 employee_id 等于当前用户且具备组织成员关系的补录冲减调整的订单号、原提成号、
 补录费用摘要、建议金额、状态与生成时间；不要求组织级 commission.read，
 查询他人调整稳定返回不存在，不泄露记录事实。 GET /api/v1/finance/commission-adjustments/${param0}/my-supplement-source */
export async function settlementServiceGetMyFeeSupplementAdjustmentSource(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceGetMyFeeSupplementAdjustmentSourceParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetMyFeeSupplementAdjustmentSourceResponse>(
    `/api/v1/finance/commission-adjustments/${param0}/my-supplement-source`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/commission-adjustments/${param0}/paid */
export async function settlementServiceMarkCommissionAdjustmentPaid(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceMarkCommissionAdjustmentPaidParams,
  body: API.MarkCommissionAdjustmentPaidRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.MarkCommissionAdjustmentPaidResponse>(
    `/api/v1/finance/commission-adjustments/${param0}/paid`,
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
  return request<API.CreateCommissionRuleResponse>(
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
  return request<API.UpdateCommissionRuleResponse>(
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

/** CopyCommissionRule 实现【复制为新方案】：新方案以当天或未来日期生效，源方案
 终止日衔接为新方案生效日前一日。按 commission.manage 可写组织过滤。 POST /api/v1/finance/commission-rules/${param0}/copy */
export async function settlementServiceCopyCommissionRule(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCopyCommissionRuleParams,
  body: API.CopyCommissionRuleRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CopyCommissionRuleResponse>(
    `/api/v1/finance/commission-rules/${param0}/copy`,
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

/** AssignCommissionRuleEmployees / RemoveCommissionRuleEmployees 为方案名单的独立
 增删入口：expected_version 防并发覆盖；已生效方案只允许当天或未来的变更生效日，
 不物理删除历史分配。按 commission.manage 可写组织过滤。 POST /api/v1/finance/commission-rules/${param0}/employees/assign */
export async function settlementServiceAssignCommissionRuleEmployees(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceAssignCommissionRuleEmployeesParams,
  body: API.AssignCommissionRuleEmployeesRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.AssignCommissionRuleEmployeesResponse>(
    `/api/v1/finance/commission-rules/${param0}/employees/assign`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/commission-rules/${param0}/employees/remove */
export async function settlementServiceRemoveCommissionRuleEmployees(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceRemoveCommissionRuleEmployeesParams,
  body: API.RemoveCommissionRuleEmployeesRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.RemoveCommissionRuleEmployeesResponse>(
    `/api/v1/finance/commission-rules/${param0}/employees/remove`,
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
  return request<API.GetCommissionResponse>(
    `/api/v1/finance/commissions/${param0}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/commissions/${param0}/adjustments */
export async function settlementServiceCreateCommissionAdjustment(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCreateCommissionAdjustmentParams,
  body: API.CreateCommissionAdjustmentRequest,
  options?: { [key: string]: any }
) {
  const { commissionId: param0, ...queryParams } = params;
  return request<API.CreateCommissionAdjustmentResponse>(
    `/api/v1/finance/commissions/${param0}/adjustments`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/commissions/${param0}/cancel */
export async function settlementServiceCancelCommission(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCancelCommissionParams,
  body: API.CancelCommissionRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CancelCommissionResponse>(
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
  body: API.ConfirmCommissionRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ConfirmCommissionResponse>(
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
  body: API.MarkCommissionPaidRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.MarkCommissionPaidResponse>(
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

/** ListCommissionCandidates 按来源单发现「员工 + 人员身份 + 已解析方案」的计提
 候选：来源二选一，服务端按来源订单提成归属与归属日期自动解析唯一有效方案，
 不再接受客户端指定规则。按 commission.manage 可写组织过滤。 GET /api/v1/finance/commissions/candidates */
export async function settlementServiceListCommissionCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCommissionCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCommissionCandidatesResponse>(
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
export async function settlementServiceListCommissionEmployees(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCommissionEmployeesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCommissionEmployeesResponse>(
    "/api/v1/finance/commissions/employees",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/commissions/export */
export async function settlementServiceExportCommissions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceExportCommissionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ExportCommissionsResponse>(
    "/api/v1/finance/commissions/export",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** ListCommissionNettingCandidates 为对冲提成提供已确认且存在应收分摊的对冲单候选，按 commission.manage 可写组织过滤。 GET /api/v1/finance/commissions/netting-candidates */
export async function settlementServiceListCommissionNettingCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCommissionNettingCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCommissionNettingCandidatesResponse>(
    "/api/v1/finance/commissions/netting-candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
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

/** 此处后端没有提供注释 GET /api/v1/finance/commissions/verification-candidates */
export async function settlementServiceListCommissionVerificationCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListCommissionVerificationCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListCommissionVerificationCandidatesResponse>(
    "/api/v1/finance/commissions/verification-candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** GetBilledFeeEditPolicy 获取账单创建后的费用修改策略。 GET /api/v1/finance/custom-settings/billed-fee-edit-policy */
export async function settlementServiceGetBilledFeeEditPolicy(options?: {
  [key: string]: any;
}) {
  return request<API.GetBilledFeeEditPolicyResponse>(
    "/api/v1/finance/custom-settings/billed-fee-edit-policy",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** UpdateBilledFeeEditPolicy 更新账单创建后的费用修改策略。 PUT /api/v1/finance/custom-settings/billed-fee-edit-policy */
export async function settlementServiceUpdateBilledFeeEditPolicy(
  body: API.UpdateBilledFeeEditPolicyRequest,
  options?: { [key: string]: any }
) {
  return request<API.UpdateBilledFeeEditPolicyResponse>(
    "/api/v1/finance/custom-settings/billed-fee-edit-policy",
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

/** GetCreditLimitControlPolicy 获取往来单位信用额度管控策略。 GET /api/v1/finance/custom-settings/credit-limit-control-policy */
export async function settlementServiceGetCreditLimitControlPolicy(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceGetCreditLimitControlPolicyParams,
  options?: { [key: string]: any }
) {
  return request<API.GetCreditLimitControlPolicyResponse>(
    "/api/v1/finance/custom-settings/credit-limit-control-policy",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** UpdateCreditLimitControlPolicy 更新往来单位信用额度管控策略。 PUT /api/v1/finance/custom-settings/credit-limit-control-policy */
export async function settlementServiceUpdateCreditLimitControlPolicy(
  body: API.UpdateCreditLimitControlPolicyRequest,
  options?: { [key: string]: any }
) {
  return request<API.UpdateCreditLimitControlPolicyResponse>(
    "/api/v1/finance/custom-settings/credit-limit-control-policy",
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

/** ListFinanceFeeTagAssignmentOptions 仅为费用标签写入提供候选，按 fee.tag 可写组织过滤。 GET /api/v1/finance/fee-tag-assignment-options */
export async function settlementServiceListFinanceFeeTagAssignmentOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListFinanceFeeTagAssignmentOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFinanceFeeTagAssignmentOptionsResponse>(
    "/api/v1/finance/fee-tag-assignment-options",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/fee-tag-options */
export async function settlementServiceListFinanceFeeTagOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListFinanceFeeTagOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFinanceFeeTagOptionsResponse>(
    "/api/v1/finance/fee-tag-options",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/finance/fee-tags/batch-assign */
export async function settlementServiceBatchAssignFinanceFeeTags(
  body: API.BatchAssignFinanceFeeTagsRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchAssignFinanceFeeTagsResponse>(
    "/api/v1/finance/fee-tags/batch-assign",
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

/** 此处后端没有提供注释 POST /api/v1/finance/fee-tags/batch-remove */
export async function settlementServiceBatchRemoveFinanceFeeTags(
  body: API.BatchRemoveFinanceFeeTagsRequest,
  options?: { [key: string]: any }
) {
  return request<API.BatchRemoveFinanceFeeTagsResponse>(
    "/api/v1/finance/fee-tags/batch-remove",
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

/** 此处后端没有提供注释 GET /api/v1/finance/fees/orders/${param0} */
export async function settlementServiceGetFeeLedgerOrderDetail(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceGetFeeLedgerOrderDetailParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.GetFeeLedgerOrderDetailResponse>(
    `/api/v1/finance/fees/orders/${param0}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** GetFeeLedgerPreference 获取当前用户的费用明细表头、分页、排序与颜色设置。 GET /api/v1/finance/fees/preference */
export async function settlementServiceGetFeeLedgerPreference(options?: {
  [key: string]: any;
}) {
  return request<API.GetFeeLedgerPreferenceResponse>(
    "/api/v1/finance/fees/preference",
    {
      method: "GET",
      ...(options || {}),
    }
  );
}

/** UpdateFeeLedgerPreference 保存当前用户的费用明细个性化设置。 PUT /api/v1/finance/fees/preference */
export async function settlementServiceUpdateFeeLedgerPreference(
  body: API.UpdateFeeLedgerPreferenceRequest,
  options?: { [key: string]: any }
) {
  return request<API.UpdateFeeLedgerPreferenceResponse>(
    "/api/v1/finance/fees/preference",
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

/** ResetFeeLedgerPreference 删除当前用户的个性化设置并恢复系统默认值。 DELETE /api/v1/finance/fees/preference */
export async function settlementServiceResetFeeLedgerPreference(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceResetFeeLedgerPreferenceParams,
  options?: { [key: string]: any }
) {
  return request<API.ResetFeeLedgerPreferenceResponse>(
    "/api/v1/finance/fees/preference",
    {
      method: "DELETE",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
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

/** 此处后端没有提供注释 GET /api/v1/finance/invoices/creation-bills */
export async function settlementServiceListInvoiceCreationBills(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListInvoiceCreationBillsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListInvoiceCreationBillsResponse>(
    "/api/v1/finance/invoices/creation-bills",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/invoices/creation-profiles */
export async function settlementServiceListInvoiceProfilesForBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListInvoiceProfilesForBillParams,
  options?: { [key: string]: any }
) {
  return request<API.ListInvoiceProfilesForBillResponse>(
    "/api/v1/finance/invoices/creation-profiles",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 对冲单：按组织、结算单位和账单币种预览双方未结余额，创建、确认、取消与反转同币种抵销。 GET /api/v1/finance/nettings */
export async function settlementServiceListNettings(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListNettingsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListNettingsResponse>("/api/v1/finance/nettings", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/nettings */
export async function settlementServiceCreateNetting(
  body: API.CreateNettingRequest,
  options?: { [key: string]: any }
) {
  return request<API.CreateNettingResponse>("/api/v1/finance/nettings", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 GET /api/v1/finance/nettings/${param0} */
export async function settlementServiceGetNetting(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceGetNettingParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.GetNettingResponse>(`/api/v1/finance/nettings/${param0}`, {
    method: "GET",
    params: { ...queryParams },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/finance/nettings/${param0}/cancel */
export async function settlementServiceCancelNetting(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceCancelNettingParams,
  body: API.CancelNettingRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.CancelNettingResponse>(
    `/api/v1/finance/nettings/${param0}/cancel`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/nettings/${param0}/confirm */
export async function settlementServiceConfirmNetting(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceConfirmNettingParams,
  body: API.ConfirmNettingRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ConfirmNettingResponse>(
    `/api/v1/finance/nettings/${param0}/confirm`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/nettings/${param0}/reverse */
export async function settlementServiceReverseNetting(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceReverseNettingParams,
  body: API.ReverseNettingRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.ReverseNettingResponse>(
    `/api/v1/finance/nettings/${param0}/reverse`,
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

/** 此处后端没有提供注释 POST /api/v1/finance/nettings/preview */
export async function settlementServicePreviewNetting(
  body: API.PreviewNettingRequest,
  options?: { [key: string]: any }
) {
  return request<API.PreviewNettingResponse>(
    "/api/v1/finance/nettings/preview",
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

/** 此处后端没有提供注释 GET /api/v1/finance/organization-options */
export async function settlementServiceListFinanceOrganizationOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListFinanceOrganizationOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFinanceOrganizationOptionsResponse>(
    "/api/v1/finance/organization-options",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/finance/settlement-party-options */
export async function settlementServiceListFinanceSettlementPartyOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListFinanceSettlementPartyOptionsParams,
  options?: { [key: string]: any }
) {
  return request<API.ListFinanceSettlementPartyOptionsResponse>(
    "/api/v1/finance/settlement-party-options",
    {
      method: "GET",
      params: {
        ...params,
      },
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

/** 此处后端没有提供注释 GET /api/v1/finance/verifications/creation-candidates */
export async function settlementServiceListVerificationCreationCandidates(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SettlementServiceListVerificationCreationCandidatesParams,
  options?: { [key: string]: any }
) {
  return request<API.ListVerificationCreationCandidatesResponse>(
    "/api/v1/finance/verifications/creation-candidates",
    {
      method: "GET",
      params: {
        ...params,
      },
      ...(options || {}),
    }
  );
}
