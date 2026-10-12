import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import { SectionCard } from './SectionCard';

afterEach(cleanup);
it('非受控折叠按钮关联唯一内容，默认折叠、点击展开，extra不误折叠', () => {
  const onCollapseChange = vi.fn();
  const onExtra = vi.fn();
  render(
    <>
      <SectionCard
        title="结算规则"
        collapsible
        defaultCollapsed
        onCollapseChange={onCollapseChange}
        extra={
          <button type="button" onClick={onExtra}>
            编辑结算
          </button>
        }
      >
        <input aria-label="规则字段" defaultValue="保留值" />
      </SectionCard>
      <SectionCard title="结算规则" collapsible>
        另一分节
      </SectionCard>
    </>,
  );
  const [toggle, other] = screen.getAllByRole('button', { name: '结算规则' });
  expect(toggle.tagName).toBe('BUTTON');
  expect(toggle).toHaveAttribute('type', 'button');
  expect(toggle).toHaveAttribute('aria-expanded', 'false');
  const content = document.getElementById(
    toggle.getAttribute('aria-controls') ?? '',
  );
  expect(content).toHaveAttribute('hidden');
  expect(other.getAttribute('aria-controls')).not.toBe(
    toggle.getAttribute('aria-controls'),
  );
  fireEvent.click(toggle);
  expect(toggle).toHaveAttribute('aria-expanded', 'true');
  expect(content).not.toHaveAttribute('hidden');
  expect(onCollapseChange).toHaveBeenLastCalledWith(false);
  fireEvent.click(screen.getByRole('button', { name: '编辑结算' }));
  expect(onExtra).toHaveBeenCalledOnce();
  expect(onCollapseChange).toHaveBeenCalledOnce();
  fireEvent.click(toggle);
  expect(toggle).toHaveAttribute('aria-expanded', 'false');
  expect(screen.getByLabelText('规则字段')).toHaveValue('保留值');
});
it('受控折叠只通知父级且语义随props同步；非折叠标题没有按钮或焦点', () => {
  const changed = vi.fn();
  const view = render(
    <>
      <SectionCard
        title="基本资料"
        collapsible
        collapsed={false}
        onCollapseChange={changed}
      >
        基本字段
      </SectionCard>
      <SectionCard title="只读分节">展示字段</SectionCard>
    </>,
  );
  const toggle = screen.getByRole('button', { name: '基本资料' });
  fireEvent.click(toggle);
  expect(changed).toHaveBeenCalledWith(true);
  expect(toggle).toHaveAttribute('aria-expanded', 'true');
  view.rerender(
    <>
      <SectionCard
        title="基本资料"
        collapsible
        collapsed
        onCollapseChange={changed}
      >
        基本字段
      </SectionCard>
      <SectionCard title="只读分节">展示字段</SectionCard>
    </>,
  );
  expect(toggle).toHaveAttribute('aria-expanded', 'false');
  expect(
    document.getElementById(toggle.getAttribute('aria-controls') ?? ''),
  ).toHaveAttribute('hidden');
  expect(
    screen.queryByRole('button', { name: '只读分节' }),
  ).not.toBeInTheDocument();
  const title = screen.getByText('只读分节');
  expect(title.closest('[tabindex]')).toBeNull();
});
