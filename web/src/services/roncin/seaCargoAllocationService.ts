// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** GetSeaCargoAllocation 获取海运箱货分配聚合信息。 GET /api/v1/orders/${param0}/sea-cargo-allocation */
export async function seaCargoAllocationServiceGetSeaCargoAllocation(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaCargoAllocationServiceGetSeaCargoAllocationParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.GetSeaCargoAllocationResponse>(
    `/api/v1/orders/${param0}/sea-cargo-allocation`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** ConfirmSeaCargoAllocation 确认海运箱货分配（严格守恒门禁）。 POST /api/v1/orders/${param0}/sea-cargo-allocation/confirm */
export async function seaCargoAllocationServiceConfirmSeaCargoAllocation(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaCargoAllocationServiceConfirmSeaCargoAllocationParams,
  body: API.ConfirmSeaCargoAllocationRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ConfirmSeaCargoAllocationResponse>(
    `/api/v1/orders/${param0}/sea-cargo-allocation/confirm`,
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

/** SaveSeaCargoAllocationDraft 全量替换保存箱货分配草稿。 PUT /api/v1/orders/${param0}/sea-cargo-allocation/draft */
export async function seaCargoAllocationServiceSaveSeaCargoAllocationDraft(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaCargoAllocationServiceSaveSeaCargoAllocationDraftParams,
  body: API.SaveSeaCargoAllocationDraftRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.SaveSeaCargoAllocationDraftResponse>(
    `/api/v1/orders/${param0}/sea-cargo-allocation/draft`,
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

/** ApplySeaHouseBillAllocationSummary HOUSE 下用分配汇总填入目标 HBL 提单内容。 POST /api/v1/orders/${param0}/sea-cargo-allocation/house-bills/${param1}/apply-summary */
export async function seaCargoAllocationServiceApplySeaHouseBillAllocationSummary(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaCargoAllocationServiceApplySeaHouseBillAllocationSummaryParams,
  body: API.ApplySeaHouseBillAllocationSummaryRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, houseBillId: param1, ...queryParams } = params;
  return request<API.ApplySeaHouseBillAllocationSummaryResponse>(
    `/api/v1/orders/${param0}/sea-cargo-allocation/house-bills/${param1}/apply-summary`,
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

/** ApplySeaOrderCargoSummaryToMasterBill DIRECT 下用操作票货物汇总填入 MBL 提单内容。 POST /api/v1/orders/${param0}/sea-cargo-allocation/master-bill/apply-cargo-summary */
export async function seaCargoAllocationServiceApplySeaOrderCargoSummaryToMasterBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaCargoAllocationServiceApplySeaOrderCargoSummaryToMasterBillParams,
  body: API.ApplySeaOrderCargoSummaryToMasterBillRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ApplySeaOrderCargoSummaryToMasterBillResponse>(
    `/api/v1/orders/${param0}/sea-cargo-allocation/master-bill/apply-cargo-summary`,
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

/** WithdrawSeaCargoAllocation 撤回海运箱货分配确认。 POST /api/v1/orders/${param0}/sea-cargo-allocation/withdraw */
export async function seaCargoAllocationServiceWithdrawSeaCargoAllocation(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaCargoAllocationServiceWithdrawSeaCargoAllocationParams,
  body: API.WithdrawSeaCargoAllocationRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.WithdrawSeaCargoAllocationResponse>(
    `/api/v1/orders/${param0}/sea-cargo-allocation/withdraw`,
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
