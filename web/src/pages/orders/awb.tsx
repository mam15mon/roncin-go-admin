import { PageContainer } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Button, Result, Skeleton } from 'antd';
import { useMemo, useState } from 'react';
import { useLocation, useParams } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { PageHeaderShell } from '@/components/ui';
import { history } from '@/router/history';
import { generateUUID } from '@/utils/uuid';
import AwbEditor from './awb/AwbEditor';
import type { AwbBundle, AwbIdentity } from './awb/model';
import { fetchAwbBundle } from './awb/use-awb-data';
import { getOrderKindDefinition } from './order-kinds/registry';
import type { OrderKindDefinition } from './order-kinds/types';

export default function AwbProductionPage() {
  const {
    id,
    masterId,
    target = 'MAWB',
  } = useParams<{ id: string; masterId: string; target: string }>();
  const location = useLocation();
  const definition = getOrderKindDefinition(location.pathname);
  const access = useAccess();
  const { initialState } = useInitialState();
  const user = initialState?.currentUser;
  const organizationId = user?.currentOrganization?.id;
  const canRead = Boolean(
    definition?.transportMode === 'air' &&
      access.canOrder(definition.businessType, 'read'),
  );
  const validTarget = target === 'MAWB' || target === 'HAWB';
  const identity: AwbIdentity = {
    organizationId: organizationId ?? '',
    businessType: definition?.businessType ?? 0,
    masterId,
    orderId: id,
    target: target === 'HAWB' ? 'HAWB' : 'MAWB',
  };
  const scope = `${user?.id}:${organizationId}:${definition?.kind}:${id}:${masterId}:${target}`;
  const owner = useMemo(() => generateUUID(), [scope]);
  const query = useQuery({
    queryKey: ['air-orders', 'awb-production', { scope, owner }],
    enabled: Boolean(
      canRead && validTarget && user?.id && organizationId && (id || masterId),
    ),
    queryFn: ({ signal }) => fetchAwbBundle(identity, signal),
    meta: { silent: true },
  });
  if (!canRead || !validTarget)
    return (
      <PageContainer>
        <Result status="403" title="无权查看该航空运单制作" />
      </PageContainer>
    );
  if (query.data && definition)
    return (
      <LoadedAwb
        key={scope}
        initial={query.data}
        identity={identity}
        definition={definition}
      />
    );
  const listPath = `/orders/${definition?.kind}`;
  return (
    <PageContainer title={false} breadcrumbRender={false}>
      <PageHeaderShell
        title="航空运单制作"
        breadcrumbs={[
          { label: definition?.navigationTitle ?? '航空订单', href: listPath },
        ]}
        onBack={() => history.push(listPath)}
      />
      {query.isPending ? (
        <Skeleton active />
      ) : (
        <Result
          status="error"
          title="航空运单制作资料加载失败"
          subTitle={query.error?.message || '请重试或返回订单检查单证模式。'}
          extra={<Button onClick={() => void query.refetch()}>重试</Button>}
        />
      )}
    </PageContainer>
  );
}

/** 初始化后由编辑器持有保存快照，后台缓存重查不替换用户输入。 */
function LoadedAwb({
  initial,
  identity,
  definition,
}: {
  initial: AwbBundle;
  identity: AwbIdentity;
  definition: OrderKindDefinition;
}) {
  const [snapshot] = useState(() => structuredClone(initial));
  const editorIdentity =
    identity.target === 'MAWB'
      ? { ...identity, masterId: snapshot.master.id, orderId: undefined }
      : identity;
  const editorSnapshot =
    identity.target === 'MAWB'
      ? { master: snapshot.master, blockedReasons: snapshot.blockedReasons }
      : snapshot;
  return (
    <AwbEditor
      initial={editorSnapshot}
      identity={editorIdentity}
      definition={definition}
    />
  );
}
