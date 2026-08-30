import { describe, expect, it } from 'vitest';
import { getReleasePodTransition } from './release-pod-panel';

describe('getReleasePodTransition', () => {
  it('只允许待签收和已签收状态向前流转', () => {
    expect(
      getReleasePodTransition({
        status: 1,
        allowedTargetStatuses: [2],
      }),
    ).toEqual({
      currentText: '待签收',
      nextText: '已签收',
      toStatus: 2,
    });
    expect(
      getReleasePodTransition({
        status: 2,
        allowedTargetStatuses: [3],
      }),
    ).toEqual({
      currentText: '已签收',
      nextText: '已回单',
      toStatus: 3,
    });
    expect(
      getReleasePodTransition({ status: 1, allowedTargetStatuses: [] }),
    ).toBeUndefined();
    expect(
      getReleasePodTransition({ status: 3, allowedTargetStatuses: [] }),
    ).toBeUndefined();
    expect(getReleasePodTransition()).toBeUndefined();
  });
});
