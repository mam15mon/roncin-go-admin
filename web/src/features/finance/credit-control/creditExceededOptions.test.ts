import { describe, expect, it } from 'vitest';
import {
  type CreditCandidateOption,
  disableCreditExceededOptions,
} from './creditExceededOptions';

describe('信用超额候选禁用', () => {
  it('直接干预模式下禁用超额客户候选，仅提醒模式原样返回', () => {
    type Option = CreditCandidateOption & { isCasual?: boolean };

    const options: Option[] = [
      { label: '正常客户 (CUS001)', value: 'p1' },
      { label: '超额客户 (CUS002)', value: 'p2', creditExceeded: true },
      { label: '散客 (CUS003)', value: 'p3', isCasual: true },
    ];

    // 仅提醒模式（默认）：不做任何禁用。
    expect(disableCreditExceededOptions(options, false)).toEqual(options);

    // 直接干预模式：仅超额候选被置灰，其余不受影响；label 保持纯净。
    const disabled = disableCreditExceededOptions(options, true);
    expect(disabled[0].disabled).toBeUndefined();
    expect(disabled[1].disabled).toBe(true);
    expect(disabled[1].label).toBe('超额客户 (CUS002)');
    expect(disabled[2].disabled).toBeUndefined();
  });
});
