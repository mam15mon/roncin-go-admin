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
  const columns = tableProps.columns as typeof props.columns;
  return (
    <div ref={containerRef} className="roncin-responsive-table">
      <ProTable<T, U, ValueType> {...props} {...tableProps} columns={columns} />
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
  const columns = tableProps.columns as typeof props.columns;
  return (
    <div ref={containerRef} className="roncin-responsive-table">
      <EditableProTable<T, U, ValueType>
        {...props}
        {...tableProps}
        columns={columns}
      />
    </div>
  );
}
