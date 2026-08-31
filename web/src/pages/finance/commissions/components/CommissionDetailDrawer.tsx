import { PlusOutlined } from '@ant-design/icons';
import {
  Alert,
  Button,
  Descriptions,
  Drawer,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import React from 'react';
import { FinanceCommissionStatus } from '@/enums.generated';
import { formatDate } from '@/utils/format';
import {
  calculationBasisText,
  cnyExchangeRateSourceText,
  commissionStatusMeta,
  decimalText,
  getAdjustmentStatusInfo,
  personnelRoleText,
} from '../types';
import { previewColumns, renderExpandedFees } from './CommissionLineTable';

type CommissionDetailDrawerProps = {
  open: boolean;
  onClose: () => void;
  detail?: API.FinanceCommission;
  loading: boolean;
  canManage: boolean;
  onOpenAdjustment: () => void;
  onTransitionAdjustment: (
    adjustment: API.FinanceCommissionAdjustment,
    toStatus: 'CONFIRMED' | 'PAID',
  ) => void;
  onCancelAdjustment: (adjustment: API.FinanceCommissionAdjustment) => void;
};

export default function CommissionDetailDrawer({
  open,
  onClose,
  detail,
  loading,
  canManage,
  onOpenAdjustment,
  onTransitionAdjustment,
  onCancelAdjustment,
}: CommissionDetailDrawerProps) {
  const adjustmentColumns = [
    {
      title: '调整编号',
      dataIndex: 'adjustmentNo',
      key: 'adjustmentNo',
      width: 180,
    },
    { title: '归属订单', dataIndex: 'orderNo', key: 'orderNo', width: 170 },
    {
      title: '调整来源',
      dataIndex: 'sourceType',
      key: 'sourceType',
      width: 110,
      render: (value: string) =>
        value === 'VERIFICATION_REVERSAL' ? (
          <Tag color="volcano">核销撤销</Tag>
        ) : (
          <Tag>手工调整</Tag>
        ),
    },
    {
      title: '方向',
      dataIndex: 'direction',
      key: 'direction',
      width: 80,
      render: (value: string) => (
        <Tag color={value === 'INCREASE' ? 'green' : 'red'}>
          {value === 'INCREASE' ? '增提' : '冲减'}
        </Tag>
      ),
    },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      width: 140,
      align: 'right' as const,
      render: (value: string, record: API.FinanceCommissionAdjustment) => (
        <Typography.Text
          type={record.direction === 'DECREASE' ? 'danger' : 'success'}
          strong
        >
          {`${record.direction === 'DECREASE' ? '-' : '+'}${decimalText(value)} ${record.baseCurrency}`}
        </Typography.Text>
      ),
    },
    { title: '调整原因', dataIndex: 'reason', key: 'reason', ellipsis: true },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_: unknown, record: API.FinanceCommissionAdjustment) => {
        const info = getAdjustmentStatusInfo(record);
        return <Tag color={info.color}>{info.text}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: API.FinanceCommissionAdjustment) => {
        const isReversal = record.sourceType === 'VERIFICATION_REVERSAL';
        const isDecrease = record.direction === 'DECREASE';

        return (
          <Space size={8}>
            {!isReversal &&
              record.status ===
                FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT &&
              canManage && (
                <a onClick={() => onTransitionAdjustment(record, 'CONFIRMED')}>
                  确认
                </a>
              )}

            {isReversal &&
              record.status ===
                FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED &&
              canManage && (
                <a onClick={() => onTransitionAdjustment(record, 'PAID')}>
                  标记已追回
                </a>
              )}

            {!isReversal &&
              record.status ===
                FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED &&
              canManage && (
                <a onClick={() => onTransitionAdjustment(record, 'PAID')}>
                  {isDecrease ? '标记已扣回' : '标记已发放'}
                </a>
              )}

            {!isReversal &&
              (record.status ===
                FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT ||
                record.status ===
                  FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED) &&
              canManage && (
                <a onClick={() => onCancelAdjustment(record)}>取消</a>
              )}
          </Space>
        );
      },
    },
  ];

  return (
    <Drawer
      title={`提成明细${detail?.commissionNo ? ` · ${detail.commissionNo}` : ''}`}
      size={1120}
      open={open}
      loading={loading}
      extra={
        detail &&
        (detail.status ===
          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED ||
          detail.status ===
            FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID) &&
        canManage ? (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={onOpenAdjustment}
          >
            新增提成调整
          </Button>
        ) : null
      }
      onClose={onClose}
    >
      {detail ? (
        <Space vertical size={16} style={{ width: '100%' }}>
          <Descriptions
            bordered
            size="small"
            column={4}
            items={[
              {
                key: 'commissionDate',
                label: '归属日期',
                children: detail.commissionDate || '-',
              },
              {
                key: 'status',
                label: '状态',
                children: (
                  <Tag
                    color={
                      commissionStatusMeta[
                        detail.status ??
                          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
                      ]?.color
                    }
                  >
                    {
                      commissionStatusMeta[
                        detail.status ??
                          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
                      ]?.text
                    }
                  </Tag>
                ),
              },
              {
                key: 'verification',
                label: '核销编号',
                children: detail.verificationNo,
              },
              {
                key: 'employee',
                label: '提成员工',
                children: detail.employeeName,
              },
              {
                key: 'rule',
                label: '规则快照',
                children: `${detail.ruleName || '-'}（v${detail.ruleVersion || 0}）`,
              },
              {
                key: 'basis',
                label: '角色/口径',
                children: `${personnelRoleText(detail.personnelRole)} · ${calculationBasisText(detail.calculationBasis)}`,
              },
              {
                key: 'rate',
                label: '比例',
                children: `${decimalText(detail.ratePercent)}%`,
              },
              {
                key: 'coverage',
                label: '业务覆盖',
                children: `${detail.customerCount ?? 1} 个客户 · ${detail.orderCount ?? 1} 票订单 · ${detail.feeCount ?? 0} 笔费用`,
              },
              {
                key: 'profit',
                label: '已实现毛利',
                children: `${decimalText(detail.realizedProfit)} ${detail.baseCurrency}`,
              },
              {
                key: 'amount',
                label: '原始提成（本位币）',
                children: (
                  <Typography.Text strong type="success">
                    {`${decimalText(detail.commissionAmount)} ${detail.baseCurrency}`}
                  </Typography.Text>
                ),
              },
              {
                key: 'adjustmentAmount',
                label: '已确认调整（本位币）',
                children: `${Number(detail.adjustmentAmount || 0) > 0 ? '+' : ''}${decimalText(detail.adjustmentAmount)} ${detail.baseCurrency}`,
              },
              {
                key: 'effectiveAmount',
                label: '有效提成（本位币）',
                children:
                  detail.status ===
                  FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED ? (
                    <Typography.Text type="secondary" delete>
                      {`快照 ${decimalText(detail.effectiveCommissionAmount)} ${detail.baseCurrency}，不计入应发`}
                    </Typography.Text>
                  ) : (
                    <Typography.Text strong style={{ color: '#1677ff' }}>
                      {`${decimalText(detail.effectiveCommissionAmount || detail.commissionAmount)} ${detail.baseCurrency}`}
                    </Typography.Text>
                  ),
              },
              {
                key: 'cnyCommissionAmount',
                label: '原始提成（CNY）',
                children: (
                  <Typography.Text strong type="success">
                    {`${decimalText(detail.cnyCommissionAmount)} CNY`}
                  </Typography.Text>
                ),
              },
              {
                key: 'cnyAdjustmentAmount',
                label: '已确认调整（CNY）',
                children: `${Number(detail.cnyAdjustmentAmount || 0) > 0 ? '+' : ''}${decimalText(detail.cnyAdjustmentAmount)} CNY`,
              },
              {
                key: 'cnyEffectiveAmount',
                label: '有效提成（CNY）',
                children:
                  detail.status ===
                  FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED ? (
                    <Typography.Text type="secondary" delete>
                      {`快照 ${decimalText(detail.cnyEffectiveCommissionAmount)} CNY，不计入应发`}
                    </Typography.Text>
                  ) : (
                    <Typography.Text strong style={{ color: '#1677ff' }}>
                      {`${decimalText(detail.cnyEffectiveCommissionAmount)} CNY`}
                    </Typography.Text>
                  ),
              },
              {
                key: 'cnyExchangeRate',
                label: 'CNY 折算率',
                children: decimalText(detail.cnyExchangeRate),
              },
              {
                key: 'cnyExchangeRateDate',
                label: 'CNY 汇率日期',
                children: detail.cnyExchangeRateDate || '-',
              },
              {
                key: 'cnyExchangeRateSource',
                label: 'CNY 汇率来源',
                children: cnyExchangeRateSourceText(
                  detail.cnyExchangeRateSource,
                ),
              },
              {
                key: 'calculationVersion',
                label: '计算版本',
                children: detail.calculationVersion || '-',
              },
              {
                key: 'createdAt',
                label: '生成时间',
                children: formatDate(detail.createdAt),
              },
              {
                key: 'confirmedAt',
                label: '确认时间',
                children: formatDate(detail.confirmedAt),
              },
              {
                key: 'paidAt',
                label: '发放时间',
                children: formatDate(detail.paidAt),
              },
            ]}
          />
          <Alert
            type="info"
            showIcon
            title="逐订单计算快照与财务锁"
            description="下列数据是生成提成时固化的计算证据。提成确认后，关联订单的原费用会进入财务锁定；支持展开行下钻查看每笔订单的具体费用明细构成。"
          />
          <Table<API.FinanceCommissionLine>
            size="small"
            bordered
            pagination={false}
            rowKey={(line) => line.id || line.orderId || ''}
            columns={previewColumns}
            dataSource={detail.lines || []}
            scroll={{ x: 1080 }}
            expandable={{
              expandedRowRender: renderExpandedFees,
              rowExpandable: (record) =>
                Boolean(record.fees && record.fees.length > 0),
            }}
          />
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Typography.Title level={5} style={{ margin: 0 }}>
              提成调整记录
            </Typography.Title>
            <Typography.Text type="secondary">
              草稿不计入有效提成，确认后计入；已发放或已扣回调整不可取消。
            </Typography.Text>
          </Space>
          <Table<API.FinanceCommissionAdjustment>
            size="small"
            bordered
            pagination={false}
            rowKey={(item) => item.id || item.adjustmentNo || ''}
            columns={adjustmentColumns}
            dataSource={detail.adjustments || []}
            scroll={{ x: 1040 }}
            locale={{ emptyText: '暂无调整，当前有效提成等于原始提成' }}
          />
        </Space>
      ) : null}
    </Drawer>
  );
}
