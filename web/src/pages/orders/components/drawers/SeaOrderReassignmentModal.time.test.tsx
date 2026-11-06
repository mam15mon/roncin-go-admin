import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  enterMasterNo,
  preview,
  renderReassignment,
  setupReassignmentTests,
  validPreview,
} from './SeaOrderReassignmentModal.fixtures';

setupReassignmentTests();

describe('改配预览的本地时间', () => {
  it('ETD 和 ETA 原值、目标值转换本地时区，空值和普通字段保持原语义', async () => {
    const current = '2026-11-13T10:00:00.000Z';
    const target = '2026-11-13T20:30:45.000Z';
    preview.mockResolvedValue({
      ...validPreview(),
      data: {
        ...validPreview().data,
        differences: [
          {
            fieldName: 'etd',
            label: '预计开航时间(ETD)',
            currentValue: current,
            targetValue: target,
            isDifferent: true,
          },
          {
            fieldName: 'eta',
            label: '预计到港时间(ETA)',
            currentValue: current,
            targetValue: current,
            isDifferent: false,
          },
          {
            fieldName: 'voyage_no',
            label: '航次',
            currentValue: '',
            targetValue: 'TEST002',
            isDifferent: true,
          },
        ],
      },
    });
    renderReassignment();
    await enterMasterNo();
    // 以标准 Intl 独立计算期望值；同一用例分别在上海、UTC、纽约时区进程执行。
    const localTime = (value: string) => {
      const parts = new Intl.DateTimeFormat('en-CA', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hourCycle: 'h23',
      }).formatToParts(new Date(value));
      const part = (type: Intl.DateTimeFormatPartTypes) =>
        parts.find((entry) => entry.type === type)?.value;
      return `${part('year')}-${part('month')}-${part('day')} ${part('hour')}:${part('minute')}:${part('second')}`;
    };
    const etd = (await screen.findByText('预计开航时间(ETD)')).closest('tr');
    const eta = screen.getByText('预计到港时间(ETA)').closest('tr');
    if (!etd || !eta) throw new Error('预览缺少时间对比行');
    expect(within(etd).getByText(localTime(current))).toBeVisible();
    expect(within(etd).getByText(localTime(target))).toBeVisible();
    expect(within(eta).getAllByText(localTime(current))).toHaveLength(2);
    expect(screen.queryByText(current)).not.toBeInTheDocument();
    expect(screen.queryByText(target)).not.toBeInTheDocument();
    expect(screen.getByText('未填写')).toBeVisible();
    expect(screen.getByText('TEST002')).toBeVisible();
  });
});
