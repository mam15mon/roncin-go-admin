import { ModalForm, ProFormTextArea } from '@ant-design/pro-components';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Spin,
  type TableColumnsType,
  Typography,
} from 'antd';
import React, { useMemo, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { FeeReviewStatus } from '@/features/orders/fee-review';
import {
  orderFeeServiceApproveOrderFeeReviews,
  orderFeeServicePreviewOrderFeeReviews,
} from '@/services/roncin/orderFeeService';
import { formatAmount, formatDate, trimDecimal } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import { FeeReviewDetails } from './FeeReviewDetails';

export function BatchFeeReviewModal({
  orderIds,
  onClose,
  onCompleted,
}: {
  orderIds: string[];
  onClose: () => void;
  onCompleted: (ids: string[]) => void;
}) {
  const { message } = App.useApp();
  const { initialState } = useInitialState();
  const client = useQueryClient();
  const [results, setResults] = useState<API.OrderFeeReviewResult[]>();
  const [submittedComments, setSubmittedComments] = useState<
    Record<string, string>
  >({});
  const actionKeys = useMemo(
    () => new Map(orderIds.map((id) => [id, generateUUID()])),
    [orderIds],
  );
  const preview = useQuery({
    queryKey: [
      'order-fee-review',
      'batch',
      initialState?.currentUser?.currentOrganization?.id,
      orderIds,
    ],
    enabled: orderIds.length > 0,
    queryFn: async () =>
      (await orderFeeServicePreviewOrderFeeReviews({ orderIds })).data ?? [],
    meta: { errorMessage: '批量费用审核预览失败' },
  });
  const mutation = useMutation({
    mutationFn: orderFeeServiceApproveOrderFeeReviews,
  });
  // 本次窗口始终绑定首次成功预览，后台刷新不得替换人工已核对的费用版本。
  const snapshotRows = useRef<API.OrderFeeReviewResult[] | undefined>(
    undefined,
  );
  if (!snapshotRows.current && preview.data)
    snapshotRows.current = preview.data;
  const rows = snapshotRows.current ?? [];
  const processedIds = new Set((results ?? []).map((row) => row.orderId));
  const actionable = rows.filter(
    (row) =>
      row.success && row.data?.canApprove && !processedIds.has(row.orderId),
  );
  const resultMap = new Map((results ?? []).map((row) => [row.orderId, row]));
  const columns: TableColumnsType<API.OrderFeeReviewResult> = [
    {
      key: 'order',
      title: '订单',
      width: 145,
      fixed: 'left',
      render: (_, row) => (
        <Typography.Text strong>
          {row.data?.orderNo || row.orderId}
        </Typography.Text>
      ),
    },
    {
      key: 'status',
      title: '审核状态',
      width: 120,
      render: (_, row) => (
        <FeeReviewStatus
          status={
            resultMap.get(row.orderId)?.success ? 'APPROVED' : row.data?.status
          }
          pendingSupplement={row.data?.hasPendingSupplement}
        />
      ),
    },
    ...(['receivable', 'payable'] as const).map((direction) => ({
      key: direction,
      title: direction === 'receivable' ? '应收 / 笔数' : '应付 / 笔数',
      width: 145,
      align: 'right' as const,
      render: (_: unknown, row: API.OrderFeeReviewResult) => (
        <div style={{ fontVariantNumeric: 'tabular-nums' }}>
          <Typography.Text strong>
            {formatAmount(row.data?.summary?.[direction])}
          </Typography.Text>
          <div style={{ fontSize: 12, color: '#8c8c8c' }}>
            {row.data?.summary?.baseCurrency} ·{' '}
            {row.data?.summary?.[
              direction === 'receivable' ? 'receivableCount' : 'payableCount'
            ] ?? 0}
            笔
          </div>
        </div>
      ),
    })),
    {
      key: 'profit',
      title: '整票利润',
      width: 145,
      align: 'right',
      render: (_, row) => (
        <div style={{ fontVariantNumeric: 'tabular-nums' }}>
          <Typography.Text
            strong
            type={Number(row.data?.summary?.profit) < 0 ? 'danger' : undefined}
          >
            {formatAmount(row.data?.summary?.profit)}
          </Typography.Text>
          <div
            style={{
              fontSize: 12,
              color:
                Number(row.data?.summary?.profit) < 0 ? '#cf1322' : '#8c8c8c',
            }}
          >
            {row.data?.summary?.baseCurrency}
            {Number(row.data?.summary?.profit) < 0 ? ' · 亏损' : ''}
          </div>
        </div>
      ),
    },
    {
      key: 'profitRate',
      title: '利润率',
      width: 85,
      align: 'right',
      render: (_, row) =>
        row.data?.summary?.profitRate == null ||
        row.data.summary.profitRate === ''
          ? '不可计算'
          : `${trimDecimal(row.data.summary.profitRate)}%`,
    },
    {
      key: 'submittedBy',
      title: '提交人 / 时间',
      width: 155,
      render: (_, row) => (
        <div>
          {row.data?.summary?.submittedByName || '-'}
          <div style={{ fontSize: 12, color: '#8c8c8c' }}>
            {formatDate(row.data?.summary?.submittedAt, 'minute')}
          </div>
        </div>
      ),
    },
    {
      key: 'comment',
      title: '审批意见（亏损必填）',
      width: 250,
      render: (_, row) => {
        const result = resultMap.get(row.orderId);
        const loss = Number(row.data?.summary?.profit) < 0;
        return row.data?.canApprove && !result ? (
          <ProFormTextArea
            name={['comments', row.orderId ?? '']}
            formItemProps={{ style: { marginBottom: 0 } }}
            fieldProps={{
              'aria-label': loss ? '审批意见（亏损原因，必填）' : '审批意见',
              placeholder: loss ? '请填写该票亏损原因' : '填写意见（选填）',
              autoSize: { minRows: 1, maxRows: 3 },
              maxLength: 2000,
            }}
            rules={
              loss
                ? [
                    {
                      required: true,
                      whitespace: true,
                      message: '请为该亏损订单填写审批意见',
                    },
                  ]
                : []
            }
          />
        ) : (
          <Typography.Text type="secondary">
            {result
              ? submittedComments[row.orderId ?? '']?.trim() || '未填写意见'
              : '不可批准'}
          </Typography.Text>
        );
      },
    },
    {
      key: 'result',
      title: '处理结果 / 限制',
      width: 200,
      render: (_, row) => {
        const result = resultMap.get(row.orderId);
        if (result)
          return (
            <Typography.Text type={result.success ? 'success' : 'danger'}>
              {result.success
                ? '已批准'
                : result.message || '批准失败，请刷新该票后重新审核'}
            </Typography.Text>
          );
        if (!row.success)
          return (
            <Typography.Text type="danger">
              {row.message || '该票无法预览'}
            </Typography.Text>
          );
        return (
          <Typography.Text type="secondary">
            {row.data?.canApprove
              ? '待人工核对'
              : row.data?.blockedReasons?.join('；') ||
                '当前无权审批或该票不处于待审核状态'}
          </Typography.Text>
        );
      },
    },
  ];
  const columnSettings = useColumnSettings<
    TableColumnsType<API.OrderFeeReviewResult>[number]
  >({
    tableKey: 'orders:fee-review-batch',
    columns,
    lockVisibleKeys: ['order', 'comment', 'result'],
  });
  return (
    <ModalForm<{ comments?: Record<string, string> }>
      open
      title={`批量审核订单费用包（${orderIds.length}票）`}
      width="96vw"
      modalProps={{ destroyOnHidden: true, onCancel: onClose }}
      submitter={{
        searchConfig: { submitText: '批准可审核订单' },
        submitButtonProps: {
          disabled:
            preview.isFetching || preview.isError || actionable.length === 0,
        },
      }}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      onFinish={async (values) => {
        try {
          const response = await mutation.mutateAsync({
            items: actionable.map((row) => ({
              orderId: row.orderId,
              expectedReviewVersion: row.data?.reviewVersion,
              expectedContentVersion: row.data?.contentVersion,
              submissionId: row.data?.submissionId,
              previewToken: row.data?.previewToken,
              idempotencyKey: actionKeys.get(row.orderId ?? ''),
              comment: values.comments?.[row.orderId ?? '']?.trim(),
            })),
          });
          const newResults = response.data ?? [];
          setSubmittedComments((previous) => ({
            ...previous,
            ...values.comments,
          }));
          setResults((previous) => [
            ...(previous ?? []).filter((row) => row.success),
            ...newResults,
          ]);
          onCompleted(
            newResults
              .filter((row) => row.success)
              .map((row) => row.orderId ?? ''),
          );
          await client.invalidateQueries({
            queryKey: ['order-fee-review'],
            refetchType: 'none',
          });
          message.info(
            `批准成功 ${newResults.filter((row) => row.success).length} 票，失败 ${newResults.filter((row) => !row.success).length} 票`,
          );
          return false;
        } catch {
          return false;
        }
      }}
    >
      <div
        style={{
          marginBottom: 10,
          display: 'flex',
          justifyContent: 'space-between',
          gap: 12,
          flexWrap: 'wrap',
        }}
      >
        <Typography.Text type="secondary">
          一订单一包 · 全部应收、应付一起审核 · 亏损订单逐票填写原因
        </Typography.Text>
        <Typography.Text type="secondary">
          可批准 {actionable.length} 票
          {results
            ? ` · 已批准 ${results.filter((row) => row.success).length} 票 · 失败 ${results.filter((row) => !row.success).length} 票`
            : ''}
        </Typography.Text>
        {columnSettings.entry}
      </div>
      {preview.isPending && <Spin />}
      {preview.isError && (
        <Alert
          type="error"
          title="预览失败，不能批准"
          action={
            <Button onClick={() => void preview.refetch()}>重新加载</Button>
          }
        />
      )}
      {!preview.isError && (
        <Table<API.OrderFeeReviewResult>
          size="small"
          bordered
          rowKey={(row) => row.orderId ?? ''}
          pagination={false}
          dataSource={rows}
          scroll={{ x: 1490, y: 'calc(100vh - 300px)' }}
          onRow={(row) => ({
            style: {
              background:
                Number(row.data?.summary?.profit) < 0 ? '#fff2f0' : undefined,
            },
          })}
          expandable={{
            columnWidth: 48,
            rowExpandable: (row) => Boolean(row.data),
            expandIcon: ({ expanded, onExpand, record }) =>
              record.data ? (
                <Button
                  size="small"
                  type="text"
                  aria-label={`${expanded ? '收起' : '展开'} ${record.data.orderNo || record.orderId} 费用明细`}
                  onClick={(event) => onExpand(record, event)}
                >
                  {expanded ? '−' : '+'}
                </Button>
              ) : null,
            expandedRowRender: (row) =>
              row.data ? (
                <FeeReviewDetails data={row.data} showHistory={false} />
              ) : null,
          }}
          columns={columnSettings.columns}
        />
      )}
    </ModalForm>
  );
}
