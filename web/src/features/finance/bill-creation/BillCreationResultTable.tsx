import { CheckCircleOutlined } from '@ant-design/icons';
import type { TableColumnsType } from 'antd';
import { Alert, Button, Descriptions, Empty, Result, Space, Tag } from 'antd';
import React from 'react';
import { ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { history } from '@/router/history';
import { formatAmount } from '@/utils/format';
import type {
  BillBatchGroupOutcome,
  BillBatchOutcome,
} from './billWorkbenchHelpers';
import { billBatchItemLabel } from './billWorkbenchHelpers';

type BillCreationResultTableProps = {
  result?: BillBatchOutcome;
  /** 提交时的预览分组快照，用于展示失败组（失败组没有账单）的结算单位与金额。 */
  groups?: Record<string, API.BillBatchPreviewGroup>;
  retrying?: boolean;
  onRetryFailed?: () => void;
  onAdjustFailed?: () => void;
  directionText: (dir: string) => string;
};

export default function BillCreationResultTable({
  result,
  groups,
  retrying,
  onRetryFailed,
  onAdjustFailed,
  directionText,
}: BillCreationResultTableProps) {
  const groupColumns: TableColumnsType<BillBatchGroupOutcome> = [
    {
      title: '结果',
      key: 'result',
      width: 80,
      render: (_, row) =>
        row.bill ? (
          <Tag color="success">成功</Tag>
        ) : (
          <Tag color="error">失败</Tag>
        ),
    },
    {
      title: '结算单位',
      key: 'party',
      render: (_, row) =>
        row.bill?.settlementPartyName ||
        groups?.[row.groupKey]?.settlementPartyName ||
        '-',
    },
    {
      title: '方向',
      key: 'direction',
      width: 80,
      render: (_, row) => {
        const direction =
          row.bill?.direction || groups?.[row.groupKey]?.direction;
        return direction ? directionText(direction) : '-';
      },
    },
    {
      title: '金额',
      key: 'amount',
      align: 'right',
      render: (_, row) => {
        const source = row.bill || groups?.[row.groupKey];
        return source?.totalAmount
          ? `${formatAmount(source.totalAmount)} ${source.currency || ''}`
          : '-';
      },
    },
    {
      title: '账单编号 / 失败原因',
      key: 'detail',
      render: (_, row) =>
        row.bill ? (
          row.bill.billNo
        ) : (
          <Space orientation="vertical" size={0}>
            <span style={{ color: '#cf1322' }}>{row.errorMessage}</span>
            {(row.itemErrors ?? []).map((item) => (
              <span
                key={`${item.step}:${item.itemIndex}:${item.reason}`}
                style={{ color: '#cf1322' }}
              >
                {billBatchItemLabel(item)}：{item.message}
              </span>
            ))}
          </Space>
        ),
    },
    {
      title: '到期日',
      key: 'dueDate',
      width: 120,
      render: (_, row) => row.bill?.dueDate || '-',
    },
  ];
  const groupSettings = useColumnSettings<
    TableColumnsType<BillBatchGroupOutcome>[number]
  >({
    tableKey: 'finance:bill-creation-group-results',
    columns: groupColumns,
  });

  const nettingColumns: TableColumnsType<API.FinanceNetting> = [
    { title: '对冲单号', dataIndex: 'nettingNo', width: 180 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: () => <Tag color="default">草稿</Tag>,
    },
    { title: '结算单位', dataIndex: 'settlementPartyName' },
    { title: '币种', dataIndex: 'currency', width: 80 },
    {
      title: '抵销金额',
      key: 'amount',
      align: 'right',
      render: (_, row) => `${formatAmount(row.amount)} ${row.currency}`,
    },
    {
      title: '本币抵销额',
      key: 'baseAmount',
      align: 'right',
      render: (_, row) =>
        `${formatAmount(row.baseCurrencyAmount)} ${row.baseCurrency}`,
    },
    {
      title: '分摊数',
      key: 'allocationCount',
      align: 'center',
      width: 80,
      render: (_, row) => row.allocations?.length || 0,
    },
  ];
  const nettingSettings = useColumnSettings<
    TableColumnsType<API.FinanceNetting>[number]
  >({
    tableKey: 'finance:bill-creation-nettings',
    columns: nettingColumns,
  });

  if (!result) return <Empty />;

  const { batch, succeededCount, failedCount } = result;
  const nettings = batch.nettings || [];
  const status =
    failedCount === 0 ? 'success' : succeededCount > 0 ? 'warning' : 'error';
  const title =
    failedCount === 0
      ? `批次 ${batch.batchNo || ''} 生成成功`
      : `批次 ${batch.batchNo || ''}：成功 ${succeededCount} 组，失败 ${failedCount} 组`;
  const subTitle =
    failedCount === 0
      ? `已生成 ${succeededCount} 张账单${nettings.length ? `和 ${nettings.length} 张对冲结算单` : ''}，账单无需确认即可开票、收付款和核销。`
      : '各组独立提交，失败组不影响已成功的账单；失败组的费用仍为未建账，可重试或返回调整后重新生成。';

  return (
    <>
      <Result
        status={status}
        icon={failedCount === 0 ? <CheckCircleOutlined /> : undefined}
        title={title}
        subTitle={subTitle}
        extra={
          <Space wrap>
            {failedCount > 0 ? (
              <>
                <Button
                  type="primary"
                  loading={retrying}
                  onClick={onRetryFailed}
                >
                  重试失败组
                </Button>
                <Button disabled={retrying} onClick={onAdjustFailed}>
                  返回调整失败组
                </Button>
              </>
            ) : null}
            {succeededCount > 0 ? (
              <>
                <Button onClick={() => history.push('/finance/invoices')}>
                  前往开票 / 来票
                </Button>
                <Button onClick={() => history.push('/finance/verifications')}>
                  前往核销管理
                </Button>
              </>
            ) : null}
            {nettings.length ? (
              <Button onClick={() => history.push('/finance/nettings')}>
                前往对冲管理
              </Button>
            ) : null}
          </Space>
        }
      />
      <Descriptions
        bordered
        size="small"
        column={4}
        style={{ marginBottom: 16 }}
      >
        <Descriptions.Item label="批次号">{batch.batchNo}</Descriptions.Item>
        <Descriptions.Item label="已建账费用数">
          {batch.feeCount ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label="已生成账单数">
          {batch.billCount ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label="本币合计">
          {batch.baseTotals?.length
            ? batch.baseTotals.map((total) => (
                <div key={total.baseCurrency}>
                  {formatAmount(total.totalBaseAmount ?? '0')}{' '}
                  {total.baseCurrency}
                </div>
              ))
            : '-'}
        </Descriptions.Item>
      </Descriptions>
      <div
        style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}
      >
        {groupSettings.entry}
      </div>
      <Table<BillBatchGroupOutcome>
        rowKey="groupKey"
        size="small"
        bordered
        pagination={false}
        dataSource={result.groups}
        columns={groupSettings.columns}
      />
      {nettings.length ? (
        <>
          <Alert
            type="info"
            showIcon
            style={{ marginTop: 16, marginBottom: 8 }}
            title="对冲结算单初始为草稿；请在对冲管理中确认对冲单，抵销才会占用账单余额。"
          />
          <div style={{ fontWeight: 600, marginBottom: 8, fontSize: 13 }}>
            对冲结算单
          </div>
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              marginBottom: 8,
            }}
          >
            {nettingSettings.entry}
          </div>
          <Table<API.FinanceNetting>
            rowKey="id"
            size="small"
            bordered
            pagination={false}
            dataSource={nettings}
            columns={nettingSettings.columns}
          />
        </>
      ) : null}
    </>
  );
}
