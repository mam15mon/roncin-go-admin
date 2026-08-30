import { Descriptions, Space, Table, Tag } from 'antd';
import React from 'react';
import { DItem, DescriptionsDetailDrawer } from '@/components/ui';
import { FinanceBillStatus } from '@/enums.generated';
import { statusOptions } from './billConstants';

interface BillDetailDrawerProps {
  open: boolean;
  loading: boolean;
  detail?: API.FinanceBill;
  onClose: () => void;
}

export default function BillDetailDrawer({
  open,
  loading,
  detail,
  onClose,
}: BillDetailDrawerProps) {
  return (
    <DescriptionsDetailDrawer
      title={(current) => `账单详情 ${current?.billNo || ''}`}
      open={open}
      detail={detail}
      size={1020}
      loading={loading}
      column={3}
      onClose={onClose}
      descriptions={(detail) => (
        <>
            <Descriptions.Item label="状态">
              <Tag
                color={
                  statusOptions[
                    detail.status ??
                      FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT
                  ]?.color
                }
              >
                {
                  statusOptions[
                    detail.status ??
                      FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT
                  ]?.text
                }
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="方向">
              {detail.direction === 'RECEIVABLE' ? '应收' : '应付'}
            </Descriptions.Item>
            <Descriptions.Item label="结算单位">
              {detail.settlementPartyName}
            </Descriptions.Item>
            <DItem label="建单批次">{detail.batchNo}</DItem>
            <DItem label="对账抬头">{detail.statementTitle}</DItem>
            <Descriptions.Item label="含税总额">
              <strong style={{ color: '#262626' }}>
                {detail.totalAmount} {detail.currency}
              </strong>
            </Descriptions.Item>
            <Descriptions.Item label="不含税金额">
              {detail.netAmount
                ? `${detail.netAmount} ${detail.currency}`
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="税额汇总">
              {detail.taxAmount
                ? `${detail.taxAmount} ${detail.currency}`
                : '0.00'}
            </Descriptions.Item>
            <Descriptions.Item label="账单汇率">
              {detail.exchangeRate ? (
                <Space size={4}>
                  <span>{detail.exchangeRate}</span>
                  <Tag
                    color={
                      detail.exchangeRateSource === 'MANUAL'
                        ? 'purple'
                        : detail.exchangeRateSource === 'BASE_CURRENCY'
                          ? 'default'
                          : 'blue'
                    }
                  >
                    {detail.exchangeRateSource === 'MANUAL'
                      ? '手工'
                      : detail.exchangeRateSource === 'BASE_CURRENCY'
                        ? '本币'
                        : '系统'}
                  </Tag>
                </Space>
              ) : (
                '-'
              )}
            </Descriptions.Item>
            <Descriptions.Item label="折本币总额">
              <strong style={{ color: '#1677ff' }}>
                {detail.baseCurrencyAmount} {detail.baseCurrency}
              </strong>
            </Descriptions.Item>
            <Descriptions.Item label="汇率生效日期">
              {detail.exchangeRateDate || detail.billDate || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="账单日期">
              {detail.billDate}
            </Descriptions.Item>
            <Descriptions.Item label="到期日">
              {detail.dueDate || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="账期">
              {detail.paymentTermsDays == null
                ? '-'
                : `${detail.paymentTermsDays} 天`}
            </Descriptions.Item>
            <Descriptions.Item label="费用笔数">
              {detail.feeCount} 笔
            </Descriptions.Item>
            <DItem label="备注" span={3}>{detail.note}</DItem>
            {detail.cancellationReason && (
              <Descriptions.Item label="取消原因" span={3}>
                {detail.cancellationReason}
              </Descriptions.Item>
            )}
        </>
      )}
    >
      {(detail) => (
          <Table<API.FinanceBillLine>
            rowKey="id"
            size="small"
            bordered
            pagination={false}
            dataSource={detail.lines || []}
            columns={[
              { title: '订单编号', dataIndex: 'orderNo', width: 150 },
              { title: '费用代码', dataIndex: 'feeCode', width: 100 },
              { title: '费用名称', dataIndex: 'feeName', width: 130 },
              {
                title: '税率',
                dataIndex: 'taxRate',
                align: 'right',
                width: 80,
                render: (value) =>
                  value == null ? '-' : `${Number(value)}%`,
              },
              {
                title: '不含税金额',
                dataIndex: 'netAmount',
                align: 'right',
                render: (val, row) => (val ? `${val} ${row.currency}` : '-'),
              },
              {
                title: '税额',
                dataIndex: 'taxAmount',
                align: 'right',
                render: (val, row) => (val ? `${val} ${row.currency}` : '-'),
              },
              {
                title: '含税金额',
                dataIndex: 'totalAmount',
                render: (_, row) => (
                  <strong>
                    {row.totalAmount} {row.currency}
                  </strong>
                ),
                align: 'right',
              },
              {
                title: '费用折本币',
                render: (_, row) =>
                  `${row.baseCurrencyAmount} ${row.baseCurrency}`,
                align: 'right',
              },
              {
                title: '关联状态',
                render: (_, row) =>
                  row.active ? (
                    <Tag color="blue">有效</Tag>
                  ) : (
                    <Tag>已释放</Tag>
                  ),
                width: 85,
              },
            ]}
          />
      )}
    </DescriptionsDetailDrawer>
  );
}
