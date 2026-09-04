// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** GetSeaOrderDocuments 获取海运单证聚合信息。 GET /api/v1/orders/${param0}/sea-documents */
export async function seaDocumentServiceGetSeaOrderDocuments(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceGetSeaOrderDocumentsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.GetSeaOrderDocumentsResponse>(
    `/api/v1/orders/${param0}/sea-documents`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** ExecuteSeaDocumentAmendment 发布改单版本。 POST /api/v1/orders/${param0}/sea-documents/amendments */
export async function seaDocumentServiceExecuteSeaDocumentAmendment(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceExecuteSeaDocumentAmendmentParams,
  body: API.ExecuteSeaDocumentAmendmentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ExecuteSeaDocumentAmendmentResponse>(
    `/api/v1/orders/${param0}/sea-documents/amendments`,
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

/** PreviewSeaDocumentAmendment 基于当前不可变版本重算改单差异与影响。 POST /api/v1/orders/${param0}/sea-documents/amendments/preview */
export async function seaDocumentServicePreviewSeaDocumentAmendment(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServicePreviewSeaDocumentAmendmentParams,
  body: API.PreviewSeaDocumentAmendmentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.PreviewSeaDocumentAmendmentResponse>(
    `/api/v1/orders/${param0}/sea-documents/amendments/preview`,
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

/** CancelSeaOrderDirect 取消直单标记，回到未确定状态。 POST /api/v1/orders/${param0}/sea-documents/cancel-direct */
export async function seaDocumentServiceCancelSeaOrderDirect(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceCancelSeaOrderDirectParams,
  body: API.CancelSeaOrderDirectRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.CancelSeaOrderDirectResponse>(
    `/api/v1/orders/${param0}/sea-documents/cancel-direct`,
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

/** ListSeaDocumentEvents 分页读取改单、作废与 Switch 历史。 GET /api/v1/orders/${param0}/sea-documents/events */
export async function seaDocumentServiceListSeaDocumentEvents(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceListSeaDocumentEventsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListSeaDocumentEventsResponse>(
    `/api/v1/orders/${param0}/sea-documents/events`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** ExecuteSeaHouseBillSwitch 在同订单和当前 MBL 下建立真实替代 HBL。 POST /api/v1/orders/${param0}/sea-documents/house-bill-switches */
export async function seaDocumentServiceExecuteSeaHouseBillSwitch(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceExecuteSeaHouseBillSwitchParams,
  body: API.ExecuteSeaHouseBillSwitchRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ExecuteSeaHouseBillSwitchResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bill-switches`,
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

/** PreviewSeaHouseBillSwitch 预览 HBL Switch 的新旧差异与影响。 POST /api/v1/orders/${param0}/sea-documents/house-bill-switches/preview */
export async function seaDocumentServicePreviewSeaHouseBillSwitch(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServicePreviewSeaHouseBillSwitchParams,
  body: API.PreviewSeaHouseBillSwitchRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.PreviewSeaHouseBillSwitchResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bill-switches/preview`,
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

/** AddSeaHouseBill 添加海运分单（HBL）。 POST /api/v1/orders/${param0}/sea-documents/house-bills */
export async function seaDocumentServiceAddSeaHouseBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceAddSeaHouseBillParams,
  body: API.AddSeaHouseBillRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.AddSeaHouseBillResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bills`,
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

/** UpdateSeaHouseBill 更新海运分单。 PUT /api/v1/orders/${param0}/sea-documents/house-bills/${param1} */
export async function seaDocumentServiceUpdateSeaHouseBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceUpdateSeaHouseBillParams,
  body: API.UpdateSeaHouseBillRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.UpdateSeaHouseBillResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bills/${param1}`,
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

/** RemoveSeaHouseBill 移除海运分单。 DELETE /api/v1/orders/${param0}/sea-documents/house-bills/${param1} */
export async function seaDocumentServiceRemoveSeaHouseBill(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceRemoveSeaHouseBillParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RemoveSeaHouseBillResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bills/${param1}`,
    {
      method: "DELETE",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** ListSeaHouseBillVersions 分页读取一张 HBL 的不可变版本。 GET /api/v1/orders/${param0}/sea-documents/house-bills/${param1}/versions */
export async function seaDocumentServiceListSeaHouseBillVersions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceListSeaHouseBillVersionsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, houseBillId: param1, ...queryParams } = params;
  return request<API.ListSeaHouseBillVersionsResponse>(
    `/api/v1/orders/${param0}/sea-documents/house-bills/${param1}/versions`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** MarkSeaOrderDirect 明确标记海运订单为直单。 POST /api/v1/orders/${param0}/sea-documents/mark-direct */
export async function seaDocumentServiceMarkSeaOrderDirect(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceMarkSeaOrderDirectParams,
  body: API.MarkSeaOrderDirectRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.MarkSeaOrderDirectResponse>(
    `/api/v1/orders/${param0}/sea-documents/mark-direct`,
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

/** UpdateSeaMasterBillContent 更新共享 MBL 提单内容。 PUT /api/v1/orders/${param0}/sea-documents/master-bill-content */
export async function seaDocumentServiceUpdateSeaMasterBillContent(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceUpdateSeaMasterBillContentParams,
  body: API.UpdateSeaMasterBillContentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.UpdateSeaMasterBillContentResponse>(
    `/api/v1/orders/${param0}/sea-documents/master-bill-content`,
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

/** ListSeaMasterBillVersions 分页读取当前订单共享 MBL 的不可变版本。 GET /api/v1/orders/${param0}/sea-documents/master-bill/versions */
export async function seaDocumentServiceListSeaMasterBillVersions(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceListSeaMasterBillVersionsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListSeaMasterBillVersionsResponse>(
    `/api/v1/orders/${param0}/sea-documents/master-bill/versions`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** GetSeaDocumentVersion 读取一条不可变版本，不回读当前工作字段。 GET /api/v1/orders/${param0}/sea-documents/versions/${param1} */
export async function seaDocumentServiceGetSeaDocumentVersion(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceGetSeaDocumentVersionParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, versionId: param1, ...queryParams } = params;
  return request<API.GetSeaDocumentVersionResponse>(
    `/api/v1/orders/${param0}/sea-documents/versions/${param1}`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** ExecuteSeaDocumentVoid 作废单证身份并追加不可变版本与事件。 POST /api/v1/orders/${param0}/sea-documents/voids */
export async function seaDocumentServiceExecuteSeaDocumentVoid(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServiceExecuteSeaDocumentVoidParams,
  body: API.ExecuteSeaDocumentVoidRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ExecuteSeaDocumentVoidResponse>(
    `/api/v1/orders/${param0}/sea-documents/voids`,
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

/** PreviewSeaDocumentVoid 基于当前不可变版本预览作废影响。 POST /api/v1/orders/${param0}/sea-documents/voids/preview */
export async function seaDocumentServicePreviewSeaDocumentVoid(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.SeaDocumentServicePreviewSeaDocumentVoidParams,
  body: API.PreviewSeaDocumentVoidRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.PreviewSeaDocumentVoidResponse>(
    `/api/v1/orders/${param0}/sea-documents/voids/preview`,
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
