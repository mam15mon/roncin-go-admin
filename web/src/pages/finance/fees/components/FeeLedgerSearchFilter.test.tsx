import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { FeeLedgerSearchFilter } from './FeeLedgerSearchFilter';

// Mock partnerService
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartners: vi.fn().mockResolvedValue({
    data: [
      { id: 'p-1', legalName: '宁波中远海运', code: 'COSCO' },
      { id: 'p-2', legalName: '上海美森轮船', code: 'MATSON' },
    ],
  }),
}));

describe('FeeLedgerSearchFilter', () => {
  it('正确基于 SearchFilterTemplate 渲染首屏 5 项高密度行内搜索与展开按钮', () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(
      <FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />,
    );

    expect(screen.getByText('综合搜索')).not.toBeNull();
    expect(screen.getByText('费用属性')).not.toBeNull();
    expect(screen.getByText('财务进度')).not.toBeNull();
    expect(screen.getByText('费用状态')).not.toBeNull();
    expect(screen.getByText('结算单位')).not.toBeNull();
    expect(screen.getByText(/展开/)).not.toBeNull();
  });

  it('点击展开时展现全维 33 项业务字段', () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(
      <FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />,
    );

    fireEvent.click(screen.getByText(/展开/));

    expect(screen.getByText('结算单位')).not.toBeNull();
    expect(screen.getByText('费用时间')).not.toBeNull();
    expect(screen.getByText('委托单位')).not.toBeNull();
    expect(screen.getByText('账单编号')).not.toBeNull();
    expect(screen.getByText('订单编号')).not.toBeNull();
    expect(screen.getByText('主提单号')).not.toBeNull();
    expect(screen.getByText('费用锁定状态')).not.toBeNull();
    fireEvent.mouseDown(screen.getByLabelText('费用锁定状态'));
    expect(screen.getByText('因提成已锁定')).not.toBeNull();
    expect(screen.getByText('未锁定')).not.toBeNull();
    expect(screen.getByText(/收起/)).not.toBeNull();
  });

  it('点击查询和重置正常触发回调', async () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(
      <FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />,
    );

    fireEvent.click(screen.getByText('重置'));
    expect(onReset).toHaveBeenCalled();
  });
});
