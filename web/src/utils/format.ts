import dayjs from 'dayjs';

const numberFormatter = new Intl.NumberFormat('en-US');

type DateFormat = 'date' | 'datetime';

/**
 * Format a number with thousand separators.
 * Replaces numeral(val).format('0,0')
 */
export const formatNumber = (val: number | string): string => {
  const parsed = Number(val);
  return Number.isFinite(parsed) ? numberFormatter.format(parsed) : '';
};

/**
 * Format a number as yuan currency string.
 * Replaces `¥ ${numeral(val).format('0,0')}`
 */
export const formatYuan = (val: number | string) => `¥ ${formatNumber(val)}`;

export function formatDate(
  value?: string | number | Date | null,
  format: DateFormat = 'datetime',
): string {
  if (value === undefined || value === null || value === '') return '-';
  const parsed = dayjs(value);
  if (!parsed.isValid()) return '-';
  return parsed.format(format === 'date' ? 'YYYY-MM-DD' : 'YYYY-MM-DD HH:mm:ss');
}

export function formatAmount(
  value?: string | number | null,
  decimals = 2,
): string {
  if (value === undefined || value === null || value === '') return '-';
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return '-';
  return parsed.toLocaleString('zh-CN', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
}

export function trimDecimal(value?: string | number | null): string {
  if (value === undefined || value === null || value === '') return '-';
  return String(value).replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '');
}
