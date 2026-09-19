import type { TableColumnsType } from 'antd';
import {
  Button,
  Descriptions,
  Drawer,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { WorkbenchCommissionApplicationStatus } from '@/enums.generated';
import {
  workbenchServiceGetMyCommissionApplication,
  workbenchServiceListMyCommissionApplications,
} from '@/services/roncin/workbenchService';
import { formatDate } from '@/utils/format';
import { applicationStatusTag } from './applicationDisplay';
import {
  amountWithCurrency,
  workbenchCalculationBasisText,
  workbenchPersonnelRoleText,
  workbenchSourceNo,
} from './display';

const { Text } = Typography;

type Application = API.WorkbenchMyCommissionApplication;
type ApplicationLine = API.WorkbenchMyApplicationLine;
type ApplicationDetail = API.WorkbenchMyCommissionApplicationDetail;

type HistoryQuery = {
  page: number;
  pageSize: number;
  status?: number;
};

const DEFAULT_PAGE_SIZE = 20;

/** 状态过滤只提供工作台有效口径；UNSPECIFIED 不作为筛选项。 */
const APPLICATION_STATUS_FILTER_OPTIONS = [
  {
    label: '审批中',
    value:
      WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
  },
  {
    label: '已驳回',
    value:
      WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED,
  },
  {
    label: '已批准',
    value:
      WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_APPROVED,
  },
];

type Props = {
  open: boolean;
  baseCurrency?: string;
  onClose: () => void;
};

/**
 * 本人月度申请历史抽屉：服务端分页列表 + 明细下钻。
 * 只读展示提交/决策审计与明细快照；被驳回的申请展示驳回原因，
 * 不在抽屉内提供重提或任何审批动作。
 */
export default function MyApplicationHistoryDrawer({
  open,
  baseCurrency,
  onClose,
}: Props) {
  const [items, setItems] = useState<Application[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<HistoryQuery>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
  });
  const [detail, setDetail] = useState<ApplicationDetail>();
  const [detailLoading, setDetailLoading] = useState(false);
  const listSequenceRef = useRef(0);
  const detailSequenceRef = useRef(0);

  useEffect(() => {
    if (!open) return;
    const sequence = ++listSequenceRef.current;
    setLoading(true);
    workbenchServiceListMyCommissionApplications({
      page: query.page,
      pageSize: query.pageSize,
      ...(query.status !== undefined ? { status: query.status } : {}),
    })
      .then((response) => {
        if (sequence !== listSequenceRef.current) return;
        setItems(response.data ?? []);
        setTotal(Number(response.total ?? 0));
      })
      .catch(() => {
        // 失败由统一请求错误处理提示；保留当前内容并停止加载。
      })
      .finally(() => {
        if (sequence === listSequenceRef.current) setLoading(false);
      });
    return () => {
      listSequenceRef.current += 1;
    };
  }, [open, query]);

  const openDetail = (record: Application) => {
    if (!record.id) return;
    const applicationId = record.id;
    const sequence = ++detailSequenceRef.current;
    setDetailLoading(true);
    workbenchServiceGetMyCommissionApplication({ id: applicationId })
      .then((response) => {
        if (sequence !== detailSequenceRef.current) return;
        setDetail(response.data);
      })
      .catch(() => {
        // 失败由统一请求错误处理提示；留在列表视图。
      })
      .finally(() => {
        if (sequence === detailSequenceRef.current) setDetailLoading(false);
      });
  };

  const backToList = () => {
    detailSequenceRef.current += 1;
    setDetail(undefined);
    setDetailLoading(false);
  };

  const currency = baseCurrency || undefined;
  const application = detail?.application;

  const lineColumns: TableColumnsType<ApplicationLine> = [
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
      title: '人员身份',
      dataIndex: 'personnelRole',
      width: 100,
      render: (_, record) => workbenchPersonnelRoleText(record.personnelRole),
    },
    {
      title: '提成方案',
      dataIndex: 'ruleName',
      width: 150,
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
      width: 130,
      align: 'right',
      render: (_, record) =>
        amountWithCurrency(
          record.commissionAmount,
          record.baseCurrency || currency,
        ),
    },
  ];

  const columns: TableColumnsType<Application> = [
    {
      title: '申请月份',
      dataIndex: 'applicationMonth',
      width: 100,
      render: (_, record) => record.applicationMonth || '-',
    },
    {
      title: '覆盖截止日',
      dataIndex: 'coverageTo',
      width: 110,
      render: (_, record) => formatDate(record.coverageTo, 'date'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) => applicationStatusTag(record.status),
    },
    {
      title: '笔数',
      dataIndex: 'commissionCount',
      width: 70,
      align: 'right',
      render: (_, record) => record.commissionCount ?? 0,
    },
    {
      title: '总额',
      dataIndex: 'totalCommissionAmount',
      width: 140,
      align: 'right',
      render: (_, record) =>
        amountWithCurrency(
          record.totalCommissionAmount,
          record.baseCurrency || currency,
        ),
    },
    {
      title: '提交时间',
      dataIndex: 'submittedAt',
      width: 150,
      render: (_, record) => formatDate(record.submittedAt),
    },
    {
      title: '决策时间',
      dataIndex: 'decidedAt',
      width: 150,
      render: (_, record) => formatDate(record.decidedAt),
    },
    {
      title: '操作',
      key: 'actions',
      width: 80,
      render: (_, record) => <a onClick={() => openDetail(record)}>明细</a>,
    },
  ];

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="我的月度申请历史"
      width={960}
      destroyOnHidden
    >
      {detail ? (
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Space size={8}>
            <Button size="small" onClick={backToList}>
              返回列表
            </Button>
            <Text strong>{application?.applicationMonth || '-'} 月度申请</Text>
            {applicationStatusTag(application?.status)}
          </Space>
          <Descriptions
            size="small"
            column={2}
            bordered
            items={[
              {
                key: 'coverageTo',
                label: '覆盖截止日',
                children: formatDate(application?.coverageTo, 'date'),
              },
              {
                key: 'version',
                label: '当前版本',
                children: application?.version ?? '-',
              },
              {
                key: 'count',
                label: '笔数',
                children: application?.commissionCount ?? 0,
              },
              {
                key: 'amount',
                label: '总额',
                children: amountWithCurrency(
                  application?.totalCommissionAmount,
                  application?.baseCurrency || currency,
                ),
              },
              {
                key: 'submittedAt',
                label: '提交时间',
                children: formatDate(application?.submittedAt),
              },
              {
                key: 'decidedAt',
                label: '决策时间',
                children: formatDate(application?.decidedAt),
              },
            ]}
          />
          {application?.decisionReason ? (
            <Tag color="error" style={{ whiteSpace: 'normal' }}>
              {`驳回原因：${application.decisionReason}`}
            </Tag>
          ) : null}
          <Table<ApplicationLine>
            rowKey={(record) => record.id || record.commissionId || ''}
            size="small"
            loading={detailLoading}
            columns={lineColumns}
            dataSource={detail.lines ?? []}
            pagination={false}
          />
        </Space>
      ) : (
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Space size={8}>
              <span style={{ fontSize: 13 }}>申请状态</span>
              <Select
                allowClear
                placeholder="全部状态"
                style={{ width: 140 }}
                value={query.status}
                options={APPLICATION_STATUS_FILTER_OPTIONS}
                onChange={(value) =>
                  setQuery((prev) => ({ ...prev, page: 1, status: value }))
                }
              />
            </Space>
            <span style={{ fontSize: 12, color: '#64748b' }}>
              金额与明细为提交时固化的快照；被驳回的申请可在原申请上重新提交。
            </span>
          </div>
          <Table<Application>
            rowKey={(record) => record.id || record.applicationMonth || ''}
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
          />
        </Space>
      )}
    </Drawer>
  );
}
