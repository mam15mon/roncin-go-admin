import { Form, Input, Select, Space, Tooltip } from 'antd';
import React, { useEffect, useState } from 'react';
import { defaultSelectFilterOption } from '../searchable-select/utils';
import type { CurrencyAmountInputProps } from './types';

interface TooltipNumericInputProps {
  value?: string;
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  placeholder?: string;
  maxLength?: number;
  disabled?: boolean;
  style?: React.CSSProperties;
  showTooltip?: boolean;
}

function TooltipNumericInput({
  value = '',
  onChange,
  placeholder = '0.00',
  maxLength = 23,
  disabled = false,
  style,
  showTooltip = true,
}: TooltipNumericInputProps) {
  const [internalVal, setInternalVal] = useState(value);

  useEffect(() => {
    setInternalVal(value ?? '');
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInternalVal(e.target.value);
    onChange?.(e);
  };

  const inputNode = (
    <Input
      value={internalVal}
      onChange={handleChange}
      placeholder={placeholder}
      maxLength={maxLength}
      disabled={disabled}
      style={{
        textAlign: 'right',
        fontVariantNumeric: 'tabular-nums',
        ...style,
      }}
    />
  );

  if (!showTooltip || !internalVal) {
    return inputNode;
  }

  return (
    <Tooltip title={internalVal} placement="topLeft">
      {inputNode}
    </Tooltip>
  );
}

const DEFAULT_AMOUNT_PATTERN = /^(0|[1-9]\d{0,17})(\.\d{1,4})?$/;

/**
 * CurrencyAmountInput
 *
 * 复合连体货币与金额输入组件：
 * - 遵循大厂金融与物流标准（Stripe / Flexport），左侧紧凑选择币种，右侧靠右输入金额；
 * - 共享容器边框（Space.Compact），保证高度 100% 对齐，杜绝错位与裁切；
 * - 原生支持双向联动校验（填金额必选币种，选币种必填金额，均空则校验通过）；
 * - 保持表单数据契约独立，无需改造后端 DTO。
 */
function formatSelectedCurrency(
  props: { label?: React.ReactNode; value?: string | number },
  customRender?: (opt: {
    label: React.ReactNode;
    value: string | number;
  }) => React.ReactNode,
): React.ReactNode {
  if (customRender && props.value !== undefined) {
    return customRender({
      label: props.label ?? '',
      value: props.value,
    });
  }

  // 1. 如果 value 已经是标准的 2-6 位纯字母货币代码（如 'CNY', 'USD', 'EUR'），直接回显代码
  if (typeof props.value === 'string') {
    const trimmedVal = props.value.trim();
    if (
      trimmedVal.length > 0 &&
      trimmedVal.length <= 6 &&
      /^[A-Za-z]+$/.test(trimmedVal)
    ) {
      return trimmedVal.toUpperCase();
    }
  }

  // 2. 如果 label 是字符串（如 'CNY - 人民币' 或 'USD(美元)'），提取前导纯字母货币代码
  if (typeof props.label === 'string') {
    const match = props.label.trim().match(/^([A-Za-z]{2,6})/);
    if (match) {
      return match[1].toUpperCase();
    }
  }

  return props.label ?? props.value;
}

export const CurrencyAmountInput: React.FC<CurrencyAmountInputProps> = ({
  currencyName,
  amountName,
  currencyOptions,
  renderSelectedCurrency,
  showSearch = true,
  filterOption = defaultSelectFilterOption,
  currencyWidth = 80,
  currencyPlaceholder = '币种',
  amountPlaceholder = '0.00',
  disabled = false,
  allowClear = true,
  maxLength = 23,
  amountRuleMessage = '请输入有效金额，最多 4 位小数',
  requireAmountWhenCurrency = false,
  emptyAmountMessage = '请输入金额',
  emptyCurrencyMessage = '请选择币种',
  amountPattern = DEFAULT_AMOUNT_PATTERN,
  showTooltip = true,
  style,
  className,
}) => {
  return (
    <Space.Compact
      block
      className={`roncin-currency-amount-input ${className || ''}`}
      style={{ width: '100%', ...style }}
    >
      {/* 1. 左侧紧凑币种选择 */}
      <Form.Item
        noStyle
        name={currencyName}
        dependencies={[amountName]}
        rules={[
          ({ getFieldValue }) => ({
            validator: async (_, val) => {
              const amount = getFieldValue(amountName);
              const rawAmount = String(amount ?? '').trim();
              if (!rawAmount || val) return;
              throw new Error(emptyCurrencyMessage);
            },
          }),
        ]}
      >
        <Select
          showSearch={showSearch ? { filterOption } : false}
          style={{ width: currencyWidth }}
          placeholder={currencyPlaceholder}
          options={currencyOptions}
          popupMatchSelectWidth={false}
          disabled={disabled}
          allowClear={allowClear}
          labelRender={(optionProps) =>
            formatSelectedCurrency(optionProps, renderSelectedCurrency)
          }
        />
      </Form.Item>

      {/* 2. 右侧对齐金额输入 */}
      <Form.Item
        noStyle
        name={amountName}
        dependencies={[currencyName]}
        rules={[
          ({ getFieldValue }) => ({
            validator: async (_, val) => {
              const currency = getFieldValue(currencyName);
              const raw = String(val ?? '').trim();
              if (!raw) {
                if (currency && requireAmountWhenCurrency) {
                  throw new Error(emptyAmountMessage);
                }
                return;
              }
              if (!amountPattern.test(raw)) {
                throw new Error(amountRuleMessage);
              }
            },
          }),
        ]}
      >
        <TooltipNumericInput
          placeholder={amountPlaceholder}
          maxLength={maxLength}
          disabled={disabled}
          style={{ width: `calc(100% - ${currencyWidth}px)` }}
          showTooltip={showTooltip}
        />
      </Form.Item>
    </Space.Compact>
  );
};
