import type { CSSProperties, ReactNode } from 'react';

export interface CurrencyAmountOption {
  label: ReactNode;
  value: string | number;
  [key: string]: any;
}

export interface CurrencyAmountInputProps {
  /** 币种表单字段名，如 'insuranceCurrency' 或 'cargoCurrency' */
  currencyName: string;
  /** 金额表单字段名，如 'insurancePremium' 或 'cargoValue' */
  amountName: string;
  /** 可选币种下拉列表 */
  currencyOptions: CurrencyAmountOption[];
  /** 选定币种后在选择框内展示的格式化函数，默认优先展示纯货币代码（如 'CNY' 而非 'CNY - 人民币'） */
  renderSelectedCurrency?: (option: {
    label: ReactNode;
    value: string | number;
  }) => ReactNode;
  /** 是否开启币种拼音/代码/中文快速搜索，默认 true */
  showSearch?: boolean;
  /** 自定义币种下拉过滤函数，默认复用系统 defaultSelectFilterOption */
  filterOption?: boolean | ((input: string, option?: any) => boolean);
  /** 币种选择框宽度（像素），默认 80 */
  currencyWidth?: number;
  /** 币种选择框占位提示，默认 '币种' */
  currencyPlaceholder?: string;
  /** 金额输入框占位提示，默认 '0.00' */
  amountPlaceholder?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 金额最大输入字符长度，默认 23 */
  maxLength?: number;
  /** 金额格式校验提示文案 */
  amountRuleMessage?: string;
  /** 填了币种但未填金额时的错误提示，默认 '请输入金额' */
  emptyAmountMessage?: string;
  /** 填了金额但未选币种时的错误提示，默认 '请选择币种' */
  emptyCurrencyMessage?: string;
  /** 自定义金额校验正则，默认允许最多 18 位整数、4 位小数 */
  amountPattern?: RegExp;
  /** 金额超出时是否启用 Tooltip 悬浮查看，默认 true */
  showTooltip?: boolean;
  /** 容器额外内联样式 */
  style?: CSSProperties;
  className?: string;
}
