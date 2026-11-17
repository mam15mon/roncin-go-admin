import { Card, Col, Row, Switch, Typography } from 'antd';
import React from 'react';

const { Text } = Typography;

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
    <Card
      title={`已选 ${selectedCount} 笔费用，怎么拆成账单？`}
      extra={<Text type="secondary">每张账单只含一个结算单位、一种币种</Text>}
    >
      <Row gutter={[16, 16]}>
        <Col xs={24} md={12}>
          <Card size="small" title="按订单拆分">
            <Switch
              checked={splitByOrder}
              checkedChildren="已启用"
              unCheckedChildren="未启用"
              onChange={setSplitByOrder}
            />
            <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
              启用：每个订单单独一张账单。关闭：同一结算单位的多个订单合并成一张。
            </Text>
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card size="small" title="按税率拆分">
            <Switch
              checked={splitByTaxRate}
              checkedChildren="已启用"
              unCheckedChildren="未启用"
              onChange={setSplitByTaxRate}
            />
            <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
              启用：不同税率分开成账。关闭：税率不同也合并，开票时仍按每笔费用的税率处理。
            </Text>
          </Card>
        </Col>
      </Row>
    </Card>
  );
}
