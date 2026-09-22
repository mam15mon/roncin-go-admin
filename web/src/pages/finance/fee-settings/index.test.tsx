import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import FeeSettingsPage from './index';

const access = vi.hoisted(() => ({
  isSystemWorkspace: true,
  canOperateBusiness: false,
  canReadFeeSettings: true,
  canReadFinanceBills: true,
}));
vi.mock('@/app/access', () => ({ useAccess: () => access }));
vi.mock('@/components/ui', () => ({
  MultiTabCenterTemplate: ({
    items,
  }: {
    items: { key: string; visible?: boolean; children: React.ReactNode }[];
  }) => (
    <>
      {items
        .filter((item) => item.visible)
        .map((item) => (
          <div key={item.key}>{item.children}</div>
        ))}
    </>
  ),
}));
vi.mock('./components/FeeItemsPanel', () => ({
  default: () => <div>公司科目</div>,
}));
vi.mock('./components/FeeTemplatesPanel', () => ({
  default: () => <div>系统初始目录</div>,
}));
vi.mock('./components/TaxableServicesPanel', () => ({
  default: () => <div>公司开票项目</div>,
}));
vi.mock('./components/CustomSettingsPanel', () => ({
  default: () => <div>公司财务策略</div>,
}));
vi.mock('./components/BillingUnitsPanel', () => ({
  default: () => <div>公共计费单位</div>,
}));

afterEach(cleanup);
describe('费用设置按工作台划分', () => {
  it('系统管理显示模板与公共计费单位，不挂载公司税务和策略面板', () => {
    access.isSystemWorkspace = true;
    access.canOperateBusiness = false;
    render(<FeeSettingsPage />);
    expect(screen.getByText('系统初始目录')).toBeInTheDocument();
    expect(screen.getByText('公共计费单位')).toBeInTheDocument();
    expect(screen.queryByText('公司开票项目')).toBeNull();
    expect(screen.queryByText('公司财务策略')).toBeNull();
    expect(screen.queryByText('公司科目')).toBeNull();
  });
  it('公司使用自己的科目、开票项目和财务策略', () => {
    access.isSystemWorkspace = false;
    access.canOperateBusiness = true;
    render(<FeeSettingsPage />);
    expect(screen.getByText('公司科目')).toBeInTheDocument();
    expect(screen.getByText('公司开票项目')).toBeInTheDocument();
    expect(screen.getByText('公司财务策略')).toBeInTheDocument();
    expect(screen.queryByText('系统初始目录')).toBeNull();
  });
});
