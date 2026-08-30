import { Col, Space, Tag, Timeline, Typography } from 'antd';
import React from 'react';
import type { OrderFormTemplateSection } from '@/components/ui';
import { formatDate } from '@/utils/format';

const { Text } = Typography;

export function buildOrderAuditTimelineSection(
  order?: API.Order,
): OrderFormTemplateSection {
  return {
    key: 'order-operation-logs',
    title: '操作记录与历史流转日志',
    extra: <Tag color="geekblue">全生命周期审计</Tag>,
    content: (
      <Col span={24}>
        <div style={{ padding: '8px 12px' }}>
          <Timeline
            items={[
              {
                color: 'green',
                children: (
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
                children: (
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
      </Col>
    ),
  };
}
