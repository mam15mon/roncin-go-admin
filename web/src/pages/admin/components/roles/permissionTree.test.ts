import { describe, expect, it } from 'vitest';
import {
  buildPermissionTree,
  collectPermissionLeafKeys,
  filterPermissionTree,
  isPermissionGroupNode,
} from './permissionTree';

const permissions = [
  {
    key: 'system.user.read',
    name: '查看用户',
    group: '系统管理 · 用户',
    requires: [],
  },
  {
    key: 'business.order.se.read',
    name: '查看海运出口订单',
    group: '订单管理 · 海运出口（SE） · 订单',
    requires: [],
  },
  {
    key: 'business.order.se.container.read',
    name: '查看海运出口集装箱',
    group: '订单管理 · 海运出口（SE） · 集装箱',
    requires: ['business.order.se.read'],
  },
  {
    key: 'business.order.se.container.create',
    name: '新建海运出口集装箱',
    group: '订单管理 · 海运出口（SE） · 集装箱',
    description: '新增订单集装箱',
    requires: ['business.order.se.container.read', 'business.order.se.read'],
  },
];

describe('buildPermissionTree', () => {
  it('按 manifest 分组路径构造任意深度权限树', () => {
    const model = buildPermissionTree(permissions);

    expect(model.treeData.map((node) => node.title)).toEqual([
      '系统管理',
      '订单管理',
    ]);
    const orderRoot = model.treeData[1];
    expect(isPermissionGroupNode(orderRoot)).toBe(true);
    if (!isPermissionGroupNode(orderRoot)) return;
    const seaExport = orderRoot.children[0];
    expect(isPermissionGroupNode(seaExport)).toBe(true);
    if (!isPermissionGroupNode(seaExport)) return;
    expect(seaExport.children.map((node) => node.title)).toEqual([
      '订单',
      '集装箱',
    ]);
    expect(collectPermissionLeafKeys([orderRoot])).toEqual([
      'business.order.se.read',
      'business.order.se.container.read',
      'business.order.se.container.create',
    ]);
    expect(model.initialExpandedKeys).toContain('group:订单管理');
    expect(model.initialExpandedKeys).toContain(
      'group:订单管理 · 海运出口（SE）',
    );
    expect(model.initialExpandedKeys).not.toContain(
      'group:订单管理 · 海运出口（SE） · 集装箱',
    );
  });

  it('保留权限依赖和名称索引', () => {
    const model = buildPermissionTree(permissions);
    expect(
      model.requiresByPermission['business.order.se.container.create'],
    ).toEqual(['business.order.se.container.read', 'business.order.se.read']);
    expect(
      model.permissionNameByKey['business.order.se.container.create'],
    ).toBe('新建海运出口集装箱');
  });
});

describe('filterPermissionTree', () => {
  it('搜索只保留命中权限及完整祖先路径', () => {
    const model = buildPermissionTree(permissions);
    const filtered = filterPermissionTree(model.treeData, '新增订单集装箱');

    expect(filtered).toHaveLength(1);
    expect(filtered[0].title).toBe('订单管理');
    expect(collectPermissionLeafKeys(filtered)).toEqual([
      'business.order.se.container.create',
    ]);
  });
});
