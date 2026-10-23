import { useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Drawer, Space, Spin } from 'antd';
import { FeeReviewSummary } from '@/features/orders/fee-review';
import { useOrderLockState } from '../../../use-order-lock-state';
import OrderLockControl, {
  OrderLockStatusTag,
} from '../../detail/OrderLockControl';

/** 费用工作台与订单列表共用同一人工审核锁。 */
export function OrderFeeReviewSection({
  orderId,
  onChanged,
}: {
  orderId: string;
  onChanged: () => void;
}) {
  const lock = useOrderLockState(orderId);
  const client = useQueryClient();
  return (
    <Space orientation="vertical" style={{ width: '100%', marginBottom: 12 }}>
      {lock.loading && !lock.state && <Spin />}
      {lock.error && (
        <Alert
          type="error"
          showIcon
          title="订单审核状态加载失败"
          action={<Button onClick={() => void lock.refresh()}>重试</Button>}
        />
      )}
      <Space wrap>
        <OrderLockStatusTag
          state={lock.state}
          loading={lock.loading}
          error={lock.error}
        />
        <OrderLockControl
          orderId={orderId}
          state={lock.state}
          loading={lock.loading}
          error={lock.error}
          onRetry={lock.refresh}
          onSynchronize={async () => {
            await Promise.all([
              lock.refresh(),
              client.invalidateQueries({ queryKey: ['orders'] }),
              client.invalidateQueries({ queryKey: ['order-fee-review'] }),
              client.invalidateQueries({ queryKey: ['finance'] }),
            ]);
            onChanged();
          }}
        />
      </Space>
      {lock.state?.feeSummary && (
        <FeeReviewSummary summary={lock.state.feeSummary} />
      )}
    </Space>
  );
}

export function OrderFeeReviewDrawer({
  orderId,
  onClose,
  onChanged,
}: {
  orderId?: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  return (
    <Drawer
      open={Boolean(orderId)}
      title="订单审核与锁定"
      size="large"
      destroyOnHidden
      onClose={onClose}
    >
      {orderId && (
        <OrderFeeReviewSection
          key={orderId}
          orderId={orderId}
          onChanged={onChanged}
        />
      )}
    </Drawer>
  );
}
