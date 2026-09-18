import {
  AuditOutlined,
  FileDoneOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { history } from '@umijs/max';
import { Button, List, Space, Tag, Typography } from 'antd';
import React from 'react';
import { amountWithCurrency } from './display';

const { Text } = Typography;

type Finance = API.WorkbenchFinanceSummary;
type ApprovalItem = API.WorkbenchSupplementApprovalItem;

type Props = {
  finance: Finance;
};

/**
 * 财务与审批摘要卡：按服务端标志与计数独立出现，与个人提成模块互不排斥。
 * 卡内只提供摘要与既有页面入口；通过、驳回、确认、忽略、发放等动作一律
 * 跳转既有页面并调用原端点，不在工作台发明新操作。
 */
export default function FinanceSummaryCard({ finance }: Props) {
  const canRead = finance.canReadCommission === true;
  const canManage = finance.canManageCommission === true;
  const approvalCount = finance.pendingSupplementApprovalCount ?? 0;
  const approvals: ApprovalItem[] = finance.pendingSupplementApprovals ?? [];
  const hasApprovals = approvalCount > 0 || approvals.length > 0;

  if (!canRead && !hasApprovals) {
    return null;
  }

  return (
    <ProCard
      title={
        <Space size={8}>
          <AuditOutlined style={{ color: '#1677ff' }} />
          <span>财务与审批待办</span>
          {canRead ? <Tag color="blue">提成台账读取权限</Tag> : null}
        </Space>
      }
      headerBordered
      variant="outlined"
      data-testid="finance-summary-card"
    >
      <Space orientation="vertical" size={16} style={{ width: '100%' }}>
        {hasApprovals ? (
          <div>
            <Space size={8} style={{ marginBottom: 8 }}>
              <FileDoneOutlined style={{ color: '#faad14' }} />
              <Text strong>锁后费用补录审批</Text>
              <Tag color="warning">{approvalCount} 条待审批</Tag>
            </Space>
            {finance.supplementApprovalsTruncated === true ? (
              <Text
                type="secondary"
                style={{ display: 'block', fontSize: 12, marginBottom: 8 }}
              >
                待审批数量超出服务端扫描上限，以上计数仅为部分统计。
              </Text>
            ) : null}
            <List
              size="small"
              dataSource={approvals.slice(0, 5)}
              renderItem={(item) => (
                <List.Item
                  actions={[
                    <Button
                      key="open"
                      type="link"
                      size="small"
                      onClick={() =>
                        history.push(`/orders/sea-export/${item.orderId}/fees`)
                      }
                    >
                      前往处理
                    </Button>,
                  ]}
                >
                  <Space size={8} wrap>
                    <Text strong>{item.orderNo || '-'}</Text>
                    <Text>{item.feeName || item.feeCode || '-'}</Text>
                    <Text>
                      {amountWithCurrency(item.amount, item.currency)}
                    </Text>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {item.requestedByName || '-'} 申请于{' '}
                      {item.requestedAt || '-'}
                    </Text>
                  </Space>
                </List.Item>
              )}
            />
            {approvals.length === 0 ? (
              <Text type="secondary" style={{ fontSize: 12 }}>
                请进入对应订单的费用录入页处理待审批申请。
              </Text>
            ) : null}
          </div>
        ) : null}

        {canRead ? (
          <>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                gap: 12,
                flexWrap: 'wrap',
              }}
            >
              <Space size={8}>
                <MinusCircleOutlined style={{ color: '#ff7a45' }} />
                <Text strong>待处理冲减建议</Text>
                <Tag color="orange">
                  {finance.pendingDecreaseCount ?? 0} 条（尚未扣回）
                </Tag>
              </Space>
              <Button
                size="small"
                danger={canManage}
                onClick={() => history.push('/finance/commissions')}
              >
                {canManage ? '去处理冲减' : '查看冲减摘要'}
              </Button>
            </div>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                gap: 12,
                flexWrap: 'wrap',
              }}
            >
              <Space size={8}>
                <FileDoneOutlined style={{ color: '#52c41a' }} />
                <Text strong>已确认待发提成（组织范围）</Text>
                <Tag color="green">
                  {finance.confirmedCommissionCount ?? 0} 笔
                </Tag>
              </Space>
              <Space size={8}>
                <Button
                  size="small"
                  onClick={() => history.push('/finance/commissions')}
                >
                  提成台账
                </Button>
                <Button
                  size="small"
                  onClick={() => history.push('/finance/commissions')}
                >
                  台账导出
                </Button>
              </Space>
            </div>
          </>
        ) : null}
      </Space>
    </ProCard>
  );
}
