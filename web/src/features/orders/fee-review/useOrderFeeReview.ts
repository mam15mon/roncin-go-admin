import { useQuery } from '@tanstack/react-query';
import { useInitialState } from '@/app/AppProvider';
import { orderFeeServicePreviewOrderFeeReview } from '@/services/roncin/orderFeeService';

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
      const response = await orderFeeServicePreviewOrderFeeReview({
        orderId: orderId as string,
      });
      if (!response.data) throw new Error('费用审核数据为空，请刷新后重试');
      return response.data;
    },
    meta: { errorMessage: '费用审核详情加载失败' },
  });
}
