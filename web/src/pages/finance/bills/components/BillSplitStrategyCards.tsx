import { Alert, Card, Col, Descriptions, Row, Space, Switch, Typography } from 'antd';
import React from 'react';

const { Text, Title } = Typography;

type BillSplitStrategyCardsProps = {
  splitByOrder: boolean;
  setSplitByOrder: (val: boolean) => void;
  splitByTaxRate: boolean;
  setSplitByTaxRate: (val: boolean) => void;
  selectedCount: number;
};

export default function BillSplitStrategyCards({
  splitByOrder,
  setSplitByOrder,
  splitByTaxRate,
  setSplitByTaxRate,
  selectedCount,
}: BillSplitStrategyCardsProps) {
  return (
    <Card>
      <Title level={5}>拆单维度</Title>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 20 }}
        title="收付方向、结算单位、原币和本币始终是强制拆单维度"
        description="不同强制维度的费用绝不会进入同一张账单。下面两个开关只控制额外拆分，不会放宽服务端的财务边界。"
      />
      <Row gutter={[24, 16]}>
        <Col xs={24} lg={12}>
          <Card size="small" title="按订单拆分">
            <Space vertical>
              <Switch
                checked={splitByOrder}
                checkedChildren="已启用"
                unCheckedChildren="未启用"
                onChange={setSplitByOrder}
              />
              <Text type="secondary">
                启用后每个业务订单独立成账；关闭时允许同一结算单位的多票订单汇总对账。
              </Text>
            </Space>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card size="small" title="按税率拆分">
            <Space vertical>
              <Switch
                checked={splitByTaxRate}
                checkedChildren="已启用"
                unCheckedChildren="未启用"
                onChange={setSplitByTaxRate}
              />
              <Text type="secondary">
                启用后不同税率独立成账；关闭时税率仍会逐费用行固化，后续开票可按行处理。
              </Text>
            </Space>
          </Card>
        </Col>
      </Row>
      <Descriptions size="small" column={2} style={{ marginTop: 20 }}>
        <Descriptions.Item label="已选费用">
          {selectedCount} 笔
        </Descriptions.Item>
        <Descriptions.Item label="预览机制">
          服务端实时拆分并签发快照令牌
        </Descriptions.Item>
      </Descriptions>
    </Card>
  );
}
