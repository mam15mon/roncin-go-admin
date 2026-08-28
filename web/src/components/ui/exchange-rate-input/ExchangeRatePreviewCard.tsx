import { Button, Card, Space, Spin, Tag, Typography } from 'antd';
import React, { type ReactNode } from 'react';

const { Text } = Typography;

export interface ExchangeRatePreviewCardProps {
  amountPreview?: string;
  currency?: string;
  amountColor?: string;
  status: 'idle' | 'loading' | 'resolved' | 'missing' | 'error';
  ratePreview?: string;
  onEnableManual?: () => void;
  extra?: ReactNode;
}

export function ExchangeRatePreviewCard({
  amountPreview,
  currency,
  amountColor = '#1677ff',
  status,
  ratePreview,
  onEnableManual,
  extra,
}: ExchangeRatePreviewCardProps) {
  return (
    <Card
      size="small"
      style={{ backgroundColor: '#f8fafc', marginBottom: 16 }}
    >
      <Space separator={<span style={{ color: '#cbd5e1' }}>|</span>} size={16}>
        <div>
          <Text type="secondary">费用金额：</Text>
          <Text
            strong
            style={{
              fontSize: 16,
              color: amountColor,
            }}
          >
            {amountPreview
              ? `${currency ? `${currency} ` : ''}${amountPreview}`
              : '-'}
          </Text>
        </div>
        <div>
          <Text type="secondary">生效汇率：</Text>
          {status === 'loading' && <Spin size="small" />}
          {status === 'resolved' && (
            <Text strong style={{ color: '#52c41a' }}>
              {ratePreview}
            </Text>
          )}
          {status === 'missing' && (
            <Space size={4}>
              <Tag color="error">汇率未配置</Tag>
              {onEnableManual && (
                <Button
                  type="link"
                  size="small"
                  onClick={onEnableManual}
                  style={{ padding: 0 }}
                >
                  手动输入
                </Button>
              )}
            </Space>
          )}
          {status === 'idle' && <span style={{ color: '#bfbfbf' }}>-</span>}
        </div>
        {extra}
      </Space>
    </Card>
  );
}

export default ExchangeRatePreviewCard;
