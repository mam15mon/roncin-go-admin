import { renderWithApp } from '@root/tests/renderWithApp';
import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import AirMasterPage from './air-master';

const permission = vi.hoisted(() => ({ transition: false }));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: (_type: number, operation: string) =>
      operation === 'transition'
        ? permission.transition
        : operation !== 'create',
  }),
}));
vi.mock('react-router', async (original) => ({
  ...(await original<typeof import('react-router')>()),
  useParams: () => ({ masterId: 'master' }),
}));
vi.mock('./order-kinds/air-export/AirWeightPanel', () => ({
  default: () => null,
}));
vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: vi.fn().mockResolvedValue({
    data: {
      id: 'master',
      version: '1',
      content: { masterNo: '78112345675', declaredVolumeCbm: 1.8 },
    },
    members: [],
    totals: { complete: true, volumeCbm: 1.2 + 0.6 },
    history: [],
  }),
}));

describe('航空主单恢复关联权限', () => {
  it('仅编辑权限允许维护共同资料，恢复关联还必须具备流程权限', async () => {
    permission.transition = false;
    const view = renderWithApp(<AirMasterPage />, {
      user: { id: 'user', currentOrganization: { id: 'org' } },
    });
    expect(
      await screen.findByRole('button', { name: '修改共同资料' }),
    ).toBeVisible();
    expect(await screen.findByText('1.8', { exact: true })).toBeVisible();
    expect(screen.getByText('1.8 / 0', { exact: true })).toBeVisible();
    expect(
      screen.queryByText('1.7999999999999998', { exact: true }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: '恢复关联已终止订单' }),
    ).not.toBeInTheDocument();
    permission.transition = true;
    view.rerender(<AirMasterPage />);
    expect(
      screen.getByRole('button', { name: '恢复关联已终止订单' }),
    ).toBeVisible();
  });
});
