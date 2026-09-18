import { Link } from '@umijs/max';
import type { TableColumnsType } from 'antd';
import { Button, Drawer, Select, Space, Table, Tag, Tooltip } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { WorkbenchCommissionStatus } from '@/enums.generated';
import { workbenchServiceListMyCommissions } from '@/services/roncin/workbenchService';
import { formatDate } from '@/utils/format';
import {
  amountWithCurrency,
  workbenchCalculationBasisText,
  workbenchCommissionStatusMeta,
  workbenchDecreaseStatusMeta,
  workbenchPersonnelRoleText,
  workbenchSourceNo,
} from './display';

type MyCommission = API.WorkbenchMyCommission;
type MyAdjustment = API.WorkbenchMyCommissionAdjustment;

type CommissionQuery = {
  page: number;
  pageSize: number;
  status?: number;
};

const DEFAULT_PAGE_SIZE = 20;

/** 状态过滤只提供工作台有效口径；CANCELLED 不计入，不作为筛选项。 */
const STATUS_FILTER_OPTIONS = [
  {
    label: '待财务确认',
    value: WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_DRAFT,
  },
  {
    label: '已确认待发',
    value: WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_CONFIRMED,
  },
  {
    label: '已发放',
    value: WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_PAID,
  },
];

function adjustmentStatusTag(status?: number) {
  const meta =
    workbenchDecreaseStatusMeta[
      status ?? WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_DRAFT
    ];
  return <Tag color={meta?.color || 'default'}>{meta?.text || '-'}</Tag>;
}

type Props = {
  open: boolean;
  baseCurrency?: string;
  onClose: () => void;
};

/**
 * 本人提成单下钻抽屉：服务端分页 + 状态过滤，展示三桶明细与冲减调整。
 * 只读 + 既有页面入口，不发明新的审批或发放操作。
 */
export default function MyCommissionDrawer({
  open,
  baseCurrency,
  onClose,
}: Props) {
  const [items, setItems] = useState<MyCommission[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<CommissionQuery>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
  });
  const sequenceRef = useRef(0);

  useEffect(() => {
    if (!open) return;
    const sequence = ++sequenceRef.current;
    setLoading(true);
    workbenchServiceListMyCommissions({
      page: query.page,
      pageSize: query.pageSize,
      ...(query.status !== undefined ? { status: query.status } : {}),
    })
      .then((response) => {
        if (sequence !== sequenceRef.current) return;
        setItems(response.data ?? []);
        setTotal(Number(response.total ?? 0));
      })
      .catch(() => {
        // 失败由统一请求错误处理提示；保留当前内容并停止加载。
      })
      .finally(() => {
        if (sequence === sequenceRef.current) setLoading(false);
      });
    return () => {
      sequenceRef.current += 1;
    };
  }, [open, query]);

  const currency = baseCurrency || undefined;

  const columns: TableColumnsType<MyCommission> = [
    {
      title: '提成单号',
      dataIndex: 'commissionNo',
      width: 170,
      render: (_, record) => record.commissionNo || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      render: (_, record) => {
        const meta = workbenchCommissionStatusMeta[record.status ?? 0];
        return <Tag color={meta?.color || 'default'}>{meta?.text || '-'}</Tag>;
      },
    },
    {
      title: '人员身份',
      dataIndex: 'personnelRole',
      width: 100,
      render: (_, record) => workbenchPersonnelRoleText(record.personnelRole),
    },
    {
      title: '提成方案',
      dataIndex: 'ruleName',
      width: 160,
      render: (_, record) => record.ruleName || '-',
    },
    {
      title: '计提口径',
      dataIndex: 'calculationBasis',
      width: 110,
      render: (_, record) =>
        workbenchCalculationBasisText(record.calculationBasis),
    },
    {
      title: '提成金额',
      dataIndex: 'commissionAmount',
      width: 140,
      align: 'right',
      render: (_, record) =>
        amountWithCurrency(
          record.commissionAmount,
          record.baseCurrency || currency,
        ),
    },
    {
      title: '归属日期',
      dataIndex: 'commissionDate',
      width: 110,
      render: (_, record) => formatDate(record.commissionDate, 'date'),
    },
    {
      title: '来源单号',
      dataIndex: 'sourceNo',
      width: 150,
      render: (_, record) => workbenchSourceNo(record),
    },
    {
      title: '冲减调整',
      dataIndex: 'adjustments',
      width: 110,
      render: (_, record) => {
        const count = record.adjustments?.length ?? 0;
        if (count === 0) return '-';
        return <Tag color="orange">{count} 条冲减</Tag>;
      },
    },
  ];

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="我的提成明细"
      width={960}
      destroyOnHidden
    >
      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Space size={8}>
            <span style={{ fontSize: 13 }}>提成状态</span>
            <Select
              allowClear
              placeholder="全部状态"
              style={{ width: 160 }}
              value={query.status}
              options={STATUS_FILTER_OPTIONS}
              onChange={(value) =>
                setQuery((prev) => ({ ...prev, page: 1, status: value }))
              }
            />
          </Space>
          <span style={{ fontSize: 12, color: '#64748b' }}>
            金额为当前组织本位币口径；已取消提成不计入。
          </span>
        </div>
        <Table<MyCommission>
          rowKey={(record) => record.id || record.commissionNo || ''}
          size="small"
          loading={loading}
          columns={columns}
          dataSource={items}
          pagination={{
            current: query.page,
            pageSize: query.pageSize,
            total,
            showSizeChanger: true,
            showTotal: (count) => `共 ${count} 条`,
            onChange: (page, pageSize) =>
              setQuery((prev) => ({ ...prev, page, pageSize })),
          }}
          expandable={{
            rowExpandable: (record) => (record.adjustments?.length ?? 0) > 0,
            expandedRowRender: (record) => (
              <Table<MyAdjustment>
                rowKey={(item) => item.id || item.adjustmentNo || ''}
                size="small"
                pagination={false}
                dataSource={record.adjustments ?? []}
                columns={[
                  {
                    title: '调整单号',
                    dataIndex: 'adjustmentNo',
                    width: 170,
                    render: (_, item) => item.adjustmentNo || '-',
                  },
                  {
                    title: '阶段',
                    dataIndex: 'status',
                    width: 190,
                    render: (_, item) => adjustmentStatusTag(item.status),
                  },
                  {
                    title: '金额',
                    dataIndex: 'amount',
                    width: 140,
                    align: 'right',
                    render: (_, item) =>
                      amountWithCurrency(
                        item.amount,
                        record.baseCurrency || currency,
                      ),
                  },
                  {
                    title: '原因',
                    dataIndex: 'reason',
                    render: (_, item) => item.reason || '-',
                  },
                  {
                    title: '操作',
                    key: 'actions',
                    width: 130,
                    render: (_, item) =>
                      item.direction === 'DECREASE' && item.id ? (
                        <Tooltip title="查看锁后补录来源详情">
                          <Link
                            to={`/commission-adjustments/${item.id}/my-supplement-source`}
                          >
                            补录来源
                          </Link>
                        </Tooltip>
                      ) : (
                        <Button type="link" size="small" disabled>
                          -
                        </Button>
                      ),
                  },
                ]}
              />
            ),
          }}
        />
      </Space>
    </Drawer>
  );
}
