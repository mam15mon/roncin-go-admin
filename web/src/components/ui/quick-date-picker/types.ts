import type { TimeRangePickerProps } from 'antd';
import type { Dayjs } from 'dayjs';
import type { CSSProperties, ReactNode } from 'react';
import type { RangeValue } from '../date-presets';

export type { RangeValue };

export type StandardPresetKey =
  | 'today'
  | 'thisWeek'
  | 'thisMonth'
  | 'lastMonth'
  | 'last3Months'
  | 'last6Months'
  | 'thisYear'
  | 'last1Year';

export interface QuickDatePresetOption {
  key: string;
  label: string;
  value: [Dayjs, Dayjs] | (() => [Dayjs, Dayjs]);
}

export interface QuickDateRangePickerProps extends TimeRangePickerProps {
  /**
   * 是否使用默认货代财务标准预设（今天、本周、本月、上月、最近3个月、最近半年、今年以来、最近1年）
   * 默认 true
   */
  enableStandardPresets?: boolean;
  /**
   * 自定义扩展或覆盖的预设项
   */
  customPresets?: TimeRangePickerProps['presets'];
}

export interface QuickDateFilterBarProps {
  value?: RangeValue;
  onChange?: (
    dates: RangeValue,
    dateStrings: [string, string],
    presetKey?: string,
  ) => void;
  /**
   * 外露展示的快捷标签项列表，默认为 ['all', 'today', 'thisMonth', 'lastMonth', 'last3Months', 'last6Months']
   */
  visiblePresets?: ('all' | StandardPresetKey)[];
  /**
   * 包含的标签大小，默认为 'small'
   */
  size?: 'small' | 'middle' | 'large';
  /**
   * 是否在快捷标签右侧同时显示日期选择输入框
   * 默认 true
   */
  showPicker?: boolean;
  /**
   * 容器样式
   */
  style?: CSSProperties;
  className?: string;
  /**
   * 前缀图标或说明文案
   */
  prefix?: ReactNode;
}
