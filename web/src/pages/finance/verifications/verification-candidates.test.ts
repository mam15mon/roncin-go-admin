import { describe, expect, it } from 'vitest';
import {
  verificationCandidatePage,
  verificationSelectedSnapshots,
} from './verification-candidates';

describe('核销候选报文与已选快照', () => {
  it('识别ProtoJSON空列表和正常分页，不吞请求失败或真实契约缺失', () => {
    expect(
      verificationCandidatePage({ success: true }, '候选加载失败'),
    ).toEqual({ data: [], total: 0, success: true });
    expect(
      verificationCandidatePage(
        { data: [{ id: 'a', organizationId: 'company-a' }], total: '205' },
        '候选加载失败',
      ).total,
    ).toBe(205);
    for (const response of [
      undefined,
      { success: false },
      {},
      { data: [{ id: 'a', organizationId: 'company-a' }] },
      { data: [], total: '-1' },
      { data: [{ organizationId: 'company-a' }], total: '1' },
    ]) {
      expect(() => verificationCandidatePage(response, '候选加载失败')).toThrow(
        '候选加载失败',
      );
    }
  });
  it('换页只保存已选记录，移除后不复活，缺来源快照明确失败', () => {
    const first = { id: 'first', unverifiedAmount: '60' };
    const second = { id: 'second', unverifiedAmount: '40' };
    expect(
      verificationSelectedSnapshots(['first', 'second'], [second], [first]),
    ).toEqual([first, second]);
    expect(
      verificationSelectedSnapshots(
        ['second'],
        [second, { id: 'not-selected', unverifiedAmount: '100' }],
        [first],
      ),
    ).toEqual([second]);
    expect(() => verificationSelectedSnapshots(['missing'], [], [])).toThrow(
      '所选核销来源缺少数据',
    );
  });
});
