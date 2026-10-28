import {
  CopyOutlined,
  ExportOutlined,
  FileDoneOutlined,
  FilterOutlined,
  PlusOutlined,
  ReloadOutlined,
  TagOutlined,
} from '@ant-design/icons';
import { Button, Space, Switch, Tooltip, Typography } from 'antd';
import React from 'react';
import type { OrderListItem } from './types';

const { Text } = Typography;

export interface OrderListToolbarProps {
  orderKindTitle?: string;
  selectedRows: OrderListItem[];
  onRefresh: () => void;
  onCreateOrder?: () => void;
  onCopyOrder?: (selectedRows: OrderListItem[]) => void;
  onExportDocuments?: (selectedRows: OrderListItem[]) => void;
  onManageTags?: (selectedRows: OrderListItem[]) => void;
  onExportTable?: () => void;
  filterVisible?: boolean;
  onToggleFilter?: () => void;
  readonly?: boolean;
  autoRefresh?: boolean;
  onAutoRefreshChange?: (enabled: boolean) => void;
}

export function OrderListToolbar({
  orderKindTitle = '订单',
  selectedRows,
  onRefresh,
  onCreateOrder,
  onCopyOrder,
  onExportDocuments,
  onManageTags,
  onExportTable,
  filterVisible = false,
  onToggleFilter,
  readonly = false,
  autoRefresh = false,
  onAutoRefreshChange,
}: OrderListToolbarProps) {
  const hasSelected = selectedRows.length > 0;

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--roncin-page-section-gap)',
        flexWrap: 'wrap',
        gap: 8,
      }}
    >
      {/* 左侧操作区 */}
      <Space size="middle" wrap>
        {!readonly && onCreateOrder && (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={onCreateOrder}
            style={{ fontWeight: 500 }}
          >
            新增{orderKindTitle}
          </Button>
        )}

        {!readonly && onManageTags && (
          <Button
            icon={<TagOutlined />}
            disabled={!hasSelected}
            onClick={() => onManageTags(selectedRows)}
          >
            批量维护标签
          </Button>
        )}

        {!readonly && onCopyOrder && (
          <Button
            icon={<CopyOutlined />}
            disabled={!hasSelected}
            onClick={() => onCopyOrder(selectedRows)}
          >
            复制订单
          </Button>
        )}

        {onExportDocuments && (
          <Button
            icon={<FileDoneOutlined />}
            disabled={!hasSelected}
            onClick={() => onExportDocuments(selectedRows)}
          >
            导出单证
          </Button>
        )}

        <Space size="small">
          <Button icon={<ReloadOutlined />} onClick={onRefresh}>
            刷新
          </Button>
          <Tooltip title="开启后每 60 秒自动静默刷新表格数据">
            <span style={{ fontSize: 12, color: '#8c8c8c', marginLeft: 4 }}>
              自动刷新
            </span>
            <Switch
              size="small"
              checked={autoRefresh}
              aria-label="自动刷新订单列表"
              onChange={onAutoRefreshChange}
              style={{ marginLeft: 4 }}
            />
          </Tooltip>
        </Space>

        {hasSelected && (
          <Text type="secondary" style={{ fontSize: 13 }}>
            已选择{' '}
            <Text strong style={{ color: '#1677ff' }}>
              {selectedRows.length}
            </Text>{' '}
            项
          </Text>
        )}
      </Space>

      {/* 右侧工具区 */}
      <Space size="small">
        {onToggleFilter && (
          <Button
            icon={<FilterOutlined />}
            onClick={onToggleFilter}
            style={
              filterVisible
                ? { borderColor: '#1677ff', color: '#1677ff' }
                : undefined
            }
          >
            {filterVisible ? '收起筛选' : '展开筛选'}
          </Button>
        )}
        {onExportTable && (
          <Button icon={<ExportOutlined />} onClick={onExportTable}>
            导出表格
          </Button>
        )}
      </Space>
    </div>
  );
}

export default OrderListToolbar;
