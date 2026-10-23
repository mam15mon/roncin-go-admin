import { Table } from 'antd';
import type { Key } from 'react';
import type { ColumnFreezeValue } from './types';

interface FreezeColumn {
  key?: Key;
  dataIndex?: unknown;
  fixed?: 'left' | 'right' | 'start' | 'end' | boolean;
  valueType?: unknown;
  hideInTable?: boolean;
  hidden?: boolean;
  children?: readonly FreezeColumn[];
}

export interface ColumnFreezeOptions {
  /** 结构列保留锚点，不参与普通业务列的区域重排。 */
  structuralKeys?: readonly string[];
}

export function columnFreezeKey(column: FreezeColumn): string | undefined {
  if (column.key != null) return String(column.key);
  if (Array.isArray(column.dataIndex)) return column.dataIndex.join('-');
  if (
    typeof column.dataIndex === 'string' ||
    typeof column.dataIndex === 'number'
  )
    return String(column.dataIndex);
  if (column.valueType === 'option') return 'option';
  return undefined;
}

function isAuxiliary(column: FreezeColumn): boolean {
  return (
    column === Table.SELECTION_COLUMN ||
    column === Table.EXPAND_COLUMN ||
    column.valueType === 'index' ||
    column.valueType === 'indexBorder' ||
    columnFreezeKey(column) === 'index'
  );
}

export function deriveColumnFreezeDefaults(
  columns: readonly FreezeColumn[],
): ColumnFreezeValue {
  const result: ColumnFreezeValue = { left: [], right: [] };
  for (const column of columns) {
    if (isAuxiliary(column)) continue;
    if (column.children) {
      const nested = deriveColumnFreezeDefaults(column.children);
      result.left.push(...nested.left);
      result.right.push(...nested.right);
      continue;
    }
    const key = columnFreezeKey(column);
    if (!key) continue;
    if (
      column.fixed === 'left' ||
      column.fixed === 'start' ||
      column.fixed === true
    )
      result.left.push(key);
    if (column.fixed === 'right' || column.fixed === 'end')
      result.right.push(key);
  }
  return result;
}

/** 仅解析当前字段冻结契约；旧首尾开关不具有字段冻结含义。 */
export function resolveColumnFreeze(
  value: unknown,
  defaults: ColumnFreezeValue,
  knownKeys?: readonly string[],
): ColumnFreezeValue {
  const parsed =
    value && typeof value === 'object'
      ? (value as Record<string, unknown>)
      : {};
  const known = knownKeys ? new Set(knownKeys) : undefined;
  const clean = (input: unknown, fallback: string[]) =>
    [
      ...new Set(
        Array.isArray(input) && input.every((key) => typeof key === 'string')
          ? (input as string[])
          : fallback,
      ),
    ].filter((key) => key.length > 0 && (!known || known.has(key)));
  const left = clean(parsed.left, defaults.left);
  const leftKeys = new Set(left);
  return {
    left,
    right: clean(parsed.right, defaults.right).filter(
      (key) => !leftKeys.has(key),
    ),
  };
}

/** 所有展示入口共享左冻结、普通、右冻结的唯一顺序。 */
export function orderColumnKeys(
  keys: readonly string[],
  freeze: ColumnFreezeValue,
): string[] {
  const left = new Set(freeze.left);
  const right = new Set(freeze.right);
  return [
    ...keys.filter((key) => left.has(key)),
    ...keys.filter((key) => !left.has(key) && !right.has(key)),
    ...keys.filter((key) => right.has(key) && !left.has(key)),
  ];
}

/** 冻结业务列并按区域排列，结构列和官方哨兵保持锚点。 */
export function applyColumnFreeze<C extends FreezeColumn>(
  columns: readonly C[],
  freeze: ColumnFreezeValue,
  options: ColumnFreezeOptions = {},
): C[] {
  const left = new Set(freeze.left);
  const right = new Set(freeze.right);
  const structural = new Set([
    'action',
    'option',
    ...(options.structuralKeys ?? []),
  ]);
  const apply = (column: C): C => {
    if (isAuxiliary(column)) return column;
    if (column.children)
      return {
        ...column,
        fixed: false,
        children: applyColumnFreeze(column.children, freeze, options),
      };
    const key = columnFreezeKey(column);
    return key
      ? {
          ...column,
          fixed: left.has(key) ? 'left' : right.has(key) ? 'right' : false,
        }
      : column;
  };
  const applied = columns.map(apply);
  const movable = applied.filter(
    (column) =>
      !isAuxiliary(column) &&
      !column.children &&
      column.valueType !== 'option' &&
      !column.hideInTable &&
      !!columnFreezeKey(column) &&
      !structural.has(columnFreezeKey(column) as string),
  );
  const ordered = orderColumnKeys(
    movable.map((column) => columnFreezeKey(column) as string),
    freeze,
  );
  const byKey = new Map(
    movable.map((column) => [columnFreezeKey(column), column]),
  );
  let index = 0;
  return applied.map((column) =>
    movable.includes(column) ? (byKey.get(ordered[index++]) as C) : column,
  );
}
