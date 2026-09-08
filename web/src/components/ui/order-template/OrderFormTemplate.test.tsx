import { ProFormText } from '@ant-design/pro-components';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  _clearAllTabCloseGuards,
  isTabDirty,
} from '@/components/layout/tabCloseGuard';
import { OrderFormTemplate } from './OrderFormTemplate';

describe('OrderFormTemplate Component', () => {
  beforeEach(() => {
    _clearAllTabCloseGuards();
  });

  afterEach(() => {
    _clearAllTabCloseGuards();
    cleanup();
  });

  it('loading 为 true 时渲染分节骨架屏与 loadingTip，不挂载表单', () => {
    const { container } = render(
      <OrderFormTemplate
        loading={true}
        loadingTip="正在加载业务模板与主数据..."
        sections={[
          {
            key: 'basic',
            title: '真实业务信息',
            content: <div>表单内容</div>,
          },
        ]}
      />,
    );

    // 检查骨架屏与分节卡片
    expect(screen.getByText('正在加载业务模板与主数据...')).toBeInTheDocument();
    expect(screen.getByText('业务基本信息')).toBeInTheDocument();
    expect(screen.getByText('运输与订舱信息')).toBeInTheDocument();
    expect(container.querySelectorAll('.ant-skeleton')).toHaveLength(2);

    // 加载期间绝不渲染真实表单或提交按钮
    expect(screen.queryByText('真实业务信息')).not.toBeInTheDocument();
    expect(screen.queryByText('表单内容')).not.toBeInTheDocument();
    expect(container.querySelector('form')).not.toBeInTheDocument();
    expect(screen.queryByText('提交')).not.toBeInTheDocument();
  });

  it('loading 为 false 时正常渲染表单分节卡片与提交按钮', () => {
    const { container } = render(
      <OrderFormTemplate
        loading={false}
        submitText="创建海运订单"
        sections={[
          {
            key: 'basic',
            title: '真实业务信息',
            content: <div>表单内容区</div>,
          },
        ]}
      />,
    );

    expect(container.querySelector('.roncin-order-form-skeleton')).not.toBeInTheDocument();
    expect(screen.getByText('真实业务信息')).toBeInTheDocument();
    expect(screen.getByText('表单内容区')).toBeInTheDocument();
    expect(screen.getByText('创建海运订单')).toBeInTheDocument();
    expect(container.querySelector('form')).toBeInTheDocument();
  });

  it('表单输入内容发生修改时，触发 onDirtyChange 并注册 tabCloseGuard 为 dirty', async () => {
    const onDirtyChange = vi.fn();

    render(
      <OrderFormTemplate
        tabKey="/orders/sea-export"
        onDirtyChange={onDirtyChange}
        sections={[
          {
            key: 'basic',
            title: '业务信息',
            content: (
              <ProFormText
                name="orderNo"
                label="订单号"
                placeholder="请输入订单号"
              />
            ),
          },
        ]}
      />,
    );

    // 初始表单未修改
    expect(isTabDirty('/orders/sea-export')).toBe(false);

    // 模拟输入修改
    const input = screen.getByPlaceholderText('请输入订单号');
    act(() => {
      fireEvent.change(input, { target: { value: 'SE20260908' } });
    });

    expect(isTabDirty('/orders/sea-export')).toBe(true);
    expect(onDirtyChange).toHaveBeenCalledWith(true);
  });

  it('表单提交成功后，重置 dirty 状态', async () => {
    const onFinish = vi.fn().mockResolvedValue(true);

    render(
      <OrderFormTemplate
        tabKey="/orders/sea-export"
        submitText="保存订单"
        onFinish={onFinish}
        sections={[
          {
            key: 'basic',
            title: '业务信息',
            content: (
              <ProFormText
                name="orderNo"
                label="订单号"
                placeholder="请输入订单号"
              />
            ),
          },
        ]}
      />,
    );

    const input = screen.getByPlaceholderText('请输入订单号');
    act(() => {
      fireEvent.change(input, { target: { value: 'SE20260908' } });
    });
    expect(isTabDirty('/orders/sea-export')).toBe(true);

    const submitBtn = screen.getByText('保存订单');
    await act(async () => {
      fireEvent.click(submitBtn);
    });

    expect(onFinish).toHaveBeenCalled();
    expect(isTabDirty('/orders/sea-export')).toBe(false);
  });
});
