import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Space, Typography } from 'antd';
import { useRef } from 'react';
import { ShipmentType } from '@/enums.generated';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import { trimDecimal } from '@/utils/format';
import ContainerDrawer, {
  type ContainerDrawerRef,
} from '../../../components/drawers/ContainerDrawer';
import type { OrderDetailFeaturesContext } from '../../../order-kinds/types';

/** 实际箱仅维护独占箱摘要；共享箱不混入本票总量，不触发整票表单回填。 */
export function SeaActualContainers({
  context,
  onOpenShared,
}: {
  context: OrderDetailFeaturesContext;
  onOpenShared: () => void;
}) {
  const {
    order,
    orderId,
    canOrder,
    businessWritesDisabled,
    businessWriteBlockedReason,
    containerSpecOptions,
  } = context;
  const client = useQueryClient();
  const drawerRef = useRef<ContainerDrawerRef | null>(null);
  const canRead = canOrder('container.read');
  const queryKey = [
    'orders',
    'actual-containers',
    { orderId, organizationId: order.organizationId },
  ];
  const query = useQuery({
    queryKey,
    enabled: canRead && Boolean(orderId),
    queryFn: async () => {
      const response = await orderContainerServiceListContainers(
        { orderId },
        { skipErrorHandler: true },
      );
      if (
        !response ||
        response.success === false ||
        !(
          Array.isArray(response.data) ||
          (response.success === true && response.data === undefined)
        )
      )
        throw new Error('实际箱资料读取失败，请重试');
      return response.data ?? [];
    },
    meta: { silent: true },
  });
  if (!canRead)
    return (
      <Typography.Paragraph type="secondary">
        暂无实际箱查看权限，请联系管理员。
      </Typography.Paragraph>
    );
  const rows = query.data;
  const summary = rows?.reduce(
    (total, row) => ({
      packages: total.packages + (row.packageCount ?? 0),
      weight: total.weight + (row.grossWeightKg ?? 0),
      volume: total.volume + (row.volumeCbm ?? 0),
    }),
    { packages: 0, weight: 0, volume: 0 },
  );
  const supportsExclusive =
    order.shipmentType === ShipmentType.SHIPMENT_TYPE_FCL;
  const writable = !businessWritesDisabled && supportsExclusive;
  const specs = containerSpecOptions.map((option) => ({
    label: option.label,
    value: String(option.value),
  }));
  return (
    <div style={{ display: 'grid', gap: 8 }}>
      <Typography.Text type="secondary">
        普通独占箱用于本票整箱；跨订单共箱请进入共享箱工作台。
      </Typography.Text>
      {query.isError ? (
        <Alert
          type="error"
          showIcon
          title="实际箱摘要加载失败"
          action={
            <Button
              // 查询重试不写业务数据，审核锁仍允许读取。
              disabled={false}
              size="small"
              onClick={() => void query.refetch()}
            >
              重试
            </Button>
          }
        />
      ) : query.isPending ? (
        <Typography.Text type="secondary">正在读取实际箱…</Typography.Text>
      ) : summary ? (
        <Typography.Text>
          独占箱 {rows?.length} 箱 · {summary.packages} 件 ·{' '}
          {trimDecimal(summary.weight)} KGS · {trimDecimal(summary.volume)} CBM
        </Typography.Text>
      ) : null}
      {!supportsExclusive ? (
        <Typography.Text type="secondary">
          普通独占箱仅支持整箱订单；拼箱请使用跨订单共享箱，散杂货不维护集装箱。
        </Typography.Text>
      ) : null}
      <Space wrap>
        <Button
          // 只打开查看入口；箱写入能力由 Drawer 权限独立约束。
          disabled={false}
          type="primary"
          onClick={() => drawerRef.current?.open(order)}
        >
          普通实际箱
        </Button>
        <Button
          disabled={
            !order.seaMasterBill?.transportExecutionId ||
            order.shipmentType === ShipmentType.SHIPMENT_TYPE_BREAK_BULK
          }
          onClick={onOpenShared}
        >
          跨订单共享箱
        </Button>
      </Space>
      <Typography.Text type="secondary">
        共享箱单独维护，不计入以上独占箱摘要。
      </Typography.Text>
      {businessWritesDisabled && businessWriteBlockedReason ? (
        <Typography.Text type="secondary">
          {businessWriteBlockedReason}
        </Typography.Text>
      ) : null}
      <ContainerDrawer
        ref={drawerRef}
        canCreate={writable && canOrder('container.create')}
        canUpdate={writable && canOrder('container.update')}
        canRemove={writable && canOrder('container.delete')}
        containerSpecOptions={specs}
        containerSpecMap={Object.fromEntries(
          specs.map((option) => [option.value, option.label]),
        )}
        hideSharedEntry
        onChanged={() => {
          void client.invalidateQueries({ queryKey });
        }}
      />
    </div>
  );
}
