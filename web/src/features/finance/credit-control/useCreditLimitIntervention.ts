import { useQuery } from '@tanstack/react-query';
import { settlementServiceGetCreditLimitControlPolicy } from '@/services/roncin/settlementService';

/**
 * 读取组织信用额度管控策略并返回是否处于「直接干预」模式。
 *
 * - 仅当策略已保存且「超额后允许选择」为 false 时返回 true；该布尔为 false 时
 *   JSON 序列化会整体省略，必须用 `!== true` 判定，禁止 `=== false` 死分支。
 * - 默认（未保存策略 / 读取失败 / 未激活 / enabled=false）返回 false：直接干预是
 *   显式管理动作，读取失败时回退仅提醒模式，不阻断任何选择；历史 catch 静默，
 *   声明 silent 避免全局 onError 重复提示。
 */
export function useCreditLimitIntervention(enabled = true): boolean {
  const query = useQuery({
    queryKey: ['settlement', 'credit-limit-policy'],
    enabled,
    meta: { silent: true },
    queryFn: async () => {
      const response = await settlementServiceGetCreditLimitControlPolicy({});
      return response.data?.allowSelectionWhenCreditExceeded !== true;
    },
  });
  // enabled=false 时不发请求，data 恒为 undefined；读取失败同样回退 false。
  return query.data ?? false;
}
