// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

/** 此处后端没有提供注释 POST /api/v1/order-fee-reviews/approve */
export async function orderFeeServiceApproveOrderFeeReviews(
  body: API.ApproveOrderFeeReviewsRequest,
  options?: { [key: string]: any }
) {
  return request<API.ApproveOrderFeeReviewsResponse>(
    "/api/v1/order-fee-reviews/approve",
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

/** 此处后端没有提供注释 POST /api/v1/order-fee-reviews/preview */
export async function orderFeeServicePreviewOrderFeeReviews(
  body: API.PreviewOrderFeeReviewsRequest,
  options?: { [key: string]: any }
) {
  return request<API.PreviewOrderFeeReviewsResponse>(
    "/api/v1/order-fee-reviews/preview",
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

/** 此处后端没有提供注释 GET /api/v1/orders/${param0}/fee-correction-requests */
export async function orderFeeServiceListOrderFeeCorrectionRequests(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceListOrderFeeCorrectionRequestsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListOrderFeeCorrectionRequestsResponse>(
    `/api/v1/orders/${param0}/fee-correction-requests`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-correction-requests */
export async function orderFeeServiceCreateOrderFeeCorrection(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceCreateOrderFeeCorrectionParams,
  body: API.CreateOrderFeeCorrectionRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.CreateOrderFeeCorrectionResponse>(
    `/api/v1/orders/${param0}/fee-correction-requests`,
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

/** 此处后端没有提供注释 GET /api/v1/orders/${param0}/fee-correction-requests/${param1} */
export async function orderFeeServiceGetOrderFeeCorrectionRequest(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceGetOrderFeeCorrectionRequestParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.GetOrderFeeCorrectionRequestResponse>(
    `/api/v1/orders/${param0}/fee-correction-requests/${param1}`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-correction-requests/${param1}/approval-preview */
export async function orderFeeServicePreviewOrderFeeCorrectionApproval(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServicePreviewOrderFeeCorrectionApprovalParams,
  body: API.PreviewOrderFeeCorrectionApprovalRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.PreviewOrderFeeCorrectionApprovalResponse>(
    `/api/v1/orders/${param0}/fee-correction-requests/${param1}/approval-preview`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-correction-requests/${param1}/approve */
export async function orderFeeServiceApproveOrderFeeCorrection(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceApproveOrderFeeCorrectionParams,
  body: API.ApproveOrderFeeCorrectionRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.ApproveOrderFeeCorrectionResponse>(
    `/api/v1/orders/${param0}/fee-correction-requests/${param1}/approve`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-correction-requests/${param1}/reject */
export async function orderFeeServiceRejectOrderFeeCorrection(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceRejectOrderFeeCorrectionParams,
  body: API.RejectOrderFeeCorrectionRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RejectOrderFeeCorrectionResponse>(
    `/api/v1/orders/${param0}/fee-correction-requests/${param1}/reject`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-correction-requests/${param1}/withdraw */
export async function orderFeeServiceWithdrawOrderFeeCorrection(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceWithdrawOrderFeeCorrectionParams,
  body: API.WithdrawOrderFeeCorrectionRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.WithdrawOrderFeeCorrectionResponse>(
    `/api/v1/orders/${param0}/fee-correction-requests/${param1}/withdraw`,
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

/** ResolveFeeExchangeRate 按费用发生日解析币种折本位币的公共参考汇率。 GET /api/v1/orders/${param0}/fee-exchange-rate */
export async function orderFeeServiceResolveFeeExchangeRate(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceResolveFeeExchangeRateParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ResolveFeeExchangeRateResponse>(
    `/api/v1/orders/${param0}/fee-exchange-rate`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** ListFeeOptions 获取费用录入所需的费用设置、计费单位、结算单位和币种候选项。 GET /api/v1/orders/${param0}/fee-options */
export async function orderFeeServiceListFeeOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceListFeeOptionsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListFeeOptionsResponse>(
    `/api/v1/orders/${param0}/fee-options`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** PreviewFeeQuantity 预览本订单数量来源与计算依据，确认后由调用方填入表单。 GET /api/v1/orders/${param0}/fee-quantity-preview */
export async function orderFeeServicePreviewFeeQuantity(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServicePreviewFeeQuantityParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.PreviewFeeQuantityResponse>(
    `/api/v1/orders/${param0}/fee-quantity-preview`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 GET /api/v1/orders/${param0}/fee-review */
export async function orderFeeServiceGetOrderFeeReview(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceGetOrderFeeReviewParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.GetOrderFeeReviewResponse>(
    `/api/v1/orders/${param0}/fee-review`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-review/approve */
export async function orderFeeServiceApproveOrderFeeReview(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceApproveOrderFeeReviewParams,
  body: API.ApproveOrderFeeReviewRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ApproveOrderFeeReviewResponse>(
    `/api/v1/orders/${param0}/fee-review/approve`,
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

/** 此处后端没有提供注释 GET /api/v1/orders/${param0}/fee-review/preview */
export async function orderFeeServicePreviewOrderFeeReview(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServicePreviewOrderFeeReviewParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.PreviewOrderFeeReviewResponse>(
    `/api/v1/orders/${param0}/fee-review/preview`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-review/reject */
export async function orderFeeServiceRejectOrderFeeReview(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceRejectOrderFeeReviewParams,
  body: API.RejectOrderFeeReviewRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.RejectOrderFeeReviewResponse>(
    `/api/v1/orders/${param0}/fee-review/reject`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-review/revoke */
export async function orderFeeServiceRevokeOrderFeeReview(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceRevokeOrderFeeReviewParams,
  body: API.RevokeOrderFeeReviewRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.RevokeOrderFeeReviewResponse>(
    `/api/v1/orders/${param0}/fee-review/revoke`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-review/submit */
export async function orderFeeServiceSubmitOrderFeeReview(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceSubmitOrderFeeReviewParams,
  body: API.SubmitOrderFeeReviewRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.SubmitOrderFeeReviewResponse>(
    `/api/v1/orders/${param0}/fee-review/submit`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-review/withdraw */
export async function orderFeeServiceWithdrawOrderFeeReview(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceWithdrawOrderFeeReviewParams,
  body: API.WithdrawOrderFeeReviewRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.WithdrawOrderFeeReviewResponse>(
    `/api/v1/orders/${param0}/fee-review/withdraw`,
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

/** ListOrderFeeSupplementRequests 订单维度分页读取补录申请。逐行授权
 （fee.read 或发起人本人或有效费用审核权限及组织范围）在领域层执行，不得被组织级
 fee.read 注解提前挡住，也不泄露无权申请。 GET /api/v1/orders/${param0}/fee-supplement-requests */
export async function orderFeeServiceListOrderFeeSupplementRequests(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceListOrderFeeSupplementRequestsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListOrderFeeSupplementRequestsResponse>(
    `/api/v1/orders/${param0}/fee-supplement-requests`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** CreateOrderFeeSupplement 基于有效整票费用批准发起受控正数应付成本补录。
 批准事件与内容版本由服务端锁内固化；同订单只允许一条待审补录，并提醒
 有效费用审核权限且组织范围匹配的候选人。 POST /api/v1/orders/${param0}/fee-supplement-requests */
export async function orderFeeServiceCreateOrderFeeSupplement(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceCreateOrderFeeSupplementParams,
  body: API.CreateOrderFeeSupplementRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.CreateOrderFeeSupplementResponse>(
    `/api/v1/orders/${param0}/fee-supplement-requests`,
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

/** PreviewOrderFeeSupplementApproval 只读复核申请及审批资格，估算通过后的订单费用毛利。 GET /api/v1/orders/${param0}/fee-supplement-requests/${param1}/approval-preview */
export async function orderFeeServicePreviewOrderFeeSupplementApproval(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServicePreviewOrderFeeSupplementApprovalParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.PreviewOrderFeeSupplementApprovalResponse>(
    `/api/v1/orders/${param0}/fee-supplement-requests/${param1}/approval-preview`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** ApproveOrderFeeSupplement 审批通过补录申请：复核固化的有效费用批准与整票利润、逐人提成预览，
 在同一事务创建补录费用、DECREASE+CONFIRMED 冲减并逐员工通知。 POST /api/v1/orders/${param0}/fee-supplement-requests/${param1}/approve */
export async function orderFeeServiceApproveOrderFeeSupplement(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceApproveOrderFeeSupplementParams,
  body: API.ApproveOrderFeeSupplementRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.ApproveOrderFeeSupplementResponse>(
    `/api/v1/orders/${param0}/fee-supplement-requests/${param1}/approve`,
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

/** CancelApprovedOrderFeeSupplement 专用撤销已批准补录生成的费用：仅限最新有效、
 未被有效账单关联占用且未产生确认冲减的补录；同事务物理删除费用并撤销
 当前整票批准，APPROVED 申请事实及历史账单行快照保留。 POST /api/v1/orders/${param0}/fee-supplement-requests/${param1}/cancel-fee */
export async function orderFeeServiceCancelApprovedOrderFeeSupplement(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceCancelApprovedOrderFeeSupplementParams,
  body: API.CancelApprovedOrderFeeSupplementRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.CancelApprovedOrderFeeSupplementResponse>(
    `/api/v1/orders/${param0}/fee-supplement-requests/${param1}/cancel-fee`,
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

/** RejectOrderFeeSupplement 驳回补录申请：只写申请终态与审计，不产生费用或调整。 POST /api/v1/orders/${param0}/fee-supplement-requests/${param1}/reject */
export async function orderFeeServiceRejectOrderFeeSupplement(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceRejectOrderFeeSupplementParams,
  body: API.RejectOrderFeeSupplementRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RejectOrderFeeSupplementResponse>(
    `/api/v1/orders/${param0}/fee-supplement-requests/${param1}/reject`,
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

/** WithdrawOrderFeeSupplement 发起人撤回本人仍处于 PENDING 的申请；与审批并发时
 在同一申请行锁内竞争，只有先提交的一方成功，撤回成功不产生费用或调整。 POST /api/v1/orders/${param0}/fee-supplement-requests/${param1}/withdraw */
export async function orderFeeServiceWithdrawOrderFeeSupplement(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceWithdrawOrderFeeSupplementParams,
  body: API.WithdrawOrderFeeSupplementRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.WithdrawOrderFeeSupplementResponse>(
    `/api/v1/orders/${param0}/fee-supplement-requests/${param1}/withdraw`,
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

/** 此处后端没有提供注释 GET /api/v1/orders/${param0}/fee-tag-options */
export async function orderFeeServiceListOrderFeeTagOptions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceListOrderFeeTagOptionsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListOrderFeeTagOptionsResponse>(
    `/api/v1/orders/${param0}/fee-tag-options`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-tags/batch-assign */
export async function orderFeeServiceBatchAssignOrderFeeTags(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceBatchAssignOrderFeeTagsParams,
  body: API.BatchAssignOrderFeeTagsRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.BatchAssignOrderFeeTagsResponse>(
    `/api/v1/orders/${param0}/fee-tags/batch-assign`,
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

/** 此处后端没有提供注释 POST /api/v1/orders/${param0}/fee-tags/batch-remove */
export async function orderFeeServiceBatchRemoveOrderFeeTags(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceBatchRemoveOrderFeeTagsParams,
  body: API.BatchRemoveOrderFeeTagsRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.BatchRemoveOrderFeeTagsResponse>(
    `/api/v1/orders/${param0}/fee-tags/batch-remove`,
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

/** ListFees 获取指定订单的费用列表。 GET /api/v1/orders/${param0}/fees */
export async function orderFeeServiceListFees(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceListFeesParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListFeesResponse>(`/api/v1/orders/${param0}/fees`, {
    method: "GET",
    params: { ...queryParams },
    ...(options || {}),
  });
}

/** AddFee 录入订单费用，总金额由服务端按数量乘单价精确计算。 POST /api/v1/orders/${param0}/fees */
export async function orderFeeServiceAddFee(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceAddFeeParams,
  body: API.AddFeeRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.AddFeeResponse>(`/api/v1/orders/${param0}/fees`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}

/** UpdateFee 更新订单费用，总金额由服务端重新精确计算；未被有效账单关联占用
 （未建账）的费用可全量维护，已建账费用仅允许按财务策略修改并同步草稿账单。 PUT /api/v1/orders/${param0}/fees/${param1} */
export async function orderFeeServiceUpdateFee(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceUpdateFeeParams,
  body: API.UpdateFeeRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.UpdateFeeResponse>(
    `/api/v1/orders/${param0}/fees/${param1}`,
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

/** RemoveFee 物理删除未被有效账单关联占用（未建账）的订单费用；补录生成的费用
 必须通过补录申请专用撤销删除，历史账单行快照保留、来源引用置空。 DELETE /api/v1/orders/${param0}/fees/${param1} */
export async function orderFeeServiceRemoveFee(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceRemoveFeeParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RemoveFeeResponse>(
    `/api/v1/orders/${param0}/fees/${param1}`,
    {
      method: "DELETE",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** BulkRemoveOrderFees 批量物理删除订单未建账费用：整批单一事务，被有效账单
 关联占用、补录来源、版本冲突或越订单任一不满足时整批回滚并返回具体费用与
 原因；费用标签关联随删除级联清理，历史账单行快照保留、来源引用置空。 POST /api/v1/orders/${param0}/fees/bulk-remove */
export async function orderFeeServiceBulkRemoveOrderFees(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceBulkRemoveOrderFeesParams,
  body: API.BulkRemoveOrderFeesRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.BulkRemoveOrderFeesResponse>(
    `/api/v1/orders/${param0}/fees/bulk-remove`,
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

/** BulkUpdateOrderFees 批量定向修改订单未被有效账单关联占用的费用：每次仅修改
 结算单位或费用发生时间之一，整批单一事务，任一行版本冲突、已建账、越订单或
 汇率缺失时整批回滚并返回具体费用与原因，不允许部分成功。 POST /api/v1/orders/${param0}/fees/bulk-update */
export async function orderFeeServiceBulkUpdateOrderFees(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceBulkUpdateOrderFeesParams,
  body: API.BulkUpdateOrderFeesRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.BulkUpdateOrderFeesResponse>(
    `/api/v1/orders/${param0}/fees/bulk-update`,
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
