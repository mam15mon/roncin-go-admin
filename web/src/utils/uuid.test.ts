import { afterEach, describe, expect, it, vi } from 'vitest';
import { generateUUID } from './uuid';

const UUID_V4_REGEX =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

describe('generateUUID', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('在默认环境下生成合法的 UUID v4', () => {
    const uuid = generateUUID();
    expect(uuid).toMatch(UUID_V4_REGEX);
  });

  it('在无 crypto.randomUUID 的非安全上下文环境下仍能生成合法 UUID v4', () => {
    const originalCrypto = globalThis.crypto;
    Object.defineProperty(globalThis, 'crypto', {
      value: {
        getRandomValues: originalCrypto?.getRandomValues.bind(originalCrypto),
      },
      configurable: true,
      writable: true,
    });

    try {
      const uuid = generateUUID();
      expect(uuid).toMatch(UUID_V4_REGEX);
    } finally {
      Object.defineProperty(globalThis, 'crypto', {
        value: originalCrypto,
        configurable: true,
        writable: true,
      });
    }
  });

  it('在 crypto 完全缺失的极端环境下降级使用 Math.random 仍生成合法 UUID v4 格式', () => {
    const originalCrypto = globalThis.crypto;
    Object.defineProperty(globalThis, 'crypto', {
      value: undefined,
      configurable: true,
      writable: true,
    });

    try {
      const uuid = generateUUID();
      expect(uuid).toMatch(UUID_V4_REGEX);
    } finally {
      Object.defineProperty(globalThis, 'crypto', {
        value: originalCrypto,
        configurable: true,
        writable: true,
      });
    }
  });

  it('连续多次调用生成不重复的 UUID', () => {
    const set = new Set<string>();
    for (let i = 0; i < 100; i++) {
      set.add(generateUUID());
    }
    expect(set.size).toBe(100);
  });
});
