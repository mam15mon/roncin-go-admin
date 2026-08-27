import React, { useMemo, useState } from 'react';
import { DatePicker, Radio } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import { standardDateRangePresets } from '../date-presets';
import type {
  QuickDateFilterBarProps,
  QuickDateRangePickerProps,
  RangeValue,
  StandardPresetKey,
} from './types';

const { RangePicker } = DatePicker;

/**
 * 预设 Key 与计算逻辑映射表
 */
export const PRESET_CALCULATORS: Record<
  StandardPresetKey,
  { label: string; getRange: () => [Dayjs, Dayjs] }
> = {
  today: {
    label: '今天',
    getRange: () => [dayjs().startOf('day'), dayjs().endOf('day')],
  },
  thisWeek: {
    label: '本周',
    getRange: () => [dayjs().startOf('week'), dayjs().endOf('week')],
  },
  thisMonth: {
    label: '本月',
    getRange: () => [dayjs().startOf('month'), dayjs().endOf('month')],
  },
  lastMonth: {
    label: '上月',
    getRange: () => [
      dayjs().subtract(1, 'month').startOf('month'),
      dayjs().subtract(1, 'month').endOf('month'),
    ],
  },
  last3Months: {
    label: '最近3个月',
    getRange: () => [
      dayjs().subtract(3, 'month').startOf('day'),
      dayjs().endOf('day'),
    ],
  },
  last6Months: {
    label: '最近半年',
    getRange: () => [
      dayjs().subtract(6, 'month').startOf('day'),
      dayjs().endOf('day'),
    ],
  },
  thisYear: {
    label: '今年以来',
    getRange: () => [dayjs().startOf('year'), dayjs().endOf('day')],
  },
  last1Year: {
    label: '最近1年',
    getRange: () => [
      dayjs().subtract(1, 'year').startOf('day'),
      dayjs().endOf('day'),
    ],
  },
};

/**
 * 通用增强型日期区间选择器 (QuickDateRangePicker)
 * 内置货代财务标准 8 项快捷预设（日历弹出面板左侧快捷选取）
 */
export const QuickDateRangePicker: React.FC<QuickDateRangePickerProps> = ({
  enableStandardPresets = true,
  customPresets,
  presets: explicitPresets,
  style,
  ...restProps
}) => {
  const mergedPresets = useMemo(() => {
    if (explicitPresets !== undefined) {
      return explicitPresets;
    }
    if (!enableStandardPresets) {
      return customPresets;
    }
    if (!customPresets) {
      return standardDateRangePresets;
    }
    return [
      ...(standardDateRangePresets || []),
      ...(customPresets || []),
    ] as typeof standardDateRangePresets;
  }, [enableStandardPresets, customPresets, explicitPresets]);

  return (
    <RangePicker
      presets={mergedPresets}
      style={{ width: '100%', ...style }}
      {...restProps}
    />
  );
};

/**
 * 外露式快捷日期标签栏 + 日期选择器组合组件 (QuickDateFilterBar)
 * 适用于报表看板、财务工作台、台账顶部快捷一键过滤
 */
export const QuickDateFilterBar: React.FC<QuickDateFilterBarProps> = ({
  value,
  onChange,
  visiblePresets = [
    'all',
    'today',
    'thisMonth',
    'lastMonth',
    'last3Months',
    'last6Months',
  ],
  size = 'small',
  showPicker = true,
  style,
  className,
  prefix,
}) => {
  const [activeKey, setActiveKey] = useState<string>('all');

  const handleRadioChange = (key: string) => {
    setActiveKey(key);
    if (key === 'all') {
      onChange?.(null, ['', ''], 'all');
      return;
    }

    const calculator = PRESET_CALCULATORS[key as StandardPresetKey];
    if (calculator) {
      const range = calculator.getRange();
      const dateStrings: [string, string] = [
        range[0].format('YYYY-MM-DD'),
        range[1].format('YYYY-MM-DD'),
      ];
      onChange?.(range, dateStrings, key);
    }
  };

  const handlePickerChange = (
    dates: RangeValue,
    dateStrings: [string, string],
  ) => {
    setActiveKey('custom');
    onChange?.(dates, dateStrings, 'custom');
  };

  return (
    <div
      className={className}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 8,
        flexWrap: 'wrap',
        ...style,
      }}
    >
      {prefix && (
        <span style={{ fontSize: 13, color: '#595959', fontWeight: 500 }}>
          {prefix}
        </span>
      )}

      <Radio.Group
        value={activeKey}
        size={size}
        onChange={(e) => handleRadioChange(e.target.value)}
        optionType="button"
        buttonStyle="solid"
      >
        {visiblePresets.map((presetKey) => {
          if (presetKey === 'all') {
            return (
              <Radio.Button key="all" value="all">
                全部
              </Radio.Button>
            );
          }
          const item = PRESET_CALCULATORS[presetKey];
          if (!item) return null;
          return (
            <Radio.Button key={presetKey} value={presetKey}>
              {item.label}
            </Radio.Button>
          );
        })}
      </Radio.Group>

      {showPicker && (
        <QuickDateRangePicker
          size={size}
          value={value}
          onChange={handlePickerChange}
          style={{ width: 230 }}
        />
      )}
    </div>
  );
};

export default QuickDateRangePicker;
