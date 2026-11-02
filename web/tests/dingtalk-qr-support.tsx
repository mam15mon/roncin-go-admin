import { act } from '@testing-library/react';
import { vi } from 'vitest';

export function qrConfig(state = 'state-1', duration = 300000) {
  return {
    data: {
      enabled: true,
      state,
      qrCodeUrl: `https://login.dingtalk.com/oauth2/qr_confirm.htm?code=qr-${state}`,
      expiresAt: new Date(Date.now() + duration).toISOString(),
    },
  };
}

export function deferred<T>() {
  let resolve: (value: T) => void = () => {};
  let reject: (error: Error) => void = () => {};
  const promise = new Promise<T>((finish, fail) => {
    resolve = finish;
    reject = fail;
  });
  return { promise, resolve, reject };
}

export async function settleQR(milliseconds = 1) {
  await act(async () => vi.advanceTimersByTimeAsync(milliseconds));
}
