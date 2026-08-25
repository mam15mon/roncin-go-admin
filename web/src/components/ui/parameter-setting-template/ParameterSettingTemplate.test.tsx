import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ParameterSettingTemplate } from './ParameterSettingTemplate';

// Mock umi hooks
vi.mock('@umijs/max', () => ({
  history: {
    replace: vi.fn(),
  },
  useLocation: () => ({
    pathname: '/settings',
    search: '',
  }),
}));

describe('ParameterSettingTemplate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正确渲染标题、副标题与各可见 Tab', () => {
    const items = [
      {
        key: 'rules',
        label: '单据编号规则',
        children: <div>编号规则面板内容</div>,
      },
      {
        key: 'fees',
        label: '费用设置',
        children: <div>费用设置面板内容</div>,
      },
      {
        key: 'hidden-tab',
        label: '隐藏标签',
        visible: false,
        children: <div>隐藏内容</div>,
      },
    ];

    render(
      <ParameterSettingTemplate
        title="自定义参数设置"
        subTitle="系统参数测试副标题"
        items={items}
        defaultActiveKey="rules"
      />,
    );

    expect(screen.getByText('自定义参数设置')).toBeInTheDocument();
    expect(screen.getByText('系统参数测试副标题')).toBeInTheDocument();
    expect(screen.getByText('单据编号规则')).toBeInTheDocument();
    expect(screen.getByText('费用设置')).toBeInTheDocument();
    expect(screen.queryByText('隐藏标签')).not.toBeInTheDocument();
    expect(screen.getByText('编号规则面板内容')).toBeInTheDocument();
  });

  it('支持点击切换 Tab 并触发 onChange 回调', () => {
    const onChange = vi.fn();
    const items = [
      {
        key: 'rules',
        label: '单据编号规则',
        children: <div>编号规则面板内容</div>,
      },
      {
        key: 'fees',
        label: '费用设置',
        children: <div>费用设置面板内容</div>,
      },
    ];

    render(
      <ParameterSettingTemplate
        items={items}
        defaultActiveKey="rules"
        onChange={onChange}
      />,
    );

    fireEvent.click(screen.getByText('费用设置'));
    expect(onChange).toHaveBeenCalledWith('fees');
    expect(screen.getByText('费用设置面板内容')).toBeInTheDocument();
  });
});
