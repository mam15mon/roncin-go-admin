import { describe, expect, it } from 'vitest';
import { AdminOrganizationKind } from '@/enums.generated';
import {
  formatOrganizationHierarchyName,
  isWorkspaceKindValue,
} from './userConstants';

const headquarters: API.AdminOrganization = {
  id: 'org-hq',
  name: '系统管理',
  kind: AdminOrganizationKind.ORGANIZATION_KIND_SYSTEM,
};
const company: API.AdminOrganization = {
  id: 'org-company',
  name: '上海公司',
  parentId: headquarters.id,
  kind: AdminOrganizationKind.ORGANIZATION_KIND_COMPANY,
};
const department: API.AdminOrganization = {
  id: 'org-dept',
  name: '财务部',
  parentId: company.id,
  kind: AdminOrganizationKind.ORGANIZATION_KIND_DEPARTMENT,
};
const subDepartment: API.AdminOrganization = {
  id: 'org-sub-dept',
  name: '结算组',
  parentId: department.id,
  kind: AdminOrganizationKind.ORGANIZATION_KIND_DEPARTMENT,
};
const team: API.AdminOrganization = {
  id: 'org-team',
  name: '退税组',
  parentId: subDepartment.id,
  kind: AdminOrganizationKind.ORGANIZATION_KIND_TEAM,
};

const organizations: API.AdminOrganization[] = [
  headquarters,
  company,
  department,
  subDepartment,
  team,
];

describe('isWorkspaceKindValue', () => {
  it('只把系统管理与公司判定为工作台节点', () => {
    expect(
      isWorkspaceKindValue(AdminOrganizationKind.ORGANIZATION_KIND_SYSTEM),
    ).toBe(true);
    expect(
      isWorkspaceKindValue(AdminOrganizationKind.ORGANIZATION_KIND_COMPANY),
    ).toBe(true);
  });

  it('部门、团队与未提供 kind 都不判定为工作台节点', () => {
    expect(
      isWorkspaceKindValue(AdminOrganizationKind.ORGANIZATION_KIND_DEPARTMENT),
    ).toBe(false);
    expect(
      isWorkspaceKindValue(AdminOrganizationKind.ORGANIZATION_KIND_TEAM),
    ).toBe(false);
    expect(
      isWorkspaceKindValue(AdminOrganizationKind.ORGANIZATION_KIND_UNSPECIFIED),
    ).toBe(false);
    expect(isWorkspaceKindValue(undefined)).toBe(false);
  });
});

describe('formatOrganizationHierarchyName', () => {
  it('部门展示为「公司 / 部门」', () => {
    expect(formatOrganizationHierarchyName(department.id, organizations)).toBe(
      '上海公司 / 财务部',
    );
  });

  it('多级部门保留中间层，直到工作台为止', () => {
    expect(
      formatOrganizationHierarchyName(subDepartment.id, organizations),
    ).toBe('上海公司 / 财务部 / 结算组');
    expect(formatOrganizationHierarchyName(team.id, organizations)).toBe(
      '上海公司 / 财务部 / 结算组 / 退税组',
    );
  });

  it('系统管理与公司直接展示自身名称，不再向上拼接', () => {
    expect(formatOrganizationHierarchyName(company.id, organizations)).toBe(
      '上海公司',
    );
    expect(
      formatOrganizationHierarchyName(headquarters.id, organizations),
    ).toBe('系统管理');
  });

  it('支持直接传入组织对象，并保持部门拼接口径一致', () => {
    expect(formatOrganizationHierarchyName(department, organizations)).toBe(
      '上海公司 / 财务部',
    );
  });

  it('未提供组织、传入未知组织 ID 时返回空字符串', () => {
    expect(formatOrganizationHierarchyName(undefined, organizations)).toBe('');
    expect(formatOrganizationHierarchyName('org-missing', organizations)).toBe(
      '',
    );
  });

  it('父链断裂或成环时不追加不可达的上级名称', () => {
    const orphan: API.AdminOrganization = {
      id: 'org-orphan',
      name: '孤立部门',
      parentId: 'org-missing',
      kind: AdminOrganizationKind.ORGANIZATION_KIND_DEPARTMENT,
    };
    expect(
      formatOrganizationHierarchyName(orphan, [...organizations, orphan]),
    ).toBe('孤立部门');

    const loopA: API.AdminOrganization = {
      id: 'org-loop-a',
      name: 'A 部门',
      parentId: 'org-loop-b',
      kind: AdminOrganizationKind.ORGANIZATION_KIND_DEPARTMENT,
    };
    const loopB: API.AdminOrganization = {
      id: 'org-loop-b',
      name: 'B 部门',
      parentId: 'org-loop-a',
      kind: AdminOrganizationKind.ORGANIZATION_KIND_DEPARTMENT,
    };
    expect(formatOrganizationHierarchyName(loopA, [loopA, loopB])).toBe(
      'B 部门 / A 部门',
    );
  });
});
