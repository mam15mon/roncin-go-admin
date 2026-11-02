import { vi } from 'vitest';

// 原有登录流程用例聚焦扫码/密码；通行密钥原生仪式由能力专项用例和浏览器验收覆盖。
vi.mock('@/features/auth/passkeys', () => ({
  formatPasskeyError: (error: unknown, fallback: string) =>
    error instanceof Error ? error.message : fallback,
  usePasskeyLogin: () => ({
    canUse: false,
    reason: '当前环境未启用通行密钥',
    enrollAfterLogin: false,
    setEnrollAfterLogin: vi.fn(),
    login: vi.fn(),
    isPending: false,
    isPreparing: false,
    prepareAgain: vi.fn(),
    error: undefined,
  }),
  usePasskeyEnrollment: () => ({
    completeLogin: (onContinue: () => void) => onContinue(),
    enrollmentPrompt: null,
    isEnrollmentOpen: false,
  }),
}));
