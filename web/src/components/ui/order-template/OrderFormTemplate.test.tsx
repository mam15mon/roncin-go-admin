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
import { resolveTabKey } from '@/components/layout/routeUtils';
import {
  _clearAllTabCloseGuards,
  isTabDirty,
} from '@/components/layout/tabCloseGuard';
import { OrderFormTemplate } from './OrderFormTemplate';
import type { OrderFormTemplateActions } from './types';

const DRAFT_PATHNAME = '/orders/sea-export/new';

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

  it('表单输入内容发生修改时，注册 tabCloseGuard 为 dirty', async () => {
    render(
      <OrderFormTemplate
        tabKey="/orders/sea-export"
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

  it('缺少任一草稿身份输入时，输入不写入持久草稿', () => {
    const sections = [
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
    ];

    // 缺少 draftPathname：只有 tabKey + draftScope 时不持久化
    const { rerender } = render(
      <OrderFormTemplate
        tabKey="/orders/sea-export"
        draftScope={draftScope}
        sections={sections}
      />,
    );
    act(() => {
      fireEvent.change(screen.getByPlaceholderText('请输入客户参考号'), {
        target: { value: 'NO-PATHNAME' },
      });
    });
    expect(sessionStorage.length).toBe(0);

    // 缺少 draftScope：只有 tabKey + draftPathname 时不持久化
    rerender(
      <OrderFormTemplate
        tabKey="/orders/sea-export"
        draftPathname={DRAFT_PATHNAME}
        sections={sections}
      />,
    );
    act(() => {
      fireEvent.change(screen.getByPlaceholderText('请输入客户参考号'), {
        target: { value: 'NO-SCOPE' },
      });
    });
    expect(sessionStorage.length).toBe(0);
  });

  it('表单输入内容发生修改时，只按显式 draftPathname 暂存草稿', async () => {
    const tabKey = resolveTabKey(DRAFT_PATHNAME);
    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
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
    // 完整身份只命中唯一规范路径键，不依赖 window.location
    expect(sessionStorage.length).toBe(1);
    expect(sessionStorage.key(0)).toBe(
      getFormDraftKey(tabKey, DRAFT_PATHNAME, draftScope),
    );
    const draft = getFormDraft<{ customerReferenceNo: string }>(
      getFormDraftKey(tabKey, DRAFT_PATHNAME, draftScope),
    );
    expect(draft).not.toBeNull();
    expect(draft?.customerReferenceNo).toBe('CR-999');
  });

  it('重新挂载（模拟切走后切回）时，在首次绘制前恢复草稿并标记 dirty', async () => {
    const tabKey = resolveTabKey(DRAFT_PATHNAME);
    const draftKey = getFormDraftKey(tabKey, DRAFT_PATHNAME, draftScope);
    sessionStorage.setItem(
      draftKey,
      JSON.stringify({ customerReferenceNo: 'RESTORED-123' }),
    );

    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
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

    const input = screen.getByPlaceholderText(
      '请输入客户参考号',
    ) as HTMLInputElement;
    expect(input.value).toBe('RESTORED-123');
    expect(isTabDirty(tabKey)).toBe(true);
  });

  it('loading 结束后的首次可编辑时机才恢复草稿，恢复不晚于首次绘制', () => {
    const tabKey = resolveTabKey(DRAFT_PATHNAME);
    const draftKey = getFormDraftKey(tabKey, DRAFT_PATHNAME, draftScope);
    sessionStorage.setItem(
      draftKey,
      JSON.stringify({ customerReferenceNo: 'LOADED-DRAFT' }),
    );

    const sections = [
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
    ];

    const { rerender } = render(
      <OrderFormTemplate
        loading={true}
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
        draftScope={draftScope}
        sections={sections}
      />,
    );
    expect(
      screen.queryByPlaceholderText('请输入客户参考号'),
    ).not.toBeInTheDocument();

    rerender(
      <OrderFormTemplate
        loading={false}
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
        draftScope={draftScope}
        sections={sections}
      />,
    );

    const input = screen.getByPlaceholderText(
      '请输入客户参考号',
    ) as HTMLInputElement;
    expect(input.value).toBe('LOADED-DRAFT');
    expect(isTabDirty(tabKey)).toBe(true);
  });

  it('初始 readonly 时不读取持久草稿，首次解除 readonly 后只恢复一次', () => {
    const tabKey = resolveTabKey(DRAFT_PATHNAME);
    const draftKey = getFormDraftKey(tabKey, DRAFT_PATHNAME, draftScope);
    sessionStorage.setItem(
      draftKey,
      JSON.stringify({ customerReferenceNo: 'DRAFT-DIRTY' }),
    );

    const sections = [
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
    ];

    const { rerender } = render(
      <OrderFormTemplate
        readonly={true}
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
        draftScope={draftScope}
        initialValues={{ customerReferenceNo: 'SERVER-ORIGINAL' }}
        sections={sections}
      />,
    );

    expect(screen.getByText('SERVER-ORIGINAL')).toBeInTheDocument();
    expect(screen.queryByText('DRAFT-DIRTY')).not.toBeInTheDocument();
    expect(isTabDirty(tabKey)).toBe(false);

    // 首次解除 readonly：恢复草稿一次
    rerender(
      <OrderFormTemplate
        readonly={false}
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
        draftScope={draftScope}
        initialValues={{ customerReferenceNo: 'SERVER-ORIGINAL' }}
        sections={sections}
      />,
    );
    const input = screen.getByPlaceholderText(
      '请输入客户参考号',
    ) as HTMLInputElement;
    expect(input.value).toBe('DRAFT-DIRTY');
    expect(isTabDirty(tabKey)).toBe(true);

    // 编辑草稿值后回到 readonly：不再恢复，不覆盖内存中的编辑值
    act(() => {
      fireEvent.change(input, { target: { value: 'USER-EDITED' } });
    });
    rerender(
      <OrderFormTemplate
        readonly={true}
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
        draftScope={draftScope}
        initialValues={{ customerReferenceNo: 'SERVER-ORIGINAL' }}
        sections={sections}
      />,
    );
    expect(screen.getByText('USER-EDITED')).toBeInTheDocument();
    expect(screen.queryByText('DRAFT-DIRTY')).not.toBeInTheDocument();
  });

  it('只读挂载全程无草稿可恢复时，展示服务端 initialValues', () => {
    const tabKey = resolveTabKey(DRAFT_PATHNAME);

    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
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
    expect(isTabDirty(tabKey)).toBe(false);
  });

  it('成功提交时清除当前草稿，提交返回 false 时保留草稿', async () => {
    const tabKey = resolveTabKey(DRAFT_PATHNAME);
    const draftKey = getFormDraftKey(tabKey, DRAFT_PATHNAME, draftScope);
    sessionStorage.setItem(
      draftKey,
      JSON.stringify({ customerReferenceNo: 'TO-SUBMIT' }),
    );

    let submitResult = false;
    const onFinish = vi.fn().mockImplementation(async () => submitResult);

    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
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

  it('resetTo 清当前草稿、清旧 Form store、回填新值并清除 dirty', () => {
    const tabKey = resolveTabKey(DRAFT_PATHNAME);
    const actionsRef: React.MutableRefObject<
      | OrderFormTemplateActions<{
          customerReferenceNo?: string;
          internalNote?: string;
        }>
      | undefined
    > = { current: undefined };

    render(
      <OrderFormTemplate<{
        customerReferenceNo?: string;
        internalNote?: string;
      }>
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
        draftScope={draftScope}
        actionsRef={actionsRef}
        sections={[
          {
            key: 'basic',
            title: '业务信息',
            content: (
              <>
                <ProFormText
                  name="customerReferenceNo"
                  label="客户参考号"
                  placeholder="请输入客户参考号"
                />
                <ProFormText
                  name="internalNote"
                  label="内部备注"
                  placeholder="请输入内部备注"
                />
              </>
            ),
          },
        ]}
      />,
    );

    act(() => {
      fireEvent.change(screen.getByPlaceholderText('请输入客户参考号'), {
        target: { value: 'STALE-REF' },
      });
      fireEvent.change(screen.getByPlaceholderText('请输入内部备注'), {
        target: { value: 'STALE-NOTE' },
      });
    });
    expect(hasTabDraft(tabKey, draftScope)).toBe(true);
    expect(isTabDirty(tabKey)).toBe(true);

    act(() => {
      actionsRef.current?.resetTo({ customerReferenceNo: 'FRESH-REF' });
    });

    // 旧 Form store 字段被清除（新快照中不存在的字段不残留），指定字段回填新值
    expect(
      (screen.getByPlaceholderText('请输入客户参考号') as HTMLInputElement)
        .value,
    ).toBe('FRESH-REF');
    expect(
      (screen.getByPlaceholderText('请输入内部备注') as HTMLInputElement).value,
    ).toBe('');
    expect(hasTabDraft(tabKey, draftScope)).toBe(false);
    expect(isTabDirty(tabKey)).toBe(false);
  });

  it('点击重置表单时，清除暂存草稿并重置 dirty 状态', async () => {
    const tabKey = resolveTabKey(DRAFT_PATHNAME);
    render(
      <OrderFormTemplate
        tabKey={tabKey}
        draftPathname={DRAFT_PATHNAME}
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
