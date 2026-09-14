import { applyPermissionLinkage } from './permissionLinkage';

export type PermissionDefinition = {
  key?: string;
  name?: string;
  group?: string;
  description?: string;
  requires?: string[];
};

export type PermissionAction = {
  key: string;
  name: string;
  group: string;
  description?: string;
  requires?: string[];
  isRead: boolean;
};

export type PermissionResource = {
  id: string; // 唯一分组标识，例如 "订单管理 · 海运出口（SE） · 订单"
  module: string; // 顶级服务模块，例如 "订单管理"
  subModule?: string; // 二级子业务/业务线，例如 "海运出口（SE）"
  name: string; // 资源名称，例如 "订单" 或 "集装箱"
  fullPath: string; // 完整路径，例如 "订单管理 · 海运出口（SE） · 订单"
  readKeys: string[];
  writeKeys: string[];
  allKeys: string[];
  readActions: PermissionAction[];
  writeActions: PermissionAction[];
  allActions: PermissionAction[];
};

export type PermissionModule = {
  id: string;
  name: string;
  subModules: string[];
  resources: PermissionResource[];
  readKeys: string[];
  writeKeys: string[];
  allKeys: string[];
};

export type PermissionMatrixModel = {
  modules: PermissionModule[];
  allResources: PermissionResource[];
  allLeafKeys: string[];
  allReadKeys: string[];
  allWriteKeys: string[];
  requiresByPermission: Record<string, string[]>;
  permissionNameByKey: Record<string, string>;
  permissionByKey: Record<string, PermissionAction>;
};

export type ResourceState = {
  readState: 'none' | 'some' | 'all';
  writeState: 'none' | 'some' | 'all' | 'na'; // na 表示该资源无写操作（如纯字典/审计）
  overallLevel: 'none' | 'read' | 'full' | 'custom';
  selectedCount: number;
  totalCount: number;
};

const CANONICAL_MODULE_ORDER = [
  '订单管理',
  '费用管理',
  '业务资料',
  '主数据',
  '系统管理',
];

/**
 * 判断是否为查看/只读类权限。
 * 规则：以 .read 结尾，或包含 .read.，或为平台访问基础权限，或中文名称以“查看”/“访问”开头。
 */
export function isReadPermission(key: string, name?: string): boolean {
  const lowerKey = key.toLowerCase();
  const lowerName = (name ?? '').toLowerCase();
  return (
    lowerKey.endsWith('.read') ||
    lowerKey.includes('.read.') ||
    lowerKey === 'system.platform.access' ||
    lowerName.startsWith('查看') ||
    lowerName.startsWith('访问')
  );
}

/**
 * 将平铺的权限清单构造成按华为云 IAM 风格组织的服务-资源矩阵模型。
 */
export function buildPermissionMatrix(
  permissions: PermissionDefinition[],
): PermissionMatrixModel {
  const allLeafKeys: string[] = [];
  const allReadKeys: string[] = [];
  const allWriteKeys: string[] = [];
  const requiresByPermission: Record<string, string[]> = {};
  const permissionNameByKey: Record<string, string> = {};
  const permissionByKey: Record<string, PermissionAction> = {};

  // 按分组路径聚合成资源对象
  const resourceMap = new Map<string, PermissionResource>();
  const resourceList: PermissionResource[] = [];

  for (const perm of permissions) {
    if (!perm.key) continue;
    const key = perm.key;
    const name = perm.name ?? perm.key;
    const group = perm.group || '其他功能';
    const isRead = isReadPermission(key, name);

    allLeafKeys.push(key);
    if (isRead) {
      allReadKeys.push(key);
    } else {
      allWriteKeys.push(key);
    }
    requiresByPermission[key] = perm.requires ?? [];
    permissionNameByKey[key] = name;

    const action: PermissionAction = {
      key,
      name,
      group,
      description: perm.description,
      requires: perm.requires ?? [],
      isRead,
    };
    permissionByKey[key] = action;

    const segments = group
      .split('·')
      .map((s) => s.trim())
      .filter(Boolean);
    const moduleName = segments[0] || '其他功能';
    const subModuleName = segments.length >= 3 ? segments[1] : undefined;
    const resourceName =
      segments.length >= 3
        ? segments.slice(2).join(' · ')
        : segments[1] || segments[0];
    const resourceId = group;

    let resource = resourceMap.get(resourceId);
    if (!resource) {
      resource = {
        id: resourceId,
        module: moduleName,
        subModule: subModuleName,
        name: resourceName,
        fullPath: group,
        readKeys: [],
        writeKeys: [],
        allKeys: [],
        readActions: [],
        writeActions: [],
        allActions: [],
      };
      resourceMap.set(resourceId, resource);
      resourceList.push(resource);
    }

    resource.allKeys.push(key);
    resource.allActions.push(action);
    if (isRead) {
      resource.readKeys.push(key);
      resource.readActions.push(action);
    } else {
      resource.writeKeys.push(key);
      resource.writeActions.push(action);
    }
  }

  // 按服务模块聚合
  const moduleMap = new Map<string, PermissionModule>();
  for (const res of resourceList) {
    let mod = moduleMap.get(res.module);
    if (!mod) {
      mod = {
        id: res.module,
        name: res.module,
        subModules: [],
        resources: [],
        readKeys: [],
        writeKeys: [],
        allKeys: [],
      };
      moduleMap.set(res.module, mod);
    }
    mod.resources.push(res);
    mod.readKeys.push(...res.readKeys);
    mod.writeKeys.push(...res.writeKeys);
    mod.allKeys.push(...res.allKeys);
    if (res.subModule && !mod.subModules.includes(res.subModule)) {
      mod.subModules.push(res.subModule);
    }
  }

  // 稳定排序模块：常用规范模块在前，其他模块跟在后面
  const sortedModules: PermissionModule[] = [];
  for (const modName of CANONICAL_MODULE_ORDER) {
    const mod = moduleMap.get(modName);
    if (mod) {
      sortedModules.push(mod);
      moduleMap.delete(modName);
    }
  }
  for (const mod of moduleMap.values()) {
    sortedModules.push(mod);
  }

  return {
    modules: sortedModules,
    allResources: resourceList,
    allLeafKeys,
    allReadKeys,
    allWriteKeys,
    requiresByPermission,
    permissionNameByKey,
    permissionByKey,
  };
}

/**
 * 计算单个资源在当前已选权限集下的授权状态。
 */
export function getResourceState(
  resource: PermissionResource,
  selectedKeys: string[],
): ResourceState {
  const selectedSet = new Set(selectedKeys);
  const selectedRead = resource.readKeys.filter((k) =>
    selectedSet.has(k),
  ).length;
  const selectedWrite = resource.writeKeys.filter((k) =>
    selectedSet.has(k),
  ).length;
  const totalSelected = selectedRead + selectedWrite;
  const total = resource.allKeys.length;

  const readState: 'none' | 'some' | 'all' =
    resource.readKeys.length === 0
      ? 'none'
      : selectedRead === resource.readKeys.length
        ? 'all'
        : selectedRead > 0
          ? 'some'
          : 'none';

  const writeState: 'none' | 'some' | 'all' | 'na' =
    resource.writeKeys.length === 0
      ? 'na'
      : selectedWrite === resource.writeKeys.length
        ? 'all'
        : selectedWrite > 0
          ? 'some'
          : 'none';

  let overallLevel: 'none' | 'read' | 'full' | 'custom' = 'none';
  if (totalSelected === 0) {
    overallLevel = 'none';
  } else if (writeState === 'na' && readState === 'all') {
    overallLevel = 'read';
  } else if (totalSelected === total) {
    overallLevel = 'full';
  } else if (
    (writeState === 'none' || writeState === 'na') &&
    readState === 'all'
  ) {
    overallLevel = 'read';
  } else {
    overallLevel = 'custom';
  }

  return {
    readState,
    writeState,
    overallLevel,
    selectedCount: totalSelected,
    totalCount: total,
  };
}

/**
 * 切换单个资源的“查看（只读）”权限。
 * enable = true: 勾选该资源所有查看权限，并补齐可能存在的依赖。
 * enable = false: 取消该资源所有查看权限，并级联取消依赖该读权限的所有写操作权限。
 */
export function toggleResourceRead(
  resource: PermissionResource,
  enable: boolean,
  currentSelected: string[],
  requiresByPermission: Record<string, string[]>,
): string[] {
  let next: string[];
  if (enable) {
    const set = new Set(currentSelected);
    for (const key of resource.readKeys) {
      set.add(key);
    }
    next = [...set];
  } else {
    const readSet = new Set(resource.readKeys);
    next = currentSelected.filter((k) => !readSet.has(k));
  }
  return applyPermissionLinkage(currentSelected, next, requiresByPermission);
}

/**
 * 切换单个资源的“编辑/管理（读写）”权限。
 * enable = true: 勾选该资源所有写权限与读权限，并补齐所有前置依赖。
 * enable = false: 取消该资源所有写权限，但保留该资源的查看权限（平滑降级为只读）。
 */
export function toggleResourceWrite(
  resource: PermissionResource,
  enable: boolean,
  currentSelected: string[],
  requiresByPermission: Record<string, string[]>,
): string[] {
  let next: string[];
  if (enable) {
    const set = new Set(currentSelected);
    for (const key of resource.allKeys) {
      set.add(key);
    }
    next = [...set];
  } else {
    const writeSet = new Set(resource.writeKeys);
    next = currentSelected.filter((k) => !writeSet.has(k));
  }
  return applyPermissionLinkage(currentSelected, next, requiresByPermission);
}

/**
 * 快速设置单个资源的授权级别（无权限 / 只读 / 完全控制）。
 */
export function setResourceLevel(
  resource: PermissionResource,
  level: 'none' | 'read' | 'full',
  currentSelected: string[],
  requiresByPermission: Record<string, string[]>,
): string[] {
  let next: string[];
  if (level === 'none') {
    const allSet = new Set(resource.allKeys);
    next = currentSelected.filter((k) => !allSet.has(k));
  } else if (level === 'read') {
    const writeSet = new Set(resource.writeKeys);
    const set = new Set(currentSelected.filter((k) => !writeSet.has(k)));
    for (const key of resource.readKeys) {
      set.add(key);
    }
    next = [...set];
  } else {
    const set = new Set(currentSelected);
    for (const key of resource.allKeys) {
      set.add(key);
    }
    next = [...set];
  }
  return applyPermissionLinkage(currentSelected, next, requiresByPermission);
}

/**
 * 批量设置指定范围（业务线、模块或全系统）的授权级别。
 */
export function setBatchScopeLevel(
  scope: {
    readKeys?: string[];
    allReadKeys?: string[];
    writeKeys?: string[];
    allWriteKeys?: string[];
    allKeys?: string[];
    allLeafKeys?: string[];
  },
  level: 'none' | 'read' | 'full',
  currentSelected: string[],
  requiresByPermission: Record<string, string[]>,
): string[] {
  const readKeys = scope.readKeys ?? scope.allReadKeys ?? [];
  const writeKeys = scope.writeKeys ?? scope.allWriteKeys ?? [];
  const allKeys = scope.allKeys ?? scope.allLeafKeys ?? [];

  let next: string[];
  if (level === 'none') {
    const allSet = new Set(allKeys);
    next = currentSelected.filter((k) => !allSet.has(k));
  } else if (level === 'read') {
    const writeSet = new Set(writeKeys);
    const set = new Set(currentSelected.filter((k) => !writeSet.has(k)));
    for (const key of readKeys) {
      set.add(key);
    }
    next = [...set];
  } else {
    const set = new Set(currentSelected);
    for (const key of allKeys) {
      set.add(key);
    }
    next = [...set];
  }
  return applyPermissionLinkage(currentSelected, next, requiresByPermission);
}

/**
 * 搜索过滤权限资源列表。
 * 若 resource 本身路径或名称匹配，或其包含的某个 action 名称/编码/说明匹配，则保留该资源。
 */
export function filterResources(
  resources: PermissionResource[],
  keyword: string,
): { filtered: PermissionResource[]; matchedActionKeys: Set<string> } {
  const kw = keyword.trim().toLowerCase();
  const matchedActionKeys = new Set<string>();

  if (!kw) {
    return { filtered: resources, matchedActionKeys };
  }

  const filtered = resources.filter((res) => {
    const pathMatch =
      res.fullPath.toLowerCase().includes(kw) ||
      res.name.toLowerCase().includes(kw) ||
      Boolean(res.subModule?.toLowerCase().includes(kw));

    let hasActionMatch = false;
    for (const action of res.allActions) {
      const match =
        action.name.toLowerCase().includes(kw) ||
        action.key.toLowerCase().includes(kw) ||
        Boolean(action.description?.toLowerCase().includes(kw));
      if (match) {
        hasActionMatch = true;
        matchedActionKeys.add(action.key);
      }
    }

    return pathMatch || hasActionMatch;
  });

  return { filtered, matchedActionKeys };
}
