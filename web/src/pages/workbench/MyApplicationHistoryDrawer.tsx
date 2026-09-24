import {
  keepPreviousData,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { useAccess } from '@/app/access';
import type { TableColumnsType } from 'antd';
import {
  App,
  Button,
  Descriptions,
  Drawer,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import React, { useState } from 'react';
import { useColumnSettings } from '@/components/ui/column-settings';
import { WorkbenchCommissionApplicationStatus } from '@/enums.generated';
import {
  workbenchServiceGetMyCommissionApplication,
  workbenchServiceListMyCommissionApplications,
  workbenchServiceResubmitMyCommissionApplication,
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

type HistoryQuery = {
  page: number;
  pageSize: number;
  status?: number;
};

const DEFAULT_PAGE_SIZE = 20;

/** 服务端状态域前缀：本人月度申请（列表与详情）查询的统一 key 前缀。 */
const APPLICATIONS_QUERY_BASE = ['workbench', 'my-applications'] as const;

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
  /** 显式重提成功后刷新工作台 Overview（由面板透传）。 */
  onResubmitted?: () => void | Promise<void>;
};

/**
 * 本人月度申请历史抽屉：服务端分页列表 + 明细下钻。
 * 只读展示提交/决策审计与明细快照；被驳回的申请展示驳回原因并提供
 * 「重新提交」显式重提入口（按申请 ID + 当前版本定位原申请，
 * 服务端按最新上游数据刷新金额后重新进入财务审批）。
 */
export default function MyApplicationHistoryDrawer({
  open,
  baseCurrency,
  onClose,
  onResubmitted,
}: Props) {
  const { canOperateBusiness } = useAccess();
  const queryClient = useQueryClient();
  const [query, setQuery] = useState<HistoryQuery>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
  });
  /** 明细下钻只记录选中的申请 ID，数据由服务端状态层管理。 */
  const [detailId, setDetailId] = useState<string>();
  const [resubmitting, setResubmitting] = useState(false);
  const { message, modal } = App.useApp();

  const { data: listData, isFetching: listFetching } = useQuery({
    queryKey: [
      ...APPLICATIONS_QUERY_BASE,
      { page: query.page, pageSize: query.pageSize, status: query.status },
    ],
    queryFn: () =>
      workbenchServiceListMyCommissionApplications({
        page: query.page,
        pageSize: query.pageSize,
        ...(query.status !== undefined ? { status: query.status } : {}),
      }),
    enabled: open,
    // 旧行为为空 catch 静默，仅靠请求层 notification；显式声明避免全局 message 补充弹错
    meta: { silent: true },
    // 翻页与切换过滤期间保留当前内容，与既有手写层行为一致。
    placeholderData: keepPreviousData,
  });

  const { data: detailData, isFetching: detailFetching } = useQuery({
    queryKey: [...APPLICATIONS_QUERY_BASE, 'detail', detailId],
    queryFn: () =>
      workbenchServiceGetMyCommissionApplication({ id: detailId ?? '' }),
    enabled: detailId !== undefined,
    // 旧行为为空 catch 静默，仅靠请求层 notification；显式声明避免全局 message 补充弹错
    meta: { silent: true },
  });

  const items = listData?.data ?? [];
  const total = Number(listData?.total ?? 0);
  const detail = detailData?.data;

  const openDetail = (record: Application) => {
    if (record.id) setDetailId(record.id);
  };

  const backToList = () => {
    setDetailId(undefined);
  };

  const isRejected =
    detail?.application?.status ===
    WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED;

  /** 显式重提：按申请 ID + 当前版本定位原申请，服务端按最新上游数据刷新金额。 */
  const resubmit = (record: Application) => {
    if (!canOperateBusiness || !record.id || !record.version || resubmitting)
      return;
    const applicationId = record.id;
    const expectedVersion = record.version;
    const monthLabel = record.applicationMonth || '';
    modal.confirm({
      title: `重新提交 ${monthLabel} 月度申请？`,
      content: (
        <div>
          <p>
            {`将按最新上游数据刷新本申请各明细的提成金额（已失效的明细会被剔除），重新提交财务整批审批。`}
          </p>
          <p style={{ color: '#64748b' }}>
            最近一次驳回原因：{record.decisionReason || '-'}；重提沿用原申请与
            申请月份，不会产生第二张申请。
          </p>
        </div>
      ),
      okText: '重新提交',
      // antd 6 的 modal.confirm 移除 confirmLoading，经 okButtonProps 表达加载态。
      okButtonProps: { loading: resubmitting },
      cancelText: '再想想',
      onOk: async () => {
        setResubmitting(true);
        try {
          await workbenchServiceResubmitMyCommissionApplication({
            applicationId,
            expectedVersion,
          });
          message.success('申请已重新提交，等待财务审批');
          backToList();
          // 重提成功后失效申请历史域（列表与详情快照按原翻页/过滤参数重查）；
          // 刷新失败由全局查询错误处理提示，不改变本次重提的成功结果。
          queryClient
            .invalidateQueries({ queryKey: [...APPLICATIONS_QUERY_BASE] })
            .catch(() => undefined);
          await onResubmitted?.();
        } catch (error: unknown) {
          message.error((error as Error).message || '重新提交失败');
        } finally {
          setResubmitting(false);
        }
      },
    });
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

  const lineSettings = useColumnSettings<TableColumnsType<ApplicationLine>[number]>({
    tableKey: 'workbench:application-history-lines',
    columns: lineColumns,
  });

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
      width: 110,
      render: (_, record) => (
        <Space size={8}>
          <a onClick={() => openDetail(record)}>明细</a>
          {canOperateBusiness &&
          record.status ===
            WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED ? (
            <a onClick={() => resubmit(record)}>重新提交</a>
          ) : null}
        </Space>
      ),
    },
  ];

  const listSettings = useColumnSettings<TableColumnsType<Application>[number]>({
    tableKey: 'workbench:application-history',
    columns,
  });

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="我的月度申请历史"
      size={960}
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
            {canOperateBusiness && isRejected ? (
              <Button
                type="primary"
                size="small"
                loading={resubmitting}
                onClick={() => resubmit(application as Application)}
                data-testid="resubmit-application-button"
              >
                重新提交
              </Button>
            ) : null}
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
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
            {lineSettings.entry}
          </div>
          <Table<ApplicationLine>
            rowKey={(record) => record.id || record.commissionId || ''}
            size="small"
            loading={detailFetching}
            columns={lineSettings.columns}
            dataSource={detail.lines ?? []}
            pagination={false}
          />
          {lineSettings.modal}
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
              金额与明细为提交时固化的快照；被驳回的申请可在原申请上按最新上游数据重新提交。
            </span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
            {listSettings.entry}
          </div>
          <Table<Application>
            rowKey={(record) => record.id || record.applicationMonth || ''}
            size="small"
            loading={listFetching}
            columns={listSettings.columns}
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
          {listSettings.modal}
        </Space>
      )}
    </Drawer>
  );
}
