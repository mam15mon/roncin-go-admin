import { ProForm, ProFormCheckbox } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { Button, Form, type FormInstance } from 'antd';
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
  exposeForm,
}: {
  initialValues?: Record<string, unknown>;
  options?: typeof sampleCargoOptions;
  exposeForm?: (form: FormInstance) => void;
}) {
  const [form] = Form.useForm();
  exposeForm?.(form);
  return (
    <ProForm form={form} initialValues={initialValues} submitter={false}>
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

    expect(screen.getByText('UN NO.')).not.toBeVisible();
    expect(screen.getByText('CLASS NO.')).not.toBeVisible();
    expect(screen.getByText('截申报时间')).not.toBeVisible();
  });

  it('初始未选中任何品类时默认隐藏危险品字段', () => {
    render(<TestHarness initialValues={{ cargoCategoryIds: [] }} />);

    expect(screen.getByText('UN NO.')).not.toBeVisible();
    expect(screen.getByText('CLASS NO.')).not.toBeVisible();
    expect(screen.getByText('截申报时间')).not.toBeVisible();
  });

  it('勾选危险品后展开 UN NO.、CLASS NO.、截申报时间', () => {
    render(<TestHarness initialValues={{ cargoCategoryIds: ['cat-dg'] }} />);

    expect(screen.getByText('UN NO.')).toBeVisible();
    expect(screen.getByText('CLASS NO.')).toBeVisible();
    expect(screen.getByText('截申报时间')).toBeVisible();
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

    expect(screen.getByText('UN NO.')).toBeVisible();
    expect(screen.getByText('CLASS NO.')).toBeVisible();
    expect(screen.getByText('截申报时间')).toBeVisible();
  });

  it('动态勾选危险品展开字段，取消勾选后隐藏并重置相关字段值', async () => {
    render(
      <TestHarness initialValues={{ cargoCategoryIds: ['cat-general'] }} />,
    );

    // 1. 初始为普货：隐藏
    expect(screen.getByText('UN NO.')).not.toBeVisible();

    // 2. 勾选危险品：展开
    const toggleOnBtn = screen.getByTestId('toggle-dg-on');
    await act(async () => {
      fireEvent.click(toggleOnBtn);
    });

    await waitFor(() => {
      expect(screen.getByText('UN NO.')).toBeVisible();
      expect(screen.getByText('CLASS NO.')).toBeVisible();
      expect(screen.getByText('截申报时间')).toBeVisible();
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
      expect(screen.getByText('UN NO.')).not.toBeVisible();
      expect(screen.getByText('CLASS NO.')).not.toBeVisible();
      expect(screen.getByText('截申报时间')).not.toBeVisible();
      expect(screen.getByTestId('un-val').textContent).toBe('');
      expect(screen.getByTestId('class-val').textContent).toBe('');
      expect(screen.getByTestId('cutoff-val').textContent).toBe('');
    });
  });

  it('收起后停用隐藏字段交互及格式校验，不阻止保存普通货物', async () => {
    let form!: FormInstance;
    render(
      <TestHarness
        initialValues={{ cargoCategoryIds: ['cat-dg'], unNumber: '12' }}
        exposeForm={(instance) => {
          form = instance;
        }}
      />,
    );
    await act(async () => {
      await expect(form.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['unNumber'] }),
        ]),
      });
    });

    await act(async () => {
      fireEvent.click(screen.getByTestId('toggle-dg-off'));
    });

    const motion = screen.getByTestId('sea-dangerous-goods-motion');
    expect(motion).toHaveAttribute('aria-hidden', 'true');
    expect(motion).toHaveAttribute('inert');
    expect(screen.getByPlaceholderText('4位数字')).toBeDisabled();
    expect(screen.getByPlaceholderText('类别')).toBeDisabled();
    await act(async () => {
      await expect(form.validateFields()).resolves.toMatchObject({
        unNumber: undefined,
        hazardClass: undefined,
        declarationCutoffAt: undefined,
      });
    });
  });

  it('快速切换危险品后保持最后的展开状态，重新录入不被迟到的收起清空', async () => {
    render(<TestHarness initialValues={{ cargoCategoryIds: ['cat-dg'] }} />);
    await act(async () => {
      fireEvent.click(screen.getByTestId('fill-dg-fields'));
    });
    await act(async () => {
      fireEvent.click(screen.getByTestId('toggle-dg-off'));
    });
    await act(async () => {
      fireEvent.click(screen.getByTestId('toggle-dg-on'));
    });

    const motion = screen.getByTestId('sea-dangerous-goods-motion');
    expect(motion).toHaveAttribute('data-open', 'true');
    expect(motion).toHaveAttribute('aria-hidden', 'false');
    expect(motion).not.toHaveAttribute('inert');
    const unInput = screen.getByPlaceholderText('4位数字');
    expect(unInput).toBeEnabled();
    expect(unInput).toHaveValue('');
    fireEvent.change(unInput, { target: { value: '1203' } });
    fireEvent.transitionEnd(motion, { propertyName: 'grid-template-rows' });
    expect(unInput).toHaveValue('1203');
  });
});
