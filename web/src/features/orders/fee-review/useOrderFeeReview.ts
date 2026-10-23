import { useQuery } from '@tanstack/react-query';
import { useInitialState } from '@/app/AppProvider';
import { orderFeeServiceGetOrderFeeReview } from '@/services/roncin/orderFeeService';

export function useOrderFeeReview(orderId?: string, enabled = true) {
  const { initialState } = useInitialState();
  return useQuery({
    queryKey: [
      'order-fee-review',
      initialState?.currentUser?.currentOrganization?.id,
      orderId,
    ],
    enabled: Boolean(orderId) && enabled,
    queryFn: async () => {
      const response = await orderFeeServiceGetOrderFeeReview({
        orderId: orderId as string,
      });
      if (!response.data) throw new Error('订单审核信息为空，请刷新后重试');
      return response.data;
    },
    meta: { errorMessage: '订单审核信息加载失败' },
  });
}
