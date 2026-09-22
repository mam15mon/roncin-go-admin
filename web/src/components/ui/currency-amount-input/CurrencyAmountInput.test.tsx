import { act, cleanup, render, screen } from '@testing-library/react';
import { Form } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { CurrencyAmountInput } from './CurrencyAmountInput';

const mockCurrencies = [
  { label: 'CNY (人民币)', value: 'CNY' },
  { label: 'USD (美元)', value: 'USD' },
  { label: 'EUR (欧元)', value: 'EUR' },
];

describe('CurrencyAmountInput', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('正常渲染币种选择与金额输入框', () => {
    render(
      <Form>
        <CurrencyAmountInput
          currencyName="testCurrency"
          amountName="testAmount"
          currencyOptions={mockCurrencies}
          currencyPlaceholder="选择币种"
          amountPlaceholder="0.00"
        />
      </Form>,
    );

    expect(screen.getByText('选择币种')).toBeInTheDocument();
    const amountInput = screen.getByPlaceholderText('0.00') as HTMLInputElement;
    expect(amountInput).toBeInTheDocument();
  });

  it('支持 disabled 属性同步禁用币种与金额', () => {
    render(
      <Form>
        <CurrencyAmountInput
          currencyName="testCurrency"
          amountName="testAmount"
          currencyOptions={mockCurrencies}
          disabled
        />
      </Form>,
    );

    const amountInput = screen.getByPlaceholderText('0.00') as HTMLInputElement;
    expect(amountInput).toBeDisabled();
  });

  it('双空时校验通过', async () => {
    let formRef: any;
    const TestComponent = () => {
      const [form] = Form.useForm();
      formRef = form;
      return (
        <Form form={form}>
          <CurrencyAmountInput
            currencyName="testCurrency"
            amountName="testAmount"
            currencyOptions={mockCurrencies}
          />
        </Form>
      );
    };

    render(<TestComponent />);
    // 表单校验是触发内部状态更新的异步流，必须在 act 内等待收敛
    await act(async () => {
      await expect(formRef.validateFields()).resolves.toEqual({
        testCurrency: undefined,
        testAmount: undefined,
      });
    });
  });

  it('只填金额未选币种时，校验抛出错误提示选择币种', async () => {
    let formRef: any;
    const TestComponent = () => {
      const [form] = Form.useForm();
      formRef = form;
      return (
        <Form form={form} initialValues={{ testAmount: '100.5' }}>
          <CurrencyAmountInput
            currencyName="testCurrency"
            amountName="testAmount"
            currencyOptions={mockCurrencies}
            emptyCurrencyMessage="必须选择币种"
          />
        </Form>
      );
    };

    render(<TestComponent />);
    // 表单校验是触发内部状态更新的异步流，必须在 act 内等待收敛
    await act(async () => {
      await expect(formRef.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({
            name: ['testCurrency'],
            errors: ['必须选择币种'],
          }),
        ]),
      });
    });
  });

  it('默认情况下只选币种未填金额时校验通过（被忽略）', async () => {
    let formRef: any;
    const TestComponent = () => {
      const [form] = Form.useForm();
      formRef = form;
      return (
        <Form form={form} initialValues={{ testCurrency: 'USD' }}>
          <CurrencyAmountInput
            currencyName="testCurrency"
            amountName="testAmount"
            currencyOptions={mockCurrencies}
            emptyAmountMessage="必须输入金额"
          />
        </Form>
      );
    };

    render(<TestComponent />);
    await act(async () => {
      await expect(formRef.validateFields()).resolves.toEqual({
        testCurrency: 'USD',
        testAmount: undefined,
      });
    });
  });

  it('显式开启 requireAmountWhenCurrency 时只选币种未填金额抛出错误提示输入金额', async () => {
    let formRef: any;
    const TestComponent = () => {
      const [form] = Form.useForm();
      formRef = form;
      return (
        <Form form={form} initialValues={{ testCurrency: 'USD' }}>
          <CurrencyAmountInput
            currencyName="testCurrency"
            amountName="testAmount"
            currencyOptions={mockCurrencies}
            requireAmountWhenCurrency
            emptyAmountMessage="必须输入金额"
          />
        </Form>
      );
    };

    render(<TestComponent />);
    // 表单校验是触发内部状态更新的异步流，必须在 act 内等待收敛
    await act(async () => {
      await expect(formRef.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({
            name: ['testAmount'],
            errors: ['必须输入金额'],
          }),
        ]),
      });
    });
  });

  it('输入非法金额格式时校验失败', async () => {
    let formRef: any;
    const TestComponent = () => {
      const [form] = Form.useForm();
      formRef = form;
      return (
        <Form
          form={form}
          initialValues={{ testCurrency: 'USD', testAmount: '100.12345' }}
        >
          <CurrencyAmountInput
            currencyName="testCurrency"
            amountName="testAmount"
            currencyOptions={mockCurrencies}
            amountRuleMessage="最多 4 位小数"
          />
        </Form>
      );
    };

    render(<TestComponent />);
    // 表单校验是触发内部状态更新的异步流，必须在 act 内等待收敛
    await act(async () => {
      await expect(formRef.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({
            name: ['testAmount'],
            errors: ['最多 4 位小数'],
          }),
        ]),
      });
    });
  });

  it('币种与金额均合法时校验成功', async () => {
    let formRef: any;
    const TestComponent = () => {
      const [form] = Form.useForm();
      formRef = form;
      return (
        <Form
          form={form}
          initialValues={{ testCurrency: 'USD', testAmount: '8888.88' }}
        >
          <CurrencyAmountInput
            currencyName="testCurrency"
            amountName="testAmount"
            currencyOptions={mockCurrencies}
          />
        </Form>
      );
    };

    render(<TestComponent />);
    // 表单校验是触发内部状态更新的异步流，必须在 act 内等待收敛
    await act(async () => {
      await expect(formRef.validateFields()).resolves.toEqual({
        testCurrency: 'USD',
        testAmount: '8888.88',
      });
    });
  });

  it('选中币种后只展示简短代码（如 CNY 而非 CNY (人民币)）', () => {
    render(
      <Form initialValues={{ testCurrency: 'CNY' }}>
        <CurrencyAmountInput
          currencyName="testCurrency"
          amountName="testAmount"
          currencyOptions={mockCurrencies}
        />
      </Form>,
    );

    // 选中的 Select 框内应展示 CNY，而不是带有后置中文备注
    expect(screen.getByText('CNY')).toBeInTheDocument();
    expect(screen.queryByText('CNY (人民币)')).not.toBeInTheDocument();
  });

  it('默认开启并支持 showSearch 进行币种输入搜索', () => {
    const { container } = render(
      <Form>
        <CurrencyAmountInput
          currencyName="testCurrency"
          amountName="testAmount"
          currencyOptions={mockCurrencies}
        />
      </Form>,
    );

    // 默认开启 showSearch 后，Select 容器包含 ant-select-show-search 类名
    expect(
      container.querySelector('.ant-select-show-search'),
    ).toBeInTheDocument();
  });

  it('默认支持 allowClear 允许清除已选币种', () => {
    const { container } = render(
      <Form initialValues={{ testCurrency: 'USD' }}>
        <CurrencyAmountInput
          currencyName="testCurrency"
          amountName="testAmount"
          currencyOptions={mockCurrencies}
        />
      </Form>,
    );

    expect(container.querySelector('.ant-select-allow-clear')).toBeInTheDocument();
  });
});
