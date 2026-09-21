import { Tooltip, Typography } from 'antd';
import type { ReactNode } from 'react';
import React, { useState } from 'react';
import { statAmountStyle, workbenchAmount } from './display';

const { Text } = Typography;

/** 三桶配色系：主题色图标 + 同系浅底（gold/blue/green 对应待确认/待发/已发放）。 */
export type FlowStatTone = 'gold' | 'blue' | 'green';

const TONE_COLORS: Record<
  FlowStatTone,
  { main: string; bg: string; border: string }
> = {
  gold: { main: '#faad14', bg: '#fff7e6', border: '#ffe58f' },
  blue: { main: '#1677ff', bg: '#e6f4ff', border: '#91caff' },
  green: { main: '#52c41a', bg: '#f6ffed', border: '#b7eb8f' },
};

type FlowStatProps = {
  tone: FlowStatTone;
  icon: ReactNode;
  label: string;
  /** 鼠标悬停提示：沿用原三桶业务口径说明。 */
  hint?: string;
  amount?: string;
  count?: number;
  currency?: string;
};

/**
 * 三桶流程 stat 小卡：浅底容器 + 圆形图标底色块 + 等宽金额 + 笔数；
 * hover 阴影浮起（translateY(-1px)）提供轻量反馈，不可点击、不下钻。
 */
export default function FlowStat({
  tone,
  icon,
  label,
  hint,
  amount,
  count,
  currency,
}: FlowStatProps) {
  const [hovered, setHovered] = useState(false);
  const colors = TONE_COLORS[tone];

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        flex: 1,
        minWidth: 150,
        padding: '12px 14px',
        borderRadius: 8,
        background: colors.bg,
        border: `1px solid ${colors.border}`,
        transition: 'box-shadow 0.2s ease, transform 0.2s ease',
        ...(hovered
          ? {
              boxShadow: '0 4px 12px rgba(15, 23, 42, 0.08)',
              transform: 'translateY(-1px)',
            }
          : null),
      }}
      data-testid={`flow-stat-${tone}`}
    >
      <Tooltip title={hint}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span
            style={{
              width: 28,
              height: 28,
              borderRadius: '50%',
              background: colors.bg,
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            <span
              style={{
                fontSize: 14,
                color: colors.main,
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              {icon}
            </span>
          </span>
          <Text type="secondary" style={{ fontSize: 13 }}>
            {label}
          </Text>
        </div>
      </Tooltip>
      <div
        style={{
          marginTop: 8,
          display: 'flex',
          alignItems: 'baseline',
          gap: 6,
          flexWrap: 'wrap',
        }}
      >
        <Text style={statAmountStyle}>{workbenchAmount(amount)}</Text>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {currency || '-'}
        </Text>
      </div>
      <Text type="secondary" style={{ fontSize: 12 }}>
        {count ?? 0} 笔
      </Text>
    </div>
  );
}
