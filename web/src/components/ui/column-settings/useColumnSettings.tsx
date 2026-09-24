import { App } from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import { ColumnSettingsEntry } from './ColumnSettingsEntry';
import ColumnSettingsModal from './ColumnSettingsModal';
import {
  defaultColumnSettingsValue,
  loadColumnSettingsPreference,
  resolveColumnSettingsValue,
  saveColumnSettingsPreference,
} from './preference';
import type {
  ColumnSettingsField,
  ColumnSettingsScope,
  ColumnSettingsValue,
} from './types';

/** 钩子内部读取列定义所需的最小形状，兼容 antd Table 与 ProTable 列。 */
interface ColumnLike {
  key?: React.Key;
  dataIndex?: string | string[];
  title?: React.ReactNode;
  fixed?: 'left' | 'right' | boolean;
  hideInTable?: boolean;
}

export interface UseColumnSettingsOptions {
  /** 稳定表格标识：按业务视图命名（如 `orders:list`），不使用实体 ID。 */
  tableKey: string;
  /** 基准列定义（应用显隐排序前）。 */
  columns: unknown[];
  /** 偏好隔离范围（用户 + 组织）；缺省时同浏览器共享。 */
  scope?: ColumnSettingsScope;
  /** 结构列 key（选择、展开、操作等）：始终渲染，不进入设置列表。 */
  structuralKeys?: string[];
  /** 必显列 key：可排序但不可隐藏。 */
  lockVisibleKeys?: string[];
  /** 默认隐藏列 key：未保存偏好时按此显隐。 */
  defaultHiddenKeys?: string[];
  /** 列标题为 ReactNode 时的纯文本名称覆盖；未覆盖的列不进入设置列表。 */
  titleOverrides?: Record<string, string>;
  /** 禁用入口（如编辑进行中），配合 disabledReason 提示原因。 */
  disabled?: boolean;
  disabledReason?: string;
  /** 弹窗标题，默认「列设置」。 */
  title?: string;
  /** 增强版高级设置页签内容；偏好由外部保存时通常配合 persist=false。 */
  advanced?: React.ReactNode;
  /** 关闭本地持久化：设置仅当前页面生效，由调用方自行保存。 */
  persist?: false;
}

function zoneOf(fixed: ColumnLike['fixed']): 'left' | 'right' | undefined {
  if (fixed === 'left' || fixed === 'right') return fixed;
  return undefined;
}

/** 解析列稳定标识；key 与 dataIndex 均缺失的列无法参与设置。 */
function columnKeyOf(column: ColumnLike, index: number): string | null {
  if (column.key !== undefined && column.key !== null && String(column.key)) {
    return String(column.key);
  }
  if (typeof column.dataIndex === 'string' && column.dataIndex) {
    return column.dataIndex;
  }
  if (Array.isArray(column.dataIndex) && column.dataIndex.length > 0) {
    return column.dataIndex.join('-');
  }
  void index;
  return null;
}

/** 提取纯文本列名；ReactNode 标题仅识别单层文本子节点，其余交给覆盖项。 */
function columnTitle(column: ColumnLike): string {
  if (typeof column.title === 'string') return column.title;
  if (React.isValidElement<{ children?: React.ReactNode }>(column.title)) {
    const { children } = column.title.props;
    if (typeof children === 'string') return children;
  }
  return '';
}

/** 派生设置元数据：可命名、可隐藏的普通列进入设置，其余按结构列处理。 */
function deriveFields(
  columns: unknown[],
  options: Pick<UseColumnSettingsOptions, 'structuralKeys' | 'lockVisibleKeys' | 'titleOverrides'>,
): ColumnSettingsField[] {
  const structural = new Set(options.structuralKeys ?? []);
  const locked = new Set(options.lockVisibleKeys ?? []);
  const overrides = options.titleOverrides ?? {};
  const fields: ColumnSettingsField[] = [];
  columns.forEach((raw, index) => {
    const column = raw as ColumnLike;
    const key = columnKeyOf(column, index);
    if (!key || structural.has(key) || column.hideInTable) return;
    const title = overrides[key] ?? columnTitle(column);
    if (!title) return;
    fields.push({
      key,
      title,
      lockVisible: locked.has(key),
      fixed: zoneOf(column.fixed),
    });
  });
  return fields;
}

/**
 * 应用列配置：按偏好顺序输出受管列（隐藏列剔除），结构列保持在
 * 原始相对锚点（如操作列保持在最后），固定列区域不跨越。
 */
function applyColumnSettings<C>(
  columns: unknown[],
  value: ColumnSettingsValue,
  structuralKeys: string[] | undefined,
): C[] {
  const structural = new Set(structuralKeys ?? []);
  const hidden = new Set(value.hidden);
  const managed = new Map<string, ColumnLike>();
  const structuralItems: { column: ColumnLike; anchor: number }[] = [];
  let managedCount = 0;
  columns.forEach((raw, index) => {
    const column = raw as ColumnLike;
    const key = columnKeyOf(column, index);
    const isStructural =
      !key || structural.has(key) || column.hideInTable === true;
    if (isStructural) {
      structuralItems.push({ column, anchor: managedCount });
      return;
    }
    if (!managed.has(key as string)) {
      managed.set(key as string, column);
      managedCount += 1;
    }
  });

  const orderedKeys = value.order.filter((key) => managed.has(key));
  for (const key of managed.keys()) {
    if (!orderedKeys.includes(key)) orderedKeys.push(key);
  }

  const result: ColumnLike[] = [];
  const pending = [...structuralItems];
  let emitted = 0;
  for (const key of orderedKeys) {
    while (pending.length > 0 && pending[0].anchor <= emitted) {
      result.push((pending.shift() as { column: ColumnLike }).column);
    }
    if (!hidden.has(key)) {
      result.push(managed.get(key) as ColumnLike);
    }
    emitted += 1;
  }
  for (const item of pending) {
    result.push(item.column);
  }
  return result as C[];
}

export interface UseColumnSettingsResult<C> {
  /** 应用显隐与排序后的列定义，直接传给 Table/ProTable。 */
  columns: C[];
  /** 统一列设置入口按钮；放入工具栏或表格上方右侧。 */
  entry: React.ReactNode;
  /** 统一列设置弹窗；置于页面任意位置即可。 */
  modal: React.ReactNode;
}

/**
 * 全站统一列设置钩子：对列数组做显隐过滤与排序，对普通 Table 与
 * ProTable 一致适用；偏好按「表格标识 + 用户 + 组织」持久化到浏览器，
 * 写入失败时明确报错并保持弹窗打开，不静默降级。
 */
export function useColumnSettings<C = Record<string, unknown>>(
  options: UseColumnSettingsOptions,
): UseColumnSettingsResult<C> {
  const { message } = App.useApp();
  const {
    tableKey,
    columns,
    scope,
    structuralKeys,
    lockVisibleKeys,
    defaultHiddenKeys,
    titleOverrides,
    disabled,
    disabledReason,
    title,
    advanced,
    persist,
  } = options;

  const [open, setOpen] = useState(false);
  const [value, setValue] = useState<ColumnSettingsValue | null>(null);

  const fields = useMemo(() => deriveFields(columns, { structuralKeys, lockVisibleKeys, titleOverrides }), [columns, structuralKeys, lockVisibleKeys, titleOverrides]);
  const defaultValue = useMemo(
    () => defaultColumnSettingsValue(fields, defaultHiddenKeys),
    [fields, defaultHiddenKeys],
  );

  // 字段集合（key + 约束 + 固定区域）或隔离范围变化时重新解析偏好，
  // 用签名字符串做依赖，避免调用方每次渲染传入新数组引用导致循环。
  const fieldsSignature = fields
    .map((field) => `${field.key}:${field.lockVisible ? 1 : 0}:${field.fixed ?? ''}`)
    .join('|');
  const hiddenSignature = (defaultHiddenKeys ?? []).join('|');
  const persistFlag = persist !== false;

  useEffect(() => {
    const stored = persistFlag
      ? loadColumnSettingsPreference(tableKey, scope ?? {})
      : null;
    setValue(
      resolveColumnSettingsValue(
        // 签名一致时字段内容等价，仅引用不同
        fields,
        defaultColumnSettingsValue(fields, defaultHiddenKeys),
        stored,
      ),
    );
  }, [fieldsSignature, hiddenSignature, tableKey, scope?.userId, scope?.organizationId, persistFlag]);

  const effectiveColumns = useMemo(() => {
    if (!value) return columns as C[];
    return applyColumnSettings<C>(columns, value, structuralKeys);
  }, [columns, value, structuralKeys]);

  const handleSave = (next: ColumnSettingsValue) => {
    setValue(next);
    if (!persistFlag) {
      setOpen(false);
      return;
    }
    const persisted = saveColumnSettingsPreference(tableKey, scope ?? {}, next);
    if (!persisted) {
      // 存储失败：表格已应用本次设置，保持弹窗打开便于重试或取消。
      message.error('列设置保存到本地浏览器失败，本次设置仅当前页面生效');
      return;
    }
    setOpen(false);
  };

  const entry = (
    <ColumnSettingsEntry
      onClick={() => setOpen(true)}
      disabled={disabled}
      disabledReason={disabledReason}
    />
  );

  const modal = (
    <ColumnSettingsModal
      open={open}
      fields={fields}
      value={value ?? defaultValue}
      defaultValue={defaultValue}
      title={title}
      advanced={advanced}
      onCancel={() => setOpen(false)}
      onSave={handleSave}
    />
  );

  return { columns: effectiveColumns, entry, modal };
}
