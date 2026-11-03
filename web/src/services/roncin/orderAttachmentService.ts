// @ts-ignore
/* eslint-disable */
import { request } from "@/utils/requestClient";

/** ListAttachments 获取指定订单的附件列表。 GET /api/v1/orders/${param0}/attachments */
export async function orderAttachmentServiceListAttachments(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAttachmentServiceListAttachmentsParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.ListAttachmentsResponse>(
    `/api/v1/orders/${param0}/attachments`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** RegisterAttachment 注册订单附件。 POST /api/v1/orders/${param0}/attachments */
export async function orderAttachmentServiceRegisterAttachment(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAttachmentServiceRegisterAttachmentParams,
  body: API.RegisterAttachmentRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.RegisterAttachmentResponse>(
    `/api/v1/orders/${param0}/attachments`,
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

/** RemoveAttachmentReference 解除订单附件引用。 DELETE /api/v1/orders/${param0}/attachments/${param1} */
export async function orderAttachmentServiceRemoveAttachmentReference(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAttachmentServiceRemoveAttachmentReferenceParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.RemoveAttachmentReferenceResponse>(
    `/api/v1/orders/${param0}/attachments/${param1}`,
    {
      method: "DELETE",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** GetAttachmentAccess 获取短期附件访问链接。 GET /api/v1/orders/${param0}/attachments/${param1}/access */
export async function orderAttachmentServiceGetAttachmentAccess(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAttachmentServiceGetAttachmentAccessParams,
  options?: { [key: string]: any }
) {
  const { orderId: param0, id: param1, ...queryParams } = params;
  return request<API.GetAttachmentAccessResponse>(
    `/api/v1/orders/${param0}/attachments/${param1}/access`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** PrepareAttachmentUpload 为当前订单准备附件上传。 POST /api/v1/orders/${param0}/attachments/upload */
export async function orderAttachmentServicePrepareAttachmentUpload(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.OrderAttachmentServicePrepareAttachmentUploadParams,
  body: API.PrepareAttachmentUploadRequest,
  options?: { [key: string]: any }
) {
  const { orderId: param0, ...queryParams } = params;
  return request<API.PrepareAttachmentUploadResponse>(
    `/api/v1/orders/${param0}/attachments/upload`,
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
