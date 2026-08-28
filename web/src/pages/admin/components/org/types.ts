export const organizationKindMeta = {
  1: { label: '总部', color: 'purple' },
  2: { label: '公司', color: 'blue' },
  3: { label: '部门', color: 'cyan' },
  4: { label: '组', color: 'gold' },
} as const;

export function getOrganizationKindMeta(kind?: number) {
  return organizationKindMeta[kind as keyof typeof organizationKindMeta];
}

export function normalizeOrganizationKind(
  kind: API.AdminOrganization['kind'],
): number {
  if (typeof kind === 'number') return kind;
  switch (String(kind)) {
    case 'ORGANIZATION_KIND_HEADQUARTERS':
      return 1;
    case 'ORGANIZATION_KIND_COMPANY':
      return 2;
    case 'ORGANIZATION_KIND_DEPARTMENT':
      return 3;
    case 'ORGANIZATION_KIND_TEAM':
      return 4;
    default:
      return 0;
  }
}

export function getChildOrganizationKind(kind?: number): 2 | 3 | 4 | undefined {
  if (kind === 1) return 2;
  if (kind === 2) return 3;
  if (kind === 3) return 4;
  return undefined;
}

export type CreateFormValues = {
  code?: string;
  name?: string;
  baseCurrency?: string;
};

export type EditFormValues = {
  name?: string;
  enabled?: boolean;
  baseCurrency?: string;
};
