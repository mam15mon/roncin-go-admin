import { describe, expect, it } from 'vitest';
import {
  getUnlockRequestStatusMeta,
  shouldPollUnlockRequests,
} from './UnlockRequestHistoryDrawer';

describe('getUnlockRequestStatusMeta', () => {
  it('准确展示服务端全部解锁请求状态', () => {
    expect(
      [
        ['PENDING_DISPATCH', '待派发'],
        ['PENDING_APPROVAL', '审批中'],
        ['APPROVED_PENDING_APPLY', '已同意待本地生效'],
        ['APPROVED', '已解锁'],
        ['REJECTED', '已拒绝'],
        ['CONFIGURATION_FAILED', '配置失败'],
        ['DISPATCH_FAILED', '派发失败'],
        ['DISPATCH_UNKNOWN', '派发结果未知'],
        ['STALE', '已过期'],
      ].map(([status]) => [status, getUnlockRequestStatusMeta(status).label]),
    ).toEqual([
      ['PENDING_DISPATCH', '待派发'],
      ['PENDING_APPROVAL', '审批中'],
      ['APPROVED_PENDING_APPLY', '已同意待本地生效'],
      ['APPROVED', '已解锁'],
      ['REJECTED', '已拒绝'],
      ['CONFIGURATION_FAILED', '配置失败'],
      ['DISPATCH_FAILED', '派发失败'],
      ['DISPATCH_UNKNOWN', '派发结果未知'],
      ['STALE', '已过期'],
    ]);
  });

  it('只对仍可能由系统自动推进的状态轮询', () => {
    for (const status of [
      'PENDING_DISPATCH',
      'PENDING_APPROVAL',
      'APPROVED_PENDING_APPLY',
    ]) {
      expect(shouldPollUnlockRequests([{ status }])).toBe(true);
    }

    for (const status of [
      'APPROVED',
      'REJECTED',
      'CONFIGURATION_FAILED',
      'DISPATCH_FAILED',
      'DISPATCH_UNKNOWN',
      'STALE',
    ]) {
      expect(shouldPollUnlockRequests([{ status }])).toBe(false);
    }
  });
});
