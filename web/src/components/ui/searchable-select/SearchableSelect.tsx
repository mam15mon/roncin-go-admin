import { ProFormSelect } from '@ant-design/pro-components';
import { Select } from 'antd';
import React from 'react';
import type { ProFormSearchableSelectProps, SearchableSelectProps } from './types';
import { defaultSelectFilterOption } from './utils';

/**
 * 通用 Ant Design SearchableSelect 下拉组件
 * 默认开启模糊搜索（showSearch: true）与智能匹配（filterOption: defaultSelectFilterOption）
 */
export const SearchableSelect: React.FC<SearchableSelectProps> = ({
  showSearch = true,
  filterOption = defaultSelectFilterOption,
  allowClear = true,
  ...rest
}) => {
  return (
    <Select
      showSearch={showSearch}
      filterOption={filterOption}
      allowClear={allowClear}
      {...rest}
    />
  );
};

/**
 * 通用 ProFormSearchableSelect 表单下拉项
 * 默认开启模糊搜索与清除功能，无缝适配 ProForm 栅格和各类表单场景
 */
export const ProFormSearchableSelect: React.FC<ProFormSearchableSelectProps> = ({
  showSearch = true,
  allowClear = true,
  fieldProps,
  ...rest
}) => {
  const mergedFieldProps = {
    showSearch: showSearch ?? true,
    allowClear: allowClear ?? true,
    filterOption: defaultSelectFilterOption,
    ...fieldProps,
  };

  return (
    <ProFormSelect
      showSearch={showSearch}
      allowClear={allowClear}
      fieldProps={mergedFieldProps}
      {...rest}
    />
  );
};
