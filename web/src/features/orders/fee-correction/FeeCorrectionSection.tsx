import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProFormTextArea } from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, App, Button, Descriptions, Space, Spin, Tag } from 'antd';
import { type ReactNode, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useModalOwnerSignal } from '@/components/layout/useModalOwnerSignal';
import {
  GuardedModalForm,
  ResponsiveProTable,
  ResponsiveTable,
} from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import {
  feeReviewCommentFieldProps,
  feeReviewCommentRule,
} from '@/features/orders/fee-review';
import {
  orderFeeServiceApproveOrderFeeCorrection,
  orderFeeServiceListOrderFeeCorrectionRequests,
  orderFeeServicePreviewOrderFeeCorrectionApproval,
  orderFeeServiceRejectOrderFeeCorrection,
  orderFeeServiceWithdrawOrderFeeCorrection,
} from '@/services/roncin/orderFeeService';
import { isWriteResponse, toTableRequest } from '@/utils/api';
import { confirmWithReason } from '@/utils/confirmWithReason';
import { formatAmount, formatDate, trimDecimal } from '@/utils/format';

type Correction = API.OrderFeeCorrectionRequestData;
const statusLabels: Record<string, string> = {
  PENDING: '待审批',
  APPROVED: '已批准',
  REJECTED: '已驳回',
  WITHDRAWN: '已撤回',
};
const financialLabels: Record<string, string> = {
  FINANCE_FOLLOW_UP: '待财务处理',
  ADJUSTMENT_UNBILLED: '调整费用待建账',
  COMPLETED: '处理完成',
  INVOICE_PENDING: '原发票待红冲或取消',
  SETTLEMENT_PENDING: '调整账款待清偿',
  INVOICE_REISSUE_PENDING: '更正后待重新开票',
};
const operationLabels: Record<number, string> = {
  1: '追加费用',
  2: '替换费用',
  3: '作废费用',
};

function feeEvidence(fee: API.OrderFee) {
  return (
    <>
      <div>
        {fee.feeName || '-'} / {fee.direction === 1 ? '应收' : '应付'} /{' '}
        {fee.settlementPartyName || '-'} / {formatAmount(fee.totalAmount)}{' '}
        {fee.currency}
      </div>
      <div>
        数量 {trimDecimal(fee.quantity)} × 单价 {trimDecimal(fee.unitPrice)}
        ；税率 {trimDecimal(fee.taxRate)}%，
        {fee.taxInclusive === true ? '含税单价' : '不含税单价'}
      </div>
      <div>
        汇率 {trimDecimal(fee.exchangeRate)}；录入日期{' '}
        {formatDate(fee.createdAt, 'date')}
      </div>
      {fee.note && <div>备注：{fee.note}</div>}
    </>
  );
}

export function FeeCorrectionComparison({ request }: { request: Correction }) {
  return (
    <Descriptions
      size="small"
      column={2}
      items={[
        {
          key: 'before',
          label: '原费用',
          children: request.beforeFee
            ? feeEvidence(request.beforeFee)
            : '无（追加费用）',
        },
        {
          key: 'after',
          label: '更正后',
          children: request.afterFee
            ? feeEvidence(request.afterFee)
            : '整笔作废',
        },
        {
          key: 'accounting-impact',
          label: '账务影响',
          span: 2,
          children: request.beforeFee?.hasActiveBill
            ? '原费用已建账，须先按权限撤销核销、发票等依赖并取消账单，回退至账单未建立后才能补录补审。'
            : request.afterFee
              ? '批准后新费用进入待建账；原费用如未建账则保留历史，不重复建账。'
              : '原费用未建账，批准后保留作废记录，无须创建零金额账单。',
        },
        {
          key: 'reason',
          label: '更正原因',
          span: 2,
          children: request.reason || '-',
        },
        {
          key: 'reviewer',
          label: '审批人',
          children: request.decidedByName || '-',
        },
        {
          key: 'decision',
          label: '审批时间',
          children: formatDate(request.decidedAt, 'datetime'),
        },
        {
          key: 'comment',
          label: '审批意见',
          span: 2,
          children: request.decisionReason || '-',
        },
        {
          key: 'approved-profit',
          label: '批准时利润',
          span: 2,
          children: request.approvedProjectedSummary
            ? `${formatAmount(request.approvedCurrentSummary?.profit)} → ${formatAmount(request.approvedProjectedSummary.profit)} ${request.approvedProjectedSummary.baseCurrency || ''}`
            : '-',
        },
        {
          key: 'approved-commissions',
          label: '批准时提成影响',
          span: 2,
          children:
            request.commissionImpacts?.map((impact) => (
              <div key={`${impact.commissionId}:${impact.orderId}`}>
                {impact.commissionNo} {impact.employeeName}：
                {formatAmount(impact.currentAmount)} →{' '}
                {formatAmount(impact.targetAmount)} {impact.baseCurrency}
              </div>
            )) || '-',
        },
        {
          key: 'finance',
          label: '账单与票据处理',
          span: 2,
          children:
            financialLabels[request.financialStatus || ''] ||
            request.financialStatus ||
            '-',
        },
      ]}
    />
  );
}

/** 订单与财务费用详情共用；审批资格完全消费服务端当前能力。 */
export function FeeCorrectionSection({
  orderId,
  onChanged,
  renderCreate,
  onRecordsLoaded,
}: {
  orderId: string;
  onChanged?: () => void;
  /** 通知外层是否有申请记录，用于分节的默认展开。 */
  onRecordsLoaded?: (hasRecords: boolean) => void;
  renderCreate?: (refresh: () => void) => ReactNode;
}) {
  const getModalOwnerSignal = useModalOwnerSignal();
  const app = App.useApp();
  const queryClient = useQueryClient();
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const userId = initialState?.currentUser?.id;
  const actionRef = useRef<ActionType>(undefined);
  const [review, setReview] = useState<Correction>();
  const preview = useQuery({
    queryKey: [
      'fee-correction-preview',
      organizationId,
      userId,
      orderId,
      review?.id,
      review?.version,
    ],
    enabled: Boolean(review?.id),
    queryFn: async () => {
      const response = await orderFeeServicePreviewOrderFeeCorrectionApproval(
        { orderId, id: review?.id || '' },
        { orderId, id: review?.id || '', expectedVersion: review?.version },
      );
      if (!response?.data) throw new Error('费用补审预览失败，请刷新重试');
      return response.data;
    },
  });
  const refresh = () => {
    actionRef.current?.reload();
    void queryClient.invalidateQueries({ queryKey: ['order-fee-review'] });
    void queryClient.invalidateQueries({ queryKey: ['orders'] });
    void queryClient.invalidateQueries({ queryKey: ['finance'] });
    onChanged?.();
  };
  const columns: ProColumns<Correction>[] = [
    {
      title: '申请时间',
      dataIndex: 'requestedAt',
      width: 155,
      render: (_, row) => formatDate(row.requestedAt, 'minute'),
    },
    { title: '申请人', dataIndex: 'requestedByName', width: 100 },
    {
      title: '补录方式',
      dataIndex: 'operation',
      width: 100,
      render: (_, row) => operationLabels[row.operation ?? 0] || '-',
    },
    { title: '原因', dataIndex: 'reason', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, row) => (
        <Tag>{statusLabels[row.status || ''] || row.status}</Tag>
      ),
    },
    {
      title: '账务进度',
      dataIndex: 'financialStatus',
      width: 180,
      render: (_, row) =>
        financialLabels[row.financialStatus || ''] ||
        row.financialStatus ||
        '-',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 160,
      align: 'right',
      render: (_, row) => (
        <Space>
          {row.canApprove && (
            <Button type="link" size="small" onClick={() => setReview(row)}>
              审核
            </Button>
          )}
          {row.canWithdraw && (
            <Button
              type="link"
              size="small"
              onClick={() =>
                app.modal
                  .confirm({
                    title: '撤回这笔费用补录申请？',
                    onOk: async () => {
                      const response =
                        await orderFeeServiceWithdrawOrderFeeCorrection(
                          { orderId, id: row.id || '' },
                          {
                            orderId,
                            id: row.id || '',
                            expectedVersion: row.version,
                          },
                        );
                      if (!isWriteResponse(response, true))
                        return Promise.reject();
                      if (!response?.success) throw new Error('撤回未成功');
                      refresh();
                    },
                  })
                  .then(
                    () => {},
                    () => {},
                  )
              }
            >
              撤回
            </Button>
          )}
          {row.canApprove && (
            <Button
              type="link"
              size="small"
              danger
              onClick={() =>
                confirmWithReason(
                  app,
                  '驳回费用补录',
                  async (reason): Promise<false | undefined> => {
                    const response =
                      await orderFeeServiceRejectOrderFeeCorrection(
                        { orderId, id: row.id || '' },
                        {
                          orderId,
                          id: row.id || '',
                          expectedVersion: row.version,
                          reason,
                        },
                      );
                    if (!isWriteResponse(response, true)) return false;
                    if (!response?.success) throw new Error('驳回未成功');
                    refresh();

                    return undefined;
                  },
                  { signal: getModalOwnerSignal() },
                )
              }
            >
              驳回
            </Button>
          )}
        </Space>
      ),
    },
  ];
  const settings = useColumnSettings({
    tableKey: 'orders:fee-corrections',
    columns,
  });
  return (
    <>
      <Alert
        type="info"
        showIcon
        title="批准后保留原费用与审批记录，财务按更正结果处理账单、票据和收付款。业务结案状态不受影响。"
        style={{ marginBottom: 12 }}
      />
      <ResponsiveProTable<Correction>
        key={`${organizationId}:${userId}:${orderId}`}
        rowKey="id"
        actionRef={actionRef}
        search={false}
        columns={settings.columns}
        options={{ setting: false, density: false }}
        toolBarRender={() =>
          [renderCreate?.(refresh), settings.entry].filter(Boolean)
        }
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        request={async ({ current, pageSize }) => {
          const response = await orderFeeServiceListOrderFeeCorrectionRequests({
            orderId,
            page: current || 1,
            pageSize: pageSize || 10,
          });
          if (!response) throw new Error('补录申请加载失败');
          return toTableRequest(response);
        }}
        onLoad={(records) => onRecordsLoaded?.(records.length > 0)}
        expandable={{
          expandedRowRender: (row) => <FeeCorrectionComparison request={row} />,
        }}
      />
      <GuardedModalForm<{ comment: string }>
        title="审核费用补录"
        open={Boolean(review)}
        width={860}
        modalProps={{ destroyOnHidden: true }}
        onOpenChange={(open) => {
          if (!open) setReview(undefined);
        }}
        submitter={{
          searchConfig: { submitText: '批准更正' },
          submitButtonProps: {
            disabled:
              !preview.data?.previewToken ||
              preview.isFetching ||
              preview.isError,
          },
        }}
        onFinish={async ({ comment }) => {
          if (
            !review?.id ||
            !preview.data?.previewToken ||
            preview.isFetching ||
            preview.isError
          )
            return false;
          try {
            const response = await orderFeeServiceApproveOrderFeeCorrection(
              { orderId, id: review.id },
              {
                orderId,
                id: review.id,
                expectedVersion: review.version,
                previewToken: preview.data.previewToken,
                comment,
              },
            );
            if (!isWriteResponse(response, true)) {
              void preview.refetch();
              return false;
            }
            app.message.success('费用补录已批准');
            refresh();
            return true;
          } catch {
            void preview.refetch();
            return false;
          }
        }}
      >
        {review && <FeeCorrectionComparison request={review} />}
        <Spin spinning={preview.isFetching}>
          {preview.isError && (
            <Alert
              type="error"
              title="审批依据已变化或不可读取，请刷新预览"
              action={
                <Button onClick={() => void preview.refetch()}>重新预览</Button>
              }
            />
          )}
          {preview.data && (
            <>
              <Descriptions
                title={`利润影响（${preview.data.baseCurrency || ''}）`}
                column={3}
                items={[
                  {
                    key: 'receivable',
                    label: '应收',
                    children: `${formatAmount(preview.data.currentReceivable)} → ${formatAmount(preview.data.projectedReceivable)}`,
                  },
                  {
                    key: 'payable',
                    label: '成本',
                    children: `${formatAmount(preview.data.currentPayable)} → ${formatAmount(preview.data.projectedPayable)}`,
                  },
                  {
                    key: 'profit',
                    label: '利润',
                    children: `${formatAmount(preview.data.currentProfit)} → ${formatAmount(preview.data.projectedProfit)}`,
                  },
                ]}
              />
              <ResponsiveTable<API.FeeCorrectionCommissionImpactData>
                rowKey={(row) =>
                  `${row.commissionId}:${row.orderId}:${row.employeeId}`
                }
                size="small"
                pagination={false}
                dataSource={preview.data.commissionImpacts || []}
                columns={[
                  { title: '提成单', dataIndex: 'commissionNo' },
                  { title: '员工', dataIndex: 'employeeName' },
                  {
                    title: '当前提成',
                    dataIndex: 'currentAmount',
                    render: (value) => formatAmount(value),
                  },
                  {
                    title: '更正后提成',
                    dataIndex: 'targetAmount',
                    render: (value) => formatAmount(value),
                  },
                  {
                    title: '本次调整',
                    dataIndex: 'difference',
                    render: (value) => formatAmount(value),
                  },
                  { title: '币种', dataIndex: 'baseCurrency' },
                ]}
              />
            </>
          )}
        </Spin>
        <ProFormTextArea
          name="comment"
          label="审批意见"
          rules={[
            feeReviewCommentRule,
            {
              required: Number(preview.data?.projectedProfit) < 0,
              whitespace: true,
              message: '更正后亏损，请填写审批意见',
            },
          ]}
          fieldProps={feeReviewCommentFieldProps}
        />
      </GuardedModalForm>
    </>
  );
}
