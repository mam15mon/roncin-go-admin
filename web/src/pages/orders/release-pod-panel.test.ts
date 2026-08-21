import { describe, expect, it } from 'vitest';
import { getReleasePodTransition } from './release-pod-panel';

describe('getReleasePodTransition', () => {
  it('只允许待签收和已签收状态向前流转', () => {
    expect(getReleasePodTransition(1)).toEqual({
      currentText: '待签收',
      nextText: '已签收',
      toStatus: 2,
    });
    expect(getReleasePodTransition(2)).toEqual({
      currentText: '已签收',
      nextText: '已回单',
      toStatus: 3,
    });
    expect(getReleasePodTransition(3)).toBeUndefined();
    expect(getReleasePodTransition()).toBeUndefined();
  });
});
