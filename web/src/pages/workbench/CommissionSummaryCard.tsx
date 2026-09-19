import {
  ArrowUpOutlined,
  DollarOutlined,
  ProfileOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import {
  Alert,
  Button,
  Empty,
  Segmented,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React, { useState } from 'react';
import { workbenchAmount } from './display';
import MyApplicationPanel from './MyApplicationPanel';

const { Text, Paragraph } = Typography;

type Summary = API.WorkbenchCommissionSummary;
type Estimated = API.WorkbenchEstimatedOpportunity;

type BucketProps = {
  label: string;
  hint?: string;
  count?: number;
  amount?: string;
  currency?: string;
  color: string;
};

function Bucket({ label, hint, count, amount, currency, color }: BucketProps) {
  return (
    <div
      style={{
        flex: 1,
        minWidth: 140,
        borderLeft: `3px solid ${color}`,
        paddingLeft: 12,
      }}
    >
      <Tooltip title={hint}>
        <Text type="secondary" style={{ fontSize: 13 }}>
          {label}
        </Text>
      </Tooltip>
      <div style={{ marginTop: 4 }}>
        <Space size={8} align="baseline">
          <Text strong style={{ fontSize: 20 }}>
            {workbenchAmount(amount)}
          </Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {currency || '-'}
          </Text>
        </Space>
      </div>
      <Text type="secondary" style={{ fontSize: 12 }}>
        {count ?? 0} 笔
      </Text>
    </div>
  );
}

type Props = {
  data: API.GetWorkbenchOverviewData;
  onOpenCommissions: () => void;
  onOpenReceivables: () => void;
  /** 月度申请提交成功后刷新 Overview；等待刷新完成再解除提交 loading。 */
  onOverviewRefresh: () => Promise<void>;
};

/**
 * 我的提成状态卡：三桶 + 冲减三阶段 + 已发累计切换 + 预计可计提。
 * 只消费服务端门禁为 true 后返回的 summary；金额为当前组织本位币口径。
 */
export default function CommissionSummaryCard({
  data,
  onOpenCommissions,
  onOpenReceivables,
  onOverviewRefresh,
}: Props) {
  const [paidRange, setPaidRange] = useState<'year' | 'month'>('year');
  const summary: Summary | undefined = data.commissionSummary;
  const currency = data.baseCurrency || summary?.baseCurrency;
  const estimated: Estimated | undefined = summary?.estimated;

  const draftCount = summary?.draftCount ?? 0;
  const confirmedCount = summary?.confirmedCount ?? 0;
  const paidCount = summary?.paidCount ?? 0;
  const decreaseDraftCount = summary?.decreaseDraftCount ?? 0;
  const decreaseConfirmedCount = summary?.decreaseConfirmedCount ?? 0;
  const decreasePaidCount = summary?.decreasePaidCount ?? 0;
  const opportunityCount = estimated?.opportunityCount ?? 0;

  const hasAnyRecord =
    draftCount > 0 ||
    confirmedCount > 0 ||
    paidCount > 0 ||
    decreaseDraftCount > 0 ||
    decreaseConfirmedCount > 0 ||
    decreasePaidCount > 0 ||
    opportunityCount > 0;

  const paidAmountText =
    paidRange === 'year'
      ? summary?.paidAmountThisYear
      : summary?.paidAmountThisMonth;

  return (
    <ProCard
      title={
        <Space size={8}>
          <DollarOutlined style={{ color: '#1677ff' }} />
          <span>我的提成</span>
          <Tag color="blue" style={{ fontSize: 12 }}>
            本位币：{currency || '-'}
          </Tag>
        </Space>
      }
      extra={
        <Button type="primary" onClick={onOpenCommissions}>
          查看提成明细
        </Button>
      }
      headerBordered
      variant="outlined"
      data-testid="commission-summary-card"
    >
      {data.nextEffectiveDate ? (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title={`您的提成方案自 ${data.nextEffectiveDate} 起生效；生效后将自动开始按方案累计提成，当前空缺不代表金额为零。`}
        />
      ) : null}

      {!hasAnyRecord ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="当前组织暂无可计提、待发或已发的提成记录"
        />
      ) : null}

      {hasAnyRecord ? (
        <>
          <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap' }}>
            <Bucket
              label="待财务确认"
              hint="提成单已生成、尚未经财务确认，金额可能调整"
              count={summary?.draftCount}
              amount={summary?.draftAmount}
              currency={currency}
              color="#faad14"
            />
            <Bucket
              label="已确认待发"
              hint="财务已确认、等待工资发放，尚未实际到账"
              count={summary?.confirmedCount}
              amount={summary?.confirmedAmount}
              currency={currency}
              color="#1677ff"
            />
            <Bucket
              label="已发放"
              hint="已完成发放的提成累计事实"
              count={summary?.paidCount}
              amount={summary?.paidAmount}
              currency={currency}
              color="#52c41a"
            />
          </div>

          <div
            style={{
              marginTop: 16,
              paddingTop: 12,
              borderTop: '1px solid #f0f0f0',
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              flexWrap: 'wrap',
            }}
          >
            <Segmented
              value={paidRange}
              onChange={(value) => setPaidRange(value as 'year' | 'month')}
              options={[
                { label: '本年已发', value: 'year' },
                { label: '本月已发', value: 'month' },
              ]}
            />
            <Space size={6} align="baseline">
              <Text strong style={{ fontSize: 18 }}>
                {workbenchAmount(paidAmountText)}
              </Text>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {currency || '-'}
              </Text>
            </Space>
          </div>

          <div
            style={{
              marginTop: 16,
              paddingTop: 12,
              borderTop: '1px solid #f0f0f0',
              display: 'flex',
              gap: 24,
              flexWrap: 'wrap',
            }}
          >
            <Bucket
              label="冲减 · 待处理"
              hint="锁后补录冲减建议草稿：尚未扣回"
              count={summary?.decreaseDraftCount}
              amount={summary?.decreaseDraftAmount}
              currency={currency}
              color="#ff7a45"
            />
            <Bucket
              label="冲减 · 已确认"
              hint="已确认冲减：尚未实际扣回"
              count={summary?.decreaseConfirmedCount}
              amount={summary?.decreaseConfirmedAmount}
              currency={currency}
              color="#faad14"
            />
            <Bucket
              label="冲减 · 已扣回"
              hint="已实际扣回的冲减金额"
              count={summary?.decreasePaidCount}
              amount={summary?.decreasePaidAmount}
              currency={currency}
              color="#722ed1"
            />
          </div>
        </>
      ) : null}

      {data.applicationSummary ? (
        <MyApplicationPanel
          summary={data.applicationSummary}
          currency={currency}
          onOverviewRefresh={onOverviewRefresh}
        />
      ) : null}

      <div
        style={{
          marginTop: 16,
          paddingTop: 12,
          borderTop: '1px solid #f0f0f0',
          display: 'flex',
          justifyContent: 'space-between',
          gap: 12,
          flexWrap: 'wrap',
        }}
      >
        <div style={{ flex: 1, minWidth: 220 }}>
          <Space size={8}>
            <ArrowUpOutlined style={{ color: '#1677ff' }} />
            <Text strong style={{ fontSize: 13 }}>
              预计可计提
            </Text>
            <Tag color="orange" style={{ fontSize: 12 }}>
              预计，非应发承诺
            </Tag>
          </Space>
          {estimated && opportunityCount > 0 ? (
            <Paragraph style={{ marginTop: 4, marginBottom: 0, fontSize: 13 }}>
              <Space size={6} align="baseline">
                <Text strong style={{ fontSize: 18 }}>
                  {workbenchAmount(estimated.estimatedAmount)}
                </Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {currency || '-'} · {opportunityCount} 笔机会
                  {estimated.hasMore === true
                    ? '（候选超出扫描上限，金额仅为部分统计）'
                    : ''}
                </Text>
              </Space>
            </Paragraph>
          ) : (
            <Paragraph style={{ marginTop: 4, marginBottom: 0, fontSize: 13 }}>
              <Space size={6} align="baseline">
                <Text type="secondary">暂无法估算</Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  尚无可匹配的有效方案与人员归属，或计提来源尚未满足计提条件
                </Text>
              </Space>
            </Paragraph>
          )}
        </div>
        <Button icon={<ProfileOutlined />} onClick={onOpenReceivables}>
          在途回款明细
        </Button>
      </div>
    </ProCard>
  );
}
