import type { TimeRangePickerProps } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';

export type RangeValue = [Dayjs | null, Dayjs | null] | null;

/**
 * 货代与财务全站标准日期区间预设快捷选项 (Date Range Presets)
 * 支持一键选取：今天、本周、本月、上月、最近3个月、最近半年、今年以来、最近1年
 */
export const standardDateRangePresets: TimeRangePickerProps['presets'] = [
  {
    label: '今天',
    value: [dayjs().startOf('day'), dayjs().endOf('day')],
  },
  {
    label: '本周',
    value: [dayjs().startOf('week'), dayjs().endOf('week')],
  },
  {
    label: '本月',
    value: [dayjs().startOf('month'), dayjs().endOf('month')],
  },
  {
    label: '上月',
    value: [
      dayjs().subtract(1, 'month').startOf('month'),
      dayjs().subtract(1, 'month').endOf('month'),
    ],
  },
  {
    label: '最近3个月',
    value: [dayjs().subtract(3, 'month').startOf('day'), dayjs().endOf('day')],
  },
  {
    label: '最近半年',
    value: [dayjs().subtract(6, 'month').startOf('day'), dayjs().endOf('day')],
  },
  {
    label: '今年以来',
    value: [dayjs().startOf('year'), dayjs().endOf('day')],
  },
  {
    label: '最近1年',
    value: [dayjs().subtract(1, 'year').startOf('day'), dayjs().endOf('day')],
  },
];

export default standardDateRangePresets;
