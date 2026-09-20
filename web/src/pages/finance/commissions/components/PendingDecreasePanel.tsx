import { CheckOutlined, CloseCircleOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Space, Tag, Tooltip } from 'antd';
import React, { useRef } from 'react';
import { SearchFilterTemplate } from '@/components/ui';
import { FinanceCommissionStatus } from '@/enums.generated';
import { financeErrorReasons } from '@/errorReasons.generated';
import {
  settlementServiceCancelCommissionAdjustment,
  settlementServiceConfirmCommissionAdjustment,
  settlementServiceListCommissionAdjustments,
  settlementServiceMarkCommissionAdjustmentPaid,
} from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { confirmWithReason } from '@/utils/confirmWithReason';
import {
  commissionDecreaseStatusMeta,
  decimalText,
  getBusinessReason,
  LOCKED_FEE_SUPPLEMENT_SOURCE,
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
    label: '待处理（草稿）',
    value: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT,
  },
  {
    label: '已确认',
    value: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
  },
  {
    label: '已扣回',
    value: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID,
  },
  {
    label: '已取消',
    value: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED,
  },
];

/** 状态文案严格按服务端状态机投影，不混淆待处理/已确认/已扣回。 */
export function decreaseStatusTag(status?: number) {
  const meta =
    commissionDecreaseStatusMeta[
      status ?? FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
    ];
  return <Tag color={meta?.color || 'default'}>{meta?.text || '-'}</Tag>;
}

/**
 * 待处理冲减视图：默认只展示锁后费用补录生成的 DECREASE + DRAFT 建议，
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

  const confirmDecrease = (record: Adjustment) => {
    if (!record.id || !record.version) return;
    const id = record.id;
    const version = record.version;
    modal.confirm({
      title: `确认冲减 ${record.adjustmentNo || ''}？`,
      content: (
        <div>
          <p>
            确认后该建议由「待处理」转为「已确认」，表示财务认可以后应少发，
            但尚不代表已实际扣回；实际少发后请再执行「标记已扣回」。
          </p>
          <p style={{ color: '#faad14' }}>
            确认后若该提成单因反核销整体取消，本冲减将随之作废。
          </p>
        </div>
      ),
      okText: '确认冲减',
      onOk: async () => {
        try {
          await settlementServiceConfirmCommissionAdjustment(
            { id },
            { id, expectedVersion: version },
          );
          message.success('冲减建议已确认（已确认不等于已扣回）');
          reload();
        } catch (error: unknown) {
          const reason = getBusinessReason(error);
          if (
            reason === financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS
          ) {
            // 有效提成不能为负：提示刷新后重试，不静默缩小金额。
            modal.warning({
              title: '冲减金额超出可冲减余额',
              content:
                '该提成单的有效提成已不足以全额冲减（可能存在并发调整），请刷新列表后重新处理。',
            });
            reload();
            return;
          }
          if (
            reason ===
            financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_TRANSITION
          ) {
            message.warning('建议状态已变化，列表已刷新');
            reload();
            return;
          }
          message.error((error as Error).message || '确认冲减失败');
        }
      },
    });
  };

  const ignoreDecrease = (record: Adjustment) => {
    if (!record.id || !record.version) return;
    const id = record.id;
    const version = record.version;
    // 忽略建议复用现有调整取消能力，必填原因后转 CANCELLED。
    confirmWithReason(
      { modal, message },
      `忽略冲减建议 ${record.adjustmentNo || ''}？`,
      async (reason) => {
        try {
          await settlementServiceCancelCommissionAdjustment(
            { id },
            { id, expectedVersion: version, reason },
          );
          message.success('冲减建议已忽略');
          reload();
        } catch (error: unknown) {
          const reasonCode = getBusinessReason(error);
          if (
            reasonCode ===
            financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_CANCEL_NOT_ALLOWED
          ) {
            message.warning('该建议已进入已确认或已扣回状态，不能忽略');
            reload();
            return;
          }
          if (
            reasonCode ===
            financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_TRANSITION
          ) {
            message.warning('建议状态已变化，列表已刷新');
            reload();
            return;
          }
          message.error((error as Error).message || '忽略冲减建议失败');
        }
      },
      {
        danger: true,
        placeholder: '请输入忽略原因（必填）',
        requiredMessage: '请输入忽略原因',
      },
    );
  };

  const markPaid = (record: Adjustment) => {
    if (!record.id || !record.version) return;
    const id = record.id;
    const version = record.version;
    modal.confirm({
      title: `标记已扣回 ${record.adjustmentNo || ''}？`,
      content: '该操作表示冲减金额已实际扣回（线下已少发），完成后不可取消。',
      okText: '标记已扣回',
      onOk: async () => {
        try {
          await settlementServiceMarkCommissionAdjustmentPaid(
            { id },
            { id, expectedVersion: version },
          );
          message.success('冲减已标记为已扣回');
          reload();
        } catch (error: unknown) {
          const reason = getBusinessReason(error);
          if (
            reason ===
            financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_TRANSITION
          ) {
            message.warning('建议状态已变化，列表已刷新');
            reload();
            return;
          }
          message.error((error as Error).message || '标记已扣回失败');
        }
      },
    });
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
      render: (_, record) => decreaseStatusTag(record.status),
    },
    {
      title: '建议金额',
      dataIndex: 'amount',
      width: 140,
      align: 'right',
      render: (_, record) => (
        <strong style={{ color: '#d4380d' }}>
          {`-${decimalText(record.amount)} ${record.baseCurrency || ''}`}
        </strong>
      ),
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
      renderText: (value) =>
        value ? value.slice(0, 16).replace('T', ' ') : '-',
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
          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
        ) {
          actions.push(
            <a key="confirm" onClick={() => confirmDecrease(record)}>
              <CheckOutlined /> 确认冲减
            </a>,
            <a key="ignore" onClick={() => ignoreDecrease(record)}>
              <CloseCircleOutlined /> 忽略建议
            </a>,
          );
        }
        if (
          record.status ===
          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED
        ) {
          actions.push(
            <a key="paid" onClick={() => markPaid(record)}>
              标记已扣回
            </a>,
          );
        }
        actions.push(
          <Tooltip key="drill" title="查看原提成单与补录费用来源">
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
            placeholder: '待处理（默认）',
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
        headerTitle="锁后费用补录冲减建议"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        size="small"
        search={false}
        toolBarRender={false}
        cardProps={{
          style: { borderRadius: 8, border: '1px solid #f0f0f0' },
        }}
        request={async (params) => {
          // 默认只看锁后费用补录来源（方向恒为 DECREASE）；状态默认 DRAFT。
          const filters = committedFiltersRef.current;
          const response = await settlementServiceListCommissionAdjustments({
            page: params.current ?? 1,
            pageSize: params.pageSize ?? 20,
            status:
              filters.status ??
              FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT,
            sourceType: LOCKED_FEE_SUPPLEMENT_SOURCE,
            keyword: filters.keyword,
          });
          return toTableRequest(response);
        }}
      />
    </>
  );
}
