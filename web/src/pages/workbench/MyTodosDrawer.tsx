import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { Alert, Button, Drawer, Space, Tabs, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { ResponsiveProTable, useColumnSettings } from '@/components/ui';
import { history } from '@/router/history';
import {
  workbenchServiceListMyOpenAbnormalCases,
  workbenchServiceListMyUnbilledFees,
} from '@/services/roncin/workbenchService';
import { ensureListResponse, unwrapPage } from '@/utils/api';
import { formatAmount, formatDate } from '@/utils/format';
import { orderTerminationStatusText, workbenchOrderPath } from './display';
import {
  useWorkbenchTodoAccess,
  type WorkbenchTodoKind,
} from './useWorkbenchTodoAccess';

type TodoOrder = Pick<
  API.WorkbenchUnbilledFee,
  | 'orderId'
  | 'orderNo'
  | 'businessType'
  | 'customerName'
  | 'orderLocked'
  | 'terminationStatus'
  | 'closureStatus'
>;

function orderState(record: TodoOrder, fees: boolean) {
  return (
    <Space orientation="vertical" size={2}>
      {record.orderLocked ? (
        <Typography.Text type="secondary">
          {fees ? '审核已锁定（禁止修改，符合条件可建账）' : '审核已锁定'}
        </Typography.Text>
      ) : null}
      {record.terminationStatus && record.terminationStatus !== 'ACTIVE' ? (
        <Tag color="red">
          {orderTerminationStatusText(record.terminationStatus)}
        </Tag>
      ) : null}
      {record.closureStatus === 'CLOSED' ? <Tag>已结案</Tag> : null}
      {!record.orderLocked &&
      (!record.terminationStatus || record.terminationStatus === 'ACTIVE') &&
      record.closureStatus !== 'CLOSED'
        ? '进行中'
        : null}
    </Space>
  );
}

function orderColumns<T extends TodoOrder>(): ProColumns<T>[] {
  return [
    { title: '订单号', dataIndex: 'orderNo', width: 180 },
    {
      title: '业务类型',
      dataIndex: 'businessType',
      width: 110,
      render: (_, record) =>
        record.businessType === 'SE'
          ? '海运出口'
          : record.businessType === 'AE'
            ? '空运出口'
            : '-',
    },
    { title: '客户名称', dataIndex: 'customerName', width: 160 },
  ];
}

function openOrder(record: TodoOrder, fees: boolean) {
  const path = workbenchOrderPath(record.businessType, record.orderId, fees);
  if (path) history.push(fees ? path : `${path}?panel=abnormal`);
}

const feeColumns: ProColumns<API.WorkbenchUnbilledFee>[] = [
  ...orderColumns<API.WorkbenchUnbilledFee>(),
  {
    title: '方向',
    dataIndex: 'direction',
    width: 80,
    render: (_, record) =>
      record.direction === 'RECEIVABLE'
        ? '应收'
        : record.direction === 'PAYABLE'
          ? '应付'
          : '-',
  },
  { title: '费用名称', dataIndex: 'feeName', width: 140 },
  { title: '结算方', dataIndex: 'settlementPartyName', width: 160 },
  { title: '币种', dataIndex: 'currency', width: 80 },
  {
    title: '金额',
    dataIndex: 'totalAmount',
    width: 120,
    align: 'right',
    render: (_, record) => formatAmount(record.totalAmount),
  },
  {
    title: '录入时间',
    dataIndex: 'createdAt',
    width: 160,
    render: (_, record) => formatDate(record.createdAt, 'minute'),
  },
  {
    title: '状态与处理条件',
    key: 'state',
    width: 260,
    render: (_, record) => (
      <Space orientation="vertical" size={4}>
        {orderState(record, true)}
        {record.hasPendingChange ? <Tag color="warning">费用待补审</Tag> : null}
        {record.blockedReasons?.map((reason) => (
          <Typography.Text key={reason} type="warning">
            {reason}
          </Typography.Text>
        ))}
      </Space>
    ),
  },
  {
    title: '操作',
    key: 'action',
    valueType: 'option',
    width: 110,
    fixed: 'right',
    align: 'right',
    render: (_, record) => (
      <Button
        type="link"
        size="small"
        disabled={
          !workbenchOrderPath(record.businessType, record.orderId, true)
        }
        onClick={() => openOrder(record, true)}
      >
        进入费用页
      </Button>
    ),
  },
];

const abnormalColumns: ProColumns<API.WorkbenchOpenAbnormalCase>[] = [
  ...orderColumns<API.WorkbenchOpenAbnormalCase>(),
  { title: '异常类型', dataIndex: 'abnormalCaseName', width: 180 },
  {
    title: '标记时间',
    dataIndex: 'markedAt',
    width: 160,
    render: (_, record) => formatDate(record.markedAt, 'minute'),
  },
  {
    title: '状态与处理条件',
    key: 'state',
    width: 240,
    render: (_, record) => (
      <Space orientation="vertical" size={4}>
        {orderState(record, false)}
        {record.businessWriteBlockedReason ? (
          <Typography.Text type="secondary">
            只读：{record.businessWriteBlockedReason}
          </Typography.Text>
        ) : null}
      </Space>
    ),
  },
  {
    title: '操作',
    key: 'action',
    valueType: 'option',
    width: 110,
    fixed: 'right',
    align: 'right',
    render: (_, record) => (
      <Button
        type="link"
        size="small"
        disabled={!workbenchOrderPath(record.businessType, record.orderId)}
        onClick={() => openOrder(record, false)}
      >
        查看异常
      </Button>
    ),
  },
];

type PageParams = { page: number; pageSize: number };

function TodoTable<T extends object>({
  columns,
  tableKey,
  rowKey,
  unit,
  request,
  onRefresh,
}: {
  columns: ProColumns<T>[];
  tableKey: string;
  rowKey: string;
  unit: string;
  request: (params: PageParams) => Promise<{ data: T[]; total: number }>;
  onRefresh: () => void;
}) {
  const actionRef = useRef<ActionType>(undefined);
  const [failed, setFailed] = useState(false);
  const columnSettings = useColumnSettings<ProColumns<T>>({
    tableKey,
    columns,
  });
  const refresh = () => {
    actionRef.current?.reload();
    onRefresh();
  };
  return (
    <ResponsiveProTable<T>
      actionRef={actionRef}
      rowKey={rowKey}
      columns={columnSettings.columns}
      size="small"
      search={false}
      cardProps={false}
      options={false}
      request={async ({ current = 1, pageSize = 20 }) => {
        setFailed(false);
        const result = await request({ page: current, pageSize });
        return { ...result, success: true };
      }}
      onRequestError={() => setFailed(true)}
      toolBarRender={() => [
        columnSettings.entry,
        <Button key="refresh" size="small" onClick={refresh}>
          刷新待办
        </Button>,
      ]}
      tableRender={(_, table) =>
        failed ? (
          <Alert
            type="error"
            showIcon
            title="待办明细加载失败"
            action={<Button onClick={refresh}>重试</Button>}
          />
        ) : (
          table
        )
      }
      locale={{ emptyText: '当前没有此类待办' }}
      pagination={{
        defaultPageSize: 20,
        showSizeChanger: true,
        pageSizeOptions: [20, 50, 100, 200],
        showTotal: (total) => `共 ${total} ${unit}`,
      }}
    />
  );
}

async function requestFees(params: PageParams) {
  const response = await workbenchServiceListMyUnbilledFees(params);
  ensureListResponse(response, '未建账费用加载失败');
  if (response?.success === false) throw new Error('未建账费用加载失败');
  return unwrapPage(response);
}

async function requestAbnormalCases(params: PageParams) {
  const response = await workbenchServiceListMyOpenAbnormalCases(params);
  ensureListResponse(response, '进行中异常加载失败');
  if (response?.success === false) throw new Error('进行中异常加载失败');
  return unwrapPage(response);
}

/** 切换分类重建表格，重开抽屉重新请求；数据与分页完全交由 ProTable 管理。 */
export default function MyTodosDrawer({
  initialKind,
  onClose,
  onRefresh,
}: {
  initialKind: WorkbenchTodoKind;
  onClose: () => void;
  onRefresh: () => void;
}) {
  const access = useWorkbenchTodoAccess();
  const [selectedKind, setSelectedKind] = useState(initialKind);
  const kind = access[selectedKind]
    ? selectedKind
    : access.fees
      ? 'fees'
      : 'abnormal';
  if (!access.fees && !access.abnormal) return null;

  return (
    <Drawer
      open
      title="我的作业待办明细"
      size="90%"
      onClose={onClose}
      destroyOnHidden
    >
      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <Alert
          type="info"
          showIcon
          title="每行是一笔费用或一件异常，同一订单可能有多项待办。只读或阻塞项目仍可进入订单查看处理条件。"
        />
        <Tabs
          activeKey={kind}
          onChange={(value) => setSelectedKind(value as WorkbenchTodoKind)}
          items={[
            ...(access.fees ? [{ key: 'fees', label: '未建账费用' }] : []),
            ...(access.abnormal
              ? [{ key: 'abnormal', label: '进行中异常' }]
              : []),
          ]}
        />
        {kind === 'fees' ? (
          <TodoTable
            key="fees"
            columns={feeColumns}
            tableKey="workbench:my-unbilled-fees"
            rowKey="feeId"
            unit="笔"
            request={requestFees}
            onRefresh={onRefresh}
          />
        ) : (
          <TodoTable
            key="abnormal"
            columns={abnormalColumns}
            tableKey="workbench:my-open-abnormal-cases"
            rowKey="id"
            unit="件"
            request={requestAbnormalCases}
            onRefresh={onRefresh}
          />
        )}
      </Space>
    </Drawer>
  );
}
