import {
  CloudUploadOutlined,
  DownloadOutlined,
  DownOutlined,
  FileDoneOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { type ActionType, ProTable } from '@ant-design/pro-components';
import {
  App,
  Button,
  Card,
  Col,
  Dropdown,
  Row,
  Statistic,
  Tooltip,
} from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { FinanceSummaryBoard } from './FinanceSummaryBoard';
import type {
  FinanceLedgerGlobalSummary,
  FinanceLedgerSummaryItem,
  FinanceLedgerTemplateProps,
} from './types';

type DensitySize = 'middle' | 'small' | 'large' | undefined;

interface ResizableHeaderCellProps
  extends React.HTMLAttributes<HTMLTableHeaderCellElement> {
  onResize?: (width: number) => void;
  onAutoFit?: () => void;
  width?: number;
  resizable?: boolean;
}

const MIN_COLUMN_WIDTH = 50;
const MAX_COLUMN_WIDTH = 600;

export const ResizableHeaderCell: React.FC<ResizableHeaderCellProps> = ({
  onResize,
  onAutoFit,
  width,
  resizable = true,
  children,
  style,
  className,
  ...restProps
}) => {
  const [isHovered, setIsHovered] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const headerCellRef = useRef<HTMLTableCellElement>(null);
  const dragCleanupRef = useRef<(() => void) | undefined>(undefined);

  useEffect(() => {
    return () => {
      dragCleanupRef.current?.();
    };
  }, []);

  if (!resizable || !width || !onResize) {
    return (
      <th style={style} className={className} {...restProps}>
        {children}
      </th>
    );
  }

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    const headerCell = headerCellRef.current;
    if (!headerCell) return;

    setIsDragging(true);

    const cellRect = headerCell.getBoundingClientRect();
    const tableWrapper =
      headerCell.closest('.ant-table-wrapper') ||
      headerCell.closest('.ant-pro-table') ||
      document.body;
    const tableRect = tableWrapper.getBoundingClientRect();

    const startX = e.clientX;
    const startWidth = width;
    let targetWidth = startWidth;

    const previousCursor = document.body.style.cursor;
    const previousUserSelect = document.body.style.userSelect;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';

    // 方案 3：创建 Excel / 飞书 垂直辅助高亮虚线与实时像素尺寸提示浮层
    const guideLine = document.createElement('div');
    guideLine.className = 'roncin-table-resizer-guide';
    guideLine.style.cssText = `
      position: fixed;
      top: ${tableRect.top}px;
      left: ${cellRect.right}px;
      width: 2px;
      height: ${tableRect.height}px;
      background: #1677ff;
      box-shadow: 0 0 6px rgba(22, 119, 255, 0.6);
      z-index: 99999;
      pointer-events: none;
      will-change: transform;
      transform: translateX(0px);
    `;

    const tooltip = document.createElement('div');
    tooltip.style.cssText = `
      position: absolute;
      top: 4px;
      left: 6px;
      background: rgba(0, 0, 0, 0.78);
      color: #fff;
      font-size: 11px;
      font-family: monospace;
      font-weight: 500;
      padding: 2px 6px;
      border-radius: 3px;
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.2);
      white-space: nowrap;
      pointer-events: none;
    `;
    tooltip.textContent = `宽度: ${startWidth}px`;
    guideLine.appendChild(tooltip);
    document.body.appendChild(guideLine);

    let animationFrameId: number | undefined;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaX = moveEvent.clientX - startX;
      targetWidth = Math.max(
        MIN_COLUMN_WIDTH,
        Math.min(MAX_COLUMN_WIDTH, startWidth + deltaX),
      );
      const clampedDeltaX = targetWidth - startWidth;

      if (animationFrameId !== undefined) {
        window.cancelAnimationFrame(animationFrameId);
      }

      animationFrameId = window.requestAnimationFrame(() => {
        guideLine.style.transform = `translateX(${clampedDeltaX}px)`;
        tooltip.textContent = `宽度: ${targetWidth}px`;
      });
    };

    const cleanup = () => {
      if (animationFrameId !== undefined) {
        window.cancelAnimationFrame(animationFrameId);
        animationFrameId = undefined;
      }
      if (guideLine.parentNode) {
        guideLine.parentNode.removeChild(guideLine);
      }
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousUserSelect;
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      dragCleanupRef.current = undefined;
      setIsDragging(false);
    };

    const handleMouseUp = () => {
      cleanup();
      // 拖拽过程 0 次 React 重绘，仅在松手时提交 1 次 React 状态
      if (targetWidth !== startWidth) {
        onResize(targetWidth);
      }
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    dragCleanupRef.current = cleanup;
  };

  const handleDoubleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (onAutoFit) {
      onAutoFit();
    }
  };

  return (
    <th
      ref={headerCellRef}
      style={{
        ...style,
        position: 'relative',
        userSelect: isDragging ? 'none' : undefined,
      }}
      className={className}
      {...restProps}
    >
      {children}
      <div
        style={{
          position: 'absolute',
          right: -3,
          top: 0,
          bottom: 0,
          width: 7,
          cursor: 'col-resize',
          zIndex: 2,
          backgroundColor: isDragging
            ? '#1677ff'
            : isHovered
            ? 'rgba(22, 119, 255, 0.45)'
            : 'transparent',
          transition: 'background-color 0.15s',
        }}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
        onMouseDown={handleMouseDown}
        onDoubleClick={handleDoubleClick}
        data-column-resize-handle
        title="按住鼠标左右拖拽调节列宽；双击缝隙自动展开/还原"
      />
    </th>
  );
};

export function FinanceLedgerTemplate<
  T extends FinanceLedgerSummaryItem = FinanceLedgerSummaryItem,
>({
  headerTitle = '财务明细台账',
  columns,
  rowKey = 'id',
  scrollX = 'max-content',
  actionRef: externalActionRef,
  metricCards,
  primaryActionText = '创建账单',
  primaryActionIcon = <FileDoneOutlined />,
  onPrimaryAction,
  primaryActionRequiresSelection = false,
  batchActions = [],
  exportFileName = `财务明细导出_${new Date().toISOString().slice(0, 10)}.csv`,
  onExport,
  onImport,
  extraToolBarActions = [],
  request,
  showSummaryBoard = true,
  onOpenColumnConfig,
  rowColors,
  getRowStatusColorKey,
  onRowClick,
}: FinanceLedgerTemplateProps<T>) {
  const { message } = App.useApp();
  const internalActionRef = useRef<ActionType | undefined>(undefined);
  const actionRef = externalActionRef || internalActionRef;

  const [currentData, setCurrentData] = useState<T[]>([]);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [selectedRows, setSelectedRows] = useState<T[]>([]);
  const [globalSummary, setGlobalSummary] =
    useState<FinanceLedgerGlobalSummary>();
  const [densitySize, setDensitySize] = useState<DensitySize>('small');
  const [colWidths, setColWidths] = useState<Record<string, number>>({});

  const handleResize = useCallback((colKey: string, newWidth: number) => {
    setColWidths((prev) => ({ ...prev, [colKey]: newWidth }));
  }, []);

  const handleAutoFit = useCallback((colKey: string, initialWidth: number) => {
    setColWidths((prev) => {
      const current = prev[colKey] || initialWidth;
      const expandedWidth = Math.max(initialWidth * 1.6, 240);
      if (current >= expandedWidth - 10) {
        return { ...prev, [colKey]: initialWidth };
      }
      return { ...prev, [colKey]: expandedWidth };
    });
  }, []);

  // 为每个非固定列动态挂载拖拽拉伸与双击自适应属性
  const resizableColumns = React.useMemo(() => {
    return columns.map((col) => {
      const key = String(col.dataIndex || col.key || '');
      const isFixed = Boolean(col.fixed);
      const initialWidth = typeof col.width === 'number' ? col.width : 120;
      const currentWidth = colWidths[key] || initialWidth;

      if (!key || isFixed) {
        return col;
      }

      return {
        ...col,
        width: currentWidth,
        onHeaderCell: (column: any) => ({
          width: column.width,
          resizable: true,
          onResize: (newWidth: number) => handleResize(key, newWidth),
          onAutoFit: () => handleAutoFit(key, initialWidth),
        }),
      };
    });
  }, [columns, colWidths, handleAutoFit, handleResize]);

  // 默认导出 CSV 处理
  const handleDefaultExport = () => {
    const list = selectedRows.length > 0 ? selectedRows : currentData;
    if (list.length === 0) {
      message.warning('当前无数据可导出');
      return;
    }
    if (onExport) {
      onExport(selectedRows, currentData);
      return;
    }
    // 通用 CSV 导出逻辑
    const exportableCols = columns.filter(
      (col) =>
        col.dataIndex &&
        !col.hideInTable &&
        col.valueType !== 'option' &&
        col.valueType !== 'index',
    );
    const headers = exportableCols.map((col) =>
      String(col.title || col.dataIndex),
    );
    const rows = list.map((item: any) =>
      exportableCols.map((col) => {
        const key = String(col.dataIndex);
        const val = item[key];
        return `"${String(val ?? '').replace(/"/g, '""')}"`;
      }),
    );
    const csvContent =
      '\uFEFF' +
      [headers.join(','), ...rows.map((e) => e.join(','))].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute('download', exportFileName);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    message.success(`已导出 ${list.length} 条数据`);
  };

  return (
    <div
      style={{
        paddingBottom: 24,
        width: '100%',
        maxWidth: '100%',
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      {/* 1. 顶部宏观统计指标卡 */}
      {metricCards && metricCards.length > 0 && (
        <Row gutter={12} style={{ marginBottom: 12 }}>
          {metricCards.map((card) => (
            <Col
              key={card.key}
              span={Math.floor(24 / Math.min(metricCards.length, 6))}
            >
              <Card size="small">
                <Statistic
                  title={card.title}
                  value={card.value}
                  precision={card.precision}
                  suffix={card.suffix}
                  valueStyle={
                    card.valueColor ? { color: card.valueColor } : undefined
                  }
                />
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {/* 2. ProTable 宽表主体 */}
      <ProTable<T>
        headerTitle={headerTitle}
        actionRef={actionRef}
        rowKey={rowKey}
        columns={resizableColumns}
        components={{
          header: {
            cell: ResizableHeaderCell,
          },
        }}
        bordered
        size={densitySize}
        onSizeChange={(size) => setDensitySize(size || 'small')}
        scroll={{ x: scrollX }}
        pagination={{ defaultPageSize: 40, showSizeChanger: true }}
        onRow={(record) => {
          const style: React.CSSProperties = {};
          if (onRowClick) {
            style.cursor = 'pointer';
          }
          if (rowColors && getRowStatusColorKey) {
            const statusKey = getRowStatusColorKey(record);
            if (statusKey) {
              const bgColor = (rowColors as any)[statusKey];
              if (bgColor && bgColor !== '#FFFFFF') {
                style.backgroundColor = bgColor;
              }
            }
          }
          return {
            style,
            onClick: (event: React.MouseEvent) => {
              if (!onRowClick) return;
              const target = event.target as HTMLElement | null;
              if (
                target?.closest('input') ||
                target?.closest('button') ||
                target?.closest('a') ||
                target?.closest('.ant-table-selection-column') ||
                target?.closest('.ant-checkbox-wrapper') ||
                target?.closest('.ant-typography-copy')
              ) {
                return;
              }
              onRowClick(record, event);
            },
          };
        }}
        toolBarRender={() => [
          onPrimaryAction && (
            <Button
              key="primary-action"
              type="primary"
              icon={primaryActionIcon}
              disabled={
                primaryActionRequiresSelection && selectedRowKeys.length === 0
              }
              onClick={() => onPrimaryAction(selectedRowKeys, selectedRows)}
            >
              {primaryActionText}{' '}
              {primaryActionRequiresSelection && selectedRowKeys.length > 0
                ? `(${selectedRowKeys.length})`
                : ''}
            </Button>
          ),
          batchActions.length > 0 && (
            <Dropdown
              key="batch-actions"
              menu={{
                items: batchActions.map((act) => ({
                  key: act.key,
                  label: act.label,
                  disabled: act.disabled || selectedRowKeys.length === 0,
                  onClick: () => act.onClick(selectedRowKeys, selectedRows),
                })),
              }}
            >
              <Button>
                批量操作 <DownOutlined />
              </Button>
            </Dropdown>
          ),
          <Button
            key="export"
            icon={<DownloadOutlined />}
            onClick={handleDefaultExport}
          >
            导出清单
          </Button>,
          onImport && (
            <Tooltip
              key="import-tip"
              title="支持通过 Excel 标准模板批量导入数据"
            >
              <Button
                key="import"
                icon={<CloudUploadOutlined />}
                style={{
                  backgroundColor: '#faad14',
                  borderColor: '#faad14',
                  color: '#fff',
                }}
                onClick={onImport}
              >
                导入数据
              </Button>
            </Tooltip>
          ),
          onOpenColumnConfig && (
            <Tooltip key="col-config" title="列设置与表头排序">
              <Button
                type="text"
                icon={<SettingOutlined style={{ fontSize: 16, color: '#595959' }} />}
                onClick={onOpenColumnConfig}
                style={{ padding: '4px 6px' }}
              />
            </Tooltip>
          ),
          ...extraToolBarActions,
        ].filter(Boolean)}
        options={{
          fullScreen: true,
          reload: true,
          density: true,
          setting: false,
        }}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys, rows) => {
            setSelectedRowKeys(keys);
            setSelectedRows(rows);
          },
        }}
        request={async (params) => {
          const res = await request(params);
          const list = res.data || [];
          setCurrentData(list);
          setTotalCount(res.total || 0);
          if (res.summary) {
            setGlobalSummary(res.summary);
          }
          return {
            data: list,
            total: res.total || 0,
            success: res.success ?? true,
          };
        }}
      />

      {/* 3. 底部双层多币种动态汇总底栏 */}
      {showSummaryBoard && (
        <FinanceSummaryBoard
          selectedRows={selectedRows}
          allRows={currentData}
          totalCount={totalCount}
          globalSummary={globalSummary}
        />
      )}
    </div>
  );
}
