import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  billDuePayload,
  billDueResult,
  initialBillDueSettings,
} from './billDueSettings';

describe('账单到期设置', () => {
  it.each([
    [{ paymentTermsDays: 30 }, '2026-12-18 到期 · 账期 30 天', '2026-12-18'],
    [{ paymentTermsDays: 0 }, '当天到期 · 现结', '2026-11-18'],
    [{}, '未约定到期日', undefined],
    [{ dueDate: '2026-12-01' }, '2026-12-01 到期 · 指定日期', '2026-12-01'],
  ])('按真实可选字段形态生成结果 %j', (source, text, effectiveDueDate) => {
    expect(billDueResult('2026-11-18', initialBillDueSettings(source))).toEqual(
      { text, effectiveDueDate },
    );
  });

  it('缺少账单日期时不推测日期', () => {
    expect(billDueResult(undefined, { paymentTermsDays: 30 })).toEqual({
      text: '未约定到期日',
      effectiveDueDate: undefined,
    });
  });

  it('显式零优先于到期日、没有账期的指定日期保留存在性', () => {
    expect(
      initialBillDueSettings({ paymentTermsDays: 0, dueDate: '2026-11-18' })
        .dueMode,
    ).toBe('terms');
    expect(initialBillDueSettings({ dueDate: '2026-12-01' }).dueMode).toBe(
      'date',
    );
    expect(initialBillDueSettings({}).dueMode).toBe('terms');
  });

  it('指定日期只提交日期，按天数只提交账期，未约定不制造零', () => {
    expect(
      billDuePayload({
        dueMode: 'date',
        dueDate: dayjs('2026-12-01'),
        paymentTermsDays: 30,
      }),
    ).toEqual({ dueDate: '2026-12-01' });
    expect(
      billDuePayload({
        dueMode: 'terms',
        dueDate: dayjs('2026-12-01'),
        paymentTermsDays: 0,
      }),
    ).toEqual({ paymentTermsDays: 0 });
    expect(billDuePayload({ dueMode: 'terms' })).toEqual({});
    expect(() => billDuePayload({ dueMode: 'date' })).toThrow(
      '请选择指定到期日',
    );
  });
});
