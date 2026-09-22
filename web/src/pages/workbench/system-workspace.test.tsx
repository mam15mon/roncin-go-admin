import { renderWithApp } from '@root/tests/renderWithApp';
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import Workbench from './index';

const overview = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/workbenchService', () => ({
  workbenchServiceGetWorkbenchOverview: overview,
}));

describe('系统管理工作台', () => {
  it('只展示授权管理入口，不请求经营概览', () => {
    renderWithApp(<Workbench />, {
      user: {
        currentOrganization: {
          id: 'system',
          kind: AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM,
        },
        permissionCapabilities: [
          { key: 'system.organization.read', dataScope: 'all' },
          { key: 'system.master_data.port.read', dataScope: 'all' },
          { key: 'system.finance.fee_setting.read', dataScope: 'all' },
        ],
      },
    });
    expect(
      screen.getByRole('link', { name: '公共基础资料' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '公司管理' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '注册审批' })).toBeNull();
    expect(overview).not.toHaveBeenCalled();
  });
});
