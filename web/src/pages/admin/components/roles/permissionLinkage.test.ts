import { describe, expect, it } from 'vitest';
import {
  applyPermissionLinkage,
  mergeVisiblePermissionSelection,
} from './permissionLinkage';

const requiresByPermission: Record<string, string[]> = {
  'business.partner.read': [],
  'business.partner.update': ['business.partner.read'],
  'business.order.se.read': [],
  'business.order.se.cargo_item.read': ['business.order.se.read'],
  'business.order.se.cargo_item.create': [
    'business.order.se.cargo_item.read',
    'business.order.se.read',
  ],
};

describe('applyPermissionLinkage', () => {
  it('勾选操作权限时自动连带同资源读权限', () => {
    const result = applyPermissionLinkage(
      [],
      ['business.partner.update'],
      requiresByPermission,
    );
    expect(result).toEqual([
      'business.partner.update',
      'business.partner.read',
    ]);
  });

  it('勾选订单子资源操作权限时传递补齐依赖链', () => {
    const result = applyPermissionLinkage(
      [],
      ['business.order.se.cargo_item.create'],
      requiresByPermission,
    );
    expect(result).toEqual([
      'business.order.se.cargo_item.create',
      'business.order.se.cargo_item.read',
      'business.order.se.read',
    ]);
  });

  it('取消基础权限时级联取消依赖它的权限且不回填', () => {
    const previous = [
      'business.partner.read',
      'business.partner.update',
      'business.order.se.read',
      'business.order.se.cargo_item.read',
      'business.order.se.cargo_item.create',
    ];
    const result = applyPermissionLinkage(
      previous,
      previous.filter((key) => key !== 'business.order.se.read'),
      requiresByPermission,
    );
    // 订单读被取消后，依赖它的子资源读写全部级联移除；无关的单位权限保留。
    expect(result).toEqual([
      'business.partner.read',
      'business.partner.update',
    ]);
  });

  it('取消读权限时级联取消同资源的写权限', () => {
    const checked = applyPermissionLinkage(
      [],
      ['business.partner.update'],
      requiresByPermission,
    );
    const afterUncheckRead = applyPermissionLinkage(
      checked,
      checked.filter((key) => key !== 'business.partner.read'),
      requiresByPermission,
    );
    expect(afterUncheckRead).toEqual([]);
  });

  it('未知权限键原样保留且不参与联动', () => {
    const result = applyPermissionLinkage(
      [],
      ['custom.unknown', 'business.partner.read'],
      requiresByPermission,
    );
    expect(result).toEqual(['custom.unknown', 'business.partner.read']);
  });

  it('已满足依赖的集合保持不变', () => {
    const source = [
      'business.order.se.cargo_item.create',
      'business.order.se.cargo_item.read',
      'business.order.se.read',
    ];
    expect(
      applyPermissionLinkage(source, source, requiresByPermission),
    ).toEqual(source);
  });
});

describe('mergeVisiblePermissionSelection', () => {
  it('搜索过滤后勾选可见权限时保留隐藏权限', () => {
    const previous = [
      'business.partner.read',
      'business.order.se.read',
      'business.order.se.cargo_item.read',
    ];
    const merged = mergeVisiblePermissionSelection(
      previous,
      ['business.partner.read', 'business.partner.update'],
      ['business.partner.read', 'business.partner.update'],
    );
    expect(merged).toEqual([
      'business.order.se.read',
      'business.order.se.cargo_item.read',
      'business.partner.read',
      'business.partner.update',
    ]);
  });

  it('取消可见权限时只移除当前过滤树中的权限', () => {
    const merged = mergeVisiblePermissionSelection(
      ['business.partner.read', 'business.order.se.read'],
      ['business.partner.read'],
      [],
    );
    expect(merged).toEqual(['business.order.se.read']);
  });
});
