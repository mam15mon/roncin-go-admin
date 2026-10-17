export type ApiListResponse<T> = {
  data?: T[];
};

export type ApiPageResponse<T> = ApiListResponse<T> & {
  total?: number | string;
};

export type ApiTableResponse<T> = ApiPageResponse<T> & {
  success?: boolean;
};

/**
 * 写接口必须收到成功报文；实体接口另外验证返回的实体标识。
 * 标签批量接口只有计数，无 success 字段；零计数会被 protojson 省略为 {}。
 */
export function isWriteResponse(
  response: unknown,
  requireEntityId: boolean | 'count' = false,
): boolean {
  if (!response || typeof response !== 'object') return false;
  const envelope = response as { success?: boolean; data?: { id?: string } };
  if (envelope.success === false) return false;
  if (requireEntityId === 'count') return true;
  return requireEntityId
    ? Boolean(envelope.data?.id)
    : envelope.success === true;
}

export function unwrapList<T>(response: ApiListResponse<T>): T[] {
  return response.data ?? [];
}

/**
 * 请求层在错误被全局处理器消费后会 resolve undefined（见 requestClient）。
 * 列表类调用用它把 undefined 转成明确业务错误，避免 unwrapList 在
 * undefined 上读 .data 抛 TypeError 文案直达用户。
 */
export function ensureListResponse<T>(
  response: ApiListResponse<T> | undefined,
  message: string,
): ApiListResponse<T> {
  if (!response) {
    throw new Error(message);
  }
  return response;
}

export function unwrapPage<T>(response: ApiPageResponse<T>): {
  data: T[];
  total: number;
} {
  return {
    data: unwrapList(response),
    total: Number(response.total ?? 0),
  };
}

export function toTableRequest<T>(response: ApiTableResponse<T>): {
  data: T[];
  success: boolean;
  total?: number;
} {
  const result: { data: T[]; success: boolean; total?: number } = {
    data: unwrapList(response),
    success: response.success ?? true,
  };
  if (response.total !== undefined) {
    result.total = Number(response.total);
  }
  return result;
}
