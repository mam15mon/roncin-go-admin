import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { masterDataServiceListCurrencies } from '@/services/roncin/masterDataService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
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

// 展开后计价币种字段会挂载远程下拉并真实请求币种服务，这里 mock 掉
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListCurrencies: vi.fn().mockResolvedValue({
    data: [{ code: 'CNY', name: '人民币', enabled: true }],
  }),
}));

describe('FeeLedgerSearchFilter', () => {
  it('正确基于 SearchFilterTemplate 渲染首屏 5 项高密度行内搜索与展开按钮', async () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(<FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />);

    expect(screen.getByText('综合搜索')).not.toBeNull();
    expect(screen.getByText('费用属性')).not.toBeNull();
    expect(screen.getByText('财务进度')).not.toBeNull();
    expect(screen.getByText('结算单位')).not.toBeNull();
    // 费用已无独立状态，不再提供「费用状态」筛选项
    expect(screen.queryByText('费用状态')).toBeNull();
    expect(screen.getByText(/展开/)).not.toBeNull();

    // 结算单位远程下拉挂载请求收敛，避免异步状态更新落到 act 之外
    await waitFor(() => expect(partnerServiceListPartners).toHaveBeenCalled());
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
  });

  it('点击展开时展现其余 16 项业务字段', async () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(<FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />);

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

    // 展开后挂载的结算单位/委托单位/计价币种远程下拉请求收敛
    await waitFor(() =>
      expect(masterDataServiceListCurrencies).toHaveBeenCalled(),
    );
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
  });

  it('点击查询和重置正常触发回调', async () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();

    render(<FeeLedgerSearchFilter onSearch={onSearch} onReset={onReset} />);

    fireEvent.click(screen.getByText('重置'));
    expect(onReset).toHaveBeenCalled();

    // 挂载的结算单位远程下拉请求收敛，避免异步状态更新落到 act 之外
    await waitFor(() => expect(partnerServiceListPartners).toHaveBeenCalled());
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
  });

  it('结算单位筛选渲染远程搜索下拉并展示服务端候选项', async () => {
    render(<FeeLedgerSearchFilter onSearch={vi.fn()} onReset={vi.fn()} />);

    // 挂载即触发结算单位候选请求（首屏 50 条，服务端关键字过滤）
    await waitFor(() =>
      expect(partnerServiceListPartners).toHaveBeenCalledWith(
        expect.objectContaining({ page: 1, pageSize: 50 }),
      ),
    );

    fireEvent.mouseDown(screen.getByLabelText('结算单位'));
    expect(await screen.findByText('宁波中远海运 (COSCO)')).toBeInTheDocument();
    expect(screen.getByText('上海美森轮船 (MATSON)')).toBeInTheDocument();
  });
});
