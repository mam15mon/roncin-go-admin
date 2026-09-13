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
    expect(screen.getByText('隆胜货运总部')).toBeInTheDocument();
    expect(screen.getByText('RC-HQ')).toBeInTheDocument();
    expect(screen.getByText('总部')).toBeInTheDocument();
    expect(screen.getByText('2 个下级')).toBeInTheDocument();

    // 子节点
    expect(screen.getByText('上海分公司')).toBeInTheDocument();
    expect(screen.getByText('RC-SH')).toBeInTheDocument();
    expect(screen.getByText('1 个下级')).toBeInTheDocument();

    // 孙节点（操作部 - 停用）
    expect(screen.getByText('操作部')).toBeInTheDocument();
    expect(screen.getByText('RC-SH-OPS')).toBeInTheDocument();
    expect(screen.getByText('停用')).toBeInTheDocument();
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

    const shCard = screen.getByText('上海分公司');
    fireEvent.click(shCard);

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

    expect(screen.getByText('上海分公司')).toBeInTheDocument();

    // 点击“全部折叠”
    const collapseAllBtn = screen.getByRole('button', { name: /全部折叠/ });
    fireEvent.click(collapseAllBtn);

    // 上海分公司与深圳分公司被隐藏
    expect(screen.queryByText('上海分公司')).not.toBeInTheDocument();
    expect(screen.queryByText('深圳分公司')).not.toBeInTheDocument();

    // 点击“全部展开”
    const expandAllBtn = screen.getByRole('button', { name: /全部展开/ });
    fireEvent.click(expandAllBtn);

    // 重新展示
    expect(screen.getByText('上海分公司')).toBeInTheDocument();
    expect(screen.getByText('深圳分公司')).toBeInTheDocument();
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

    expect(screen.getByText('隆胜货运总部')).toBeInTheDocument();
    expect(screen.getByText('深圳分公司')).toBeInTheDocument();
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
});
