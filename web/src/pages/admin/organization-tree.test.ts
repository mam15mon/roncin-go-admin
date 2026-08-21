import { describe, expect, it } from 'vitest';
import {
  buildOrgTree,
  filterOrgTree,
  getDirectChildren,
  getTotalDescendantCount,
} from './organization-tree';

describe('organization-tree utils', () => {
  const mockOrgs: API.AdminOrganization[] = [
    { id: '1', code: 'HQ', name: '总部集团', parentId: '', enabled: true },
    { id: '2', code: 'SH', name: '上海分公司', parentId: '1', enabled: true },
    { id: '3', code: 'SH_OPS', name: '上海操作部', parentId: '2', enabled: true },
    { id: '4', code: 'SH_FIN', name: '上海财务部', parentId: '2', enabled: false },
    { id: '5', code: 'SZ', name: '深圳分公司', parentId: '1', enabled: true },
    { id: '6', code: 'ISOLATED', name: '孤立组织', parentId: '999', enabled: true },
  ];

  it('buildOrgTree builds hierarchy and treats invalid parentId as root', () => {
    const { treeData, allKeys, orgMap } = buildOrgTree(mockOrgs);

    expect(treeData).toHaveLength(2); // HQ and ISOLATED (parentId 999 not found)
    expect(treeData[0].key).toBe('1');
    expect(treeData[0].children).toHaveLength(2); // SH and SZ
    expect(treeData[0].children?.[0].children).toHaveLength(2); // SH_OPS and SH_FIN
    expect(treeData[1].key).toBe('6'); // ISOLATED

    expect(allKeys).toEqual(['1', '2', '3', '4', '5', '6']);
    expect(orgMap.get('1')?.name).toBe('总部集团');
  });

  it('getDirectChildren returns direct sub-organizations only', () => {
    const childrenOfHQ = getDirectChildren('1', mockOrgs);
    expect(childrenOfHQ.map((o) => o.id)).toEqual(['2', '5']);

    const childrenOfSH = getDirectChildren('2', mockOrgs);
    expect(childrenOfSH.map((o) => o.id)).toEqual(['3', '4']);

    const childrenOfLeaf = getDirectChildren('3', mockOrgs);
    expect(childrenOfLeaf).toEqual([]);
  });

  it('getTotalDescendantCount recursively counts all sub-levels', () => {
    expect(getTotalDescendantCount('1', mockOrgs)).toBe(4); // SH, SH_OPS, SH_FIN, SZ
    expect(getTotalDescendantCount('2', mockOrgs)).toBe(2); // SH_OPS, SH_FIN
    expect(getTotalDescendantCount('3', mockOrgs)).toBe(0);
    expect(getTotalDescendantCount('6', mockOrgs)).toBe(0);
  });

  it('filterOrgTree filters by title or code keeping ancestor chain', () => {
    const { treeData } = buildOrgTree(mockOrgs);

    // Search by leaf node name "财务"
    const searchFin = filterOrgTree(treeData, '财务');
    expect(searchFin.filteredTree).toHaveLength(1);
    expect(searchFin.filteredTree[0].key).toBe('1');
    expect(searchFin.filteredTree[0].children?.[0].key).toBe('2');
    expect(searchFin.filteredTree[0].children?.[0].children?.[0].key).toBe('4');
    expect(searchFin.matchedKeys).toContain('4');
    expect(searchFin.matchedKeys).toContain('2');
    expect(searchFin.matchedKeys).toContain('1');

    // Search by code "sz"
    const searchSZ = filterOrgTree(treeData, 'sz');
    expect(searchSZ.filteredTree).toHaveLength(1);
    expect(searchSZ.filteredTree[0].children?.[0].key).toBe('5');

    // Empty keyword returns original tree
    const emptySearch = filterOrgTree(treeData, '');
    expect(emptySearch.filteredTree).toEqual(treeData);
    expect(emptySearch.matchedKeys).toEqual([]);
  });
});
