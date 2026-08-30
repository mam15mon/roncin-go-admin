import { CheckCircleOutlined } from '@ant-design/icons';
import { history, useAccess } from '@umijs/max';
import { Button, Descriptions, Empty, Result, Space, Table, Tag } from 'antd';
import React from 'react';
import { FinanceBillStatus } from '@/enums.generated';

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

  if (!result) return <Empty />;

  return (
    <>
      <Result
        status="success"
        icon={<CheckCircleOutlined />}
        title={`批次 ${result.batchNo || ''} 生成成功`}
        subTitle={`${result.feeCount || 0} 笔费用已原子生成 ${result.billCount || 0} 张账单，当前${result.bills?.every((bill) => bill.status === FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED) ? '已全部确认' : '为草稿状态'}，未发生部分成功。`}
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
          </Space>
        }
      />
      <Descriptions
        bordered
        size="small"
        column={4}
        style={{ marginBottom: 16 }}
      >
        <Descriptions.Item label="批次号">
          {result.batchNo}
        </Descriptions.Item>
        <Descriptions.Item label="费用数">
          {result.feeCount}
        </Descriptions.Item>
        <Descriptions.Item label="账单数">
          {result.billCount}
        </Descriptions.Item>
        <Descriptions.Item label="本币合计">
          {result.totalBaseAmount} {result.baseCurrency}
        </Descriptions.Item>
      </Descriptions>
      <Table<API.FinanceBill>
        rowKey="id"
        size="small"
        bordered
        pagination={false}
        dataSource={result.bills || []}
        columns={[
          { title: '账单编号', dataIndex: 'billNo', width: 180 },
          {
            title: '状态',
            dataIndex: 'status',
            width: 90,
            render: (value) =>
              value === FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED ? (
                <Tag color="green">已确认</Tag>
              ) : (
                <Tag color="gold">草稿</Tag>
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
            align: 'right',
            render: (_, row) => `${row.totalAmount} ${row.currency}`,
          },
          { title: '到期日', dataIndex: 'dueDate', width: 120 },
        ]}
      />
    </>
  );
}
