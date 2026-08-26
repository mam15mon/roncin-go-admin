import {
  CopyOutlined,
  DeleteOutlined,
  DownloadOutlined,
  DownOutlined,
  ExportOutlined,
  FileDoneOutlined,
  LockOutlined,
  PlusOutlined,
  ReloadOutlined,
  ShareAltOutlined,
  TagOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import {
  Button,
  Dropdown,
  Popconfirm,
  Space,
  Switch,
  Tooltip,
  Typography,
} from 'antd';
import React, { useState } from 'react';
import type { BatchActionKey, OrderListItem } from './types';

const { Text } = Typography;

export interface OrderListToolbarProps {
  orderKindTitle?: string;
  selectedRows: OrderListItem[];
  onRefresh: () => void;
  onCreateOrder?: () => void;
  onCopyOrder?: (selectedRows: OrderListItem[]) => void;
  onExportDocuments?: (selectedRows: OrderListItem[]) => void;
  onBatchAction?: (actionKey: BatchActionKey, selectedRows: OrderListItem[]) => void;
  onExportTable?: () => void;
  readonly?: boolean;
}

export function OrderListToolbar({
  orderKindTitle = '订单',
  selectedRows,
  onRefresh,
  onCreateOrder,
  onCopyOrder,
  onExportDocuments,
  onBatchAction,
  onExportTable,
  readonly = false,
}: OrderListToolbarProps) {
  const [autoRefresh, setAutoRefresh] = useState(false);
  const hasSelected = selectedRows.length > 0;

  // 批量操作下拉菜单项定义
  const batchMenuItems: MenuProps['items'] = [
    {
      key: 'export-documents',
      icon: <FileDoneOutlined />,
      label: '批量导出单证',
      disabled: !hasSelected,
    },
    {
      type: 'divider',
    },
    {
      key: 'batch-collab',
      icon: <TeamOutlined />,
      label: '批量协同 / 取消协同',
      disabled: !hasSelected,
    },
    {
      key: 'share-orders',
      icon: <ShareAltOutlined />,
      label: '分享订单',
      disabled: !hasSelected,
    },
    {
      key: 'finish-orders',
      label: '完结 / 取消完结',
      disabled: !hasSelected,
    },
    {
      key: 'surrender',
      label: '退关 / 取消退关',
      disabled: !hasSelected,
    },
    {
      key: 'archive',
      label: '订单归档',
      disabled: !hasSelected,
    },
    {
      type: 'divider',
    },
    {
      key: 'lock-unlock',
      icon: <LockOutlined />,
      label: '锁定 / 解锁订单',
      disabled: !hasSelected,
    },
    {
      key: 'modify-lock-time',
      label: '修改锁定时间',
      disabled: !hasSelected,
    },
    {
      key: 'interest-to-receivable',
      label: '利息带入应收',
      disabled: !hasSelected,
    },
    {
      key: 'batch-modify-fields',
      label: '批量修改字段',
      disabled: !hasSelected,
    },
    {
      key: 'manage-tags',
      icon: <TagOutlined />,
      label: '标签管理',
      disabled: !hasSelected,
    },
    {
      key: 'batch-import',
      icon: <DownloadOutlined />,
      label: '批量导入订单',
    },
    {
      type: 'divider',
    },
    {
      key: 'batch-delete',
      icon: <DeleteOutlined />,
      label: '批量删除',
      danger: true,
      disabled: !hasSelected,
    },
  ];

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (onBatchAction) {
      onBatchAction(key as BatchActionKey, selectedRows);
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 12,
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

        <Dropdown menu={{ items: batchMenuItems, onClick: handleMenuClick }} disabled={readonly}>
          <Button>
            批量操作 <DownOutlined />
          </Button>
        </Dropdown>

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
              onChange={setAutoRefresh}
              style={{ marginLeft: 4 }}
            />
          </Tooltip>
        </Space>

        {hasSelected && (
          <Text type="secondary" style={{ fontSize: 13 }}>
            已选择 <Text strong style={{ color: '#1677ff' }}>{selectedRows.length}</Text> 项
          </Text>
        )}
      </Space>

      {/* 右侧工具区 */}
      <Space size="small">
        {onExportTable && (
          <Button icon={<ExportOutlined />} onClick={onExportTable}>
            导出表格
          </Button>
        )}
        {!readonly && (
          <Popconfirm
            title="确定批量删除选中的订单？"
            disabled={!hasSelected}
            onConfirm={() => onBatchAction?.('batch-delete', selectedRows)}
          >
            <Button danger icon={<DeleteOutlined />} disabled={!hasSelected}>
              批量删除
            </Button>
          </Popconfirm>
        )}
      </Space>
    </div>
  );
}

export default OrderListToolbar;
