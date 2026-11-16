import type { BaseOptionType } from 'antd/es/select';
import { isValidElement } from 'react';
import { matchesSearch } from '@/utils/search';

function labelText(value: unknown): string {
  if (typeof value === 'string' || typeof value === 'number')
    return String(value);
  if (Array.isArray(value)) return value.map(labelText).join(' ');
  if (isValidElement<{ children?: unknown }>(value))
    return labelText(value.props.children);
  return '';
}

/** 全站本地下拉统一匹配名称、代码、别名、全拼与首字母。远程搜索关闭本地过滤。 */
export const defaultSelectFilterOption = (
  input: string,
  option?: BaseOptionType,
): boolean => {
  if (!input?.trim()) return true;
  if (!option) return false;
  return matchesSearch(
    input,
    labelText(option.label),
    option.value,
    ...[
      'code',
      'name',
      'nameZh',
      'nameEn',
      'aliasName',
      'feeCode',
      'title',
      'key',
      'search',
      'keywords',
    ].map((field) => option[field]),
    labelText(option.children),
  );
};
