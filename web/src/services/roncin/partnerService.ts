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

/** 此处后端没有提供注释 GET /api/v1/partners/${param0} */
export async function partnerServiceGetPartner(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceGetPartnerParams,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.PartnerReply>(`/api/v1/partners/${param0}`, {
    method: "GET",
    params: { ...queryParams },
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

/** 此处后端没有提供注释 GET /api/v1/partners/${param0}/accounts */
export async function partnerServiceListPartnerAccounts(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceListPartnerAccountsParams,
  options?: { [key: string]: any }
) {
  const { partnerId: param0, ...queryParams } = params;
  return request<API.PartnerAccountListReply>(
    `/api/v1/partners/${param0}/accounts`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/partners/${param0}/accounts */
export async function partnerServiceCreatePartnerAccount(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceCreatePartnerAccountParams,
  body: API.CreatePartnerAccountRequest,
  options?: { [key: string]: any }
) {
  const { partnerId: param0, ...queryParams } = params;
  return request<API.PartnerAccountReply>(
    `/api/v1/partners/${param0}/accounts`,
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

/** 此处后端没有提供注释 PUT /api/v1/partners/${param0}/accounts/${param1} */
export async function partnerServiceUpdatePartnerAccount(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceUpdatePartnerAccountParams,
  body: API.UpdatePartnerAccountRequest,
  options?: { [key: string]: any }
) {
  const { partnerId: param0, id: param1, ...queryParams } = params;
  return request<API.PartnerAccountReply>(
    `/api/v1/partners/${param0}/accounts/${param1}`,
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

/** 此处后端没有提供注释 GET /api/v1/partners/${param0}/contracts */
export async function partnerServiceListPartnerContracts(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceListPartnerContractsParams,
  options?: { [key: string]: any }
) {
  const { partnerId: param0, ...queryParams } = params;
  return request<API.PartnerContractListReply>(
    `/api/v1/partners/${param0}/contracts`,
    {
      method: "GET",
      params: {
        ...queryParams,
      },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/partners/${param0}/contracts */
export async function partnerServiceCreatePartnerContract(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceCreatePartnerContractParams,
  body: API.CreatePartnerContractRequest,
  options?: { [key: string]: any }
) {
  const { partnerId: param0, ...queryParams } = params;
  return request<API.PartnerContractReply>(
    `/api/v1/partners/${param0}/contracts`,
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

/** 此处后端没有提供注释 PUT /api/v1/partners/${param0}/contracts/${param1} */
export async function partnerServiceUpdatePartnerContract(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceUpdatePartnerContractParams,
  body: API.UpdatePartnerContractRequest,
  options?: { [key: string]: any }
) {
  const { partnerId: param0, id: param1, ...queryParams } = params;
  return request<API.PartnerContractReply>(
    `/api/v1/partners/${param0}/contracts/${param1}`,
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

/** 此处后端没有提供注释 GET /api/v1/partners/${param0}/roles/${param1}/settlement-rules */
export async function partnerServiceListPartnerSettlementRules(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceListPartnerSettlementRulesParams,
  options?: { [key: string]: any }
) {
  const { partnerId: param0, roleType: param1, ...queryParams } = params;
  return request<API.PartnerSettlementRuleListReply>(
    `/api/v1/partners/${param0}/roles/${param1}/settlement-rules`,
    {
      method: "GET",
      params: { ...queryParams },
      ...(options || {}),
    }
  );
}

/** 此处后端没有提供注释 POST /api/v1/partners/${param0}/roles/${param1}/settlement-rules */
export async function partnerServiceCreatePartnerSettlementRule(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceCreatePartnerSettlementRuleParams,
  body: API.CreatePartnerSettlementRuleRequest,
  options?: { [key: string]: any }
) {
  const { partnerId: param0, roleType: param1, ...queryParams } = params;
  return request<API.PartnerSettlementRuleReply>(
    `/api/v1/partners/${param0}/roles/${param1}/settlement-rules`,
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

/** 此处后端没有提供注释 PUT /api/v1/partners/${param0}/roles/${param1}/settlement-rules/${param2} */
export async function partnerServiceUpdatePartnerSettlementRule(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceUpdatePartnerSettlementRuleParams,
  body: API.UpdatePartnerSettlementRuleRequest,
  options?: { [key: string]: any }
) {
  const {
    partnerId: param0,
    roleType: param1,
    id: param2,
    ...queryParams
  } = params;
  return request<API.PartnerSettlementRuleReply>(
    `/api/v1/partners/${param0}/roles/${param1}/settlement-rules/${param2}`,
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

/** 此处后端没有提供注释 POST /api/v1/partners/${param0}/supplier-blacklist */
export async function partnerServiceSetSupplierBlacklist(
  // 叠加生成的Param类型 (非body参数swagger默认没有生成对象)
  params: API.PartnerServiceSetSupplierBlacklistParams,
  body: API.SetSupplierBlacklistRequest,
  options?: { [key: string]: any }
) {
  const { id: param0, ...queryParams } = params;
  return request<API.PartnerReply>(
    `/api/v1/partners/${param0}/supplier-blacklist`,
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
