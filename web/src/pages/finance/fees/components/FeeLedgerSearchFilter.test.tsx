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
  it('正确渲染首屏核心 6 项与展开更多筛选按钮', () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(
      <FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />,
    );

    expect(screen.getByText('综合搜索')).not.toBeNull();
    expect(screen.getByText('属性')).not.toBeNull();
    expect(screen.getByText('财务进度')).not.toBeNull();
    expect(screen.getByText('状态')).not.toBeNull();
    expect(screen.getByText('结算单位')).not.toBeNull();
    expect(screen.getByText('费用时间')).not.toBeNull();
    expect(screen.getByText('展开')).not.toBeNull();
  });

  it('点击展开时完整展现多组专业维度字段', () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(
      <FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />,
    );

    fireEvent.click(screen.getByText('展开'));

    expect(screen.getByText(/单据编号与往来实体/)).not.toBeNull();
    expect(screen.getByText(/航次船期与业务责任人/)).not.toBeNull();
    expect(screen.getByText(/账期审计、合约与风控标记/)).not.toBeNull();
    expect(screen.getByText('收起')).not.toBeNull();
  });

  it('点击查询和重置正常触发回调', () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(
      <FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />,
    );

    fireEvent.click(screen.getByText('查询'));
    expect(onSearch).toHaveBeenCalled();

    fireEvent.click(screen.getByText('重置'));
    expect(onReset).toHaveBeenCalled();
  });
});
