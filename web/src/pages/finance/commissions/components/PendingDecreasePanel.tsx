import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { App, Button, Space, Tag, Tooltip, Typography } from 'antd';
import React, { useRef } from 'react';
import { useAccess } from '@/app/access';
import {
  ResponsiveProTable as ProTable,
  SearchFilterTemplate,
} from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { FinanceCommissionStatus } from '@/enums.generated';
import { financeErrorReasons } from '@/errorReasons.generated';
import {
  settlementServiceListCommissionAdjustments,
  settlementServiceMarkCommissionAdjustmentPaid,
} from '@/services/roncin/settlementService';
import { isWriteResponse, toTableRequest } from '@/utils/api';
import { formatDate } from '@/utils/format';
import {
  commissionDecreaseStatusMeta,
  decimalText,
  getBusinessReason,
} from '../types';

type Adjustment = API.FinanceCommissionAdjustment;

type PendingDecreasePanelProps = {
  /** 下钻来源：打开原提成单详情（含补录费用明细）。 */
  onOpenCommissionDetail: (commissionId: string) => void;
};

type DecreaseFilterValues = {
  keyword?: string;
  status?: number;
};

export const DECREASE_STATUS_FILTER_OPTIONS = [
  {
    label: '已确认',
    value: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
  },
  {
    label: '已执行',
    value: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID,
  },
  {
    label: '已取消',
    value: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED,
  },
];

/** 状态文案严格按服务端状态机投影，不混淆待处理/已确认/已扣回。 */
export function decreaseStatusTag(status?: number, direction?: string) {
  const meta =
    commissionDecreaseStatusMeta[
      status ?? FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
    ];
  return (
    <Tag color={meta?.color || 'default'}>
      {status === FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID &&
      direction === 'INCREASE'
        ? '已补付'
        : meta?.text || '-'}
    </Tag>
  );
}

/**
 * 待处理冲减视图：默认展示费用变更与手工产生的已确认调整，
 * 服务端分页 + 过滤，不在前端循环翻页。
 */
export default function PendingDecreasePanel({
  onOpenCommissionDetail,
}: PendingDecreasePanelProps) {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  // 列表与动作读取同一份已提交筛选；表单编辑值在提交时才固化。
  const committedFiltersRef = useRef<DecreaseFilterValues>({});

  const canManage = access.canManageFinanceCommissions === true;

  const reload = () => actionRef.current?.reload();

  const markPaid = (record: Adjustment) => {
    if (!record.id || !record.version) return;
    const id = record.id;
    const version = record.version;
    const increase = record.direction === 'INCREASE';
    const action = increase ? '补付' : '扣回';
    modal.confirm({
      title: `标记已${action} ${record.adjustmentNo || ''}？`,
      content: increase
        ? '确认该笔提成已实际补付。'
        : '原提成已发放后，确认该笔金额已在线下实际扣回。父提成尚未发放时，请登记父单净额发放，系统会一并抵扣。',
      okText: `标记已${action}`,
      onOk: async () => {
        try {
          const writeResponse1 =
            await settlementServiceMarkCommissionAdjustmentPaid(
              { id },
              { id, expectedVersion: version },
            );
          if (!isWriteResponse(writeResponse1, true)) return Promise.reject();
          message.success(`调整已标记为已${action}`);
          reload();
        } catch (error: unknown) {
          const reason = getBusinessReason(error);
          if (
            reason ===
            financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_TRANSITION
          ) {
            message.warning('建议状态已变化，列表已刷新');
            reload();
            throw error;
          }
          message.error((error as Error).message || '标记执行失败');
          throw error;
        }
      },
    }).then(
      () => {},
      () => {},
    );
  };

  const columns: ProColumns<Adjustment>[] = [
    {
      title: '员工',
      dataIndex: 'employeeName',
      width: 110,
      renderText: (value) => value || '-',
    },
    {
      title: '订单',
      dataIndex: 'orderNo',
      width: 170,
      copyable: true,
      renderText: (value) => value || '-',
    },
    {
      title: '原提成',
      dataIndex: 'commissionNo',
      width: 210,
      copyable: true,
      renderText: (value) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) => decreaseStatusTag(record.status, record.direction),
    },
    {
      title: '调整金额',
      dataIndex: 'amount',
      width: 140,
      align: 'right',
      render: (_, record) => (
        <strong
          style={{
            color: record.direction === 'INCREASE' ? '#389e0d' : '#d4380d',
          }}
        >
          {`${record.direction === 'INCREASE' ? '+' : '-'}${decimalText(record.amount)} ${record.baseCurrency || ''}`}
        </strong>
      ),
    },
    {
      title: '调整来源',
      dataIndex: 'sourceType',
      width: 120,
      renderText: (value: string) =>
        ({
          LOCKED_FEE_SUPPLEMENT: '费用补录',
          FEE_CORRECTION: '费用更正',
          ECONOMIC_REALLOCATION: '收付款重算',
          VERIFICATION_REVERSAL: '核销撤销',
          NETTING_REVERSAL: '对冲撤销',
          MANUAL: '手工调整',
        })[value] ||
        value ||
        '-',
    },
    {
      title: '原因',
      dataIndex: 'reason',
      width: 240,
      ellipsis: true,
      renderText: (value) => value || '-',
    },
    {
      title: '发起时间',
      dataIndex: 'createdAt',
      width: 150,
      renderText: (value) => formatDate(value, 'minute'),
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 230,
      render: (_, record) => {
        if (!canManage || !access.canOperateOrganization(record.organizationId))
          return ['-'];
        const actions: React.ReactNode[] = [];
        if (
          record.status ===
          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED
        ) {
          actions.push(
            record.canExecute ? (
              <a key="paid" onClick={() => markPaid(record)}>
                {record.direction === 'INCREASE' ? '标记已补付' : '标记已扣回'}
              </a>
            ) : (
              <Tooltip key="blocked" title={record.executionBlockedReason}>
                <Typography.Text type="secondary">
                  随父提成净额发放
                </Typography.Text>
              </Tooltip>
            ),
          );
        }
        actions.push(
          <Tooltip key="drill" title="查看原提成单与调整来源">
            <a
              onClick={() =>
                record.commissionId &&
                onOpenCommissionDetail(record.commissionId)
              }
            >
              来源
            </a>
          </Tooltip>,
        );
        return actions;
      },
    },
  ];

  const columnSettings = useColumnSettings<ProColumns<Adjustment>>({
    tableKey: 'finance:commission-pending-decrease',
    columns,
  });

  return (
    <>
      <SearchFilterTemplate<DecreaseFilterValues>
        layout="grid"
        collapsible={false}
        colSpan={6}
        items={[
          {
            name: 'keyword',
            label: '关键词',
            placeholder: '订单号或提成号',
            span: 8,
          },
          {
            name: 'status',
            label: '状态',
            type: 'select',
            placeholder: '已确认待执行（默认）',
            span: 6,
            options: DECREASE_STATUS_FILTER_OPTIONS,
          },
        ]}
        extraRight={
          <Space size={8}>
            <Button onClick={reload}>刷新</Button>
          </Space>
        }
        onSearch={(values) => {
          const status = values.status;
          committedFiltersRef.current = {
            keyword: values.keyword?.trim() || undefined,
            status:
              status === undefined || status === null
                ? undefined
                : Number(status),
          };
          actionRef.current?.reload();
        }}
        onReset={() => {
          committedFiltersRef.current = {};
          actionRef.current?.reload();
        }}
      />
      <ProTable<Adjustment>
        headerTitle="待执行提成调整"
        actionRef={actionRef}
        rowKey="id"
        columns={columnSettings.columns}
        size="small"
        search={false}
        toolBarRender={() => [columnSettings.entry]}
        options={{ reload: true, density: true, setting: false }}
        cardProps={{
          style: { borderRadius: 8, border: '1px solid #f0f0f0' },
        }}
        request={async (params) => {
          // 默认展示全部已确认调整，执行资格由服务端返回。
          const filters = committedFiltersRef.current;
          const response = await settlementServiceListCommissionAdjustments({
            page: params.current ?? 1,
            pageSize: params.pageSize ?? 20,
            status:
              filters.status ??
              FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
            keyword: filters.keyword,
          });
          return toTableRequest(response);
        }}
      />
    </>
  );
}
