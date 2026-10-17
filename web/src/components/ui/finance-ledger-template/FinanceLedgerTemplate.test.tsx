import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { FinanceLedgerTemplate } from './FinanceLedgerTemplate';

describe('FinanceLedgerTemplate', () => {
  it('正确渲染顶层骨架标题、副标题与 topBar 工具栏插槽', async () => {
    const { getByText } = render(
      <App>
        <FinanceLedgerTemplate
          pageTitle="测试账单管理"
          pageSubTitle="测试副标题描述"
          topBar={<div>所属公司筛选器</div>}
          exportColumns={[]}
          columns={[]}
          request={async () => ({ data: [], total: 0 })}
        />
      </App>,
    );

    expect(getByText('测试账单管理')).toBeTruthy();
    expect(getByText('测试副标题描述')).toBeTruthy();
    expect(getByText('所属公司筛选器')).toBeTruthy();
    await act(async () => {});
  });

  it('费用模板消费公共调宽，保留数据并不重新发起台账请求', async () => {
    const request = vi.fn(async () => ({
      data: [{ id: '1', feeName: '报关费' }],
      total: 1,
    }));
    const { container } = render(
      <App>
        <FinanceLedgerTemplate
          pageTitle="费用台账"
          exportColumns={[]}
          columns={[
            {
              key: 'feeName',
              title: '费用名称',
              dataIndex: 'feeName',
              width: 120,
            },
          ]}
          request={request}
          showSummaryBoard={false}
          rowSelection={false}
        />
      </App>,
    );
    await screen.findByText('报关费');
    const handle = container.querySelector<HTMLElement>(
      '[data-column-resize-handle]',
    );
    if (!handle) throw new Error('没有调宽手柄');
    fireEvent.mouseDown(handle, { clientX: 100 });
    fireEvent.mouseMove(window, { clientX: 180 });
    fireEvent.mouseUp(window);
    await waitFor(() =>
      expect(
        container.querySelector<HTMLTableColElement>('colgroup col')?.style
          .width,
      ).toBe('200px'),
    );
    fireEvent.doubleClick(handle);
    expect(
      container.querySelector<HTMLTableColElement>('colgroup col')?.style.width,
    ).toBe('240px');
    expect(request).toHaveBeenCalledTimes(1);
    expect(screen.getByText('报关费')).toBeInTheDocument();
    await act(async () => {});
  });
});
