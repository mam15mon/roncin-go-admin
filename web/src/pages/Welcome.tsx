import { ApartmentOutlined } from '@ant-design/icons';
import { PageContainer, ProCard } from '@ant-design/pro-components';
import { useModel } from '@umijs/max';
import { Alert, Avatar, Button, Skeleton, Space, Tag, Typography } from 'antd';
import React, { useState } from 'react';
import CommissionSummaryCard from './workbench/CommissionSummaryCard';
import FinanceSummaryCard from './workbench/FinanceSummaryCard';
import MyCommissionDrawer from './workbench/MyCommissionDrawer';
import MyReceivablesDrawer from './workbench/MyReceivablesDrawer';
import { useWorkbenchOverview } from './workbench/useWorkbenchOverview';
import {
  AccountBoundaryCard,
  QuickEntriesCard,
  RecentOrdersCard,
  TodosCard,
} from './workbench/WorkbenchSideCards';

const { Text, Title } = Typography;

const GRID_STYLE: React.CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fit, minmax(360px, 1fr))',
  gap: 16,
};

export default function Welcome() {
  const { initialState } = useModel('@@initialState');
  const user = initialState?.currentUser;
  const displayName = user?.displayName || user?.username || '用户';
  const orgName = user?.currentOrganization?.name || '默认组织';

  // 请求键：当前工作区组织 ID。组织切换时 hook 重新拉取并丢弃旧组织在途响应。
  const { loading, data, error, reload } = useWorkbenchOverview(
    user?.currentOrganization?.id,
  );

  const [commissionDrawerOpen, setCommissionDrawerOpen] = useState(false);
  const [receivablesDrawerOpen, setReceivablesDrawerOpen] = useState(false);

  // 提成门禁：optional bool 的 false 可能被省略为 undefined，一律 !== true 判假；
  // 门禁为假时从首次渲染（含加载中）起就不出现提成模块 DOM。
  const showCommission = data?.hasCommissionEligibility === true;
  const finance = data?.finance;
  const hasTodos =
    (data?.todos?.draftFeeCount ?? 0) > 0 ||
    (data?.todos?.openAbnormalCount ?? 0) > 0;

  return (
    <PageContainer
      title={
        <Space size={12} align="center">
          <Avatar
            size={44}
            src={user?.avatarUrl}
            style={{
              backgroundColor: '#1677ff',
              fontSize: 18,
              fontWeight: 600,
            }}
          >
            {displayName.charAt(0).toUpperCase()}
          </Avatar>
          <div>
            <Title level={4} style={{ margin: 0 }}>
              您好，{displayName}
            </Title>
            <Text type="secondary" style={{ fontSize: 13 }}>
              欢迎登录 Roncin 国际货代协同管理平台
            </Text>
          </div>
        </Space>
      }
      extra={
        <Space size={8}>
          <Tag
            icon={<ApartmentOutlined />}
            color="blue"
            style={{ padding: '4px 10px', fontSize: 12 }}
          >
            当前组织：{orgName}
          </Tag>
        </Space>
      }
    >
      <Space orientation="vertical" size={16} style={{ width: '100%' }}>
        {error ? (
          <Alert
            type="error"
            showIcon
            message="工作台数据加载失败"
            description="稍后重试；失败不会展示任何组织的旧数据。"
            action={
              <Button size="small" danger onClick={reload}>
                重试
              </Button>
            }
          />
        ) : null}

        {loading ? (
          <ProCard
            headerBordered
            variant="outlined"
            data-testid="workbench-loading"
          >
            <Skeleton active paragraph={{ rows: 5 }} />
          </ProCard>
        ) : data ? (
          <>
            {showCommission ? (
              <CommissionSummaryCard
                data={data}
                onOpenCommissions={() => setCommissionDrawerOpen(true)}
                onOpenReceivables={() => setReceivablesDrawerOpen(true)}
              />
            ) : null}

            {finance ? <FinanceSummaryCard finance={finance} /> : null}

            {(data.recentOrders?.length ?? 0) > 0 || hasTodos ? (
              <div style={GRID_STYLE}>
                <RecentOrdersCard orders={data.recentOrders} />
                <TodosCard todos={data.todos} />
              </div>
            ) : null}
          </>
        ) : null}

        <div style={GRID_STYLE}>
          <AccountBoundaryCard user={user} />
          <QuickEntriesCard />
        </div>
      </Space>

      {commissionDrawerOpen ? (
        <MyCommissionDrawer
          open
          baseCurrency={data?.baseCurrency}
          onClose={() => setCommissionDrawerOpen(false)}
        />
      ) : null}
      {receivablesDrawerOpen ? (
        <MyReceivablesDrawer
          open
          onClose={() => setReceivablesDrawerOpen(false)}
        />
      ) : null}
    </PageContainer>
  );
}
