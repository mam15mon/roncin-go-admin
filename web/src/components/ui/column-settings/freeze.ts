import { Table } from 'antd';
import type { Key } from 'react';
import type { ColumnFreezeValue } from './types';

/** 只描述展示层需要的官方列属性，保持业务列对象的类型和回调。 */
interface FreezeColumn {
  key?: Key;
  dataIndex?: unknown;
  title?: unknown;
  fixed?: 'left' | 'right' | 'start' | 'end' | boolean;
  valueType?: unknown;
  hideInTable?: boolean;
  hidden?: boolean;
  children?: readonly FreezeColumn[];
}

export interface ColumnFreezeOptions {
  /** 操作等结构列不作为首个业务列；仍可作为末尾冻结目标。 */
  firstExcludedKeys?: readonly string[];
}

function keyOf(column: FreezeColumn): string | undefined {
  if (column.key != null) return String(column.key);
  if (Array.isArray(column.dataIndex)) return column.dataIndex.join('-');
  return typeof column.dataIndex === 'string' ||
    typeof column.dataIndex === 'number'
    ? String(column.dataIndex)
    : undefined;
}

function isAuxiliary(column: FreezeColumn): boolean {
  return (
    column === Table.SELECTION_COLUMN ||
    column === Table.EXPAND_COLUMN ||
    column.valueType === 'index' ||
    column.valueType === 'indexBorder' ||
    keyOf(column) === 'index'
  );
}

function visibleLeaves(columns: readonly FreezeColumn[]): FreezeColumn[] {
  return columns.flatMap((column) => {
    if (column.hideInTable || column.hidden) return [];
    return column.children ? visibleLeaves(column.children) : [column];
  });
}

/** 缺省开关从原业务固定区域推导，选择/展开/序号自身不产生开关。 */
export function deriveColumnFreezeDefaults(
  columns: readonly FreezeColumn[],
): ColumnFreezeValue {
  const data = visibleLeaves(columns).filter((column) => !isAuxiliary(column));
  return {
    first: data.some(
      (column) =>
        column.fixed === 'left' ||
        column.fixed === 'start' ||
        column.fixed === true,
    ),
    last: data.some(
      (column) => column.fixed === 'right' || column.fixed === 'end',
    ),
  };
}

/** 缺失或非法开关使用业务默认，合法 false 必须保留。 */
export function resolveColumnFreeze(
  value: unknown,
  defaults: ColumnFreezeValue,
): ColumnFreezeValue {
  const parsed = value && typeof value === 'object' ? value : {};
  const first = 'first' in parsed ? parsed.first : undefined;
  const last = 'last' in parsed ? parsed.last : undefined;
  return {
    first: typeof first === 'boolean' ? first : defaults.first,
    last: typeof last === 'boolean' ? last : defaults.last,
  };
}

/** 在最终可见顺序中识别首尾，供列应用和浮层目标提示共用。 */
export function getColumnFreezeTargets<C extends FreezeColumn>(
  columns: readonly C[],
  options: ColumnFreezeOptions = {},
): { first?: C; last?: C } {
  const leaves = visibleLeaves(columns).filter(
    (column) => !isAuxiliary(column),
  );
  const excluded = new Set(options.firstExcludedKeys);
  const first = leaves.find(
    (column) =>
      column.valueType !== 'option' &&
      keyOf(column) !== 'option' &&
      keyOf(column) !== 'action' &&
      !excluded.has(keyOf(column) ?? '') &&
      (keyOf(column) !== undefined || column.dataIndex !== undefined),
  );
  return {
    first: first as C | undefined,
    last: leaves.at(-1) as C | undefined,
  };
}

/** 统一按首尾开关替换普通列的原固定设置，辅助结构列保持原规则。 */
export function applyColumnFreeze<C extends FreezeColumn>(
  columns: readonly C[],
  freeze: ColumnFreezeValue,
  options: ColumnFreezeOptions = {},
): C[] {
  const targets = getColumnFreezeTargets(columns, options);
  const apply = (column: FreezeColumn): FreezeColumn => {
    if (isAuxiliary(column)) return column;
    if (column.children) {
      return { ...column, fixed: false, children: column.children.map(apply) };
    }
    const fixed =
      freeze.first && column === targets.first
        ? 'left'
        : freeze.last && column === targets.last
          ? 'right'
          : false;
    return { ...column, fixed };
  };
  return columns.map(apply) as C[];
}
