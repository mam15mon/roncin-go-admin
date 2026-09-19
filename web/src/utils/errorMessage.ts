/** 从未知错误对象中安全取出可展示文案（请求错误、未知异常的统一兜底）。 */
export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const messageValue = (error as { message?: unknown }).message;
    if (typeof messageValue === 'string' && messageValue) return messageValue;
  }
  return fallback;
}
