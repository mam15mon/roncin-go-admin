import { ProFormText } from '@ant-design/pro-components';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getFormDraft,
  getFormDraftKey,
  getFormDraftScope,
  hasTabDraft,
} from '@/components/layout/formDraft';
import {
  _clearAllTabCloseGuards,
  isTabDirty,
} from '@/components/layout/tabCloseGuard';
import { OrderFormTemplate } from './OrderFormTemplate';

describe('OrderFormTemplate Component', () => {
  const draftScope = getFormDraftScope('user-1', 'org-1');

  beforeEach(() => {
    _clearAllTabCloseGuards();
    sessionStorage.clear();
  });

  afterEach(() => {
    _clearAllTabCloseGuards();
    sessionStorage.clear();
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

    expect(
      container.querySelector('.roncin-order-form-skeleton'),
    ).not.toBeInTheDocument();
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

  it('表单输入内容发生修改时，自动暂存草稿至 sessionStorage', async () => {
    const tabKey = '/orders/sea-export';
    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftScope={draftScope}
        sections={[
          {
            key: 'basic',
            title: '业务信息',
            content: (
              <ProFormText
                name="customerReferenceNo"
                label="客户参考号"
                placeholder="请输入客户参考号"
              />
            ),
          },
        ]}
      />,
    );

    const input = screen.getByPlaceholderText('请输入客户参考号');
    act(() => {
      fireEvent.change(input, { target: { value: 'CR-999' } });
    });

    expect(hasTabDraft(tabKey, draftScope)).toBe(true);
    const draft = getFormDraft<{ customerReferenceNo: string }>(
      getFormDraftKey(tabKey, window.location?.pathname, draftScope),
    );
    expect(draft).not.toBeNull();
    expect(draft?.customerReferenceNo).toBe('CR-999');
  });

  it('重新挂载（模拟切走后切回）时，自动恢复草稿至表单字段并触发 onDirtyChange', async () => {
    const tabKey = '/orders/sea-export';
    const draftKey = getFormDraftKey(
      tabKey,
      window.location?.pathname,
      draftScope,
    );
    sessionStorage.setItem(
      draftKey,
      JSON.stringify({ customerReferenceNo: 'RESTORED-123' }),
    );

    const onDirtyChange = vi.fn();

    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftScope={draftScope}
        onDirtyChange={onDirtyChange}
        sections={[
          {
            key: 'basic',
            title: '业务信息',
            content: (
              <ProFormText
                name="customerReferenceNo"
                label="客户参考号"
                placeholder="请输入客户参考号"
              />
            ),
          },
        ]}
      />,
    );

    const input = screen.getByPlaceholderText(
      '请输入客户参考号',
    ) as HTMLInputElement;
    expect(input.value).toBe('RESTORED-123');
    expect(onDirtyChange).toHaveBeenCalledWith(true);
    expect(isTabDirty(tabKey)).toBe(true);
  });

  it('只读挂载时不读取持久草稿，展示服务端 initialValues', () => {
    const tabKey = '/orders/sea-export';
    const draftKey = getFormDraftKey(
      tabKey,
      window.location?.pathname,
      draftScope,
    );
    sessionStorage.setItem(
      draftKey,
      JSON.stringify({ customerReferenceNo: 'DRAFT-DIRTY' }),
    );

    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftScope={draftScope}
        readonly={true}
        initialValues={{ customerReferenceNo: 'SERVER-ORIGINAL' }}
        sections={[
          {
            key: 'basic',
            title: '业务信息',
            content: (
              <ProFormText
                name="customerReferenceNo"
                label="客户参考号"
                placeholder="请输入客户参考号"
              />
            ),
          },
        ]}
      />,
    );

    expect(screen.getByText('SERVER-ORIGINAL')).toBeInTheDocument();
    expect(screen.queryByText('DRAFT-DIRTY')).not.toBeInTheDocument();
    expect(isTabDirty(tabKey)).toBe(false);
  });

  it('成功提交时清除当前草稿，提交返回 false 时保留草稿', async () => {
    const tabKey = '/orders/sea-export';
    const draftKey = getFormDraftKey(
      tabKey,
      window.location?.pathname,
      draftScope,
    );
    sessionStorage.setItem(
      draftKey,
      JSON.stringify({ customerReferenceNo: 'TO-SUBMIT' }),
    );

    let submitResult = false;
    const onFinish = vi.fn().mockImplementation(async () => submitResult);

    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftScope={draftScope}
        submitText="保存订单"
        onFinish={onFinish}
        sections={[
          {
            key: 'basic',
            title: '业务信息',
            content: (
              <ProFormText
                name="customerReferenceNo"
                label="客户参考号"
                placeholder="请输入客户参考号"
              />
            ),
          },
        ]}
      />,
    );

    const submitBtn = screen.getByText('保存订单');

    // 1. 提交返回 false，草稿保留
    await act(async () => {
      fireEvent.click(submitBtn);
    });
    expect(onFinish).toHaveBeenCalledTimes(1);
    expect(hasTabDraft(tabKey, draftScope)).toBe(true);

    // 2. 提交返回 true，草稿清除
    submitResult = true;
    await act(async () => {
      fireEvent.click(submitBtn);
    });
    expect(onFinish).toHaveBeenCalledTimes(2);
    expect(hasTabDraft(tabKey, draftScope)).toBe(false);
    expect(isTabDirty(tabKey)).toBe(false);
  });

  it('点击重置表单时，清除暂存草稿并重置 dirty 状态', async () => {
    const tabKey = '/orders/sea-export';
    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftScope={draftScope}
        resetText="重置表单"
        sections={[
          {
            key: 'basic',
            title: '业务信息',
            content: (
              <ProFormText
                name="customerReferenceNo"
                label="客户参考号"
                placeholder="请输入客户参考号"
              />
            ),
          },
        ]}
      />,
    );

    const input = screen.getByPlaceholderText('请输入客户参考号');
    act(() => {
      fireEvent.change(input, { target: { value: 'TEMP-DATA' } });
    });
    expect(hasTabDraft(tabKey, draftScope)).toBe(true);

    const resetBtn = screen.getByText('重置表单');
    act(() => {
      fireEvent.click(resetBtn);
    });

    expect(hasTabDraft(tabKey, draftScope)).toBe(false);
    expect(isTabDirty(tabKey)).toBe(false);
  });
});
