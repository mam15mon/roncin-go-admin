import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { OrgTreeNode } from '../../organization-tree';
import OrgChartCanvas from './OrgChartCanvas';

describe('OrgChartCanvas', () => {
  const mockTreeData: OrgTreeNode[] = [
    {
      key: 'org-root',
      title: '隆胜货运总部',
      code: 'RC-HQ',
      kind: 1, // 总部
      enabled: true,
      raw: {
        id: 'org-root',
        name: '隆胜货运总部',
        code: 'RC-HQ',
        kind: 1,
        enabled: true,
      },
      children: [
        {
          key: 'org-comp-1',
          title: '上海分公司',
          code: 'RC-SH',
          kind: 2, // 公司
          enabled: true,
          raw: {
            id: 'org-comp-1',
            name: '上海分公司',
            code: 'RC-SH',
            kind: 2,
            enabled: true,
            parentId: 'org-root',
          },
          children: [
            {
              key: 'org-dept-1',
              title: '操作部',
              code: 'RC-SH-OPS',
              kind: 3, // 部门
              enabled: false,
              raw: {
                id: 'org-dept-1',
                name: '操作部',
                code: 'RC-SH-OPS',
                kind: 3,
                enabled: false,
                parentId: 'org-comp-1',
              },
            },
          ],
        },
        {
          key: 'org-comp-2',
          title: '深圳分公司',
          code: 'RC-SZ',
          kind: 2,
          enabled: true,
          raw: {
            id: 'org-comp-2',
            name: '深圳分公司',
            code: 'RC-SZ',
            kind: 2,
            enabled: true,
            parentId: 'org-root',
          },
        },
      ],
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正确渲染组织节点卡片、名称、编码、类型 Tag 与下级数量', () => {
    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={mockTreeData}
          chartDirection="vertical"
          selectedId="org-root"
          onSelectNode={vi.fn()}
          onOpenDrawer={vi.fn()}
        />
      </App>,
    );

    // 根节点
    expect(screen.getAllByText('隆胜货运总部').length).toBeGreaterThanOrEqual(
      1,
    );
    expect(screen.getAllByText('RC-HQ').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('总部').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('2 个下级')).toBeInTheDocument();

    // 子节点
    expect(screen.getAllByText('上海分公司').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('RC-SH').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('1 个下级')).toBeInTheDocument();

    // 孙节点（操作部 - 停用）
    expect(screen.getByText('操作部')).toBeInTheDocument();
    expect(screen.getByText('RC-SH-OPS')).toBeInTheDocument();
    expect(screen.getAllByText('停用').length).toBeGreaterThanOrEqual(1);
  });

  it('点击组织节点触发 onSelectNode 与 onOpenDrawer', () => {
    const handleSelect = vi.fn();
    const handleOpenDrawer = vi.fn();

    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={mockTreeData}
          chartDirection="vertical"
          selectedId="org-root"
          onSelectNode={handleSelect}
          onOpenDrawer={handleOpenDrawer}
        />
      </App>,
    );

    const shCard = document.querySelector('[data-node-id="org-comp-1"]');
    expect(shCard).not.toBeNull();
    if (shCard) {
      fireEvent.click(shCard);
    }

    expect(handleSelect).toHaveBeenCalledWith('org-comp-1');
    expect(handleOpenDrawer).toHaveBeenCalled();
  });

  it('支持折叠与展开子分支', () => {
    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={mockTreeData}
          chartDirection="vertical"
          selectedId="org-root"
          onSelectNode={vi.fn()}
          onOpenDrawer={vi.fn()}
        />
      </App>,
    );

    expect(screen.getByText('操作部')).toBeInTheDocument();

    // 上海分公司有 1 个下级，找到其折叠按钮
    const collapseBtn = screen.getAllByTitle('折叠下级')[1];
    fireEvent.click(collapseBtn);

    // 操作部应该被折叠隐藏，出现展开按钮 "+1"
    expect(screen.queryByText('操作部')).not.toBeInTheDocument();
    const expandBtn = screen.getByTitle('展开 1 个下级');
    expect(expandBtn).toBeInTheDocument();

    // 再次点击展开
    fireEvent.click(expandBtn);
    expect(screen.getByText('操作部')).toBeInTheDocument();
  });

  it('支持全部折叠与全部展开工具栏操作', () => {
    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={mockTreeData}
          chartDirection="vertical"
          selectedId="org-root"
          onSelectNode={vi.fn()}
          onOpenDrawer={vi.fn()}
        />
      </App>,
    );

    expect(screen.getAllByText('上海分公司').length).toBeGreaterThanOrEqual(1);

    // 点击“全部折叠”
    const collapseAllBtn = screen.getByRole('button', { name: /全部折叠/ });
    fireEvent.click(collapseAllBtn);

    // 画布中的子分支卡片被隐藏（仅在收起的下级数或详情中）
    const cards = document.querySelectorAll('[data-node-id]');
    expect(cards.length).toBe(1); // 只有根节点卡片

    // 点击“全部展开”
    const expandAllBtn = screen.getByRole('button', { name: /全部展开/ });
    fireEvent.click(expandAllBtn);

    // 重新展示全部节点
    const expandedCards = document.querySelectorAll('[data-node-id]');
    expect(expandedCards.length).toBe(4);
  });

  it('水平左右布局正常渲染且不报错', () => {
    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={mockTreeData}
          chartDirection="horizontal"
          selectedId="org-root"
          onSelectNode={vi.fn()}
          onOpenDrawer={vi.fn()}
        />
      </App>,
    );

    expect(screen.getAllByText('隆胜货运总部').length).toBeGreaterThanOrEqual(
      1,
    );
    expect(screen.getAllByText('深圳分公司').length).toBeGreaterThanOrEqual(1);
  });

  it('无数据时展示空状态', () => {
    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={[]}
          chartDirection="vertical"
          selectedId=""
          onSelectNode={vi.fn()}
          onOpenDrawer={vi.fn()}
        />
      </App>,
    );

    expect(screen.getByText('暂无组织架构数据')).toBeInTheDocument();
  });

  it('画布内嵌浮动面板正常渲染选中组织详情，并支持收起与重新展开', () => {
    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={mockTreeData}
          chartDirection="vertical"
          selectedId="org-root"
          onSelectNode={vi.fn()}
        />
      </App>,
    );

    // 浮动面板应处于展示状态，包含选中组织信息
    expect(screen.getByTestId('org-inspector-panel')).toBeInTheDocument();
    expect(screen.getByText('本位币种')).toBeInTheDocument();
    expect(screen.getByText('直属下级组织 (2)')).toBeInTheDocument();

    // 点击收起面板
    const closeBtn = screen.getByRole('button', { name: /收起面板/ });
    fireEvent.click(closeBtn);

    // 面板收起，出现“组织详情”唤起按钮
    expect(screen.queryByTestId('org-inspector-panel')).not.toBeInTheDocument();
    const expandToggleBtn = screen.getByRole('button', { name: /组织详情/ });
    expect(expandToggleBtn).toBeInTheDocument();

    // 点击重新展开
    fireEvent.click(expandToggleBtn);
    expect(screen.getByTestId('org-inspector-panel')).toBeInTheDocument();
  });

  it('支持按 Esc 快捷键收起浮动检查器面板', () => {
    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={mockTreeData}
          chartDirection="vertical"
          selectedId="org-root"
          onSelectNode={vi.fn()}
        />
      </App>,
    );

    expect(screen.getByTestId('org-inspector-panel')).toBeInTheDocument();

    // 触发 Esc 按键
    fireEvent.keyDown(window, { key: 'Escape' });

    // 面板应收起
    expect(screen.queryByTestId('org-inspector-panel')).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /组织详情/ }),
    ).toBeInTheDocument();
  });

  it('浮动面板提供居中对焦按钮，并在点击下级组织链接时触发对焦跳转', () => {
    const handleSelect = vi.fn();
    render(
      <App>
        <OrgChartCanvas
          loading={false}
          treeData={mockTreeData}
          chartDirection="vertical"
          selectedId="org-root"
          onSelectNode={handleSelect}
        />
      </App>,
    );

    expect(screen.getByTestId('org-inspector-panel')).toBeInTheDocument();

    // 存在居中对焦按钮
    const locateBtn = screen.getByRole('button', {
      name: /在画布中居中对焦/,
    });
    expect(locateBtn).toBeInTheDocument();
    fireEvent.click(locateBtn);

    // 点击直属下级组织中的链接跳转
    const childLink = screen.getByRole('button', { name: '上海分公司' });
    expect(childLink).toBeInTheDocument();
    fireEvent.click(childLink);

    expect(handleSelect).toHaveBeenCalledWith('org-comp-1');
  });
});
