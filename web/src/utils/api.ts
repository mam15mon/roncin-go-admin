export type ApiListResponse<T> = {
  data?: T[];
};

export type ApiPageResponse<T> = ApiListResponse<T> & {
  total?: number | string;
};

export type ApiTableResponse<T> = ApiPageResponse<T> & {
  success?: boolean;
};

export function unwrapList<T>(response: ApiListResponse<T>): T[] {
  return response.data ?? [];
}

export function unwrapPage<T>(response: ApiPageResponse<T>): { data: T[]; total: number } {
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
