import { describe, expect, it } from 'vitest';
import {
  backgroundTaskExecutionSummary,
  backgroundTaskHasNextRunAt,
  backgroundTaskPresentation,
} from './background-task-presentation';

describe('后台任务展示', () => {
  it('将账号授权任务展示为业务名称', () => {
    const presentation = backgroundTaskPresentation({
      kind: 5,
      idempotencyKey: 'user-authorized:task-id',
    });

    expect(presentation.label).toBe('账号授权完成通知');
    expect(presentation.description).toContain('组织和角色授权');
  });

  it('将订单人员任务展示为业务名称', () => {
    expect(
      backgroundTaskPresentation({
        kind: 5,
        idempotencyKey: 'order-personnel:task-id',
      }).label,
    ).toBe('订单人员分配通知');
  });

  it('用自然语言说明执行和重试情况', () => {
    expect(backgroundTaskExecutionSummary({ status: 3, attempts: 0 })).toBe(
      '首次执行成功，无重试',
    );
    expect(backgroundTaskExecutionSummary({ status: 3, attempts: 2 })).toBe(
      '失败 2 次后执行成功',
    );
    expect(backgroundTaskExecutionSummary({ status: 4, attempts: 1 })).toBe(
      '已失败 1 次，等待自动重试',
    );
  });

  it('只为等待执行和等待重试的任务显示下次执行时间', () => {
    expect(backgroundTaskHasNextRunAt({ status: 1 })).toBe(true);
    expect(backgroundTaskHasNextRunAt({ status: 4 })).toBe(true);
    expect(backgroundTaskHasNextRunAt({ status: 2 })).toBe(false);
    expect(backgroundTaskHasNextRunAt({ status: 3 })).toBe(false);
    expect(backgroundTaskHasNextRunAt({ status: 5 })).toBe(false);
  });
});
