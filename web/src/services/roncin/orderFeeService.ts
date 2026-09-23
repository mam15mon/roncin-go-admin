// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

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

/** ListOrderFeeSupplementRequests 订单维度分页读取补录申请。逐行授权
 （fee.read 或发起人本人或实时 lock grant）在领域层执行，不得被组织级
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

/** CreateOrderFeeSupplement 在业务锁或提成净额财务锁成立期间发起锁后应付费用补录申请。
 锁类型、锁代次与财务锁证据均由服务端在 Order 行锁内判定固化；双锁均不存在时
 稳定拒绝并引导普通费用新增。 POST /api/v1/orders/${param0}/fee-supplement-requests */
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

/** ApproveOrderFeeSupplement 审批通过补录申请：按申请固化的锁依据复核原始依据，
 在同一事务创建 UNBILLED 补录费用、DECREASE+DRAFT 冲减建议并逐员工通知。 POST /api/v1/orders/${param0}/fee-supplement-requests/${param1}/approve */
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

/** CancelApprovedOrderFeeSupplement 专用作废已批准补录生成的费用：仅限最新有效、
 UNBILLED、无活动账单行且关联冲减从未确认/扣回的补录；费用与仍为 DRAFT 的
 关联冲减建议在同一事务转为 CANCELLED，APPROVED 申请保持不变。 POST /api/v1/orders/${param0}/fee-supplement-requests/${param1}/cancel-fee */
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

/** UpdateFee 更新订单费用，总金额由服务端重新精确计算；未建账费用可全量维护，
 已建账费用仅允许按财务策略修改并同步草稿账单。 PUT /api/v1/orders/${param0}/fees/${param1} */
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

/** RemoveFee 作废尚未进入账单的订单费用，并保留完整历史数据。 DELETE /api/v1/orders/${param0}/fees/${param1} */
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

/** BulkRemoveOrderFees 批量删除订单未建账费用：整批单一事务，被未取消账单
 占用、版本冲突或越订单任一不满足时整批回滚，费用标签关联随删除级联清理。 POST /api/v1/orders/${param0}/fees/bulk-remove */
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

/** BulkUpdateOrderFees 批量定向修改订单未建账费用：每次仅修改结算单位或
 费用发生时间之一，整批单一事务，任一行版本冲突、状态不符、越订单或
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
