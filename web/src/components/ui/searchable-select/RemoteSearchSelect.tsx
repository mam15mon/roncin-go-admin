import { Select, Spin } from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import type { RemoteSearchSelectProps, SearchableSelectProps } from './types';

/**
 * 远程搜索下拉组件：候选项由 `request` 按关键字异步返回（服务端过滤）。
 * - 挂载即以空关键字加载首屏候选；
 * - 输入联想默认 300ms 防抖，`filterOption` 固定关闭本地过滤；
 * - 竞态防护：仅采纳最后一次请求的响应，卸载后不再写入状态。
 */
export const RemoteSearchSelect: React.FC<RemoteSearchSelectProps> = ({
  request,
  debounceMs = 300,
  allowClear = true,
  ...rest
}) => {
  const [options, setOptions] = useState<SearchableSelectProps['options']>([]);
  const [loading, setLoading] = useState(false);
  const seqRef = useRef(0);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const requestRef = useRef(request);

  useEffect(() => {
    requestRef.current = request;
  }, [request]);

  const load = useCallback(async (keyWords?: string) => {
    const seq = seqRef.current + 1;
    seqRef.current = seq;
    setLoading(true);
    try {
      const result = await requestRef.current(keyWords);
      if (seqRef.current !== seq) return;
      setOptions(result ?? []);
    } catch (error) {
      if (seqRef.current !== seq) return;
      setOptions([]);
      console.error('RemoteSearchSelect 候选项加载失败', error);
    } finally {
      if (seqRef.current === seq) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void load();
    return () => {
      seqRef.current += 1;
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [load]);

  const handleSearch = useCallback(
    (value: string) => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
      const keyword = value.trim();
      timerRef.current = setTimeout(() => {
        void load(keyword || undefined);
      }, debounceMs);
    },
    [debounceMs, load],
  );

  return (
    <Select
      {...rest}
      showSearch
      filterOption={false}
      allowClear={allowClear}
      loading={loading}
      notFoundContent={loading ? <Spin size="small" /> : undefined}
      options={options}
      onSearch={handleSearch}
    />
  );
};

export default RemoteSearchSelect;
