import { describe, expect, it } from 'vitest';
import { statusOptions } from './billConstants';

describe('账单状态选项', () => {
  it('仅包含后端账单状态机支持的状态', () => {
    expect(Object.keys(statusOptions)).toEqual([
      'DRAFT',
      'CONFIRMED',
      'CANCELLED',
    ]);
  });
});
