import { ProForm } from '@ant-design/pro-components';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import type { TemplateProps } from '../../types';
import { getSeaBaseInfoFields } from './SeaBasicInfoSection';

describe('海运船代远程候选', () => {
  it('拼音检索命中的中文结果保持可选且提交公共主数据 ID', async () => {
    const searchShippingAgents = vi
      .fn()
      .mockImplementation(async (keyword) =>
        keyword === 'shcd'
          ? [{ label: '上海船代', value: 'shipping-agent-1' }]
          : [],
      );
    const onFinish = vi.fn();
    const field = getSeaBaseInfoFields({
      searchShippingAgents,
    } as unknown as TemplateProps).shippingAgent;
    render(
      <ProForm submitter={false} onFinish={onFinish}>
        {field}
        <button type="submit">保存船代</button>
      </ProForm>,
    );
    const input = screen.getByRole('combobox');
    fireEvent.mouseDown(input);
    fireEvent.change(input, { target: { value: 'shcd' } });
    const option = await screen.findByText('上海船代');
    fireEvent.click(option);
    fireEvent.click(screen.getByRole('button', { name: '保存船代' }));
    await waitFor(() =>
      expect(onFinish).toHaveBeenCalledWith({
        shippingAgentId: 'shipping-agent-1',
      }),
    );
    expect(searchShippingAgents).toHaveBeenCalledWith('shcd');
  });
});
