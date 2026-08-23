// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

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
