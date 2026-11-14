import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  loadFilterSettingsPreference,
  omitHiddenFilterValues,
  resolveFilterVisibleNames,
  saveFilterSettingsPreference,
} from './preference';
import type { FilterSettingsField } from './types';

const fields: FilterSettingsField[] = [
  { name: 'a', label: 'A', defaultVisible: true },
  { name: 'b', label: 'B', defaultVisible: true },
  { name: 'c', label: 'C', defaultVisible: false },
  { name: 'r', label: 'R', defaultVisible: true, required: true },
];
const scope = { userId: 'u1', organizationId: 'o1' };

describe('筛选设置偏好', () => {
  const spies: { mockRestore: () => void }[] = [];
  const track = <T extends { mockRestore: () => void }>(spy: T) => {
    spies.push(spy);
    return spy;
  };
  afterEach(() => {
    for (const spy of spies.splice(0)) spy.mockRestore();
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it('按页面、用户、公司隔离存取', () => {
    saveFilterSettingsPreference('p1', scope, {
      visible: ['a'],
      hidden: ['b'],
    });
    expect(loadFilterSettingsPreference('p1', scope)?.hidden).toEqual(['b']);
    expect(loadFilterSettingsPreference('p2', scope)).toBeNull();
    expect(
      loadFilterSettingsPreference('p1', { ...scope, userId: 'u2' }),
    ).toBeNull();
    expect(
      loadFilterSettingsPreference('p1', { ...scope, organizationId: 'o2' }),
    ).toBeNull();
    expect(
      window.localStorage.getItem('roncin:filter-settings:v1:p1:u1:o1'),
    ).toContain('"visible":["a"]');
  });

  it('身份未就绪不读不写', () => {
    const set = track(vi.spyOn(window.localStorage, 'setItem'));
    set.mockClear();
    const get = track(vi.spyOn(window.localStorage, 'getItem'));
    get.mockClear();
    expect(
      saveFilterSettingsPreference(
        'p1',
        { userId: 'u1' },
        {
          visible: [],
          hidden: [],
        },
      ),
    ).toBe(false);
    expect(loadFilterSettingsPreference('p1', {})).toBeNull();
    expect(set).not.toHaveBeenCalled();
    expect(get).not.toHaveBeenCalled();
  });

  it('存储抛错时保存返回 false、读取返回 null', () => {
    track(vi.spyOn(window.localStorage, 'setItem')).mockImplementation(() => {
      throw new Error('quota');
    });
    expect(
      saveFilterSettingsPreference('p1', scope, { visible: [], hidden: [] }),
    ).toBe(false);
  });

  it('未知字段忽略、新字段按默认补入、必选字段不可隐藏', () => {
    expect(
      resolveFilterVisibleNames(fields, {
        visible: ['a', 'gone'],
        hidden: ['b', 'r'],
      }),
    ).toEqual(['a', 'r']);
    // 偏好未提及的 c(默认隐藏) 不补入，新增默认可见字段补入。
    expect(
      resolveFilterVisibleNames(fields, { visible: ['a'], hidden: [] }),
    ).toEqual(['a', 'b', 'r']);
    expect(resolveFilterVisibleNames(fields, null)).toEqual(['a', 'b', 'r']);
  });

  it('页签恢复只剔除已隐藏字段的值', () => {
    expect(
      omitHiddenFilterValues({ a: 1, b: 2, other: 3 }, fields, ['a', 'r']),
    ).toEqual({ a: 1, other: 3 });
  });
});
