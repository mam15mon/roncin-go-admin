import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  cancelledServices,
  handlingChanged,
  handlingFormValues,
  handlingPayload,
  handlingSnapshot,
  IMPORT_SERVICES,
  needsHandlingReason,
  serviceStatus,
} from './model';

function snapshot(extra: Partial<API.OrderImportHandling> = {}) {
  return handlingSnapshot(
    {
      orderId: 'order',
      orderVersion: '9007199254740993',
      steps: IMPORT_SERVICES.map(({ kind }) => ({ kind })),
      ...extra,
    },
    'order',
  );
}

describe('进口服务wire与提交映射', () => {
  it('非optional false省略表示不承接，完整五项仍必需，optional时间保持存在性', () => {
    const value = snapshot();
    expect(
      value.steps.every(
        (step) => step.undertaken === false && step.occurredAt === undefined,
      ),
    ).toBe(true);
    expect(serviceStatus(value.steps[0])).toBe('不适用');
    expect(() => snapshot({ steps: [] })).toThrow('资料不完整');
    expect(() =>
      snapshot({ steps: [...value.steps.slice(0, 4), value.steps[0]] }),
    ).toThrow('环节重复');
    expect(() => snapshot({ orderVersion: '0' })).toThrow('资料不完整');
    expect(() =>
      snapshot({
        steps: value.steps.map((step, index) =>
          index === 0 ? { ...step, undertaken: true, occurredAt: '' } : step,
        ),
      }),
    ).toThrow('完成事实不合法');
    expect(() =>
      snapshot({
        steps: value.steps.map((step, index) =>
          index === 0 ? { ...step, occurredAt: '2026-11-16T08:00:00Z' } : step,
        ),
      }),
    ).toThrow('完成事实不合法');
  });

  it('任意子集、并行时间和未完成事实原样映射，取消仅清完成时间并保留资料', () => {
    const value = snapshot();
    value.steps[1] = {
      ...value.steps[1],
      undertaken: true,
      occurredAt: '2026-11-16T08:00:00+08:00',
      referenceNo: 'EX-001',
      notes: ' 换单备注 ',
      attachmentIds: ['attachment'],
    };
    value.steps[3] = { ...value.steps[3], undertaken: true };
    const form = handlingFormValues(value);
    expect(dayjs.isDayjs(form.steps[1].occurredAt)).toBe(true);
    const payload = handlingPayload(value, form);
    expect(payload.expectedVersion).toBe('9007199254740993');
    expect(payload.steps?.[1]).toMatchObject({
      undertaken: true,
      occurredAt: '2026-11-16T08:00:00+08:00',
      referenceNo: 'EX-001',
      notes: '换单备注',
      attachmentIds: ['attachment'],
    });
    expect(serviceStatus(payload.steps?.[3] ?? {})).toBe('未完成');
    form.steps[1].undertaken = false;
    const cancelled = handlingPayload(value, form);
    expect(cancelled.steps?.[1]).toMatchObject({
      undertaken: false,
      occurredAt: undefined,
      referenceNo: 'EX-001',
      attachmentIds: ['attachment'],
    });
    expect(cancelledServices(value, cancelled)).toEqual(['换单']);
    expect(needsHandlingReason(value, cancelled)).toBe(true);
    expect(value.steps[1].occurredAt).toBe('2026-11-16T08:00:00+08:00');
  });

  it('已完成事实改参考号、附件、责任人或换单代理需原因，未完成项不误拦', () => {
    const value = snapshot();
    value.steps[1] = {
      ...value.steps[1],
      undertaken: true,
      occurredAt: '2026-11-16T00:00:00Z',
    };
    const form = handlingFormValues(value);
    const unchanged = handlingPayload(value, form);
    expect(needsHandlingReason(value, unchanged)).toBe(false);
    for (const field of [
      'referenceNo',
      'notes',
      'responsibleUserId',
    ] as const) {
      const changed = structuredClone(unchanged);
      if (changed.steps?.[1]) changed.steps[1][field] = 'changed';
      expect(needsHandlingReason(value, changed)).toBe(true);
    }
    expect(
      needsHandlingReason(value, {
        ...unchanged,
        exchangeAgentId: 'new-agent',
      }),
    ).toBe(true);
    const other = structuredClone(unchanged);
    if (other.steps?.[0]) other.steps[0].notes = '未承接资料备注';
    expect(needsHandlingReason(value, other)).toBe(false);
  });

  it('未修改完成时间保留wire纳秒，实际调整时提交新时刻且要求原因', () => {
    const value = snapshot();
    value.steps[1] = {
      ...value.steps[1],
      undertaken: true,
      occurredAt: '2026-11-16T08:00:00.123456789+08:00',
    };
    const form = handlingFormValues(value);
    const unchanged = handlingPayload(value, form);
    expect(unchanged.steps?.[1].occurredAt).toBe(
      '2026-11-16T08:00:00.123456789+08:00',
    );
    expect(needsHandlingReason(value, unchanged)).toBe(false);
    form.steps[1].occurredAt = dayjs('2026-11-16T08:00:01.123+08:00');
    const changed = handlingPayload(value, form);
    expect(changed.steps?.[1].occurredAt).toBe('2026-11-16T00:00:01.123Z');
    expect(needsHandlingReason(value, changed)).toBe(true);
    const nanosOnly = structuredClone(value);
    nanosOnly.steps[1].occurredAt = '2026-11-16T08:00:00.123456790+08:00';
    expect(handlingChanged(value, nanosOnly)).toBe(true);
  });

  it('变动判断关注事实与版本，输出姓名投影变化不替换编辑基线', () => {
    const value = snapshot();
    expect(
      handlingChanged(value, { ...value, orderVersion: '9007199254740994' }),
    ).toBe(true);
    expect(
      handlingChanged(value, {
        ...value,
        steps: value.steps.map((step) => ({
          ...step,
          responsibleUserName: '投影姓名',
        })),
      }),
    ).toBe(false);
    expect(() =>
      handlingPayload(
        { ...value, orderVersion: '' },
        handlingFormValues(value),
      ),
    ).toThrow('提交基线缺失');
  });
});
