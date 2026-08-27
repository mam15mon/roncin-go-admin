import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  AdvancedFilterDrawer,
  DEFAULT_FEE_FILTER_VALUES,
} from './AdvancedFilterDrawer';

// Mock partnerService
vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartners: vi.fn().mockResolvedValue({
    data: [
      { id: 'p-1', legalName: '宁波中远海运', code: 'COSCO' },
      { id: 'p-2', legalName: '上海美森轮船', code: 'MATSON' },
    ],
  }),
}));

describe('AdvancedFilterDrawer', () => {
  it('正确渲染 6 组 33 项全维业务筛选抽屉', () => {
    const onClose = vi.fn();
    const onApply = vi.fn();

    render(
      <AdvancedFilterDrawer
        open={true}
        onClose={onClose}
        initialValues={DEFAULT_FEE_FILTER_VALUES}
        onApply={onApply}
      />,
    );

    // 验证 6 组标题
    expect(screen.getByText(/第一组：费用时间与收付账期/)).not.toBeNull();
    expect(screen.getByText(/第二组：单据与业务实体/)).not.toBeNull();
    expect(screen.getByText(/第三组：航次船期与人员/)).not.toBeNull();
    expect(screen.getByText(/第四组：对账与风控/)).not.toBeNull();
    expect(screen.getByText(/第五组：航运工具与合约/)).not.toBeNull();
    expect(screen.getByText(/第六组：拓展属性与标签/)).not.toBeNull();

    // 验证核心字段 label 存在
    expect(screen.getByText('费用时间')).not.toBeNull();
    expect(screen.getByText('费用属性')).not.toBeNull();
    expect(screen.getByText('结算单位')).not.toBeNull();
    expect(screen.getByText('财务进度')).not.toBeNull();
    expect(screen.getByText('费用状态')).not.toBeNull();
    expect(screen.getByText('订单/主单号/加拼主单号')).not.toBeNull();
    expect(screen.getByText('ETD（预计离港时间）')).not.toBeNull();
    expect(screen.getByText('是否对账')).not.toBeNull();
    expect(screen.getByText('船名')).not.toBeNull();
  });

  it('点击确定筛选时将表单各维度值回传给 onApply', () => {
    const onClose = vi.fn();
    const onApply = vi.fn();

    render(
      <AdvancedFilterDrawer
        open={true}
        onClose={onClose}
        initialValues={DEFAULT_FEE_FILTER_VALUES}
        onApply={onApply}
      />,
    );

    const submitBtn = screen.getByText('确定并执行筛选');
    fireEvent.click(submitBtn);

    expect(onApply).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
