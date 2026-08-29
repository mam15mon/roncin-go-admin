import type {
  PermissionGroupNode,
  PermissionLeafNode,
  PermissionTreeNode,
} from './roleConstants';

type PermissionDefinition = {
  key?: string;
  name?: string;
  group?: string;
  description?: string;
  requires?: string[];
};

export type PermissionTreeModel = {
  treeData: PermissionTreeNode[];
  allBranchKeys: string[];
  initialExpandedKeys: string[];
  allLeafKeys: string[];
  requiresByPermission: Record<string, string[]>;
  permissionNameByKey: Record<string, string>;
};

const branchKey = (path: string[]) => `group:${path.join(' · ')}`;

export function isPermissionGroupNode(
  node: PermissionTreeNode,
): node is PermissionGroupNode {
  return 'children' in node;
}

export function buildPermissionTree(
  permissions: PermissionDefinition[],
): PermissionTreeModel {
  const treeData: PermissionTreeNode[] = [];
  const branches = new Map<string, PermissionGroupNode>();
  const allBranchKeys: string[] = [];
  const initialExpandedKeys: string[] = [];
  const allLeafKeys: string[] = [];
  const requiresByPermission: Record<string, string[]> = {};
  const permissionNameByKey: Record<string, string> = {};

  for (const permission of permissions) {
    if (!permission.key) continue;
    const path = (permission.group || '其他功能权限')
      .split('·')
      .map((segment) => segment.trim())
      .filter(Boolean);
    let siblings = treeData;

    for (let index = 0; index < path.length; index += 1) {
      const currentPath = path.slice(0, index + 1);
      const key = branchKey(currentPath);
      let branch = branches.get(key);
      if (!branch) {
        branch = {
          key,
          title: currentPath[index],
          groupName: currentPath.join(' · '),
          path: currentPath,
          isLeaf: false,
          children: [],
        };
        branches.set(key, branch);
        siblings.push(branch);
        allBranchKeys.push(key);
        if (currentPath.length <= 2) initialExpandedKeys.push(key);
      }
      siblings = branch.children;
    }

    const leaf: PermissionLeafNode = {
      key: permission.key,
      title: permission.name ?? permission.key,
      name: permission.name ?? permission.key,
      group: path.join(' · '),
      description: permission.description,
      requires: permission.requires ?? [],
      isLeaf: true,
    };
    siblings.push(leaf);
    allLeafKeys.push(permission.key);
    requiresByPermission[permission.key] = permission.requires ?? [];
    permissionNameByKey[permission.key] = leaf.name;
  }

  return {
    treeData,
    allBranchKeys,
    initialExpandedKeys,
    allLeafKeys,
    requiresByPermission,
    permissionNameByKey,
  };
}

export function filterPermissionTree(
  nodes: PermissionTreeNode[],
  keyword: string,
): PermissionTreeNode[] {
  const normalized = keyword.trim().toLowerCase();
  if (!normalized) return nodes;

  const filterNode = (node: PermissionTreeNode): PermissionTreeNode | null => {
    if (!isPermissionGroupNode(node)) {
      return node.name.toLowerCase().includes(normalized) ||
        node.key.toLowerCase().includes(normalized) ||
        node.description?.toLowerCase().includes(normalized)
        ? node
        : null;
    }
    const children = node.children
      .map(filterNode)
      .filter((child): child is PermissionTreeNode => child !== null);
    return children.length > 0 ? { ...node, children } : null;
  };

  return nodes
    .map(filterNode)
    .filter((node): node is PermissionTreeNode => node !== null);
}

export function collectPermissionLeafKeys(
  nodes: PermissionTreeNode[],
): string[] {
  const keys: string[] = [];
  const visit = (node: PermissionTreeNode) => {
    if (isPermissionGroupNode(node)) {
      node.children.forEach(visit);
      return;
    }
    keys.push(node.key);
  };
  nodes.forEach(visit);
  return keys;
}
