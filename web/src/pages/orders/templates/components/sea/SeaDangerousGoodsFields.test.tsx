import { ProForm, ProFormCheckbox } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { Button, Form } from 'antd';
import React from 'react';
import { describe, expect, it } from 'vitest';
import { SeaDangerousGoodsFields } from './SeaBasicInfoSection';

const sampleCargoOptions = [
  { label: '普货', value: 'cat-general', code: 'GENERAL' },
  { label: '危险品', value: 'cat-dg', code: 'DANGEROUS' },
  { label: '冷藏品', value: 'cat-reefer', code: 'REEFER' },
];

function DgController() {
  const form = Form.useFormInstance();
  return (
    <>
      <Button
        onClick={() => {
          form.setFieldsValue({
            cargoCategoryIds: ['cat-general', 'cat-dg'],
          });
        }}
        data-testid="toggle-dg-on"
      >
        Select DG
      </Button>
      <Button
        onClick={() => {
          form.setFieldsValue({
            cargoCategoryIds: ['cat-general'],
          });
        }}
        data-testid="toggle-dg-off"
      >
        Unselect DG
      </Button>
      <Button
        onClick={() => {
          form.setFieldsValue({
            unNumber: '1993',
            hazardClass: '3',
            declarationCutoffAt: '2026-09-15 12:00',
          });
        }}
        data-testid="fill-dg-fields"
      >
        Fill DG Fields
      </Button>
      <div data-testid="un-val">
        {String(form.getFieldValue('unNumber') ?? '')}
      </div>
      <div data-testid="class-val">
        {String(form.getFieldValue('hazardClass') ?? '')}
      </div>
      <div data-testid="cutoff-val">
        {String(form.getFieldValue('declarationCutoffAt') ?? '')}
      </div>
    </>
  );
}

function TestHarness({
  initialValues,
  options = sampleCargoOptions,
}: {
  initialValues?: Record<string, unknown>;
  options?: typeof sampleCargoOptions;
}) {
  return (
    <ProForm initialValues={initialValues} submitter={false}>
      <ProFormCheckbox.Group
        name="cargoCategoryIds"
        label="货物品类"
        options={options}
      />
      <SeaDangerousGoodsFields options={options} />
      <DgController />
    </ProForm>
  );
}

describe('SeaDangerousGoodsFields 品类联动', () => {
  it('普货场景（默认未勾选危险品）默认隐藏 UN NO.、CLASS NO.、截申报时间', () => {
    render(
      <TestHarness initialValues={{ cargoCategoryIds: ['cat-general'] }} />,
    );

    expect(screen.queryByText('UN NO.')).not.toBeInTheDocument();
    expect(screen.queryByText('CLASS NO.')).not.toBeInTheDocument();
    expect(screen.queryByText('截申报时间')).not.toBeInTheDocument();
  });

  it('初始未选中任何品类时默认隐藏危险品字段', () => {
    render(<TestHarness initialValues={{ cargoCategoryIds: [] }} />);

    expect(screen.queryByText('UN NO.')).not.toBeInTheDocument();
    expect(screen.queryByText('CLASS NO.')).not.toBeInTheDocument();
    expect(screen.queryByText('截申报时间')).not.toBeInTheDocument();
  });

  it('勾选危险品后展开 UN NO.、CLASS NO.、截申报时间', () => {
    render(<TestHarness initialValues={{ cargoCategoryIds: ['cat-dg'] }} />);

    expect(screen.getByText('UN NO.')).toBeInTheDocument();
    expect(screen.getByText('CLASS NO.')).toBeInTheDocument();
    expect(screen.getByText('截申报时间')).toBeInTheDocument();
  });

  it('通过 code: DG 匹配危险品', () => {
    const dgOptions = [
      { label: '一般化工品', value: 'c-1', code: 'GENERAL' },
      { label: '易燃品', value: 'c-2', code: 'DG' },
    ];
    render(
      <TestHarness
        options={dgOptions}
        initialValues={{ cargoCategoryIds: ['c-2'] }}
      />,
    );

    expect(screen.getByText('UN NO.')).toBeInTheDocument();
    expect(screen.getByText('CLASS NO.')).toBeInTheDocument();
    expect(screen.getByText('截申报时间')).toBeInTheDocument();
  });

  it('动态勾选危险品展开字段，取消勾选后隐藏并重置相关字段值', async () => {
    render(
      <TestHarness initialValues={{ cargoCategoryIds: ['cat-general'] }} />,
    );

    // 1. 初始为普货：隐藏
    expect(screen.queryByText('UN NO.')).not.toBeInTheDocument();

    // 2. 勾选危险品：展开
    const toggleOnBtn = screen.getByTestId('toggle-dg-on');
    await act(async () => {
      fireEvent.click(toggleOnBtn);
    });

    await waitFor(() => {
      expect(screen.getByText('UN NO.')).toBeInTheDocument();
      expect(screen.getByText('CLASS NO.')).toBeInTheDocument();
      expect(screen.getByText('截申报时间')).toBeInTheDocument();
    });

    // 3. 填入危险品字段
    const fillButton = screen.getByTestId('fill-dg-fields');
    await act(async () => {
      fireEvent.click(fillButton);
    });

    await waitFor(() => {
      const unInput = screen.getByPlaceholderText(
        '4位数字',
      ) as HTMLInputElement;
      expect(unInput.value).toBe('1993');
    });

    // 4. 取消勾选危险品：重新隐藏并清空
    const toggleOffBtn = screen.getByTestId('toggle-dg-off');
    await act(async () => {
      fireEvent.click(toggleOffBtn);
    });

    await waitFor(() => {
      expect(screen.queryByText('UN NO.')).not.toBeInTheDocument();
      expect(screen.queryByText('CLASS NO.')).not.toBeInTheDocument();
      expect(screen.queryByText('截申报时间')).not.toBeInTheDocument();
      expect(screen.getByTestId('un-val').textContent).toBe('');
      expect(screen.getByTestId('class-val').textContent).toBe('');
      expect(screen.getByTestId('cutoff-val').textContent).toBe('');
    });
  });
});
