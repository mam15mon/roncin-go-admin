// @ts-ignore
/* eslint-disable */
import { request } from "@umijs/max";

/** 此处后端没有提供注释 GET /api/v1/partners */
export async function partnerServiceListPartners(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceListPartnersParams,
  options?: { [key: string]: any }
) {
  return request<API.PartnerListReply>("/api/v1/partners", {
    method: "GET",
    params: {
      ...params,
    },
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 POST /api/v1/partners */
export async function partnerServiceCreatePartner(
  body: API.CreatePartnerRequest,
  options?: { [key: string]: any }
) {
  return request<API.PartnerReply>("/api/v1/partners", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    ...(options || {}),
  });
}

/** 此处后端没有提供注释 PUT /api/v1/partners/${param0} */
export async function partnerServiceUpdatePartner(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceUpdatePartnerParams,
  body: API.UpdatePartnerRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.PartnerReply>(`/api/v1/partners/${param0}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    params: { ...queryParams },
    data: body,
    ...(options || {}),
  });
}
