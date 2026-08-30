export const DEFAULT_REQUEST_TIMEOUT = 30_000;
export const LONG_REQUEST_TIMEOUT = 120_000;

export const longRequestOptions = {
  timeout: LONG_REQUEST_TIMEOUT,
} as const;
