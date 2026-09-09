import type { ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import {
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Row,
  Space,
  Tag,
  Typography,
} from 'antd';
import React from 'react';

const { Text } = Typography;

type BillGroupCardProps = {
  group: API.BillBatchPreviewGroup;
  index: number;
  feeColumns: ProColumns<API.FeeLedgerItem>[];
  directionText: (dir?: string) => string;
};

export default function BillGroupCard({
  group,
  index,
  feeColumns,
  directionText,
}: BillGroupCardProps) {
  return (
    <Card
      key={group.groupKey}
      size="small"
      style={{
        marginBottom: 16,
        border: '1px solid #e8e8e8',
        borderRadius: 6,
      }}
      title={
        <Space wrap>
          <Tag color={group.direction === 'RECEIVABLE' ? 'green' : 'volcano'}>
            {directionText(group.direction)}
          </Tag>
          <span style={{ fontWeight: 600 }}>{group.settlementPartyName}</span>
          <Text type="secondary">
            {group.orderNo ? `订单 ${group.orderNo}` : '多订单汇总'}
          </Text>
          {group.taxRate != null && <Tag>{Number(group.taxRate)}% 税率</Tag>}
          <Tag color="geekblue">{group.fees?.length || 0} 笔费用</Tag>
        </Space>
      }
      extra={
        <Text strong style={{ color: '#1677ff', fontSize: 14 }}>
          {group.totalAmount} {group.currency}
        </Text>
      }
    >
      <Row gutter={16} style={{ marginBottom: 8 }}>
        <Col xs={24} md={8}>
          <Form.Item
            name={['groups', index, 'statementTitle']}
            label="对账抬头"
            rules={[
              {
                required: true,
                whitespace: true,
                message: '请输入对账抬头',
              },
              { max: 200, message: '对账抬头不能超过 200 字' },
            ]}
          >
            <Input placeholder="默认使用结算单位名称，可编辑" />
          </Form.Item>
        </Col>
        <Col xs={24} md={5}>
          <Form.Item
            name={['groups', index, 'billDate']}
            label="账单日期"
            rules={[{ required: true, message: '请选择账单日期' }]}
          >
            <DatePicker allowClear={false} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={4}>
          <Form.Item
            name={['groups', index, 'paymentTermsDays']}
            label="账期（天）"
          >
            <InputNumber
              min={0}
              max={3650}
              precision={0}
              placeholder="天数"
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Col>
        <Col xs={24} md={7}>
          <Form.Item
            name={['groups', index, 'note']}
            label="备注"
            rules={[{ max: 500, message: '备注不能超过 500 字' }]}
          >
            <Input maxLength={500} placeholder="选填，账单备注" />
          </Form.Item>
        </Col>
      </Row>
      <ProTable<API.FeeLedgerItem>
        rowKey="id"
        size="small"
        bordered
        search={false}
        options={false}
        toolBarRender={false}
        pagination={false}
        columns={feeColumns}
        dataSource={group.fees || []}
        scroll={{ x: 880 }}
      />
    </Card>
  );
}
