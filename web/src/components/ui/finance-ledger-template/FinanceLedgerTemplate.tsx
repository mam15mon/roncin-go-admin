import {
  CloudUploadOutlined,
  DownOutlined,
  DownloadOutlined,
  FileDoneOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import {
  type ActionType,
  ProTable,
} from '@ant-design/pro-components';

type DensitySize = 'middle' | 'small' | 'large' | undefined;
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
import React, { useRef, useState } from 'react';
import { FinanceSummaryBoard } from './FinanceSummaryBoard';
import type {
  FinanceLedgerGlobalSummary,
  FinanceLedgerSummaryItem,
  FinanceLedgerTemplateProps,
} from './types';

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
        columns={columns}
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
