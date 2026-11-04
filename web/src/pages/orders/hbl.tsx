import { PageContainer } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Button, Result, Skeleton } from 'antd';
import { useState } from 'react';
import { useParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { getFormDraftScope } from '@/components/layout/formDraft';
import { PageHeaderShell } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import { history } from '@/router/history';
import HblEditor from './hbl/HblEditor';
import type { HblBundle } from './hbl/model';
import { hblUnavailableReason } from './hbl/model';
import { fetchHblBundle } from './hbl/use-hbl-data';

export default function HblProductionPage() {
  const { id } = useParams<{ id: string }>();
  const { initialState } = useInitialState();
  const access = useAccess();
  const user = initialState?.currentUser;
  const scope = getFormDraftScope(user?.id, user?.currentOrganization?.id);
  const canRead = access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'read');
  const query = useQuery({
    queryKey: ['orders', 'hbl-production', { id, scope }],
    enabled: Boolean(id && user?.id && user.currentOrganization?.id && canRead),
    queryFn: ({ signal }) => fetchHblBundle(id as string, signal),
    meta: { silent: true },
  });
  if (!canRead || !id)
    return (
      <PageContainer>
        <Result status="403" title="无权查看 HBL 制作" />
      </PageContainer>
    );
  if (query.data && scope)
    return (
      <LoadedHbl key={`${scope}:${id}`} initial={query.data} scope={scope} />
    );
  return (
    <PageContainer title={false} breadcrumbRender={false}>
      <PageHeaderShell
        title="HBL 制作"
        breadcrumbs={[{ label: '海运出口', href: '/orders/sea-export' }]}
        onBack={() => history.push('/orders/sea-export')}
      />
      {query.isPending ? (
        <Skeleton active />
      ) : (
        <Result
          status={query.isError ? 'error' : 'info'}
          title={query.isError ? 'HBL 制作资料加载失败' : '暂不能制作 HBL'}
          subTitle="请重试，或返回订单检查资料。"
          extra={[
            query.isError && (
              <Button key="retry" onClick={() => void query.refetch()}>
                重试
              </Button>
            ),
            <Button
              key="order"
              onClick={() => history.push(`/orders/sea-export/${id}`)}
            >
              返回订单资料
            </Button>,
          ]}
        />
      )}
    </PageContainer>
  );
}

/** 首次加载后由编辑器独占完整快照，后台失效重查不能卸载用户正在编辑的表单。 */
function LoadedHbl({ initial, scope }: { initial: HblBundle; scope: string }) {
  const [snapshot] = useState(initial);
  const reason = hblUnavailableReason(snapshot);
  if (!reason) return <HblEditor initial={snapshot} scope={scope} />;
  return (
    <PageContainer title={false} breadcrumbRender={false}>
      <PageHeaderShell
        title="HBL 制作"
        subTitle={snapshot.order.orderNo}
        breadcrumbs={[{ label: '海运出口', href: '/orders/sea-export' }]}
        onBack={() => history.push('/orders/sea-export')}
      />
      <Result
        status="info"
        title="暂不能制作 HBL"
        subTitle={reason}
        extra={
          <Button
            onClick={() =>
              history.push(`/orders/sea-export/${snapshot.order.id}`)
            }
          >
            返回订单资料
          </Button>
        }
      />
    </PageContainer>
  );
}
