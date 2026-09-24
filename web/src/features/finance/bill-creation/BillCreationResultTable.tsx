import { CheckCircleOutlined } from '@ant-design/icons';
import type { TableColumnsType } from 'antd';
import {
  Alert,
  Button,
  Descriptions,
  Empty,
  Result,
  Space,
  Table,
  Tag,
} from 'antd';
import React from 'react';
import { useAccess } from '@/app/access';
import { useColumnSettings } from '@/components/ui/column-settings';
import { statusTag } from '@/constants/statusMeta';
import { FinanceBillStatus } from '@/enums.generated';
import { billStatusMeta } from '@/features/finance/bill-status';
import { history } from '@/router/history';
import { formatAmount } from '@/utils/format';

type BillCreationResultTableProps = {
  result?: API.FinanceBillBatch;
  confirming: boolean;
  onConfirmBatch: () => void;
  directionText: (dir: string) => string;
};

export default function BillCreationResultTable({
  result,
  confirming,
  onConfirmBatch,
  directionText,
}: BillCreationResultTableProps) {
  const access = useAccess();

  const billColumns: TableColumnsType<API.FinanceBill> = [
    { title: '账单编号', dataIndex: 'billNo', width: 180 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      // 与账单列表共用 billStatusMeta 统一映射，替换原先内联的草稿/已确认判断。
      render: (value) =>
        statusTag(
          billStatusMeta,
          value ?? FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT,
        ),
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 80,
      render: (value) => directionText(String(value)),
    },
    { title: '结算单位', dataIndex: 'settlementPartyName' },
    { title: '对账抬头', dataIndex: 'statementTitle' },
    {
      title: '金额',
      key: 'amount',
      align: 'right',
      render: (_, row) => `${formatAmount(row.totalAmount)} ${row.currency}`,
    },
    { title: '到期日', dataIndex: 'dueDate', width: 120 },
  ];
  const billSettings = useColumnSettings<
    TableColumnsType<API.FinanceBill>[number]
  >({
    tableKey: 'finance:bill-creation-results',
    columns: billColumns,
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

  return (
    <>
      <Result
        status="success"
        icon={<CheckCircleOutlined />}
        title={`批次 ${result.batchNo || ''} 生成成功`}
        subTitle={`${result.feeCount || 0} 笔费用已原子生成 ${result.billCount || 0} 张账单${result.nettings?.length ? `和 ${result.nettings.length} 张对冲结算单` : ''}，当前${result.bills?.every((bill) => bill.status === FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED) ? '已全部确认' : '为草稿状态'}，未发生部分成功。`}
        extra={
          <Space wrap>
            {access.canConfirmFinanceBills &&
              result.bills?.every(
                (bill) =>
                  bill.status === FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT,
              ) && (
                <Button
                  type="primary"
                  loading={confirming}
                  onClick={onConfirmBatch}
                >
                  确认本批全部账单
                </Button>
              )}
            <Button onClick={() => history.push('/finance/invoices')}>
              前往开票 / 来票
            </Button>
            <Button onClick={() => history.push('/finance/verifications')}>
              前往核销管理
            </Button>
            {result.nettings?.length ? (
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
        <Descriptions.Item label="批次号">{result.batchNo}</Descriptions.Item>
        <Descriptions.Item label="费用数">{result.feeCount}</Descriptions.Item>
        <Descriptions.Item label="账单数">{result.billCount}</Descriptions.Item>
        <Descriptions.Item label="本币合计">
          {formatAmount(result.totalBaseAmount ?? '0')} {result.baseCurrency}
        </Descriptions.Item>
      </Descriptions>
      <div
        style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}
      >
        {billSettings.entry}
      </div>
      <Table<API.FinanceBill>
        rowKey="id"
        size="small"
        bordered
        pagination={false}
        dataSource={result.bills || []}
        columns={billSettings.columns}
      />
      {billSettings.modal}
      {result.nettings?.length ? (
        <>
          <Alert
            type="info"
            showIcon
            style={{ marginTop: 16, marginBottom: 8 }}
            title="对冲结算单初始为草稿；请先确认本批账单，再在对冲管理中确认对冲单，抵销才会占用账单余额。"
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
            dataSource={result.nettings}
            columns={nettingSettings.columns}
          />
          {nettingSettings.modal}
        </>
      ) : null}
    </>
  );
}
