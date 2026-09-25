import type { ProFormSelectProps } from '@ant-design/pro-components';
import type { SelectProps } from 'antd';
import type { BaseOptionType, DefaultOptionType } from 'antd/es/select';

export interface SearchableSelectProps<
  // biome-ignore lint/suspicious/noExplicitAny: 模板泛型默认/边界保持消费方零改动的宽松度；收紧需模板泛型化改造（后续任务）
  ValueType = any,
  OptionType extends BaseOptionType | DefaultOptionType = DefaultOptionType,
> extends SelectProps<ValueType, OptionType> {
  /** 是否启用全局默认智能模糊搜索，默认 true */
  showSearch?: SelectProps<ValueType, OptionType>['showSearch'];
}

export interface ProFormSearchableSelectProps extends ProFormSelectProps {
  /** 是否启用全局默认智能模糊搜索，默认 true */
  showSearch?: ProFormSelectProps['showSearch'];
}

export interface RemoteSearchSelectProps
  extends Omit<SearchableSelectProps, 'options' | 'filterOption'> {
  /** 远程候选项请求：按关键字返回候选（服务端过滤），挂载时以空关键字调用一次 */
  request: (keyWords?: string) => Promise<SearchableSelectProps['options']>;
  /** 输入联想防抖毫秒数，默认 300 */
  debounceMs?: number;
}
