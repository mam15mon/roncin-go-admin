import {
  EditableProTable,
  type EditableProTableProps,
  type ParamsType,
  ProTable,
  type ProTableProps,
} from '@ant-design/pro-components';
import { Table, type TableProps, type TableRef } from 'antd';
import React, { forwardRef } from 'react';
import type { LayoutColumn } from './layout';
import { useResponsiveTableLayout } from './useResponsiveTableLayout';

// 列显隐、顺序与 fixed 已由公共设置和响应式层应用。空的受控 map 使
// ProComponents 每次转换都使用当前列，避免首次挂载的内部 fixed 缓存覆盖。
const columnsStateFromProps = { value: {} };

/** Pro 内部按 left/right 再排序；官方逻辑固定位置可保留输入锚点与结构列顺序。 */
function logicalFixedColumns<C extends LayoutColumn>(
  columns: C[] | undefined,
): C[] | undefined {
  return columns?.map((column) => {
    if (Object.keys(column).length === 0) return column;
    return {
      ...column,
      fixed:
        column.fixed === 'left'
          ? 'start'
          : column.fixed === 'right'
            ? 'end'
            : column.fixed,
      ...(column.children
        ? { children: logicalFixedColumns([...column.children]) }
        : {}),
    };
  });
}

function useLayout<T extends object, C extends LayoutColumn>(
  props: Pick<TableProps<T>, 'scroll' | 'sticky' | 'expandable'> & {
    columns?: C[];
    rowSelection?: TableProps<T>['rowSelection'] | false;
  },
) {
  const rowSelection = props.rowSelection || undefined;
  const layout = useResponsiveTableLayout({
    columns: props.columns,
    scroll: props.scroll,
    sticky: props.sticky,
    auxiliaryWidth:
      (rowSelection ? Number(rowSelection.columnWidth) || 32 : 0) +
      (props.expandable?.expandedRowRender || props.expandable?.rowExpandable
        ? Number(props.expandable.columnWidth) || 48
        : 0),
    auxiliaryFixedWidth:
      (rowSelection?.fixed ? Number(rowSelection.columnWidth) || 32 : 0) +
      (props.expandable?.fixed ? 48 : 0),
  });
  return {
    containerRef: layout.containerRef,
    tableProps: {
      columns: layout.columns,
      scroll: layout.scroll,
      sticky: layout.sticky,
      rowSelection:
        props.rowSelection && layout.releaseFixed
          ? { ...props.rowSelection, fixed: false }
          : props.rowSelection,
      expandable:
        props.expandable && layout.releaseFixed
          ? { ...props.expandable, fixed: false }
          : props.expandable,
    },
  };
}

function PlainTable<T extends object>(
  props: TableProps<T>,
  ref: React.ForwardedRef<TableRef>,
) {
  const { containerRef, tableProps } = useLayout(props);
  return (
    <div ref={containerRef} className="roncin-responsive-table">
      <Table<T>
        {...props}
        {...tableProps}
        rowSelection={tableProps.rowSelection || undefined}
        ref={ref}
      />
    </div>
  );
}

/** 保留官方泛型调用签名和公开静态组件，ref 仍指向实际 antd Table。 */
export const ResponsiveTable = forwardRef(
  PlainTable,
) as unknown as typeof Table;
for (const key of [
  'SELECTION_COLUMN',
  'EXPAND_COLUMN',
  'SELECTION_ALL',
  'SELECTION_INVERT',
  'SELECTION_NONE',
  'Column',
  'ColumnGroup',
  'Summary',
] as const) {
  Object.defineProperty(ResponsiveTable, key, { get: () => Table[key] });
}

function ResponsiveProTableComponent<
  T extends object,
  U extends ParamsType = ParamsType,
  ValueType = 'text',
>(props: ProTableProps<T, U, ValueType>) {
  const { containerRef, tableProps } = useLayout(props);
  // Pro 列在布局映射时仅变更 fixed，完整保留其 render/valueType 等字段。
  const columns = logicalFixedColumns(
    tableProps.columns,
  ) as typeof props.columns;
  return (
    <div ref={containerRef} className="roncin-responsive-table">
      <ProTable<T, U, ValueType>
        {...props}
        {...tableProps}
        columns={columns}
        columnsState={props.columnsState ?? columnsStateFromProps}
      />
    </div>
  );
}

export const ResponsiveProTable = Object.defineProperty(
  ResponsiveProTableComponent,
  'Summary',
  { get: () => ProTable.Summary },
) as typeof ResponsiveProTableComponent & Pick<typeof ProTable, 'Summary'>;

export function ResponsiveEditableProTable<
  T extends object,
  U extends ParamsType = ParamsType,
  ValueType = 'text',
>(props: EditableProTableProps<T, U, ValueType>) {
  const { containerRef, tableProps } = useLayout(props);
  const columns = logicalFixedColumns(
    tableProps.columns,
  ) as typeof props.columns;
  return (
    <div ref={containerRef} className="roncin-responsive-table">
      <EditableProTable<T, U, ValueType>
        {...props}
        {...tableProps}
        columns={columns}
        columnsState={props.columnsState ?? columnsStateFromProps}
      />
    </div>
  );
}
