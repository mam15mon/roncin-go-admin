import type { TabsProps } from 'antd';
import { Col, Space, Tabs, Tag, Timeline, Typography } from 'antd';
import React from 'react';
import type { OrderFormTemplateSection } from '@/components/ui';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

/** 类型扩展向「关联与记录」卡片贡献的记录页签。 */
export interface OrderRecordTab {
  key: string;
  label: React.ReactNode;
  content: React.ReactNode;
}

export function OrderAuditTimelineContent({ order }: { order?: API.Order }) {
  return (
    <div style={{ padding: '8px 12px' }}>
      <Timeline
        items={[
          {
            color: 'green',
            content: (
              <div>
                <Space>
                  <Text strong>初始建单成功</Text>
                  <Tag color="default">系统录入</Tag>
                </Space>
                <div
                  style={{
                    fontSize: 12,
                    color: '#94a3b8',
                    marginTop: 2,
                  }}
                >
                  {formatDate(order?.createdAt)}
                </div>
              </div>
            ),
          },
          {
            color: 'blue',
            content: (
              <div>
                <Space>
                  <Text strong>业务信息与配舱已录入</Text>
                  <Tag color="processing">主操作员</Tag>
                </Space>
                <div
                  style={{
                    fontSize: 12,
                    color: '#94a3b8',
                    marginTop: 2,
                  }}
                >
                  {formatDate(order?.updatedAt)}
                </div>
              </div>
            ),
          },
        ]}
      />
    </div>
  );
}

/**
 * 详情后置「关联与记录」卡片：操作记录为默认页签，类型扩展可追加
 * 同批订单、拆票与改配记录等专属页签；原独立后置卡片不再单独成卡。
 */
export function buildOrderRecordsSection(
  order: API.Order | undefined,
  extraTabs: OrderRecordTab[] = [],
): OrderFormTemplateSection {
  const tabItems: TabsProps['items'] = [
    {
      key: 'operation-logs',
      label: '操作记录',
      children: <OrderAuditTimelineContent order={order} />,
      forceRender: true,
    },
    ...extraTabs.map((tab) => ({
      key: tab.key,
      label: tab.label,
      children: tab.content,
      forceRender: true,
    })),
  ];
  return {
    key: 'order-related-records',
    title: '关联与记录',
    content: (
      <Col span={24}>
        <Tabs defaultActiveKey="operation-logs" items={tabItems} />
      </Col>
    ),
  };
}
