import {
  ArrowRightOutlined,
  CheckCircleOutlined,
  HourglassOutlined,
  WalletOutlined,
} from '@ant-design/icons';
import { Alert, Button, Divider, Empty, Tag, Typography } from 'antd';
import React from 'react';
import { SectionCard } from '@/components/ui';
import { amountFont, heroAmountStyle, workbenchAmount } from './display';
import FlowStat from './FlowStat';

const { Text } = Typography;

type Props = {
  data: API.GetWorkbenchOverviewData;
  onOpenCommissions: () => void;
  onOpenReceivables: () => void;
};

/** hero 副指标（本月已发）：16px、加粗、等宽数字，与主数字同屏错落展示。 */
const secondaryAmountStyle: React.CSSProperties = {
  fontSize: 16,
  fontWeight: 600,
  ...amountFont,
};

/** 三桶流程条衔接箭头：窄屏 flexWrap 时随 flex 自然折行。 */
function FlowArrow() {
  return (
    <ArrowRightOutlined
      style={{
        color: '#cbd5e1',
        fontSize: 18,
        alignSelf: 'center',
        flexShrink: 0,
      }}
    />
  );
}

/**
 * 提成总览卡：hero 双指标（本年/本月已发同屏，取代切换交互）+ 三桶流程条
 * （待财务确认 → 已确认待发 → 已发放）+ 冲减单行摘要。
 * 只消费服务端门禁为 true 后返回的 summary；金额为当前组织本位币口径。
 */
export default function CommissionOverviewCard({
  data,
  onOpenCommissions,
  onOpenReceivables,
}: Props) {
  const summary = data.commissionSummary;
  const currency = data.baseCurrency || summary?.baseCurrency;
  const estimated = summary?.estimated;

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

  // 冲减降级为单行摘要：有任一冲减记录才渲染，不再占用整层三桶大展示。
  const hasDecreaseRecord =
    decreaseDraftCount > 0 ||
    decreaseConfirmedCount > 0 ||
    decreasePaidCount > 0;

  return (
    <SectionCard
      title={
        <>
          我的提成
          <Tag color="blue" style={{ fontSize: 12 }}>
            本位币：{currency || '-'}
          </Tag>
        </>
      }
      extra={
        <>
          <Button onClick={onOpenReceivables}>在途回款</Button>
          <Button type="primary" onClick={onOpenCommissions}>
            提成明细
          </Button>
        </>
      }
      style={{ marginBottom: 0 }}
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
      ) : (
        <>
          {/* hero 行：本年已发主数字 + 本月已发副指标同屏，无需切换。 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'baseline',
              gap: 40,
              flexWrap: 'wrap',
            }}
          >
            <div>
              <Text type="secondary" style={{ fontSize: 13 }}>
                本年已发
              </Text>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'baseline',
                  gap: 6,
                  marginTop: 2,
                }}
              >
                <Text style={heroAmountStyle}>
                  {workbenchAmount(summary?.paidAmountThisYear)}
                </Text>
                <Text type="secondary" style={{ fontSize: 13 }}>
                  {currency || '-'}
                </Text>
              </div>
            </div>
            <div>
              <Text type="secondary" style={{ fontSize: 13 }}>
                本月已发
              </Text>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'baseline',
                  gap: 6,
                  marginTop: 2,
                }}
              >
                <Text style={secondaryAmountStyle}>
                  {workbenchAmount(summary?.paidAmountThisMonth)}
                </Text>
                <Text type="secondary" style={{ fontSize: 13 }}>
                  {currency || '-'}
                </Text>
              </div>
            </div>
          </div>

          {/* 三桶流程条：浅底 stat 小卡横向排列，箭头衔接。 */}
          <div
            style={{
              display: 'flex',
              alignItems: 'stretch',
              gap: 8,
              flexWrap: 'wrap',
              marginTop: 16,
            }}
          >
            <FlowStat
              tone="gold"
              icon={<HourglassOutlined />}
              label="待财务确认"
              hint="提成单已生成、尚未经财务确认，金额可能调整"
              count={summary?.draftCount}
              amount={summary?.draftAmount}
              currency={currency}
            />
            <FlowArrow />
            <FlowStat
              tone="blue"
              icon={<WalletOutlined />}
              label="已确认待发"
              hint="财务已确认、等待工资发放，尚未实际到账"
              count={summary?.confirmedCount}
              amount={summary?.confirmedAmount}
              currency={currency}
            />
            <FlowArrow />
            <FlowStat
              tone="green"
              icon={<CheckCircleOutlined />}
              label="已发放"
              hint="已完成发放的提成累计事实"
              count={summary?.paidCount}
              amount={summary?.paidAmount}
              currency={currency}
            />
          </div>

          {/* 冲减单行摘要：待处理带笔数，已确认/已扣回只列金额。 */}
          {hasDecreaseRecord ? (
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
              <Text strong style={{ fontSize: 13 }}>
                冲减
              </Text>
              <Text type="secondary" style={{ fontSize: 13 }}>
                {`待处理 ${workbenchAmount(summary?.decreaseDraftAmount)} · ${decreaseDraftCount} 笔`}
              </Text>
              <Divider orientation="vertical" />
              <Text type="secondary" style={{ fontSize: 13 }}>
                {`已确认 ${workbenchAmount(summary?.decreaseConfirmedAmount)}`}
              </Text>
              <Divider orientation="vertical" />
              <Text type="secondary" style={{ fontSize: 13 }}>
                {`已扣回 ${workbenchAmount(summary?.decreasePaidAmount)}`}
              </Text>
            </div>
          ) : null}
        </>
      )}
    </SectionCard>
  );
}
