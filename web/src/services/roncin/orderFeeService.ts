// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

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
 （未建账）且订单未锁定的费用可维护；已建账费用必须按权限回退账务。 PUT /api/v1/orders/${param0}/fees/${param1} */
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
 原因；历史账单行快照保留、来源引用置空。 POST /api/v1/orders/${param0}/fees/bulk-remove */
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

/** SaveOrderFees 按单侧（应收或应付）整表保存订单费用的新增与修改：单一事务，
 逐行完整校验后才写入，任一行失败整体回滚；不含删除，不连带另一侧。
 新增行幂等键由 request_key 与 client_row_id 派生，响应丢失后沿用同一
 request_key 重试不重复新增；半应用情形返回 409。新增行要求 fee.create，
 修改行要求 fee.update，在服务端分别校验。行级错误通过错误 metadata 的
 client_row_id、field、reason 定位。 POST /api/v1/orders/${param0}/fees:save */
export async function orderFeeServiceSaveOrderFees(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderFeeServiceSaveOrderFeesParams,
  body: API.SaveOrderFeesRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.SaveOrderFeesResponse>(
    `/api/v1/orders/${param0}/fees:save`,
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
