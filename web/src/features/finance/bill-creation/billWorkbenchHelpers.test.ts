import { describe, expect, it } from 'vitest';
import { normalizeBillBatchResponse } from './billWorkbenchHelpers';

describe('批量建账逐组响应映射', () => {
  it.each([undefined, false])(
    '失败组 success=%s 时保留失败，成功组可不含 error',
    (success) => {
      const result = normalizeBillBatchResponse(
        {
          data: { id: 'batch-1' },
          results: [
            { groupKey: 'ar', success: true, bill: { id: 'bill-1' } },
            {
              groupKey: 'ap',
              success,
              error: {
                reason: 'FINANCE_BILL_PREVIEW_STALE',
                message: '预览已变化',
              },
            },
          ],
        },
        ['ar', 'ap'],
      );
      expect(result.succeededCount).toBe(1);
      expect(result.failedCount).toBe(1);
      expect(result.groups[0].bill?.id).toBe('bill-1');
      expect(result.groups[1]).toEqual({
        groupKey: 'ap',
        errorReason: 'FINANCE_BILL_PREVIEW_STALE',
        errorMessage: '费用或创建日期已发生变化，请返回调整失败组后重新预览',
      });
    },
  );

  it.each([
    { groupKey: 'ar', success: true },
    { groupKey: 'ar', success: true, bill: {} },
    { groupKey: 'ar' },
    { groupKey: 'ar', error: { message: '缺少原因码' } },
  ])('真实必填结果缺失时显式失败：%j', (item) => {
    expect(() =>
      normalizeBillBatchResponse({ data: { id: 'batch-1' }, results: [item] }, [
        'ar',
      ]),
    ).toThrow();
  });

  it('拒绝重复、遗漏或额外分组结果', () => {
    const item = { groupKey: 'ar', success: true, bill: { id: 'bill-1' } };
    for (const results of [
      [item, item],
      [],
      [item, { ...item, groupKey: 'ap' }],
    ]) {
      expect(() =>
        normalizeBillBatchResponse({ data: { id: 'batch-1' }, results }, [
          'ar',
        ]),
      ).toThrow();
    }
  });
});
