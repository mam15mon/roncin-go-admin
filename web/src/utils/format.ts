import dayjs from 'dayjs';

type DateFormat = 'date' | 'datetime';

export function formatDate(
  value?: string | number | Date | null,
  format: DateFormat = 'datetime',
): string {
  if (value === undefined || value === null || value === '') return '-';
  const parsed = dayjs(value);
  if (!parsed.isValid()) return '-';
  return parsed.format(
    format === 'date' ? 'YYYY-MM-DD' : 'YYYY-MM-DD HH:mm:ss',
  );
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
  return String(value)
    .replace(/(\.\d*?)0+$/, '$1')
    .replace(/\.$/, '');
}
