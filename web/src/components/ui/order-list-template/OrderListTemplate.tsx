import {
  AlertOutlined,
  AppstoreOutlined,
  DollarOutlined,
  DownOutlined,
  EditOutlined,
  EyeOutlined,
  FileDoneOutlined,
  InboxOutlined,
  LockOutlined,
  NodeIndexOutlined,
  PaperClipOutlined,
  SwapOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Badge, Button, Card, Dropdown, Space, Tabs, Tag, Tooltip } from 'antd';
import React, { useMemo, useRef, useState } from 'react';
import { toTableRequest } from '@/utils/api';
import OrderListSearchFilter from './OrderListSearchFilter';
import OrderListToolbar from './OrderListToolbar';
import type {
  OrderListFilterParams,
  OrderListItem,
  OrderListTemplateProps,
  OrderStatusTabItem,
} from './types';

const defaultStatusTabs: OrderStatusTabItem[] = [
  { key: 'all', label: '全部订单' },
  { key: 'draft', label: '草稿待提交', count: 0 },
  { key: 'booking', label: '待订舱', count: 0, badgeColor: '#faad14' },
  { key: 'loaded', label: '已配载/订舱确认', count: 0 },
  { key: 'in_transit', label: '在途运输', count: 0, badgeColor: '#1677ff' },
  { key: 'released', label: '已放货/放行', count: 0 },
  { key: 'completed', label: '已完结', count: 0, badgeColor: '#52c41a' },
  { key: 'abnormal', label: '异常预警', count: 0, badgeColor: '#ff4d4f' },
];

export function OrderListTemplate({
  actionRef: externalActionRef,
  orderKind,
  title = '业务订单管理',
  subTitle = '支持多维复杂筛选、主分单跟踪、集装箱调度、费用结算与履约状态流转',
  statusTabs = defaultStatusTabs,
  activeStatusTab = 'all',
  onStatusTabChange,
  customColumns,
  extraColumns = [],
  queryOrders,
  onCreateOrder,
  onCopyOrder,
  onExportDocuments,
  onBatchAction,
  onViewDetail,
  onEditOrder,
  onOpenFees,
  onOpenMilestones,
  onOpenDocuments,
  documentsActionLabel = '主分单据管理',
  onOpenContainers,
  onOpenCargo,
  onOpenCargoAllocation,
  canOpenCargoAllocation,
  onOpenAttachments,
  onOpenPersonnel,
  onOpenConsolidations,
  onOpenAbnormal,
  onTransitionStatus,
  options,
  readonly = false,
  showManageTags = true,
}: OrderListTemplateProps) {
  const internalActionRef = useRef<ActionType | undefined>(undefined);
  const actionRef =
    (externalActionRef as React.MutableRefObject<ActionType | undefined>) ||
    internalActionRef;
  const [selectedRows, setSelectedRows] = useState<OrderListItem[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const currentFilterRef = useRef<OrderListFilterParams>({});
  const [currentTab, setCurrentTab] = useState<string>(activeStatusTab);
  const [filterVisible, setFilterVisible] = useState(false);

  // 1. 构建完整表头列定义
  const defaultColumns: ProColumns<OrderListItem>[] = useMemo(
    () => [
      // 1. 序号
      {
        title: '序号',
        valueType: 'index',
        width: 50,
        fixed: 'left',
      },
      // 2. 进程
      {
        title: '进程',
        dataIndex: 'stage',
        width: 90,
        sorter: true,
        render: (_, record) => {
          const val =
            record.stage ||
            (record.status === 'COMPLETED'
              ? '已完结'
              : record.status === 'CANCELLED'
                ? '已退关'
                : '未退关');
          const color =
            val === '已完结'
              ? 'success'
              : val === '已退关'
                ? 'error'
                : 'processing';
          return <Tag color={color}>{val}</Tag>;
        },
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
      // 4. 业务类型
      {
        title: '业务类型',
        dataIndex: 'businessType',
        width: 100,
        sorter: true,
        render: (_, record) => {
          const kindMap: Record<string, string> = {
            'sea-export': '海运出口',
            'sea-import': '海运进口',
            'air-export': '空运出口',
            'air-import': '空运进口',
            rail: '铁路运输',
            truck: '内陆拖车',
            customs: '报关业务',
          };
          return (
            kindMap[record.orderKind || orderKind] ||
            record.businessType ||
            '海运出口'
          );
        },
      },
      // 5. 委托单位
      {
        title: '委托单位',
        dataIndex: 'customerName',
        width: 180,
        ellipsis: true,
        sorter: true,
      },
      // 6. 客户业务编号
      {
        title: '客户业务编号',
        dataIndex: 'customerReferenceNo',
        width: 130,
        ellipsis: true,
        sorter: true,
        renderText: (val) => val || '-',
      },
      // 6.1 标签
      {
        title: '标签',
        dataIndex: 'tags',
        width: 150,
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
        renderText: (val) => (val ? val.slice(0, 16).replace('T', ' ') : '-'),
      },

      // 8. 航程与船运信息
      {
        title: '订舱代理',
        dataIndex: 'bookingAgentName',
        width: 130,
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
        title: '船名航次',
        dataIndex: 'vesselVoyage',
        width: 140,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '起运港',
        dataIndex: 'originPortName',
        width: 130,
        ellipsis: true,
        render: (_, record) =>
          record.originPortName
            ? `${record.originPortName} (${record.originPortCode || ''})`
            : record.originPortCode || '-',
      },
      {
        title: '目的港',
        dataIndex: 'destinationPortName',
        width: 130,
        ellipsis: true,
        render: (_, record) =>
          record.destinationPortName
            ? `${record.destinationPortName} (${record.destinationPortCode || ''})`
            : record.destinationPortCode || '-',
      },
      {
        title: '目的地',
        dataIndex: 'finalDestination',
        width: 120,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '箱型箱量',
        dataIndex: 'containerSummary',
        width: 130,
        ellipsis: true,
        render: (_, record) =>
          record.containerSummary ? (
            <Tag color="blue">{record.containerSummary}</Tag>
          ) : (
            '-'
          ),
      },
      {
        title: '箱号',
        dataIndex: 'containerNos',
        width: 140,
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
        width: 130,
        render: (_, record) =>
          record.operatorName
            ? `${record.operatorName}${record.operatorBranch ? ` (${record.operatorBranch})` : ''}`
            : '-',
      },
      {
        title: '业务人员',
        dataIndex: 'salesName',
        width: 130,
        render: (_, record) =>
          record.salesName
            ? `${record.salesName}${record.salesBranch ? ` (${record.salesBranch})` : ''}`
            : '-',
      },
      {
        title: '创建人',
        dataIndex: 'creatorName',
        width: 120,
        renderText: (val) => val || '-',
      },

      // 10. 货物与计量
      {
        title: '委托总件数',
        dataIndex: 'totalPackages',
        width: 110,
        align: 'right',
        render: (_, record) =>
          record.totalPackages !== undefined
            ? `${record.totalPackages} ${record.packageUnit || '件'}`
            : '-',
      },
      {
        title: '委托总毛重(KGS)',
        dataIndex: 'grossWeightKg',
        width: 130,
        align: 'right',
        renderText: (val) => (val !== undefined ? `${val} KGS` : '-'),
      },
      {
        title: '委托总体积(CBM)',
        dataIndex: 'volumeCbm',
        width: 130,
        align: 'right',
        renderText: (val) => (val !== undefined ? `${val} CBM` : '-'),
      },

      // 11. 单证与商务条款
      {
        title: '付款方式',
        dataIndex: 'paymentTerm',
        width: 130,
        renderText: (val) => val || '-',
      },
      {
        title: '贸易条款',
        dataIndex: 'tradeTerm',
        width: 100,
        renderText: (val) => val || '-',
      },
      {
        title: '合约号',
        dataIndex: 'contractNo',
        width: 120,
        renderText: (val) => val || '-',
      },
      {
        title: '收货人简称',
        dataIndex: 'consigneeName',
        width: 130,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '发货人简称',
        dataIndex: 'shipperName',
        width: 130,
        ellipsis: true,
        renderText: (val) => val || '-',
      },

      // 12. 备注与控制
      {
        title: '配舱备注',
        dataIndex: 'spaceNotes',
        width: 130,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '订舱备注',
        dataIndex: 'bookingNotes',
        width: 130,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '操作备注',
        dataIndex: 'operationNotes',
        width: 130,
        ellipsis: true,
        renderText: (val) => val || '-',
      },
      {
        title: '附件',
        dataIndex: 'attachmentCount',
        width: 70,
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
        renderText: (val) => (val ? val.slice(0, 16).replace('T', ' ') : '-'),
      },

      // 13. 右侧固定状态与异常
      {
        title: '订单状态',
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
        valueType: 'option',
        width: 180,
        fixed: 'right',
        render: (_, record) => (
          <Space size="small">
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => onViewDetail?.(record)}
            >
              详情
            </Button>
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
                    key: 'milestones',
                    icon: <NodeIndexOutlined />,
                    label: '履约里程碑',
                    onClick: () => onOpenMilestones?.(record),
                  },
                  {
                    key: 'documents',
                    icon: <FileDoneOutlined />,
                    label: documentsActionLabel,
                    onClick: () => onOpenDocuments?.(record),
                  },
                  {
                    key: 'containers',
                    icon: <InboxOutlined />,
                    label: '集装箱管理',
                    onClick: () => onOpenContainers?.(record),
                  },
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
                  ...(onOpenCargoAllocation && (canOpenCargoAllocation?.(record) ?? true)
                    ? [
                        {
                          key: 'cargo_allocation',
                          icon: <AppstoreOutlined />,
                          label: '箱货分配',
                          onClick: () => onOpenCargoAllocation(record),
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
                  ...(onOpenPersonnel
                    ? [
                        {
                          key: 'personnel',
                          icon: <NodeIndexOutlined />,
                          label: '协作人员',
                          onClick: () => onOpenPersonnel(record),
                        },
                      ]
                    : []),
                  ...(onOpenConsolidations
                    ? [
                        {
                          key: 'consolidations',
                          icon: <NodeIndexOutlined />,
                          label: '自拼汇总',
                          onClick: () => onOpenConsolidations(record),
                        },
                      ]
                    : []),
                  {
                    key: 'abnormal',
                    icon: <WarningOutlined />,
                    label: '异常情况登记',
                    onClick: () => onOpenAbnormal?.(record),
                  },
                  {
                    type: 'divider',
                  },
                  {
                    key: 'status',
                    icon: <SwapOutlined />,
                    label: '流转状态',
                    onClick: () => onTransitionStatus?.(record),
                  },
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
    ],
    [
      orderKind,
      extraColumns,
      onViewDetail,
      onEditOrder,
      onOpenFees,
      onOpenMilestones,
      onOpenDocuments,
      documentsActionLabel,
      onOpenContainers,
      onOpenCargo,
      onOpenCargoAllocation,
      canOpenCargoAllocation,
      onOpenAttachments,
      onOpenPersonnel,
      onOpenConsolidations,
      onOpenAbnormal,
      onTransitionStatus,
      readonly,
    ],
  );

  const columns = customColumns || defaultColumns;

  // 状态切签切换
  const handleTabChange = (key: string) => {
    setCurrentTab(key);
    onStatusTabChange?.(key);
    actionRef.current?.reload();
  };

  return (
    <PageContainer
      header={{
        title,
        subTitle,
      }}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      {/* 顶部状态快捷切签卡片 */}
      {statusTabs && statusTabs.length > 0 && (
        <Card
          variant="borderless"
          style={{
            borderRadius: 8,
            border: '1px solid #f0f0f0',
            backgroundColor: '#ffffff',
            marginBottom: 12,
          }}
          styles={{ body: { padding: '4px 16px 0' } }}
        >
          <Tabs
            activeKey={currentTab}
            onChange={handleTabChange}
            items={statusTabs.map((tab) => ({
              key: tab.key,
              label: (
                <Space size={4}>
                  <span>{tab.label}</span>
                  {tab.count !== undefined && tab.count > 0 && (
                    <Badge
                      count={tab.count}
                      color={tab.badgeColor || '#1677ff'}
                      style={{ boxShadow: 'none' }}
                    />
                  )}
                </Space>
              ),
            }))}
          />
        </Card>
      )}

      {/* 展开/收起的专业多维筛选面板（默认收起） */}
      {filterVisible && (
        <OrderListSearchFilter
          options={options}
          onSearch={(values) => {
            currentFilterRef.current = values;
            actionRef.current?.reload();
          }}
          onReset={() => {
            currentFilterRef.current = {};
            actionRef.current?.reload();
          }}
        />
      )}

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
          onBatchAction={onBatchAction}
          filterVisible={filterVisible}
          onToggleFilter={() => setFilterVisible((prev) => !prev)}
          readonly={readonly}
          showManageTags={showManageTags}
        />

        <ProTable<OrderListItem>
          actionRef={actionRef}
          rowKey="id"
          columns={columns}
          search={false}
          pagination={{
            defaultPageSize: 20,
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
          scroll={{ x: 2600 }}
          request={async (params, sorter) => {
            const sorterField = Object.keys(sorter)[0];
            const sorterOrder = sorterField ? sorter[sorterField] : undefined;
            const currentFilter = currentFilterRef.current;

            const res = await queryOrders({
              ...currentFilter,
              stage: currentTab !== 'all' ? currentTab : currentFilter.stage,
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
