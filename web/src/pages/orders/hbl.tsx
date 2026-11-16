import { PageContainer } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Button, Result, Skeleton } from 'antd';
import { useState } from 'react';
import { useLocation, useParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { getFormDraftScope } from '@/components/layout/formDraft';
import { PageHeaderShell } from '@/components/ui';
import { history } from '@/router/history';
import HblEditor from './hbl/HblEditor';
import type { HblBundle } from './hbl/model';
import { hblUnavailableReason } from './hbl/model';
import { fetchHblBundle } from './hbl/use-hbl-data';
import {
  getOrderKindDefinition,
  getOrderKindDefinitionByBusinessType,
} from './order-kinds/registry';

export default function HblProductionPage() {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const definition = getOrderKindDefinition(location.pathname);
  const kind = definition?.kind;
  const listPath = `/orders/${kind}`;
  const { initialState } = useInitialState();
  const access = useAccess();
  const user = initialState?.currentUser;
  const scope = getFormDraftScope(user?.id, user?.currentOrganization?.id);
  const canRead = Boolean(
    definition?.transportMode === 'sea' &&
      access.canOrder(definition.businessType, 'read'),
  );
  const query = useQuery({
    queryKey: ['orders', 'hbl-production', { id, scope, kind }],
    enabled: Boolean(id && user?.id && user.currentOrganization?.id && canRead),
    queryFn: async ({ signal }) => {
      const bundle = await fetchHblBundle(
        id as string,
        {
          businessType: definition?.businessType,
          organizationId: user?.currentOrganization?.id,
        },
        signal,
      );
      if (
        bundle.order.businessType !== definition?.businessType ||
        bundle.order.organizationId !== user?.currentOrganization?.id
      )
        throw new Error('订单与当前业务类型或公司不一致。');
      return bundle;
    },
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
        breadcrumbs={[
          { label: definition?.navigationTitle ?? '海运订单', href: listPath },
        ]}
        onBack={() => history.push(listPath)}
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
              onClick={() => history.push(`/orders/${kind}/${id}`)}
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
  const definition = getOrderKindDefinitionByBusinessType(
    snapshot.order.businessType ?? 0,
  );
  const listPath = `/orders/${definition?.kind}`;
  const reason = hblUnavailableReason(snapshot);
  if (!reason) return <HblEditor initial={snapshot} scope={scope} />;
  return (
    <PageContainer title={false} breadcrumbRender={false}>
      <PageHeaderShell
        title="HBL 制作"
        subTitle={snapshot.order.orderNo}
        breadcrumbs={[
          { label: definition?.navigationTitle ?? '海运订单', href: listPath },
        ]}
        onBack={() => history.push(listPath)}
      />
      <Result
        status="info"
        title="暂不能制作 HBL"
        subTitle={reason}
        extra={
          <Button
            onClick={() =>
              history.push(`/orders/${definition?.kind}/${snapshot.order.id}`)
            }
          >
            返回订单资料
          </Button>
        }
      />
    </PageContainer>
  );
}
