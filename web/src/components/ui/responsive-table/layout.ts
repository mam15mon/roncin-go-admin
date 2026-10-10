/** 仅读取布局字段，兼容普通列、Pro 列和分组列。 */
export interface LayoutColumn {
  width?: number | string;
  fixed?: boolean | 'left' | 'right' | 'start' | 'end';
  hidden?: boolean;
  hideInTable?: boolean;
  children?: readonly LayoutColumn[];
}

function pixelWidth(width: LayoutColumn['width'], defaultWidth: number) {
  if (typeof width === 'number') return width;
  if (typeof width === 'string' && /^\d+(\.\d+)?(px)?$/.test(width)) {
    return Number.parseFloat(width);
  }
  return defaultWidth;
}

/** 没有列宽的普通列预留阅读空间，避免 max-content 把可换行内容拉成超宽单行。 */
export function visibleColumnsWidth(
  columns: readonly LayoutColumn[],
): number | 'max-content' {
  let total = 0;
  for (const column of columns) {
    if (Object.keys(column).length === 0) continue;
    if (column.hidden || column.hideInTable) continue;
    if (column.children) {
      const width = visibleColumnsWidth(column.children);
      if (width === 'max-content') return width;
      total += width;
    } else if (
      typeof column.width === 'string' &&
      !/^\d+(\.\d+)?(px)?$/.test(column.width)
    ) {
      return 'max-content';
    } else {
      total += pixelWidth(column.width, 160);
    }
  }
  return total;
}

function fixedWidth(
  columns: readonly LayoutColumn[],
  inherited = false,
): number {
  return columns.reduce((total, column) => {
    if (column.hidden || column.hideInTable) return total;
    const fixed = inherited || Boolean(column.fixed);
    return (
      total +
      (column.children
        ? fixedWidth(column.children, fixed)
        : fixed
          ? pixelWidth(column.width, 160)
          : 0)
    );
  }, 0);
}

/** 固定区之外至少保留一列阅读空间；尚未测量时先允许横滚访问全部列。 */
export function shouldReleaseFixedColumns(
  columns: readonly LayoutColumn[],
  containerWidth: number,
  auxiliaryFixedWidth = 0,
): boolean {
  const occupiedWidth = fixedWidth(columns) + auxiliaryFixedWidth;
  return (
    containerWidth === 0 ||
    containerWidth < 640 ||
    occupiedWidth > containerWidth - 240 ||
    occupiedWidth > containerWidth * 0.6
  );
}

export function releaseFixedColumns<C extends LayoutColumn>(
  columns: readonly C[],
): C[] {
  return columns.map((column) => {
    // 官方选择/展开哨兵为空对象，必须保留对象身份才能让 Table 识别。
    if (Object.keys(column).length === 0) return column;
    return {
      ...column,
      fixed: false,
      ...(column.children
        ? { children: releaseFixedColumns(column.children) }
        : {}),
    };
  });
}
