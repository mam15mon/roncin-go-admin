import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { PageHeaderShell } from './PageHeaderShell';

const mockPush = vi.fn();

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    Link: ({ to, children, onClick, ...rest }: any) => (
      <a
        href={to}
        onClick={(e) => {
          e.preventDefault();
          onClick?.(e);
          mockPush(to);
        }}
        {...rest}
      >
        {children}
      </a>
    ),
  };
});

describe('PageHeaderShell', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('支持在面包屑中使用 href 渲染真实链接', () => {
    render(
      <PageHeaderShell
        title="当前功能"
        breadcrumbs={[
          { label: '订单管理', href: '/orders' },
          { label: '海运出口', href: '/orders/sea-export' },
        ]}
      />,
    );

    const link1 = screen.getByText('订单管理');
    expect(link1.tagName).toBe('A');
    expect(link1).toHaveAttribute('href', '/orders');

    const link2 = screen.getByText('海运出口');
    expect(link2.tagName).toBe('A');
    expect(link2).toHaveAttribute('href', '/orders/sea-export');

    fireEvent.click(link1);
    expect(mockPush).toHaveBeenCalledWith('/orders');
  });

  it('兼容仅使用 onClick 的纯按钮面包屑', () => {
    const handleCustomClick = vi.fn();
    render(
      <PageHeaderShell
        title="客户详情"
        breadcrumbs={[{ label: '客户管理', onClick: handleCustomClick }]}
      />,
    );

    const btn = screen.getByRole('button', { name: '客户管理' });
    fireEvent.click(btn);
    expect(handleCustomClick).toHaveBeenCalledTimes(1);
  });

  it('无 href/onClick 时渲染不可点击的文本面包屑', () => {
    render(
      <PageHeaderShell
        title="只读层级"
        breadcrumbs={[{ label: '静态分类' }]}
      />,
    );

    const staticItem = screen.getByText('静态分类');
    expect(staticItem.tagName).not.toBe('A');
    expect(staticItem.tagName).not.toBe('BUTTON');
  });

  it('正确渲染主要返回按钮与点击回调', () => {
    const onBack = vi.fn();
    render(
      <PageHeaderShell title="订单详情" onBack={onBack} backText="返回列表" />,
    );

    const backBtn = screen.getByRole('button', { name: /返回列表/ });
    fireEvent.click(backBtn);
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it('单行结构：标题作为面包屑末级展示，每级后接分隔符', () => {
    render(
      <PageHeaderShell
        title="新建订单"
        subTitle="填写业务委托与配舱信息"
        breadcrumbs={[{ label: '海运出口', href: '/orders/sea-export' }]}
      />,
    );

    const shell = document.querySelector('.roncin-page-header-shell');
    expect(shell).not.toBeNull();
    // 整个页头只有一行主体行
    expect(shell?.querySelectorAll('.roncin-page-header-main-row').length).toBe(
      1,
    );
    // 每个上级面包屑后都接分隔符，当前标题紧随其后作为末级
    expect(
      shell?.querySelectorAll('.roncin-page-header-crumb-sep').length,
    ).toBe(1);
    const headingTitle = shell?.querySelector(
      '.roncin-page-header-heading-title',
    );
    expect(headingTitle?.textContent).toBe('新建订单');
    // 副标题与标题同行渲染在主体行内
    expect(
      shell?.querySelector('.roncin-page-header-main-row')?.textContent,
    ).toContain('填写业务委托与配舱信息');
  });

  it('无面包屑时退化为返回按钮加标题的单行结构', () => {
    render(<PageHeaderShell title="费用台账详情" />);

    const shell = document.querySelector('.roncin-page-header-shell');
    expect(shell?.querySelector('.roncin-page-header-breadcrumbs')).toBeNull();
    expect(
      shell?.querySelectorAll('.roncin-page-header-crumb-sep').length,
    ).toBe(0);
    expect(
      shell?.querySelector('.roncin-page-header-heading-title')?.textContent,
    ).toBe('费用台账详情');
  });

  it('提供 actions 时在主体行下方紧贴渲染操作按钮栏', () => {
    render(
      <PageHeaderShell
        title="新建订单"
        breadcrumbs={[{ label: '海运出口', href: '/orders/sea-export' }]}
        actions={<button type="button">创建订单</button>}
      />,
    );

    const shell = document.querySelector('.roncin-page-header-shell');
    const actionsRow = shell?.querySelector('.roncin-page-header-actions-row');
    expect(actionsRow).not.toBeNull();
    expect(actionsRow?.textContent).toContain('创建订单');
  });
});
