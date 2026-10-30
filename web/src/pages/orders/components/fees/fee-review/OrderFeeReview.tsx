import { useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Drawer, Space, Spin } from 'antd';
import {
  FeeReviewSummary,
  useOrderFeeReview,
} from '@/features/orders/fee-review';
import { useOrderLockState } from '../../../use-order-lock-state';
import OrderLockControl, {
  OrderLockStatusTag,
} from '../../detail/OrderLockControl';
import OrderLockFeeDetails from '../../detail/OrderLockFeeDetails';

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

/** 抽屉直接展示全量只读明细；费用工作台继续使用自身的可编辑表格。 */
function OrderFeeReviewDetails({ orderId }: { orderId: string }) {
  const review = useOrderFeeReview(orderId);
  if (review.isFetching) {
    return (
      <div role="status" style={{ padding: 24, textAlign: 'center' }}>
        <Spin /> 正在加载全部费用明细
      </div>
    );
  }
  if (review.isError) {
    return (
      <Alert
        type="error"
        showIcon
        title="费用明细加载失败，请重试"
        action={
          <Button onClick={() => void review.refetch()}>重试费用明细</Button>
        }
      />
    );
  }
  return <OrderLockFeeDetails fees={review.data?.fees || []} />;
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
      title="订单审核"
      size="min(1280px, 100vw)"
      destroyOnHidden
      onClose={onClose}
    >
      {orderId && (
        <div key={orderId}>
          <OrderFeeReviewSection orderId={orderId} onChanged={onChanged} />
          <OrderFeeReviewDetails orderId={orderId} />
        </div>
      )}
    </Drawer>
  );
}
