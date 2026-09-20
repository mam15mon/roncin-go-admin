import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { workbenchServiceGetWorkbenchOverview } from '@/services/roncin/workbenchService';

/** 服务端状态域前缀：工作台 Overview 查询的统一 key 前缀。 */
const OVERVIEW_QUERY_BASE = ['workbench', 'overview'] as const;

/**
 * 工作台 Overview 数据 Hook。
 *
 * 组织切换契约：organizationId 进入 queryKey，切换即换 key 重查；
 * 旧组织的在途响应归属旧 key，不会覆盖当前页面，缓存与竞态由库收敛。
 *
 * reload 返回 Promise：解析在本次刷新请求落定（成功或失败）之后，
 * 供月度申请提交等调用方「等待刷新完成再解除 loading」。
 */
export function useWorkbenchOverview(organizationId?: string) {
  const queryClient = useQueryClient();

  const { data, isPending, isError } = useQuery({
    queryKey: [...OVERVIEW_QUERY_BASE, { organizationId }],
    queryFn: async () => {
      const response = await workbenchServiceGetWorkbenchOverview();
      return response.data;
    },
    // 历史空 catch 静默：失败仅置 error 态由页面展示重试，不重复弹全局 message。
    meta: { silent: true },
  });

  const reload = useCallback(
    () => queryClient.refetchQueries({ queryKey: OVERVIEW_QUERY_BASE }),
    [queryClient],
  );

  return { loading: isPending, data, error: isError, reload };
}
