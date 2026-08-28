import { SectionCard } from '@/components/ui';
import { history } from '@umijs/max';
import {
  Alert,
  Card,
  Col,
  Descriptions,
  Row,
  Space,
  Statistic,
  Tag,
} from 'antd';
import dayjs from 'dayjs';
import React from 'react';

type OrderFeeHeaderProps = {
  order: API.Order;
  kind: string;
  orderId: string;
  configTitle: string;
  customerName?: string;
  financeLocked: boolean;
  financeLockReason?: string;
  financeLockCommissionNos: string[];
  receivableSummary: { totalAmount: number; count: number };
  payableSummary: { totalAmount: number; count: number };
  profitCny: number;
  profitRate: string;
};

export default function OrderFeeHeader({
  order,
  kind,
  orderId,
  configTitle,
  customerName,
  financeLocked,
  financeLockReason,
  financeLockCommissionNos,
  receivableSummary,
  payableSummary,
  profitCny,
  profitRate,
}: OrderFeeHeaderProps) {
  return (
    <>
      {financeLocked && (
        <Alert
          type="warning"
          showIcon
          title="该订单费用已进入财务锁定"
          description={`${financeLockReason || '关联提成已确认或已发放，原费用事实不可再修改。'}${financeLockCommissionNos.length > 0 ? ` 关联提成：${financeLockCommissionNos.join('、')}。` : ''} 后续提成差异请在提成管理中新增独立调整记录。`}
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 1. 基础信息卡片 */}
      <SectionCard title="订单基础信息" style={{ marginBottom: 16 }}>
        <Descriptions
          size="small"
          column={{ xs: 1, sm: 2, md: 3, lg: 4, xl: 4 }}
        >
          <Descriptions.Item label="订单编号">
            <a
              style={{
                fontWeight: 600,
                color: '#1677ff',
                fontFamily: 'monospace',
              }}
              onClick={() => history.push(`/orders/${kind}/${orderId}`)}
            >
              {order.orderNo || order.id}
            </a>
          </Descriptions.Item>
          <Descriptions.Item label="委托单位">
            {customerName || order.customerId || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="业务类型">{configTitle}</Descriptions.Item>
          <Descriptions.Item label="贸易条款">
            {order.tradeTerm ? 'FOB / CIF' : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="主单号 (MBL)">
            {order.shippingDocuments?.[0]?.masterNo || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="船名航次">
            {order.vesselVoyage || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="起运港 (POL)">
            {order.originLocationId || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="目的港 (POD)">
            {order.destinationLocationId || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="承运人 (船司)">
            {order.carrierId || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="订舱代理">
            {order.bookingAgentId || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="ETD">
            {order.etd ? dayjs(order.etd).format('YYYY-MM-DD') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="件重尺">
            {order.totalPackages || '-'} 件 / {order.totalGrossWeightKg || '-'}{' '}
            kg / {order.totalVolumeCbm || '-'} m³
          </Descriptions.Item>
        </Descriptions>
      </SectionCard>

      {/* 2. 费用汇总统计指标卡 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <Card
            variant="borderless"
            style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
          >
            <Statistic
              title={<span style={{ color: '#64748b' }}>应收总计</span>}
              value={receivableSummary.totalAmount}
              precision={2}
              prefix="¥"
              styles={{ content: { color: '#1677ff', fontWeight: 700 } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card
            variant="borderless"
            style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
          >
            <Statistic
              title={<span style={{ color: '#64748b' }}>应付总计</span>}
              value={payableSummary.totalAmount}
              precision={2}
              prefix="¥"
              styles={{ content: { color: '#fa8c16', fontWeight: 700 } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card
            variant="borderless"
            style={{ borderRadius: 6, border: '1px solid #f0f0f0' }}
          >
            <Statistic
              title={
                <Space>
                  <span style={{ color: '#64748b' }}>预计毛利</span>
                  <Tag color={profitCny >= 0 ? 'success' : 'error'}>
                    {profitRate}%
                  </Tag>
                </Space>
              }
              value={profitCny}
              precision={2}
              prefix="¥"
              styles={{
                content: {
                  color: profitCny >= 0 ? '#52c41a' : '#ff4d4f',
                  fontWeight: 700,
                },
              }}
            />
          </Card>
        </Col>
      </Row>
    </>
  );
}
