import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MultiTabCenterTemplate, ParameterSettingTemplate } from './ParameterSettingTemplate';
import { SettingTableTemplate } from './SettingTableTemplate';

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

describe('MultiTabCenterTemplate / ParameterSettingTemplate', () => {
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
      <MultiTabCenterTemplate
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

describe('SettingTableTemplate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正确渲染表格列、数据与新建按钮', async () => {
    const mockQuery = vi.fn().mockResolvedValue({
      data: [
        { id: '1', code: 'BOX', name: '集装箱' },
        { id: '2', code: 'CBM', name: '立方米' },
      ],
      success: true,
    });

    render(
      <SettingTableTemplate
        entityName="计费单位"
        columns={[
          { title: '代码', dataIndex: 'code' },
          { title: '名称', dataIndex: 'name' },
        ]}
        query={mockQuery}
        createItem={vi.fn()}
        updateItem={vi.fn()}
        renderFormItems={() => <div>表单项内容</div>}
      />,
    );

    expect(screen.getByText('新建计费单位')).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText('BOX')).toBeInTheDocument();
      expect(screen.getByText('CBM')).toBeInTheDocument();
    });
  });
});
