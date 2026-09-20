import {
  ApartmentOutlined,
  AppstoreOutlined,
  BellOutlined,
  SafetyCertificateOutlined,
  TableOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { history, useAccess } from '@umijs/max';
import {
  Button,
  Descriptions,
  Empty,
  Listy,
  Space,
  Tag,
  Typography,
} from 'antd';
import React from 'react';
import { formatDate } from '@/utils/format';
import { orderFlowStatusText, orderTerminationStatusText } from './display';

const { Text, Paragraph } = Typography;

type RecentOrder = API.WorkbenchRecentOrder;
type Todos = API.WorkbenchTodoSummary;

/** 我负责的近期订单：仅本人真实协作归属的最近海运出口订单，无数据时不渲染本卡。 */
export function RecentOrdersCard({ orders }: { orders?: RecentOrder[] }) {
  const list = orders ?? [];
  if (list.length === 0) {
    return null;
  }
  return (
    <ProCard
      title={
        <Space size={8}>
          <TableOutlined style={{ color: '#1677ff' }} />
          <span>我负责的近期订单</span>
          <Tag color="blue">{list.length} 票</Tag>
        </Space>
      }
      extra={
        <Button size="small" onClick={() => history.push('/orders/sea-export')}>
          订单列表
        </Button>
      }
      headerBordered
      variant="outlined"
      data-testid="recent-orders-card"
    >
      <Listy<RecentOrder>
        rowKey="orderId"
        items={list}
        itemRender={(item) => (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 8,
              flexWrap: 'wrap',
            }}
          >
            <Space size={8} wrap>
              <Text strong copyable={Boolean(item.orderNo)}>
                {item.orderNo || '-'}
              </Text>
              <Text type="secondary">{item.customerName || '-'}</Text>
              <Tag color="processing">
                {orderFlowStatusText(item.flowStatus)}
              </Tag>
              {item.terminationStatus && item.terminationStatus !== 'ACTIVE' ? (
                <Tag color="red">
                  {orderTerminationStatusText(item.terminationStatus)}
                </Tag>
              ) : null}
              <Text type="secondary" style={{ fontSize: 12 }}>
                下单 {formatDate(item.orderDate || item.createdAt, 'date')}
              </Text>
            </Space>
            <Button
              key="open"
              type="link"
              size="small"
              onClick={() =>
                history.push(`/orders/sea-export/${item.orderId}`)
              }
            >
              订单详情
            </Button>
          </div>
        )}
      />
    </ProCard>
  );
}

/**
 * 可靠作业待办：只统计本人协作订单上可由现有事实准确判定的草稿费用与
 * 进行中异常；无任何待办时不渲染本卡，不虚构未建模的责任与时限。
 */
export function TodosCard({ todos }: { todos?: Todos }) {
  const draftFeeCount = todos?.draftFeeCount ?? 0;
  const openAbnormalCount = todos?.openAbnormalCount ?? 0;
  if (draftFeeCount === 0 && openAbnormalCount === 0) {
    return null;
  }
  return (
    <ProCard
      title={
        <Space size={8}>
          <BellOutlined style={{ color: '#faad14' }} />
          <span>我的作业待办</span>
        </Space>
      }
      headerBordered
      variant="outlined"
      data-testid="todos-card"
    >
      <Space orientation="vertical" size={8} style={{ width: '100%' }}>
        {draftFeeCount > 0 ? (
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Text>协作订单上的费用草稿</Text>
            <Space size={8}>
              <Tag color="warning">{draftFeeCount} 笔</Tag>
              <Button
                type="link"
                size="small"
                onClick={() => history.push('/finance/fees')}
              >
                费用明细
              </Button>
            </Space>
          </div>
        ) : null}
        {openAbnormalCount > 0 ? (
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Text>协作订单上的进行中异常</Text>
            <Space size={8}>
              <Tag color="red">{openAbnormalCount} 件</Tag>
              <Text type="secondary" style={{ fontSize: 12 }}>
                请进入相关订单详情处理
              </Text>
            </Space>
          </div>
        ) : null}
      </Space>
    </ProCard>
  );
}

/** 账号边界：登录身份、当前组织与授权范围的固定事实。 */
export function AccountBoundaryCard({ user }: { user?: API.CurrentUser }) {
  const orgName = user?.currentOrganization?.name || '默认组织';
  const orgCode = user?.currentOrganization?.code || '-';
  return (
    <ProCard
      title={
        <Space size={8}>
          <SafetyCertificateOutlined style={{ color: '#52c41a' }} />
          <span>账号与数据边界</span>
        </Space>
      }
      headerBordered
      variant="outlined"
      data-testid="account-card"
    >
      <Descriptions column={{ xs: 1, sm: 2 }} size="small">
        <Descriptions.Item label="用户登录名">
          <Text copyable style={{ fontFamily: 'monospace' }}>
            {user?.username || '-'}
          </Text>
        </Descriptions.Item>
        <Descriptions.Item label="显示名称">
          <Text strong>{user?.displayName || '-'}</Text>
        </Descriptions.Item>
        <Descriptions.Item label="当前工作区组织">
          <Space size={6}>
            <ApartmentOutlined style={{ color: '#1677ff' }} />
            <Text strong>{orgName}</Text>
          </Space>
        </Descriptions.Item>
        <Descriptions.Item label="组织统一编码">
          <Text copyable style={{ fontFamily: 'monospace' }}>
            {orgCode}
          </Text>
        </Descriptions.Item>
        <Descriptions.Item label="数据授权范围" span={2}>
          {user?.roleScopes && user.roleScopes.length > 0 ? (
            <Space wrap size={[6, 6]}>
              {user.roleScopes.map((scope) => (
                <Tag
                  key={`${scope.roleCode}:${scope.dataScope}`}
                  color="cyan"
                  variant="filled"
                  style={{ padding: '2px 8px', fontSize: 12 }}
                >
                  {scope.roleName || scope.roleCode} · 数据范围:{' '}
                  {scope.dataScope}
                </Tag>
              ))}
            </Space>
          ) : (
            <Text type="secondary">按系统默认规则隔离</Text>
          )}
        </Descriptions.Item>
        <Descriptions.Item label="功能权限授权数" span={2}>
          <Text strong>{user?.permissions?.length ?? 0} 项</Text>
        </Descriptions.Item>
      </Descriptions>
      <Paragraph style={{ fontSize: 12, color: '#64748b', marginBottom: 0 }}>
        所有单据创建、流转与核销均受当前组织边界和授权策略约束；工作台仅展示
        服务端判定您有权查看的模块与数据。
      </Paragraph>
    </ProCard>
  );
}

type QuickEntry = { title: string; path: string };

/** 已授权快捷入口：与全站 access 同一权限真相，不新增第二套规则。 */
export function QuickEntriesCard() {
  const access = useAccess();
  const entries: QuickEntry[] = [
    access.canReadSEOrders
      ? { title: '海运出口订单', path: '/orders/sea-export' }
      : null,
    access.canReadFinanceFees
      ? { title: '集运费用明细', path: '/finance/fees' }
      : null,
    access.canReadFinanceBills
      ? { title: '账单管理', path: '/finance/bills' }
      : null,
    access.canReadFinanceVerifications
      ? { title: '核销管理', path: '/finance/verifications' }
      : null,
    access.canReadFinanceCommissions
      ? { title: '提成台账', path: '/finance/commissions' }
      : null,
    access.canReadPartners
      ? { title: '客户管理', path: '/partners/customers' }
      : null,
    access.canReadMasterData ? { title: '主数据', path: '/master-data' } : null,
  ].filter((entry): entry is QuickEntry => entry !== null);

  return (
    <ProCard
      title={
        <Space size={8}>
          <AppstoreOutlined style={{ color: '#1677ff' }} />
          <span>快捷入口</span>
        </Space>
      }
      headerBordered
      variant="outlined"
      data-testid="quick-entries-card"
    >
      {entries.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="当前账号暂无已授权的业务入口"
        />
      ) : (
        <Space wrap size={[8, 8]}>
          {entries.map((entry) => (
            <Button
              key={entry.path}
              size="small"
              onClick={() => history.push(entry.path)}
            >
              {entry.title}
            </Button>
          ))}
        </Space>
      )}
    </ProCard>
  );
}
