import { buildOrgTree, type OrgTreeNode } from '../../organization-tree';

/** 组织画布树数据的纯计算逻辑：树回填、按 key 检索与折叠键收集。 */

/** 组织图数据（G6 风格）：节点载荷挂 data 字段，缺失时节点自身承载组织字段。 */
export type OrgGraphData = {
  nodes: { id?: string; data?: API.AdminOrganization }[];
  edges: { source: string; target: string }[];
};

// 优先使用 treeData prop，缺失时由 graphData 重建组织树
export function resolveTreeData(
  treeData: OrgTreeNode[] | undefined,
  graphData: OrgGraphData | undefined,
): OrgTreeNode[] {
  if (treeData && treeData.length > 0) return treeData;
  if (!graphData?.nodes || graphData.nodes.length === 0) return [];

  const orgs: API.AdminOrganization[] = graphData.nodes.map((n) => {
    const d = (n.data || n) as API.AdminOrganization;
    return {
      id: d.id,
      name: d.name,
      code: d.code,
      kind: d.kind,
      enabled: d.enabled,
      parentId: d.parentId,
    };
  });
  return buildOrgTree(orgs).treeData;
}

// 深度优先查找 key 对应的节点原始组织数据
export function findNodeRawByKey(
  nodes: OrgTreeNode[],
  key: string,
): API.AdminOrganization | null {
  for (const node of nodes) {
    if (node.key === key) return node.raw;
    if (node.children) {
      const res = findNodeRawByKey(node.children, key);
      if (res) return res;
    }
  }
  return null;
}

// 深度优先查找 key 对应节点的直属下级组织列表
export function findChildRawsByKey(
  nodes: OrgTreeNode[],
  key: string,
): API.AdminOrganization[] {
  for (const node of nodes) {
    if (node.key === key) return node.children?.map((c) => c.raw) ?? [];
    if (node.children) {
      const res = findChildRawsByKey(node.children, key);
      if (res.length > 0) return res;
    }
  }
  return [];
}

// 查找目标节点的全部祖先 key 路径（不含目标自身）
export function findAncestorKeys(
  nodes: OrgTreeNode[],
  targetId: string,
  path: string[] = [],
): string[] | null {
  for (const n of nodes) {
    if (n.key === targetId) return path;
    if (n.children && n.children.length > 0) {
      const res = findAncestorKeys(n.children, targetId, [...path, n.key]);
      if (res) return res;
    }
  }
  return null;
}

// 收集全部拥有下级的节点 key，供“全部折叠”使用
export function collectBranchKeys(nodes: OrgTreeNode[]): Set<string> {
  const keys = new Set<string>();
  const traverse = (list: OrgTreeNode[]) => {
    for (const node of list) {
      if (node.children && node.children.length > 0) {
        keys.add(node.key);
        traverse(node.children);
      }
    }
  };
  traverse(nodes);
  return keys;
}
