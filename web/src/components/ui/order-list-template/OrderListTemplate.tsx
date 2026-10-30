import {
  AlertOutlined,
  DollarOutlined,
  DownOutlined,
  EditOutlined,
  FileDoneOutlined,
  InboxOutlined,
  LockOutlined,
  NodeIndexOutlined,
  PaperClipOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer } from '@ant-design/pro-components';
import { Button, Card, Dropdown, Space, Tag, Tooltip } from 'antd';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { toTableRequest } from '@/utils/api';
import { formatDate } from '@/utils/format';
import { useColumnSettings } from '../column-settings';
import { useListQueryState } from '../list-query-state';
import { ResponsiveProTable } from '../responsive-table';
import OrderListSearchFilter from './OrderListSearchFilter';
import OrderListToolbar from './OrderListToolbar';
import type {
  OrderListFilterParams,
  OrderListItem,
  OrderListTemplateProps,
} from './types';
import { useOrderAutoRefresh } from './useOrderAutoRefresh';

/** 低频辅助信息默认收起，仍可通过列设置按需开启。 */
const DEFAULT_HIDDEN_COLUMN_KEYS = [
  'rowNumber',
  'businessType',
  'creatorName',
  'lockedAt',
  'spaceNotes',
  'bookingNotes',
  'operationNotes',
  'commissionSummary',
];

export function OrderListTemplate({
  queryStateIdentity,
  actionRef: externalActionRef,
  title = '业务订单管理',
  subTitle = '支持多维复杂筛选、主分单跟踪、集装箱调度、费用结算与履约状态流转',
  customColumns,
  extraColumns = [],
  trailingColumns = [],
  completedSelectionIds,
  queryOrders,
  onCreateOrder,
  onCopyOrder,
  onExportDocuments,
  onManageTags,
  onViewDetail,
  onEditOrder,
  onOpenFees,
  onOpenDocuments,
  documentsActionLabel = '主分单据管理',
  onOpenContainers,
  onOpenCargo,
  onOpenAttachments,
  onOpenConsolidations,
  onOpenAbnormal,
  options,
  readonly = false,
  columnSettingsKey = 'orders:list',
}: OrderListTemplateProps) {
  const internalActionRef = useRef<ActionType | undefined>(undefined);
  const actionRef =
    (externalActionRef as React.MutableRefObject<ActionType | undefined>) ||
    internalActionRef;
  const [selectedRows, setSelectedRows] = useState<OrderListItem[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  useEffect(() => {
    if (!completedSelectionIds?.length) return;
    const completed = new Set(completedSelectionIds);
    setSelectedRows((rows) => rows.filter((row) => !completed.has(row.id)));
    setSelectedRowKeys((keys) =>
      keys.filter((key) => !completed.has(String(key))),
    );
  }, [completedSelectionIds]);
  const queryState = useListQueryState<OrderListFilterParams>({
    tabKey: queryStateIdentity?.tabKey ?? '',
    listId: queryStateIdentity?.listId ?? '',
    defaultPageSize: 20,
    defaultFilters: {},
  });
  const currentFilterRef = useRef<OrderListFilterParams>(
    queryState.initial.filters,
  );
  const [autoRefresh, setAutoRefresh] = useState(false);
  const busyRef = useRef(false);
  useOrderAutoRefresh(autoRefresh, actionRef, busyRef);
  const [filterVisible, setFilterVisible] = useState(false);

  // 1. 构建完整表头列定义
  const defaultColumns: ProColumns<OrderListItem>[] = useMemo(
    () => [
      // 1. 序号
      {
        title: '序号',
        key: 'rowNumber',
        width: 50,
        fixed: 'left',
        render: (_, _record, index) => index + 1,
      },
      // 2. 进程
      {
        title: '进程',
        dataIndex: 'stage',
        width: 112,
        sorter: true,
        render: (_, record) =>
          record.stage?.length ? (
            <Space size={4} wrap>
              {record.stage.map((stage) => (
                <Tag
                  key={stage}
                  color={
                    stage === '已完结'
                      ? 'success'
                      : stage === '已退关'
                        ? 'error'
                        : stage === '退关中'
                          ? 'warning'
                          : 'processing'
                  }
                  style={{ marginInlineEnd: 0 }}
                >
                  {stage}
                </Tag>
              ))}
            </Space>
          ) : (
            '-'
          ),
      },
      // 3. 订单编号
      {
        title: '订单编号',
        dataIndex: 'orderNo',
        width: 150,
        sorter: true,
        fixed: 'left',
        render: (_, record) => (
          <Space orientation="horizontal" size={4}>
            <a
              style={{ fontWeight: 600, color: '#1677ff' }}
              onClick={() => onViewDetail?.(record)}
            >
              {record.orderNo}
            </a>
            {record.isLocked && (
              <Tooltip title="订单已锁定">
                <LockOutlined style={{ color: '#faad14' }} />
              </Tooltip>
            )}
          </Space>
        ),
      },
      // 4. 业务类型（页面查询已按注册定义映射好显示文案）
      {
        title: '业务类型',
        dataIndex: 'businessType',
        width: 88,
        sorter: true,
        renderText: (val) => val || '-',
      },
      // 5. 委托单位
      {
        title: '委托单位',
        dataIndex: 'customerName',
        width: 160,
        ellipsis: true,
        sorter: true,
      },
      // 6. 客户业务编号
      {
        title: '客户业务编号',
        dataIndex: 'customerReferenceNo',
        width: 120,
        ellipsis: true,
        sorter: true,
        renderText: (val) => val || '-',
      },
      // 6.1 标签
      {
        title: '标签',
        dataIndex: 'tags',
        width: 112,
        render: (_, record) =>
          record.tags?.length ? (
            <Space size={4} wrap>
              {record.tags.map((tag) => (
                <Tag
                  key={tag.id}
                  color={tag.groupColor || undefined}
                  style={
                    tag.groupColor
                      ? { color: tag.groupColor, borderColor: tag.groupColor }
                      : undefined
                  }
                >
                  {tag.name}
                </Tag>
              ))}
            </Space>
          ) : (
            '-'
          ),
      },
      // 7. 创建时间
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        width: 140,
        sorter: true,
        renderText: (val) => formatDate(val, 'minute'),
      },

      // 8. 航程与船运信息
      {
        title: '订舱代理',
        dataIndex: 'bookingAgentName',
        width: 112,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '主单号',
        dataIndex: 'masterBlNo',
        width: 140,
        copyable: true,
        renderText: (val) => val || '-',
      },
      {
        title: options?.transportMode === 'air' ? '航班号' : '船名航次',
        dataIndex: 'vesselVoyage',
        width: 128,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: options?.transportMode === 'air' ? '起运机场' : '起运港',
        dataIndex: 'originPortName',
        width: 112,
        ellipsis: true,
        render: (_, record) =>
          record.originPortName
            ? record.originPortCode
              ? `${record.originPortName} (${record.originPortCode})`
              : record.originPortName
            : record.originPortCode || '-',
      },
      {
        title: options?.transportMode === 'air' ? '目的机场' : '目的港',
        dataIndex: 'destinationPortName',
        width: 112,
        ellipsis: true,
        render: (_, record) =>
          record.destinationPortName
            ? record.destinationPortCode
              ? `${record.destinationPortName} (${record.destinationPortCode})`
              : record.destinationPortName
            : record.destinationPortCode || '-',
      },
      {
        title: '目的地',
        dataIndex: 'finalDestination',
        width: 104,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '箱型箱量',
        hideInTable: options?.transportMode === 'air',
        dataIndex: 'containerSummary',
        width: 112,
        ellipsis: true,
        render: (_, record) =>
          record.containerSummary ? (
            <Tag
              color="blue"
              style={{
                maxWidth: '100%',
                marginInlineEnd: 0,
                whiteSpace: 'normal',
                overflowWrap: 'anywhere',
              }}
            >
              {record.containerSummary}
            </Tag>
          ) : (
            '-'
          ),
      },
      {
        title: '箱号',
        hideInTable: options?.transportMode === 'air',
        dataIndex: 'containerNos',
        width: 128,
        ellipsis: true,
        render: (_, record) =>
          record.containerNos?.length ? record.containerNos.join(', ') : '-',
      },

      // 插入品类扩展列
      ...extraColumns,

      // 9. 干系人信息
      {
        title: '操作人员',
        dataIndex: 'operatorName',
        width: 88,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '业务人员',
        dataIndex: 'salesName',
        width: 112,
        ellipsis: true,
        render: (_, record) =>
          record.salesName
            ? `${record.salesName}${record.salesBranch ? ` (${record.salesBranch})` : ''}`
            : '-',
      },
      {
        title: '创建人',
        dataIndex: 'creatorName',
        width: 88,
        ellipsis: true,
        renderText: (val) => val || '-',
      },

      // 10. 货物与计量
      {
        title: '委托总件数',
        dataIndex: 'totalPackages',
        width: 96,
        align: 'right',
        render: (_, record) =>
          record.totalPackages !== undefined
            ? `${record.totalPackages} ${record.packageUnit || '件'}`
            : '-',
      },
      {
        title: '委托总毛重(KGS)',
        dataIndex: 'grossWeightKg',
        width: 120,
        align: 'right',
        renderText: (val) => (val !== undefined ? `${val} KGS` : '-'),
      },
      {
        title: '委托总体积(CBM)',
        dataIndex: 'volumeCbm',
        width: 120,
        align: 'right',
        renderText: (val) => (val !== undefined ? `${val} CBM` : '-'),
      },

      // 11. 单证与商务条款
      {
        title: '付款方式',
        dataIndex: 'paymentTerm',
        width: 88,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '贸易条款',
        dataIndex: 'tradeTerm',
        width: 88,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '合约号',
        dataIndex: 'contractNo',
        width: 112,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '收货人简称',
        dataIndex: 'consigneeName',
        width: 112,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '发货人简称',
        dataIndex: 'shipperName',
        width: 112,
        ellipsis: true,
        renderText: (val) => val || '-',
      },

      // 12. 备注与控制
      {
        title: '配舱备注',
        dataIndex: 'spaceNotes',
        width: 112,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '订舱备注',
        dataIndex: 'bookingNotes',
        width: 112,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '操作备注',
        dataIndex: 'operationNotes',
        width: 112,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '附件',
        dataIndex: 'attachmentCount',
        width: 56,
        align: 'center',
        render: (_, record) => (
          <Space size={2}>
            <PaperClipOutlined
              style={{ color: record.attachmentCount ? '#1677ff' : '#bfbfbf' }}
            />
            <span>{record.attachmentCount || 0}</span>
          </Space>
        ),
      },
      {
        title: '订单锁定时间',
        dataIndex: 'lockedAt',
        width: 140,
        renderText: (val) => formatDate(val, 'minute'),
      },

      // 13. 右侧固定状态与异常
      {
        title: '操作进度',
        dataIndex: 'statusName',
        width: 100,
        fixed: 'right',
        render: (_, record) => {
          const name = record.statusName || record.status || '待处理';
          return <Tag color="blue">{name}</Tag>;
        },
      },
      {
        title: '异常',
        dataIndex: 'abnormalLevel',
        width: 80,
        fixed: 'right',
        render: (_, record) => {
          const level = record.abnormalLevel || 'normal';
          if (level === 'normal') {
            return <span style={{ color: '#52c41a' }}>正常</span>;
          }
          const colorMap = { low: 'orange', medium: 'volcano', high: 'red' };
          return (
            <Tag color={colorMap[level] || 'red'} icon={<AlertOutlined />}>
              {record.abnormalName || '异常'}
            </Tag>
          );
        },
      },

      // 14. 操作列
      {
        title: '操作',
        key: 'option',
        valueType: 'option',
        width: 144,
        fixed: 'right',
        render: (_, record) => (
          <Space size="small">
            {!readonly && onEditOrder && (
              <Button
                type="link"
                size="small"
                icon={<EditOutlined />}
                onClick={() => onEditOrder(record)}
              >
                编辑
              </Button>
            )}
            <Dropdown
              menu={{
                items: [
                  {
                    key: 'fees',
                    icon: <DollarOutlined />,
                    label: '费用核算',
                    onClick: () => onOpenFees?.(record),
                  },
                  {
                    key: 'documents',
                    icon: <FileDoneOutlined />,
                    label: documentsActionLabel,
                    onClick: () => onOpenDocuments?.(record),
                  },
                  ...(options?.transportMode !== 'air'
                    ? [
                        {
                          key: 'containers',
                          icon: <InboxOutlined />,
                          label: '集装箱管理',
                          onClick: () => onOpenContainers?.(record),
                        },
                      ]
                    : []),
                  ...(onOpenCargo
                    ? [
                        {
                          key: 'cargo',
                          icon: <InboxOutlined />,
                          label: '货物明细',
                          onClick: () => onOpenCargo(record),
                        },
                      ]
                    : []),
                  ...(onOpenAttachments
                    ? [
                        {
                          key: 'attachments',
                          icon: <PaperClipOutlined />,
                          label: '附件档案',
                          onClick: () => onOpenAttachments(record),
                        },
                      ]
                    : []),
                  ...(onOpenConsolidations && options?.transportMode !== 'air'
                    ? [
                        {
                          key: 'consolidations',
                          icon: <NodeIndexOutlined />,
                          label: '自拼汇总',
                          onClick: () => onOpenConsolidations(record),
                        },
                      ]
                    : []),
                  ...(onOpenAbnormal
                    ? [
                        {
                          key: 'abnormal',
                          icon: <WarningOutlined />,
                          label: '异常情况',
                          onClick: () => onOpenAbnormal(record),
                        },
                      ]
                    : []),
                ],
              }}
            >
              <Button type="link" size="small">
                更多 <DownOutlined />
              </Button>
            </Dropdown>
          </Space>
        ),
      },
      ...trailingColumns,
    ],
    [
      options?.transportMode,
      trailingColumns,
      extraColumns,
      onViewDetail,
      onEditOrder,
      onOpenFees,
      onOpenDocuments,
      documentsActionLabel,
      onOpenContainers,
      onOpenCargo,
      onOpenAttachments,
      onOpenConsolidations,
      onOpenAbnormal,
      readonly,
    ],
  );

  const columns = customColumns || defaultColumns;

  // 统一列设置：订单列表各业务视图共享同一业务标识。
  const columnSettings = useColumnSettings<ProColumns<OrderListItem>>({
    tableKey: columnSettingsKey,
    columns,
    structuralKeys: ['option'],
    defaultHiddenKeys: DEFAULT_HIDDEN_COLUMN_KEYS,
  });

  return (
    <PageContainer
      breadcrumbRender={false}
      header={{
        title,
        subTitle,
      }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      {/* 展开/收起的专业多维筛选面板（默认收起） */}
      <div hidden={!filterVisible}>
        <OrderListSearchFilter
          options={options}
          initialValues={queryState.initial.filters}
          initialLabels={queryState.initial.labels}
          onSearch={(values, labels) => {
            currentFilterRef.current = values;
            queryState.commit(values, labels);
            actionRef.current?.reload(true);
          }}
          onReset={() => {
            currentFilterRef.current = {};
            queryState.commit({}, {});
            actionRef.current?.reload(true);
          }}
        />
      </div>

      {/* 数据表格卡片 */}
      <Card
        variant="borderless"
        style={{
          borderRadius: 8,
          border: '1px solid #f0f0f0',
          backgroundColor: '#ffffff',
        }}
        styles={{ body: { padding: '14px 16px' } }}
      >
        {/* 操作工具栏 */}
        <OrderListToolbar
          orderKindTitle={`${title.replace('管理', '').replace('订单', '')}订单`}
          selectedRows={selectedRows}
          onRefresh={() => actionRef.current?.reload()}
          onCreateOrder={onCreateOrder}
          onCopyOrder={onCopyOrder}
          onExportDocuments={onExportDocuments}
          onManageTags={onManageTags}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          filterVisible={filterVisible}
          onToggleFilter={() => setFilterVisible((prev) => !prev)}
          readonly={readonly}
        />

        <ResponsiveProTable<OrderListItem>
          actionRef={actionRef}
          rowKey="id"
          columns={columnSettings.columns}
          search={false}
          toolBarRender={() => [columnSettings.entry]}
          options={{
            reload: true,
            density: true,
            setting: false,
          }}
          onLoadingChange={(loading) => {
            busyRef.current = Boolean(loading);
          }}
          pagination={{
            ...queryState.pagination,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50', '100'],
            showTotal: (total) => `共 ${total} 笔订单`,
          }}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys, rows) => {
              setSelectedRowKeys(keys);
              setSelectedRows(rows);
            },
          }}
          request={async (params, sorter) => {
            const sorterField = Object.keys(sorter)[0];
            const sorterOrder = sorterField ? sorter[sorterField] : undefined;
            const currentFilter = currentFilterRef.current;

            const res = await queryOrders({
              ...currentFilter,
              page: params.current,
              pageSize: params.pageSize,
              sorterField,
              sorterOrder: sorterOrder as 'ascend' | 'descend',
            });

            return toTableRequest(res);
          }}
        />
      </Card>
    </PageContainer>
  );
}

export default OrderListTemplate;
