import { Table, type TableProps } from 'antd';
import type { ColumnType } from 'antd/es/table';
import React, { useCallback, useMemo, useRef, useState } from 'react';
import type { LayoutColumn } from './layout';
import {
  clampColumnWidth,
  ResizableHeaderCell,
  type ResizableHeaderCellProps,
} from './ResizableHeaderCell';

function initialPixelWidth(width: LayoutColumn['width']) {
  if (typeof width === 'number' && Number.isFinite(width) && width > 0)
    return width;
  if (typeof width === 'string' && /^\d+(\.\d+)?(px)?$/.test(width))
    return Number.parseFloat(width);
  return undefined;
}

function columnIdentity<T extends object>(column: ColumnType<T>) {
  const key = column.key ?? column.dataIndex;
  return key === undefined || key === null || key === ''
    ? undefined
    : JSON.stringify(key);
}

/** 调宽状态属于当前表格实例，以稳定字段身份匹配显隐和重排后的列。 */
export function useResizableColumns<T extends object, C extends LayoutColumn>(
  columns: C[] | undefined,
  components: TableProps<T>['components'],
) {
  const [widths, setWidths] = useState<Record<string, number>>({});
  const measuredInitialWidths = useRef(new Map<string, number>());
  const measuredWidths = useRef<Record<string, number>>({});
  const measure = useCallback(
    (key: string, width: number, cell: HTMLElement) => {
      if (!measuredInitialWidths.current.has(key))
        measuredInitialWidths.current.set(key, width);
      const wrapper = cell.closest('.ant-table-wrapper');
      const snapshot: Record<string, number> = {};
      wrapper?.querySelectorAll<HTMLElement>('thead th').forEach((header) => {
        if (header.closest('.ant-table-wrapper') !== wrapper) return;
        const identity =
          header.dataset.columnResizeKey ??
          (header.classList.contains('ant-table-selection-column')
            ? '@selection'
            : header.classList.contains('ant-table-row-expand-icon-cell')
              ? '@expand'
              : undefined);
        const renderedWidth = header.getBoundingClientRect().width;
        if (identity && renderedWidth > 0) snapshot[identity] = renderedWidth;
      });
      measuredWidths.current = snapshot;
    },
    [],
  );
  const resize = useCallback((key: string, width: number) => {
    setWidths((previous) => ({
      ...previous,
      ...measuredWidths.current,
      [key]: width,
    }));
  }, []);
  const autoFit = useCallback((key: string, initialWidth: number) => {
    const baseline = clampColumnWidth(initialWidth);
    const expandedWidth = clampColumnWidth(Math.max(baseline * 1.6, 240));
    setWidths((previous) => ({
      ...previous,
      ...measuredWidths.current,
      [key]:
        (previous[key] ?? baseline) >= expandedWidth - 10
          ? baseline
          : expandedWidth,
    }));
  }, []);

  const resizableColumns = useMemo(() => {
    const mapColumns = (input: readonly C[]): C[] =>
      input.map((column) => {
        if (column === Table.SELECTION_COLUMN || column === Table.EXPAND_COLUMN)
          return column;
        if (column.children)
          return { ...column, children: mapColumns(column.children as C[]) };
        const businessColumn = column as C &
          ColumnType<T> & { valueType?: unknown };
        const key = columnIdentity(businessColumn);
        if (!key) return column;
        const structural =
          businessColumn.valueType === 'index' ||
          businessColumn.valueType === 'indexBorder' ||
          businessColumn.key === 'index' ||
          businessColumn.dataIndex === 'index';
        const initialWidth = initialPixelWidth(column.width);
        return {
          ...column,
          ...(widths[key] === undefined ? {} : { width: widths[key] }),
          onHeaderCell: (
            headerColumn: Parameters<
              NonNullable<ColumnType<T>['onHeaderCell']>
            >[0],
          ) => ({
            ...businessColumn.onHeaderCell?.(headerColumn),
            'data-column-resize-key': key,
            ...(structural
              ? {}
              : {
                  columnResize: {
                    fallbackWidth: widths[key] ?? initialWidth ?? 160,
                    onResize: (width: number) => resize(key, width),
                    onMeasure: (width: number, cell: HTMLElement) =>
                      measure(key, width, cell),
                    onAutoFit: (measuredWidth: number) => {
                      if (!measuredInitialWidths.current.has(key))
                        measuredInitialWidths.current.set(key, measuredWidth);
                      autoFit(
                        key,
                        initialWidth ??
                          measuredInitialWidths.current.get(key) ??
                          measuredWidth,
                      );
                    },
                  },
                }),
          }),
        };
      });
    return columns && mapColumns(columns);
  }, [columns, widths, resize, autoFit, measure]);

  const customCell = components?.header?.cell;
  // 组件类型不随调宽状态变化，避免重挂表头或可编辑内容。
  const HeaderCell = useMemo(() => {
    if (!customCell) return ResizableHeaderCell;
    return function ComposedHeaderCell(props: ResizableHeaderCellProps) {
      return <ResizableHeaderCell {...props} component={customCell} />;
    };
  }, [customCell]);
  const resizableComponents = useMemo(
    () => ({
      ...components,
      header: { ...components?.header, cell: HeaderCell },
    }),
    [components, HeaderCell],
  );
  return {
    columns: resizableColumns,
    components: resizableComponents,
    resized: Object.keys(widths).length > 0,
    selectionWidth: widths['@selection'],
    expandWidth: widths['@expand'],
  };
}
