import { App } from 'antd';
import React, { useContext, useMemo, useRef, useState } from 'react';
import { appInitialStateContext } from '@/app/initialStateContext';
import { ColumnSettingsEntry } from './ColumnSettingsEntry';
import {
  applyColumnFreeze,
  deriveColumnFreezeDefaults,
  getColumnFreezeTargets,
} from './freeze';
import {
  defaultColumnSettingsValue,
  loadColumnSettingsPreference,
  resolveColumnSettingsValue,
  saveColumnSettingsPreference,
} from './preference';
import type {
  ColumnSettingsField,
  ColumnSettingsPreference,
  ColumnSettingsScope,
  ColumnSettingsValue,
} from './types';

/** 钩子内部读取列定义所需的最小形状，兼容 antd Table 与 ProTable 列。 */
interface ColumnLike {
  key?: React.Key;
  dataIndex?: string | string[];
  title?: React.ReactNode;
  fixed?: 'left' | 'right' | boolean;
  valueType?: unknown;
  hideInTable?: boolean;
}

export interface UseColumnSettingsOptions {
  /** 稳定表格标识：按业务视图命名（如 `orders:list`），不使用实体 ID。 */
  tableKey: string;
  /** 基准列定义（应用显隐排序前）。 */
  columns: unknown[];
  /** 偏好隔离范围（用户 + 组织）；缺省时读取当前认证上下文。 */
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
  /**
   * 增强版高级设置内容（如行配色）：提供时浮层底部出现「更多设置」，
   * 点击打开二级弹窗展示该内容；内容状态由调用方自行维护与持久化。
   */
  advanced?: React.ReactNode;
  /** 高级设置二级弹窗标题，默认「高级设置」。 */
  advancedTitle?: string;
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

/** 沿用公共模板的稳定结构标识与 ProTable 官方类型，不根据列名猜测。 */
function isStructuralColumn(
  column: ColumnLike,
  key: string | null,
  structural: Set<string>,
): boolean {
  return (
    !key ||
    structural.has(key) ||
    column.hideInTable === true ||
    column.valueType === 'option' ||
    column.valueType === 'index' ||
    column.valueType === 'indexBorder' ||
    key === 'index' ||
    key === 'action' ||
    key === 'option'
  );
}

/** 派生设置元数据：可命名、可隐藏的普通列进入设置，其余按结构列处理。 */
function deriveFields(
  columns: unknown[],
  options: Pick<
    UseColumnSettingsOptions,
    'structuralKeys' | 'lockVisibleKeys' | 'titleOverrides'
  >,
): ColumnSettingsField[] {
  const structural = new Set(options.structuralKeys ?? []);
  const locked = new Set(options.lockVisibleKeys ?? []);
  const overrides = options.titleOverrides ?? {};
  const fields: ColumnSettingsField[] = [];
  columns.forEach((raw, index) => {
    const column = raw as ColumnLike;
    const key = columnKeyOf(column, index);
    if (isStructuralColumn(column, key, structural) || !key) return;
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
 * 原始相对锚点（如操作列保持在最后）。首尾冻结在排序完成后统一应用。
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
    const isStructural = isStructuralColumn(column, key, structural);
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
  /** 统一列设置入口按钮（含锚定浮层）；放入工具栏或表格上方右侧。 */
  entry: React.ReactNode;
}

/**
 * 全站统一列设置钩子：对列数组做显隐过滤与排序，对普通 Table 与
 * ProTable 一致适用。浮层内每次显隐、排序操作即时生效并自动持久化到
 * 浏览器本地（按「表格标识 + 用户 + 组织」隔离）；写入失败时明确提示
 * 且页面内设置保持生效，不静默降级。
 */
export function useColumnSettings<C = Record<string, unknown>>(
  options: UseColumnSettingsOptions,
): UseColumnSettingsResult<C> {
  const { message } = App.useApp();
  const {
    tableKey,
    columns,
    scope: explicitScope,
    structuralKeys,
    lockVisibleKeys,
    defaultHiddenKeys,
    titleOverrides,
    disabled,
    disabledReason,
    advanced,
    advancedTitle,
  } = options;

  const initialState = useContext(appInitialStateContext)?.initialState;
  const scope = explicitScope ?? {
    userId: initialState?.currentUser?.id,
    organizationId: initialState?.currentUser?.currentOrganization?.id,
  };
  // 存储连续失败时只提示一次，写入成功后复位，避免连续操作刷屏。
  const persistFailedRef = useRef<string | null>(null);

  const fields = useMemo(
    () =>
      deriveFields(columns, {
        structuralKeys,
        lockVisibleKeys,
        titleOverrides,
      }),
    [columns, structuralKeys, lockVisibleKeys, titleOverrides],
  );
  const defaultValue = useMemo(
    () =>
      defaultColumnSettingsValue(
        fields,
        defaultHiddenKeys,
        deriveColumnFreezeDefaults(columns as ColumnLike[]),
      ),
    [columns, fields, defaultHiddenKeys],
  );

  // 字段集合（key + 约束 + 固定区域）或隔离范围变化时重新解析偏好，
  // 用签名字符串做依赖，避免调用方每次渲染传入新数组引用导致循环。
  const fieldsSignature = fields
    .map(
      (field) =>
        `${field.key}:${field.lockVisible ? 1 : 0}:${field.fixed ?? ''}`,
    )
    .join('|');
  const hiddenSignature = (defaultHiddenKeys ?? []).join('|');

  const identity = JSON.stringify([
    fieldsSignature,
    hiddenSignature,
    tableKey,
    scope.userId,
    scope.organizationId,
    defaultValue.freeze,
  ]);

  const [local, setLocal] = useState<{
    identity: string;
    value: ColumnSettingsPreference | null;
  }>(() => ({
    identity,
    value: loadColumnSettingsPreference(tableKey, scope),
  }));
  // 切换作用域的当前渲染直接读取新偏好，不等待 effect 清理旧组织配置。
  const storedValue = useMemo(
    () =>
      local.identity === identity
        ? local.value
        : loadColumnSettingsPreference(tableKey, scope),
    [identity, local],
  );
  const value = useMemo(
    () => resolveColumnSettingsValue(fields, defaultValue, storedValue),
    [fields, defaultValue, storedValue],
  );

  const effectiveColumns = useMemo(() => {
    const ordered = applyColumnSettings<ColumnLike>(
      columns,
      value,
      structuralKeys,
    );
    return applyColumnFreeze(
      ordered,
      value.freeze ?? deriveColumnFreezeDefaults(columns as ColumnLike[]),
      {
        firstExcludedKeys: structuralKeys,
      },
    ) as C[];
  }, [columns, value, structuralKeys]);

  const targets = getColumnFreezeTargets(effectiveColumns as ColumnLike[], {
    firstExcludedKeys: structuralKeys,
  });

  const handleValueChange = (next: ColumnSettingsValue) => {
    setLocal({ identity, value: next });
    const persisted = saveColumnSettingsPreference(tableKey, scope, next);
    if (!persisted) {
      // 存储失败：表格已应用本次设置，提示一次便于感知未持久化。
      if (persistFailedRef.current !== identity) {
        persistFailedRef.current = identity;
        message.error('列设置保存到本地浏览器失败，本次设置仅当前页面生效');
      }
      return;
    }
    persistFailedRef.current = null;
  };

  const handleReset = () => {
    handleValueChange(defaultValue);
  };

  // 包一层带 key 的 Fragment：entry 常被直接放进 toolBarRender 数组。
  const entry = (
    <React.Fragment key="column-settings-entry">
      <ColumnSettingsEntry
        key={identity}
        fields={fields}
        value={value}
        freezeTargets={{
          first: targets.first
            ? columnTitle(targets.first) ||
              titleOverrides?.[columnKeyOf(targets.first, 0) ?? ''] ||
              '未命名列'
            : undefined,
          last: targets.last
            ? columnTitle(targets.last) ||
              titleOverrides?.[columnKeyOf(targets.last, 0) ?? ''] ||
              '未命名列'
            : undefined,
        }}
        onChange={handleValueChange}
        onReset={handleReset}
        advanced={advanced}
        advancedTitle={advancedTitle}
        disabled={disabled}
        disabledReason={disabledReason}
      />
    </React.Fragment>
  );

  return { columns: effectiveColumns, entry };
}
