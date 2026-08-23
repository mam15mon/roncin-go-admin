import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { Form } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { TooltipInput } from './TooltipInput';

describe('TooltipInput', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正常渲染输入框并支持输入', () => {
    render(<TooltipInput placeholder="请输入内部编号" />);
    const input = screen.getByPlaceholderText('请输入内部编号') as HTMLInputElement;
    expect(input).toBeInTheDocument();

    fireEvent.change(input, { target: { value: 'RC20260823001' } });
    expect(input.value).toBe('RC20260823001');
  });

  it('鼠标悬停时当输入框有值触发 Tooltip 展示', () => {
    render(<TooltipInput defaultValue="LONG_CODE_123456789" />);
    const input = screen.getByDisplayValue('LONG_CODE_123456789');

    fireEvent.mouseEnter(input);
    expect(input).toBeInTheDocument();
  });

  it('支持在 Form.Item 中作为受控表单控件运行', () => {
    const TestForm = () => (
      <Form initialValues={{ code: 'INIT_CODE_888' }}>
        <Form.Item name="code" label="编号">
          <TooltipInput />
        </Form.Item>
      </Form>
    );

    render(<TestForm />);
    const input = screen.getByDisplayValue('INIT_CODE_888');
    expect(input).toBeInTheDocument();
  });

  it('支持自定义 tooltipTitle', () => {
    render(
      <TooltipInput
        value="12345"
        tooltipTitle="自定义提示文本"
      />,
    );

    const input = screen.getByDisplayValue('12345');
    fireEvent.mouseEnter(input);
    expect(input).toBeInTheDocument();
  });
});
