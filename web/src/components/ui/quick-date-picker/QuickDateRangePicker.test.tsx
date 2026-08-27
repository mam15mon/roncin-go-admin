import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import {
  QuickDateRangePicker,
  QuickDateFilterBar,
} from './QuickDateRangePicker';

describe('QuickDateRangePicker & QuickDateFilterBar', () => {
  it('renders QuickDateRangePicker with default inputs', () => {
    const { container } = render(<QuickDateRangePicker />);
    expect(container.querySelector('.ant-picker-range')).toBeTruthy();
  });

  it('renders QuickDateFilterBar with preset buttons and handles click', () => {
    const handleChange = vi.fn();
    render(
      <QuickDateFilterBar
        onChange={handleChange}
        visiblePresets={['all', 'today', 'thisMonth']}
      />,
    );

    expect(screen.getByText('全部')).toBeTruthy();
    expect(screen.getByText('今天')).toBeTruthy();
    expect(screen.getByText('本月')).toBeTruthy();

    fireEvent.click(screen.getByText('今天'));
    expect(handleChange).toHaveBeenCalledTimes(1);
    expect(handleChange).toHaveBeenCalledWith(
      expect.any(Array),
      expect.any(Array),
      'today',
    );

    fireEvent.click(screen.getByText('全部'));
    expect(handleChange).toHaveBeenCalledWith(null, ['', ''], 'all');
  });
});
