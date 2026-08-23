export type OrgTreeNode = {
  key: string;
  title: string;
  code: string;
  kind: number;
  enabled: boolean;
  raw: API.AdminOrganization;
  children?: OrgTreeNode[];
};

export type BuiltOrgTree = {
  treeData: OrgTreeNode[];
  allKeys: string[];
  orgMap: Map<string, API.AdminOrganization>;
};

/**
 * 将组织平铺列表构建为树结构
 * parentId 为空或指向不存在组织的节点统一作为根节点
 */
export function buildOrgTree(
  organizations: API.AdminOrganization[],
): BuiltOrgTree {
  const orgMap = new Map<string, API.AdminOrganization>();
  for (const org of organizations) {
    if (org.id) {
      orgMap.set(org.id, org);
    }
  }

  const childrenMap = new Map<string, API.AdminOrganization[]>();
  const roots: API.AdminOrganization[] = [];

  for (const org of organizations) {
    const parentId = org.parentId?.trim();
    if (!parentId || !orgMap.has(parentId)) {
      roots.push(org);
    } else {
      const arr = childrenMap.get(parentId) ?? [];
      arr.push(org);
      childrenMap.set(parentId, arr);
    }
  }

  const allKeys: string[] = [];

  const createNode = (org: API.AdminOrganization): OrgTreeNode => {
    const id = org.id ?? '';
    allKeys.push(id);
    const subList = childrenMap.get(id);
    return {
      key: id,
      title: org.name ?? '',
      code: org.code ?? '',
      kind: org.kind ?? 0,
      enabled: org.enabled ?? true,
      raw: org,
      children:
        subList && subList.length > 0 ? subList.map(createNode) : undefined,
    };
  };

  const treeData = roots.map(createNode);
  return { treeData, allKeys, orgMap };
}

/**
 * 获取指定组织的直接下级列表
 */
export function getDirectChildren(
  orgId: string,
  organizations: API.AdminOrganization[],
): API.AdminOrganization[] {
  if (!orgId) return [];
  return organizations.filter((org) => org.parentId === orgId);
}

/**
 * 递归计算指定组织的所有下级节点总数
 */
export function getTotalDescendantCount(
  orgId: string,
  organizations: API.AdminOrganization[],
): number {
  if (!orgId) return 0;
  let count = 0;
  const childrenMap = new Map<string, API.AdminOrganization[]>();

  for (const org of organizations) {
    const pId = org.parentId?.trim();
    if (pId) {
      const list = childrenMap.get(pId) ?? [];
      list.push(org);
      childrenMap.set(pId, list);
    }
  }

  const queue = [...(childrenMap.get(orgId) ?? [])];
  while (queue.length > 0) {
    const curr = queue.shift();
    if (!curr?.id) continue;
    count += 1;
    const nextSubs = childrenMap.get(curr.id);
    if (nextSubs && nextSubs.length > 0) {
      queue.push(...nextSubs);
    }
  }

  return count;
}

/**
 * 过滤组织树，保留命中节点及其全部祖先链，并收集需要展开的节点 key
 */
export function filterOrgTree(
  treeData: OrgTreeNode[],
  keyword: string,
): { filteredTree: OrgTreeNode[]; matchedKeys: string[] } {
  const kw = keyword.trim().toLowerCase();
  if (!kw) {
    return { filteredTree: treeData, matchedKeys: [] };
  }

  const matchedKeys: string[] = [];

  const filterNode = (node: OrgTreeNode): OrgTreeNode | null => {
    const titleMatch = node.title.toLowerCase().includes(kw);
    const codeMatch = node.code.toLowerCase().includes(kw);
    const isSelfMatch = titleMatch || codeMatch;

    const filteredChildren = node.children
      ?.map(filterNode)
      .filter((child): child is OrgTreeNode => child !== null);

    const hasMatchingChildren = Boolean(
      filteredChildren && filteredChildren.length > 0,
    );

    if (isSelfMatch || hasMatchingChildren) {
      matchedKeys.push(node.key);
      return {
        ...node,
        children: hasMatchingChildren ? filteredChildren : undefined,
      };
    }

    return null;
  };

  const filteredTree = treeData
    .map(filterNode)
    .filter((node): node is OrgTreeNode => node !== null);

  return { filteredTree, matchedKeys };
}
