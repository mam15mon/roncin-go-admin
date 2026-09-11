import { fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { FormAnchorNav } from './FormAnchorNav';

describe('FormAnchorNav', () => {
  const items = [
    { key: 'basic', title: '基础信息' },
    { key: 'transport', title: '运输订舱' },
    { key: 'cargo', title: '货物清单' },
  ];

  it('正常渲染所有楼层分节项与标题', () => {
    render(<FormAnchorNav items={items} />);

    expect(screen.getByText('表单导航')).toBeInTheDocument();
    expect(screen.getByText('基础信息')).toBeInTheDocument();
    expect(screen.getByText('运输订舱')).toBeInTheDocument();
    expect(screen.getByText('货物清单')).toBeInTheDocument();
  });

  it('分节存在错误时显示对应的红色错误徽标', () => {
    render(
      <FormAnchorNav
        items={items}
        sectionErrors={{
          transport: 2,
        }}
      />,
    );

    // 楼层头部总错误数
    const badges = screen.getAllByText('2');
    expect(badges.length).toBeGreaterThanOrEqual(1);
  });

  it('点击带有错误的分节时优先触发 onErrorClick 回调', () => {
    const onErrorClick = vi.fn();
    const onSelect = vi.fn();

    render(
      <FormAnchorNav
        items={items}
        sectionErrors={{
          transport: 3,
        }}
        onErrorClick={onErrorClick}
        onSelect={onSelect}
      />,
    );

    fireEvent.click(screen.getByText('运输订舱'));

    expect(onErrorClick).toHaveBeenCalledWith('transport');
    expect(onSelect).not.toHaveBeenCalled();
  });
});
