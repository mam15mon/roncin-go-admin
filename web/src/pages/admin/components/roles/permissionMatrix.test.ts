import { describe, expect, it } from 'vitest';
import {
  buildPermissionMatrix,
  filterResources,
  getResourceState,
  isReadPermission,
  setBatchScopeLevel,
  setResourceLevel,
  toggleResourceRead,
  toggleResourceWrite,
} from './permissionMatrix';

const mockPermissions = [
  {
    key: 'system.platform.access',
    name: '访问工作台',
    group: '系统管理 · 平台',
  },
  {
    key: 'system.user.read',
    name: '查看用户',
    group: '系统管理 · 用户',
    requires: [],
  },
  {
    key: 'system.user.create',
    name: '新建用户',
    group: '系统管理 · 用户',
    requires: ['system.user.read'],
  },
  {
    key: 'system.user.update',
    name: '编辑用户',
    group: '系统管理 · 用户',
    requires: ['system.user.read'],
  },
  {
    key: 'system.user.delete',
    name: '办理离职',
    group: '系统管理 · 用户',
    requires: ['system.user.read'],
  },
  {
    key: 'system.audit.read',
    name: '查看审计日志',
    group: '系统管理 · 审计',
  },
  {
    key: 'business.order.se.read',
    name: '查看海运出口订单',
    group: '订单管理 · 海运出口（SE） · 订单',
    requires: [],
  },
  {
    key: 'business.order.se.create',
    name: '新建海运出口订单',
    group: '订单管理 · 海运出口（SE） · 订单',
    requires: ['business.order.se.read'],
  },
  {
    key: 'business.order.se.container.read',
    name: '查看海运出口集装箱',
    group: '订单管理 · 海运出口（SE） · 集装箱',
    requires: ['business.order.se.read'],
  },
  {
    key: 'business.order.se.container.create',
    name: '新增海运出口集装箱',
    group: '订单管理 · 海运出口（SE） · 集装箱',
    requires: ['business.order.se.container.read', 'business.order.se.read'],
  },
  {
    key: 'system.finance.bill.read',
    name: '查看账单',
    group: '费用管理 · 账单',
    requires: [],
  },
  {
    key: 'system.finance.bill.create',
    name: '创建账单',
    group: '费用管理 · 账单',
    requires: ['system.finance.bill.read'],
  },
];

describe('isReadPermission', () => {
  it('正确识别各类只读权限', () => {
    expect(isReadPermission('system.user.read', '查看用户')).toBe(true);
    expect(isReadPermission('system.platform.access', '访问工作台')).toBe(
      true,
    );
    expect(
      isReadPermission('business.order.se.cargo_item.read', '查看货物明细'),
    ).toBe(true);
  });

  it('正确识别编辑与操作类权限', () => {
    expect(isReadPermission('system.user.create', '新建用户')).toBe(false);
    expect(isReadPermission('system.user.delete', '办理离职')).toBe(false);
    expect(
      isReadPermission('business.order.se.split', '海运出口（SE） 拆票'),
    ).toBe(false);
  });
});

describe('buildPermissionMatrix', () => {
  it('聚合为模块与资源，区分读写权限集合', () => {
    const matrix = buildPermissionMatrix(mockPermissions);

    expect(matrix.modules.map((m) => m.name)).toEqual([
      '订单管理',
      '费用管理',
      '系统管理',
    ]);

    const sysModule = matrix.modules.find((m) => m.name === '系统管理');
    expect(sysModule).toBeDefined();
    expect(sysModule?.resources.map((r) => r.name)).toEqual([
      '平台',
      '用户',
      '审计',
    ]);

    const userRes = sysModule?.resources.find((r) => r.name === '用户');
    expect(userRes).toBeDefined();
    expect(userRes?.readKeys).toEqual(['system.user.read']);
    expect(userRes?.writeKeys).toEqual([
      'system.user.create',
      'system.user.update',
      'system.user.delete',
    ]);

    const orderModule = matrix.modules.find((m) => m.name === '订单管理');
    expect(orderModule?.subModules).toEqual(['海运出口（SE）']);
  });
});

function findResource(
  matrix: ReturnType<typeof buildPermissionMatrix>,
  id: string,
) {
  const res = matrix.allResources.find((r) => r.id === id);
  if (!res) throw new Error(`Resource ${id} not found in test`);
  return res;
}

function findModule(
  matrix: ReturnType<typeof buildPermissionMatrix>,
  name: string,
) {
  const mod = matrix.modules.find((m) => m.name === name);
  if (!mod) throw new Error(`Module ${name} not found in test`);
  return mod;
}

describe('getResourceState', () => {
  const matrix = buildPermissionMatrix(mockPermissions);
  const userRes = findResource(matrix, '系统管理 · 用户');
  const auditRes = findResource(matrix, '系统管理 · 审计');

  it('未选择任何权限时返回 none 状态', () => {
    const state = getResourceState(userRes, []);
    expect(state.readState).toBe('none');
    expect(state.writeState).toBe('none');
    expect(state.overallLevel).toBe('none');
    expect(state.selectedCount).toBe(0);
  });

  it('仅勾选只读权限时返回 read 状态', () => {
    const state = getResourceState(userRes, ['system.user.read']);
    expect(state.readState).toBe('all');
    expect(state.writeState).toBe('none');
    expect(state.overallLevel).toBe('read');
    expect(state.selectedCount).toBe(1);
  });

  it('勾选全部权限时返回 full 状态', () => {
    const state = getResourceState(userRes, [
      'system.user.read',
      'system.user.create',
      'system.user.update',
      'system.user.delete',
    ]);
    expect(state.readState).toBe('all');
    expect(state.writeState).toBe('all');
    expect(state.overallLevel).toBe('full');
    expect(state.selectedCount).toBe(4);
  });

  it('部分勾选写操作时返回 custom 状态', () => {
    const state = getResourceState(userRes, [
      'system.user.read',
      'system.user.create',
    ]);
    expect(state.readState).toBe('all');
    expect(state.writeState).toBe('some');
    expect(state.overallLevel).toBe('custom');
    expect(state.selectedCount).toBe(2);
  });

  it('纯只读资源无写权限时 writeState 为 na', () => {
    const state = getResourceState(auditRes, ['system.audit.read']);
    expect(state.readState).toBe('all');
    expect(state.writeState).toBe('na');
    expect(state.overallLevel).toBe('read');
  });
});

describe('toggleResourceRead & toggleResourceWrite', () => {
  const matrix = buildPermissionMatrix(mockPermissions);
  const userRes = findResource(matrix, '系统管理 · 用户');

  it('开启查看时勾选该资源查看权限', () => {
    const result = toggleResourceRead(
      userRes,
      true,
      [],
      matrix.requiresByPermission,
    );
    expect(result).toEqual(['system.user.read']);
  });

  it('关闭查看时级联取消依赖该查看权限的全部写权限', () => {
    const initial = [
      'system.user.read',
      'system.user.create',
      'system.user.update',
    ];
    const result = toggleResourceRead(
      userRes,
      false,
      initial,
      matrix.requiresByPermission,
    );
    expect(result).toEqual([]);
  });

  it('开启编辑时自动补齐查看权限与全部写权限', () => {
    const result = toggleResourceWrite(
      userRes,
      true,
      [],
      matrix.requiresByPermission,
    );
    expect(result).toEqual(
      expect.arrayContaining([
        'system.user.read',
        'system.user.create',
        'system.user.update',
        'system.user.delete',
      ]),
    );
  });

  it('关闭编辑时保留查看权限，实现平滑降级为只读', () => {
    const initial = [
      'system.user.read',
      'system.user.create',
      'system.user.update',
    ];
    const result = toggleResourceWrite(
      userRes,
      false,
      initial,
      matrix.requiresByPermission,
    );
    expect(result).toEqual(['system.user.read']);
  });
});

describe('setResourceLevel & setBatchScopeLevel', () => {
  const matrix = buildPermissionMatrix(mockPermissions);
  const userRes = findResource(matrix, '系统管理 · 用户');

  it('快捷设置资源级别为 read / full / none', () => {
    const readOnly = setResourceLevel(
      userRes,
      'read',
      [],
      matrix.requiresByPermission,
    );
    expect(readOnly).toEqual(['system.user.read']);

    const full = setResourceLevel(
      userRes,
      'full',
      readOnly,
      matrix.requiresByPermission,
    );
    expect(full).toHaveLength(4);

    const none = setResourceLevel(
      userRes,
      'none',
      full,
      matrix.requiresByPermission,
    );
    expect(none).toEqual([]);
  });

  it('批量设置模块级别', () => {
    const sysModule = findModule(matrix, '系统管理');
    const readOnlySys = setBatchScopeLevel(
      sysModule,
      'read',
      [],
      matrix.requiresByPermission,
    );
    expect(readOnlySys).toEqual(
      expect.arrayContaining([
        'system.platform.access',
        'system.user.read',
        'system.audit.read',
      ]),
    );
    expect(readOnlySys).not.toContain('system.user.create');

    const fullSys = setBatchScopeLevel(
      sysModule,
      'full',
      readOnlySys,
      matrix.requiresByPermission,
    );
    expect(fullSys).toContain('system.user.create');
    expect(fullSys).toContain('system.user.delete');

    const clearedSys = setBatchScopeLevel(
      sysModule,
      'none',
      fullSys,
      matrix.requiresByPermission,
    );
    expect(clearedSys).toEqual([]);
  });
});

describe('filterResources', () => {
  const matrix = buildPermissionMatrix(mockPermissions);

  it('按操作名称或编码搜索时保留命中资源并记录高亮键', () => {
    const { filtered, matchedActionKeys } = filterResources(
      matrix.allResources,
      '离职',
    );
    expect(filtered).toHaveLength(1);
    expect(filtered[0].id).toBe('系统管理 · 用户');
    expect(matchedActionKeys.has('system.user.delete')).toBe(true);
  });

  it('按业务线或模块名称搜索', () => {
    const { filtered } = filterResources(matrix.allResources, '海运出口');
    expect(filtered).toHaveLength(2); // 订单 + 集装箱
  });
});
