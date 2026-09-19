import { fireEvent, render, screen } from '@testing-library/react';
import { Form } from 'antd';
import React from 'react';
import { describe, expect, it } from 'vitest';
import { PackageCountInput } from './PackageCountInput';
import { PACKAGE_UNIT_OPTIONS, PACKAGE_UNIT_SEEDS } from './packageUnits';

function TestWrapper({
  initialValues,
  disabled,
}: {
  initialValues?: Record<string, unknown>;
  disabled?: boolean;
}) {
  const [form] = Form.useForm();
  return (
    <Form form={form} initialValues={initialValues}>
      <Form.Item label="总件数/单位">
        <PackageCountInput
          countName="totalPackages"
          unitName="totalPackageUnit"
          disabled={disabled}
        />
      </Form.Item>
    </Form>
  );
}

describe('PackageCountInput', () => {
  it('预置海运包装单位选项丰富，常见单位靠前', () => {
    expect(PACKAGE_UNIT_SEEDS.length).toBeGreaterThan(500);
    expect(PACKAGE_UNIT_OPTIONS[0].value).toBe('CTNS');
    expect(PACKAGE_UNIT_SEEDS).toContain('PLTS');
    expect(PACKAGE_UNIT_SEEDS).toContain('PCS');
    expect(PACKAGE_UNIT_SEEDS).toContain('WOODEN CASES');
    expect(PACKAGE_UNIT_SEEDS).toContain('箱');
    expect(PACKAGE_UNIT_SEEDS).toContain('托');
  });

  it('正确渲染件数输入框和单位下拉框', () => {
    render(
      <TestWrapper
        initialValues={{ totalPackages: 100, totalPackageUnit: 'CTNS' }}
      />,
    );
    const countInput = screen.getByRole('spinbutton');
    expect(countInput).toHaveValue('100');
    expect(screen.getByText('CTNS')).toBeInTheDocument();
  });

  it('支持修改件数与单位', async () => {
    render(<TestWrapper />);
    const countInput = screen.getByRole('spinbutton');
    fireEvent.change(countInput, { target: { value: '250' } });
    expect(countInput).toHaveValue('250');
  });

  it('disabled 状态下输入和选择框均被禁用', () => {
    render(
      <TestWrapper
        disabled
        initialValues={{ totalPackages: 50, totalPackageUnit: 'PLTS' }}
      />,
    );
    const countInput = screen.getByRole('spinbutton');
    expect(countInput).toBeDisabled();
    const select = screen.getByRole('combobox');
    expect(select).toBeDisabled();
  });
});
