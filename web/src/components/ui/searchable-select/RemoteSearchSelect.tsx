import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Select, Spin } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { RemoteSearchSelectProps } from './types';

/** 远程候选按资源身份及防抖关键词查询，已选名称在渲染侧合并保活。 */
export const RemoteSearchSelect: React.FC<RemoteSearchSelectProps> = ({
  queryKey,
  request,
  selectedOptions = [],
  debounceMs = 300,
  allowClear = true,
  ...rest
}) => {
  const client = useQueryClient();
  const [keyword, setKeyword] = useState('');
  const [debouncedKeyword, setDebouncedKeyword] = useState('');
  const requestRef = useRef(request);
  requestRef.current = request;

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(keyword), debounceMs);
    return () => clearTimeout(timer);
  }, [debounceMs, keyword]);

  const query = useQuery({
    queryKey: [...queryKey, { keyword: debouncedKeyword }],
    queryFn: async ({ signal }) => {
      const options = await requestRef.current(debouncedKeyword || undefined, {
        signal,
        skipErrorHandler: true,
      });
      if (!Array.isArray(options))
        throw new Error('候选项加载失败，请稍后重试');
      return options;
    },
    meta: { errorMessage: '候选项加载失败，请稍后重试' },
  });
  const options = useMemo(() => {
    const values: unknown[] = Array.isArray(rest.value)
      ? rest.value
      : [rest.value];
    const selectedValues = new Set(
      values.map((value) =>
        value && typeof value === 'object' && 'value' in value
          ? value.value
          : value,
      ),
    );
    const retainedOptions = client
      .getQueriesData<NonNullable<RemoteSearchSelectProps['selectedOptions']>>({
        queryKey,
      })
      .flatMap(([, items]) => items ?? [])
      .filter(
        (option) =>
          option.value !== undefined && selectedValues.has(option.value),
      );
    return Array.from(
      new Map(
        [...selectedOptions, ...retainedOptions, ...(query.data ?? [])].map(
          (option) => [option.value, option],
        ),
      ).values(),
    );
  }, [client, query.data, queryKey, rest.value, selectedOptions]);

  return (
    <Select
      {...rest}
      showSearch={{
        filterOption: false,
        onSearch: (value) => setKeyword(value.trim()),
      }}
      allowClear={allowClear}
      loading={query.isFetching}
      notFoundContent={query.isFetching ? <Spin size="small" /> : undefined}
      options={options}
    />
  );
};

export default RemoteSearchSelect;
