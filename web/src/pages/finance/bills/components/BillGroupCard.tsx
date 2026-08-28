import { PlusOutlined } from '@ant-design/icons';
import type { ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import {
  AutoComplete,
  Button,
  Card,
  Col,
  DatePicker,
  Divider,
  Form,
  Input,
  InputNumber,
  Row,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React from 'react';

const { Text } = Typography;

type BillGroupCardProps = {
  group: API.BillBatchPreviewGroup;
  index: number;
  invoiceProfilesMap: Record<string, API.PartnerInvoiceProfile[]>;
  feeColumns: ProColumns<API.FeeLedgerItem>[];
  directionText: (dir?: string) => string;
  onOpenQuickAddProfile: (
    index: number,
    partnerId?: string,
    partnerName?: string,
  ) => void;
};

export default function BillGroupCard({
  group,
  index,
  invoiceProfilesMap,
  feeColumns,
  directionText,
  onOpenQuickAddProfile,
}: BillGroupCardProps) {
  const profileOptions = (() => {
    const profiles = (
      invoiceProfilesMap[group.settlementPartyId || ''] || []
    ).filter((p) => p.enabled !== false);
    const list = profiles.map((p) => ({
      value: p.invoiceTitle || '',
      label: (
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <span style={{ fontWeight: 500 }}>{p.invoiceTitle}</span>
          <Space size="small">
            {p.isDefault && (
              <Tag color="blue" style={{ margin: 0, fontSize: 11 }}>
                默认
              </Tag>
            )}
            {p.taxpayerIdentificationNo && (
              <Text type="secondary" style={{ fontSize: 11 }}>
                税号: {p.taxpayerIdentificationNo}
              </Text>
            )}
          </Space>
        </div>
      ),
    }));
    if (!list.some((opt) => opt.value === group.settlementPartyName)) {
      list.unshift({
        value: group.settlementPartyName || '',
        label: <span>{group.settlementPartyName}（结算单位全称）</span>,
      });
    }
    return list;
  })();

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
          <Tag
            color={group.direction === 'RECEIVABLE' ? 'green' : 'volcano'}
          >
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
            label={
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  width: '100%',
                }}
              >
                <span>对账抬头</span>
                <Tooltip title="为该结算单位新增开票抬头并自动选中">
                  <Button
                    type="link"
                    size="small"
                    icon={<PlusOutlined />}
                    style={{
                      padding: 0,
                      height: 'auto',
                      fontSize: 12,
                      fontWeight: 'normal',
                    }}
                    onClick={() =>
                      onOpenQuickAddProfile(
                        index,
                        group.settlementPartyId,
                        group.settlementPartyName,
                      )
                    }
                  >
                    新增抬头
                  </Button>
                </Tooltip>
              </div>
            }
            rules={[
              {
                required: true,
                whitespace: true,
                message: '请输入对账抬头',
              },
              { max: 200, message: '对账抬头不能超过 200 字' },
            ]}
          >
            <AutoComplete
              options={profileOptions}
              popupRender={(menu) => (
                <>
                  {menu}
                  <Divider style={{ margin: '4px 0' }} />
                  <div
                    style={{
                      padding: '6px 12px',
                      cursor: 'pointer',
                      color: '#1677ff',
                      fontSize: 12,
                      display: 'flex',
                      alignItems: 'center',
                      gap: 4,
                      background: '#f6faff',
                    }}
                    onMouseDown={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      onOpenQuickAddProfile(
                        index,
                        group.settlementPartyId,
                        group.settlementPartyName,
                      );
                    }}
                  >
                    <PlusOutlined /> 为【{group.settlementPartyName}】新增开票抬头
                  </div>
                </>
              )}
              placeholder="输入或下拉选择对账抬头"
            />
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
