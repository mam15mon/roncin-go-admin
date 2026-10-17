import { trimDecimal } from './format';

export type CsvValue = string | number | null | undefined;

/** 导出列独立于界面渲染；数值列只允许经过字段格式化的合法十进制数。 */
export interface CsvColumn<T> {
  header: string;
  kind?: 'text' | 'number';
  value: (item: T) => CsvValue;
}

export function csvAmount(value: CsvValue): string {
  if (value === undefined || value === null || value === '') return '';
  const number = Number(value);
  return Number.isFinite(number) ? number.toFixed(2) : '';
}

export function csvDecimal(value: CsvValue): string {
  return value === undefined || value === null || value === ''
    ? ''
    : trimDecimal(value);
}

function escapeCell(value: CsvValue, kind: CsvColumn<unknown>['kind']): string {
  let text = String(value ?? '');
  const numeric = kind === 'number' && /^-?\d+(?:\.\d+)?$/.test(text);
  if (!numeric && /^\s*[=+\-@]/u.test(text)) text = `'${text}`;
  return /[",\r\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
}

export function serializeCsv<T>(
  items: readonly T[],
  columns: readonly CsvColumn<T>[],
): string | null {
  if (items.length === 0) return null;
  const header = columns.map((column) => escapeCell(column.header, 'text'));
  const rows = items.map((item) =>
    columns.map((column) => escapeCell(column.value(item), column.kind)),
  );
  return `\uFEFF${[header, ...rows].map((row) => row.join(',')).join('\r\n')}`;
}

export function downloadCsv(content: string, fileName: string): void {
  const url = URL.createObjectURL(
    new Blob([content], { type: 'text/csv;charset=utf-8;' }),
  );
  const link = document.createElement('a');
  try {
    link.href = url;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
  } finally {
    link.remove();
    URL.revokeObjectURL(url);
  }
}
