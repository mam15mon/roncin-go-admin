import {
  CloudUploadOutlined,
  DownloadOutlined,
  DownOutlined,
  FileDoneOutlined,
} from '@ant-design/icons';
import {
  type ActionType,
  PageContainer,
  type ProColumns,
} from '@ant-design/pro-components';
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
import dayjs from 'dayjs';
import React, { useRef, useState } from 'react';
import { toTableRequest, unwrapPage } from '@/utils/api';
import { downloadCsv, serializeCsv } from '@/utils/csv';
import { useColumnSettings } from '../column-settings';
import { ResponsiveProTable } from '../responsive-table';
import { FinanceSummaryBoard } from './FinanceSummaryBoard';
import type {
  FinanceLedgerGlobalSummary,
  FinanceLedgerRequestParams,
  FinanceLedgerSummaryItem,
  FinanceLedgerTemplateProps,
} from './types';

type DensitySize = 'middle' | 'small' | 'large' | undefined;

export function FinanceLedgerTemplate<
  T extends FinanceLedgerSummaryItem = FinanceLedgerSummaryItem,
  TFilter extends Record<string, unknown> = Record<string, unknown>,
>({
  pagination,
  formRef,
  form,
  onSubmit,
  onReset,
  pageTitle,
  pageSubTitle,
  headerExtra,
  topBar,
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
  rowSelection,
  batchActions = [],
  exportFileName = `财务明细导出_${dayjs().format('YYYY-MM-DD')}.csv`,
  exportColumns,
  onImport,
  extraToolBarActions = [],
  request,
  showSummaryBoard = true,
  columnSettingsKey,
  columnSettingsStructuralKeys,
  advancedSettings,
  advancedSettingsTitle,
  rowColors,
  getRowStatusColorKey,
  onRowClick,
  search,
  customSearch,
}: FinanceLedgerTemplateProps<T, TFilter>) {
  const { message } = App.useApp();
  const internalActionRef = useRef<ActionType | undefined>(undefined);
  const actionRef = externalActionRef || internalActionRef;

  const [currentData, setCurrentData] = useState<T[]>([]);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [selectedRows, setSelectedRows] = useState<T[]>([]);
  const [globalSummary, setGlobalSummary] =
    useState<FinanceLedgerGlobalSummary>();
  const loadedSnapshots = useRef(
    new WeakMap<T[], { total: number; summary?: FinanceLedgerGlobalSummary }>(),
  );
  const [densitySize, setDensitySize] = useState<DensitySize>('small');
  // 统一列设置：表格标识由调用方按业务视图传入，不使用实体 ID。
  const enableColumnSettings = Boolean(columnSettingsKey);
  const columnSettings = useColumnSettings<ProColumns<T>>({
    tableKey: columnSettingsKey ?? 'finance-ledger:unset',
    columns,
    // 消费者的序号列（dataIndex: 'index'）属结构列，不进入设置。
    structuralKeys: ['index', ...(columnSettingsStructuralKeys ?? [])],
    advanced: advancedSettings,
    advancedTitle: advancedSettingsTitle,
  });
  const ledgerColumns = enableColumnSettings ? columnSettings.columns : columns;

  // 默认导出 CSV 处理
  const handleDefaultExport = () => {
    const list = selectedRows.length > 0 ? selectedRows : currentData;
    if (list.length === 0) {
      message.warning('当前无数据可导出');
      return;
    }
    const content = serializeCsv(list, exportColumns);
    if (!content) return;
    downloadCsv(content, exportFileName);
    message.success(
      `已导出${selectedRows.length > 0 ? '选中' : '当前页'} ${list.length} 条数据`,
    );
  };

  return (
    <PageContainer
      header={{
        title: pageTitle,
        subTitle: pageSubTitle,
        extra: headerExtra,
        breadcrumb: undefined,
      }}
      style={{
        minHeight: 'calc(100vh - var(--roncin-header-height))',
        backgroundColor: '#f5f7fa',
      }}
    >
      <div
        style={{
          paddingBottom: 24,
          width: '100%',
          maxWidth: '100%',
          overflow: 'hidden',
          position: 'relative',
        }}
      >
        {/* 0. 顶层全局筛选工具栏（如所属公司选择器、全局标签筛选） */}
        {topBar && (
          <Card
            size="small"
            style={{
              marginBottom: 'var(--roncin-page-section-gap)',
              borderRadius: 8,
              border: '1px solid #f0f0f0',
              backgroundColor: '#ffffff',
            }}
            styles={{ body: { padding: '10px 16px' } }}
          >
            {topBar}
          </Card>
        )}

        {/* 1. 顶部宏观统计指标卡 */}
        {metricCards && metricCards.length > 0 && (
          <Row
            gutter={12}
            style={{ marginBottom: 'var(--roncin-page-section-gap)' }}
          >
            {metricCards.map((card) => (
              <Col
                key={card.key}
                span={Math.floor(24 / Math.min(metricCards.length, 6))}
              >
                <Card
                  size="small"
                  style={{
                    borderRadius: 8,
                    border: '1px solid #f0f0f0',
                  }}
                >
                  <Statistic
                    title={card.title}
                    value={card.value}
                    precision={card.precision}
                    suffix={card.suffix}
                    styles={
                      card.valueColor
                        ? { content: { color: card.valueColor } }
                        : undefined
                    }
                  />
                </Card>
              </Col>
            ))}
          </Row>
        )}

        {/* 2. 自定义嵌入式搜索筛选栏（位于指标统计卡与表格台账之间） */}
        {customSearch}

        {/* 3. ProTable 宽表主体 */}
        <ResponsiveProTable<T>
          headerTitle={headerTitle}
          actionRef={actionRef}
          rowKey={rowKey}
          columns={ledgerColumns}
          cardProps={{
            style: {
              borderRadius: 8,
              border: '1px solid #f0f0f0',
            },
          }}
          size={densitySize}
          onSizeChange={(size) => setDensitySize(size || 'small')}
          scroll={{ x: scrollX }}
          pagination={
            pagination === false
              ? false
              : { defaultPageSize: 40, showSizeChanger: true, ...pagination }
          }
          formRef={formRef}
          form={form}
          onSubmit={(params) =>
            onSubmit?.(params as TFilter & FinanceLedgerRequestParams)
          }
          onReset={onReset}
          search={
            search === false
              ? false
              : {
                  labelWidth: 80,
                  defaultCollapsed: false,
                  searchText: '查询',
                  resetText: '重置',
                  span: { xs: 24, sm: 12, md: 8, lg: 6, xl: 6, xxl: 6 },
                  ...search,
                }
          }
          onRow={(record) => {
            const style: React.CSSProperties = {};
            if (onRowClick) {
              style.cursor = 'pointer';
            }
            if (rowColors && getRowStatusColorKey) {
              const statusKey = getRowStatusColorKey(record);
              if (statusKey) {
                const bgColor = rowColors[statusKey];
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
          toolBarRender={() =>
            [
              onPrimaryAction && (
                <Button
                  key="primary-action"
                  type="primary"
                  icon={primaryActionIcon}
                  disabled={
                    primaryActionRequiresSelection &&
                    selectedRowKeys.length === 0
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
                title="按固定业务字段导出选中行；未选择时仅导出当前页"
                icon={<DownloadOutlined />}
                onClick={handleDefaultExport}
              >
                {selectedRows.length > 0
                  ? `导出选中 (${selectedRows.length})`
                  : '导出当前页'}
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
              enableColumnSettings && (
                <React.Fragment key="col-settings">
                  {columnSettings.entry}
                </React.Fragment>
              ),
              ...extraToolBarActions,
            ].filter(Boolean)
          }
          options={{
            fullScreen: true,
            reload: true,
            density: true,
            setting: false,
          }}
          rowSelection={
            rowSelection === false
              ? undefined
              : {
                  selectedRowKeys,
                  preserveSelectedRowKeys: true,
                  onChange: (keys, rows, info) => {
                    setSelectedRowKeys(keys);
                    setSelectedRows(rows);
                    if (
                      typeof rowSelection === 'object' &&
                      rowSelection?.onChange
                    ) {
                      rowSelection.onChange(keys, rows, info);
                    }
                  },
                  ...(typeof rowSelection === 'object' ? rowSelection : {}),
                }
          }
          request={async (params) => {
            // ProTable 以 ParamsType 提供参数，此处断言到 TFilter 边界由消费方契约保证
            const res = await request(
              params as TFilter & FinanceLedgerRequestParams,
            );
            const page = unwrapPage(res);
            loadedSnapshots.current.set(page.data, {
              total: page.total,
              summary: res.summary,
            });
            return {
              ...toTableRequest(res),
              data: page.data,
            };
          }}
          onLoad={(data) => {
            // 官方 request 引擎采纳数据后再同步快照，迟到或失败请求不影响导出。
            const snapshot = loadedSnapshots.current.get(data);
            setCurrentData(data);
            if (snapshot) {
              setTotalCount(snapshot.total);
              setGlobalSummary(snapshot.summary);
            }
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
    </PageContainer>
  );
}
