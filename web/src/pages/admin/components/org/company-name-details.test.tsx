import { renderWithApp } from '@root/tests/renderWithApp';
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import OrgDetailCard from './OrgDetailCard';
import OrgInspectorPanel from './OrgInspectorPanel';

vi.mock('@/components/ui/column-settings', () => ({
  useColumnSettings: ({ columns }: { columns: unknown[] }) => ({
    columns,
    entry: null,
  }),
}));

const company: API.AdminOrganization = {
  id: 'company-1',
  kind: 2,
  code: 'TEST',
  name: '测试公司',
  nameEn: 'Confirmed Company Name Co., Ltd',
};
const common = {
  selectedOrg: company,
  directChildren: [],
  totalDescendantCount: 0,
  canCreate: false,
  canUpdate: false,
  onOpenCreateChild: vi.fn(),
  onOpenEdit: vi.fn(),
};

describe('公司详情英文名称', () => {
  it('树表详情展示英文名称，非公司不展示该档案字段', () => {
    const view = renderWithApp(
      <OrgDetailCard {...common} onSelectOrg={vi.fn()} />,
    );
    expect(screen.getByText('公司英文名称')).toBeVisible();
    expect(screen.getByText(company.nameEn as string)).toBeVisible();
    view.rerender(
      <OrgDetailCard
        {...common}
        selectedOrg={{ ...company, kind: 3 }}
        onSelectOrg={vi.fn()}
      />,
    );
    expect(screen.queryByText('公司英文名称')).toBeNull();
  });

  it('默认拓扑视图的详情面板也展示公司英文名称', () => {
    renderWithApp(
      <OrgInspectorPanel
        {...common}
        open
        onClose={vi.fn()}
        onSelectNode={vi.fn()}
      />,
    );
    expect(screen.getByText('公司英文名称')).toBeVisible();
    expect(screen.getByText(company.nameEn as string)).toBeVisible();
  });
});
